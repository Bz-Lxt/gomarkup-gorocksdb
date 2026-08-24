package engine

import (
	"os"
	"time"

	"gorocksdb/internal/encoding"
	"gorocksdb/internal/logger"
	"gorocksdb/internal/memtable"
	"gorocksdb/internal/sstable"
	"gorocksdb/internal/version"
)

func (db *DB) bgFlushLoop() {
	defer db.bgWg.Done()
	for {
		select {
		case <-db.stopCh:
			db.flushAll()
			return
		case <-db.bgCh:
			db.flushOnce()
		case <-time.After(200 * time.Millisecond):
			db.flushOnce()
		}
	}
}

func (db *DB) flushAll() {
	for {
		db.mu.Lock()
		if len(db.imm) == 0 {
			db.mu.Unlock()
			return
		}
		db.mu.Unlock()
		db.flushOnce()
	}
}

func (db *DB) Flush() error {
	db.mu.Lock()
	if db.mem != nil && db.mem.Len() > 0 {
		old := db.mem
		old.MarkImmutable()
		db.imm = append(db.imm, old)
		oldLog := db.log
		if err := db.newMemLocked(); err != nil {
			db.mu.Unlock()
			return err
		}
		if oldLog != nil {
			_ = oldLog.Unref()
		}
		db.bus.Publish("memtable.rotate", map[string]any{
			"old_id": old.ID(), "forced": true, "bytes": old.ApproximateMemory(),
		})
	}
	db.mu.Unlock()
	db.flushOnce()
	return nil
}

func (db *DB) flushOnce() {
	db.mu.Lock()
	if db.flushing || len(db.imm) == 0 {
		db.mu.Unlock()
		return
	}
	m := db.imm[0]
	db.flushing = true
	db.mu.Unlock()

	if err := db.flushMem(m); err != nil {
		logger.L().Error("flush failed", "err", err, "mem", m.ID())
	}

	db.mu.Lock()
	if len(db.imm) > 0 && db.imm[0] == m {
		db.imm = db.imm[1:]
	}
	db.flushing = false
	db.mu.Unlock()
	db.kickBG()
}

func (db *DB) flushMem(m *memtable.MemTable) error {
	if m.Len() == 0 {
		_ = os.Remove(WALPath(db.opts.Dir, m.LogNum()))
		return nil
	}
	start := time.Now()
	num := db.vs.NewFileNumber()
	db.bus.Publish("flush.start", map[string]any{
		"mem_id": m.ID(), "file": num, "entries": m.Len(),
	})
	w, err := sstable.Create(db.opts.Dir, num, db.opts.Profile, int(m.Len()))
	if err != nil {
		return err
	}
	it := m.NewIterator()
	it.SeekToFirst()
	for it.Valid() {
		if err := w.Add(it.Key(), it.Value()); err != nil {
			w.Abort()
			return err
		}
		it.Next()
	}
	meta, err := w.Finish()
	if err != nil {
		return err
	}
	meta.Level = 0
	edit := &version.VersionEdit{}
	edit.AddFile(meta)
	edit.SetLogNumber(db.currentLogNumber())
	edit.SetLastSeq(db.seq.Load())
	if _, _, err := db.vs.Apply(edit); err != nil {
		return err
	}
	_ = os.Remove(WALPath(db.opts.Dir, m.LogNum()))
	db.Met.Flushes.Add(1)
	ms := time.Since(start).Milliseconds()
	db.bus.Publish("flush.done", map[string]any{
		"mem_id": m.ID(), "file": num, "size": meta.Size,
		"entries": meta.Entries, "duration_ms": ms,
		"min_key": meta.SmallestUser(), "max_key": meta.LargestUser(),
	})
	db.bus.Publish("sstable.created", map[string]any{
		"number": num, "level": 0, "size": meta.Size, "entries": meta.Entries,
		"min_key": meta.SmallestUser(), "max_key": meta.LargestUser(),
	})
	_ = encoding.MaxSequence
	return nil
}

func (db *DB) currentLogNumber() uint64 {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.log == nil {
		return 0
	}
	return db.log.Number()
}
