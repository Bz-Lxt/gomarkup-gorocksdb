package engine

import (
	"gorocksdb/internal/config"
	"gorocksdb/internal/sstable"
)

type FileView struct {
	Number   uint64 `json:"number"`
	Level    int    `json:"level"`
	Size     uint64 `json:"size"`
	Entries  uint64 `json:"entries"`
	MinKey   string `json:"min_key"`
	MaxKey   string `json:"max_key"`
}

type LevelView struct {
	Level int        `json:"level"`
	Files []FileView `json:"files"`
	Bytes int64      `json:"bytes"`
	Limit int64      `json:"limit"`
}

type LSMState struct {
	Profile          string      `json:"profile"`
	MemBytes         int64       `json:"mem_bytes"`
	MemLimit         int64       `json:"mem_limit"`
	MemEntries       int64       `json:"mem_entries"`
	MemRatio         float64     `json:"mem_ratio"`
	Immutable        []ImmView   `json:"immutable"`
	Levels           []LevelView `json:"levels"`
	LastSequence     uint64      `json:"last_sequence"`
	WriteStall       bool        `json:"write_stall"`
	Compacting       bool        `json:"compacting"`
}

type ImmView struct {
	ID      uint64 `json:"id"`
	Bytes   int64  `json:"bytes"`
	Entries int64  `json:"entries"`
}

func fileView(f *sstable.FileMeta) FileView {
	return FileView{
		Number:  f.Number,
		Level:   f.Level,
		Size:    f.Size,
		Entries: f.Entries,
		MinKey:  f.SmallestUser(),
		MaxKey:  f.LargestUser(),
	}
}

func (db *DB) SnapshotState() LSMState {
	db.mu.Lock()
	memBytes := db.mem.ApproximateMemory()
	memEnt := db.mem.Len()
	imm := make([]ImmView, 0, len(db.imm))
	for _, m := range db.imm {
		imm = append(imm, ImmView{ID: m.ID(), Bytes: m.ApproximateMemory(), Entries: m.Len()})
	}
	stall := db.stalling
	comp := db.compacting
	prof := db.opts.Profile
	db.mu.Unlock()

	v := db.vs.Current()
	v.Ref()
	defer v.Unref()

	levels := make([]LevelView, 0, config.NumLevels)
	for i := 0; i < config.NumLevels; i++ {
		lv := LevelView{Level: i, Limit: prof.MaxBytesForLevel(i), Bytes: v.Bytes(i)}
		for _, f := range v.Files[i] {
			lv.Files = append(lv.Files, fileView(f))
		}
		if lv.Files == nil {
			lv.Files = []FileView{}
		}
		levels = append(levels, lv)
	}
	ratio := 0.0
	if prof.WriteBufferSize > 0 {
		ratio = float64(memBytes) / float64(prof.WriteBufferSize)
	}
	return LSMState{
		Profile:      prof.Name,
		MemBytes:     memBytes,
		MemLimit:     prof.WriteBufferSize,
		MemEntries:   memEnt,
		MemRatio:     ratio,
		Immutable:    imm,
		Levels:       levels,
		LastSequence: db.seq.Load(),
		WriteStall:   stall,
		Compacting:   comp,
	}
}
