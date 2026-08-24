package engine

import (
	"os"
	"time"

	"gorocksdb/internal/compaction"
	"gorocksdb/internal/config"
	"gorocksdb/internal/encoding"
	"gorocksdb/internal/iterator"
	"gorocksdb/internal/logger"
	"gorocksdb/internal/sstable"
	"gorocksdb/internal/version"
)

func (db *DB) bgCompactLoop() {
	defer db.bgWg.Done()
	t := time.NewTicker(300 * time.Millisecond)
	defer t.Stop()
	for {
		select {
		case <-db.stopCh:
			return
		case <-t.C:
			db.compactOnce()
		case <-db.bgCh:
			db.compactOnce()
		}
	}
}

func (db *DB) Compact() error {
	for i := 0; i < 8; i++ {
		if !db.compactOnce() {
			break
		}
	}
	return nil
}

func (db *DB) compactOnce() bool {
	db.mu.Lock()
	if db.compacting {
		db.mu.Unlock()
		return false
	}
	v := db.vs.Current()
	job := compaction.Pick(v, db.opts.Profile)
	if job == nil || len(job.Inputs) == 0 {
		db.mu.Unlock()
		return false
	}
	db.compacting = true
	db.mu.Unlock()

	defer func() {
		db.mu.Lock()
		db.compacting = false
		db.mu.Unlock()
	}()

	if err := db.runCompaction(job); err != nil {
		logger.L().Error("compaction failed", "err", err, "level", job.Level)
		return false
	}
	return true
}

