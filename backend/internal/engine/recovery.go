package engine

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gorocksdb/internal/logger"
	"gorocksdb/internal/memtable"
	"gorocksdb/internal/wal"
)

func (db *DB) recover() error {
	logNum := db.vs.LogNumber()
	logs, err := listWAL(db.opts.Dir)
	if err != nil {
		return err
	}
	var replay []uint64
	for _, n := range logs {
		if n >= logNum {
			replay = append(replay, n)
		}
	}
	sort.Slice(replay, func(i, j int) bool { return replay[i] < replay[j] })

	if len(replay) == 0 {
		return db.newMemLocked()
	}

	// last WAL becomes the live memtable
	for i, n := range replay {
		path := WALPath(db.opts.Dir, n)
		isLast := i == len(replay)-1
		m := memtable.New(n, n)
		truncated, err := wal.Replay(path, func(rec []byte) error {
			seq, items, err := wal.DecodeBatch(rec)
			if err != nil {
				return err
			}
			for j, it := range items {
				m.Add(seq+uint64(j), it.Type, it.Key, it.Value)
			}
			if last := seq + uint64(len(items)) - 1; last > db.seq.Load() {
				db.seq.Store(last)
				db.vs.SetLastSequence(last)
			}
			return nil
		})
		if err != nil {
			logger.L().Error("wal replay aborted: logical corruption detected, refusing to start in half-replayed state", "file", path, "err", err)
			return err
		}
		if truncated {
			logger.L().Warn("wal tail truncated", "file", path)
		}
		if isLast {
			w, err := reopenWAL(path, n)
			if err != nil {
				return err
			}
			db.mem = m
			db.log = w
			db.nextMemID = n
		} else {
			m.MarkImmutable()
			db.imm = append(db.imm, m)
		}
	}
	if db.mem == nil {
		return db.newMemLocked()
	}
	db.kickBG()
	return nil
}

func reopenWAL(path string, number uint64) (*wal.Writer, error) {
	// append to existing file via a fresh writer that opens APPEND is safer:
	// we create a new writer type by opening the file for append.
	return wal.OpenAppend(path, number)
}

func listWAL(dir string) ([]uint64, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []uint64
	for _, e := range ents {
		name := e.Name()
		if !strings.HasSuffix(name, ".log") {
			continue
		}
		n, err := strconv.ParseUint(strings.TrimSuffix(name, ".log"), 10, 64)
		if err != nil {
			continue
		}
		out = append(out, n)
	}
	return out, nil
}

func listSST(dir string) ([]uint64, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []uint64
	for _, e := range ents {
		name := e.Name()
		if !strings.HasSuffix(name, ".sst") {
			continue
		}
		n, err := strconv.ParseUint(strings.TrimSuffix(name, ".sst"), 10, 64)
		if err != nil {
			continue
		}
		out = append(out, n)
	}
	_ = filepath.Separator
	return out, nil
}