func (db *DB) runCompaction(job *compaction.Job) error {
	start := time.Now()
	inputs := uniqueFiles(job.Inputs)
	inNums := make([]uint64, 0, len(inputs))
	var bytesRead int64
	for _, f := range inputs {
		inNums = append(inNums, f.Number)
		bytesRead += int64(f.Size)
	}
	db.bus.Publish("compaction.start", map[string]any{
		"level": job.Level, "output_level": job.OutputLevel,
		"input_files": inNums, "score": job.Score,
	})

	var children []iterator.Iterator
	var opened []*sstable.Reader
	for _, f := range inputs {
		r, err := db.reader(f)
		if err != nil {
			return err
		}
		it, err := r.NewIterator()
		if err != nil {
			return err
		}
		children = append(children, iterator.WrapSST(it))
		opened = opened
	}
	_ = opened
	merge := iterator.NewMerging(children)
	defer merge.Close()

	p := db.opts.Profile
	bottom := job.OutputLevel == config.NumLevels-1 || db.vs.Current().NumFiles(job.OutputLevel+1) == 0 && job.OutputLevel >= 1
	// tombstones droppable when compacting into the last occupied level below
	dropTomb := job.OutputLevel >= config.NumLevels-1
	if !dropTomb {
		higherEmpty := true
		for lv := job.OutputLevel + 1; lv < config.NumLevels; lv++ {
			if db.vs.Current().NumFiles(lv) > 0 {
				higherEmpty = false
				break
			}
		}
		dropTomb = higherEmpty
	}
	_ = bottom

	var stats compaction.Stats
	stats.Level = job.Level
	stats.OutputLevel = job.OutputLevel
	stats.InputFiles = inNums
	stats.BytesRead = bytesRead

	var writer *sstable.Writer
	var outputs []*sstable.FileMeta
	finishWriter := func() error {
		if writer == nil {
			return nil
		}
		meta, err := writer.Finish()
		if err != nil {
			return err
		}
		meta.Level = job.OutputLevel
		outputs = append(outputs, meta)
		stats.OutputFiles = append(stats.OutputFiles, meta.Number)
		stats.BytesWritten += int64(meta.Size)
		db.bus.Publish("sstable.created", map[string]any{
			"number": meta.Number, "level": meta.Level, "size": meta.Size,
			"entries": meta.Entries, "min_key": meta.SmallestUser(), "max_key": meta.LargestUser(),
		})
		writer = nil
		return nil
	}

	merge.SeekToFirst()
	var lastUser []byte
	for merge.Valid() {
		ikey := append([]byte(nil), merge.Key()...)
		ival := append([]byte(nil), merge.Value()...)
		uk, seq, typ, ok := encoding.SplitInternalKey(ikey)
		if !ok {
			merge.Next()
			continue
		}
		stats.KeysRead++
		if stats.KeysRead == 1 || stats.KeysRead%64 == 0 {
			db.bus.Publish("compaction.progress", map[string]any{
				"keys_read": stats.KeysRead, "keys_written": stats.KeysWritten,
				"dropped_versions": stats.DroppedVersions, "dropped_tombs": stats.DroppedTombs,
			})
		}

		if lastUser != nil && encoding.CompareUser(uk, lastUser) == 0 {
			stats.DroppedVersions++
			db.Met.DroppedVersions.Add(1)
			merge.Next()
			continue
		}
		lastUser = append(lastUser[:0], uk...)

		if typ == encoding.TypeDeletion && dropTomb {
			stats.DroppedTombs++
			db.Met.DroppedTombs.Add(1)
			merge.Next()
			continue
		}
		_ = seq

		if writer == nil {
			num := db.vs.NewFileNumber()
			var err error
			writer, err = sstable.Create(db.opts.Dir, num, p, 1024)
			if err != nil {
				return err
			}
		}
		if err := writer.Add(ikey, ival); err != nil {
			writer.Abort()
			return err
		}
		stats.KeysWritten++
		if int64(writerSize(writer)) >= p.TargetFileSize {
			if err := finishWriter(); err != nil {
				return err
			}
		}
		merge.Next()
	}
	if err := finishWriter(); err != nil {
		return err
	}

	edit := &version.VersionEdit{}
	edit.SetLastSeq(db.seq.Load())
	for _, f := range outputs {
		edit.AddFile(f)
	}
	for _, f := range inputs {
		edit.DeleteFile(f.Level, f.Number)
	}
	_, obsolete, err := db.vs.Apply(edit)
	if err != nil {
		return err
	}
	for _, f := range obsolete {
		db.dropFile(f.Number)
		db.bus.Publish("sstable.deleted", map[string]any{"number": f.Number, "level": f.Level})
	}
	for _, f := range inputs {
		db.dropFile(f.Number)
		db.bus.Publish("sstable.deleted", map[string]any{"number": f.Number, "level": f.Level})
	}

	stats.DurationMS = time.Since(start).Milliseconds()
	db.Met.Compactions.Add(1)
	db.bus.Publish("compaction.done", map[string]any{
		"level": stats.Level, "output_level": stats.OutputLevel,
		"input_files": stats.InputFiles, "output_files": stats.OutputFiles,
		"keys_read": stats.KeysRead, "keys_written": stats.KeysWritten,
		"dropped_versions": stats.DroppedVersions, "dropped_tombs": stats.DroppedTombs,
		"bytes_read": stats.BytesRead, "bytes_written": stats.BytesWritten,
		"duration_ms": stats.DurationMS,
	})
	return nil
}

func writerSize(w *sstable.Writer) int {
	// approximate via unexported fields is not possible; use Entries via reflection-free hook
	return int(w.ApproxSize())
}

func uniqueFiles(in []*version.FileMeta) []*version.FileMeta {
	seen := map[uint64]struct{}{}
	var out []*version.FileMeta
	for _, f := range in {
		if f == nil {
			continue
		}
		if _, ok := seen[f.Number]; ok {
			continue
		}
		seen[f.Number] = struct{}{}
		out = append(out, f)
	}
	return out
}

func (db *DB) dropFile(num uint64) {
	if v, ok := db.readers.LoadAndDelete(num); ok {
		v.(*sstable.Reader).Close()
	}
	_ = os.Remove(sstable.SSTPath(db.opts.Dir, num))
}
