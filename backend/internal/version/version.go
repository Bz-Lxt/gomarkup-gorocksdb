package version

import (
	"sort"
	"sync/atomic"

	"gorocksdb/internal/config"
	"gorocksdb/internal/encoding"
)

type Version struct {
	Files [config.NumLevels][]*FileMeta
	refs  atomic.Int32
}

func NewVersion() *Version {
	v := &Version{}
	v.refs.Store(1)
	return v
}

func (v *Version) Ref()   { v.refs.Add(1) }
func (v *Version) Unref() int32 { return v.refs.Add(-1) }
func (v *Version) Refs() int32  { return v.refs.Load() }

func (v *Version) Clone() *Version {
	nv := NewVersion()
	for i := 0; i < config.NumLevels; i++ {
		nv.Files[i] = make([]*FileMeta, len(v.Files[i]))
		for j, f := range v.Files[i] {
			nv.Files[i][j] = cloneMeta(f)
		}
	}
	return nv
}

func (v *Version) NumFiles(level int) int {
	if level < 0 || level >= config.NumLevels {
		return 0
	}
	return len(v.Files[level])
}

func (v *Version) Bytes(level int) int64 {
	var n int64
	for _, f := range v.Files[level] {
		n += int64(f.Size)
	}
	return n
}

func (v *Version) AllFiles() []*FileMeta {
	var out []*FileMeta
	for i := 0; i < config.NumLevels; i++ {
		out = append(out, v.Files[i]...)
	}
	return out
}

func (v *Version) GetCandidates(userKey []byte) []*FileMeta {
	var out []*FileMeta
	// L0 newest first (higher file number typically newer)
	l0 := append([]*FileMeta(nil), v.Files[0]...)
	sort.Slice(l0, func(i, j int) bool { return l0[i].Number > l0[j].Number })
	for _, f := range l0 {
		if f.OverlapsUser(userKey, userKey) {
			out = append(out, f)
		}
	}
	for lv := 1; lv < config.NumLevels; lv++ {
		files := v.Files[lv]
		if len(files) == 0 {
			continue
		}
		i := sort.Search(len(files), func(i int) bool {
			return encoding.CompareUser(encoding.UserKey(files[i].Largest), userKey) >= 0
		})
		if i < len(files) && files[i].OverlapsUser(userKey, userKey) {
			out = append(out, files[i])
		}
	}
	return out
}

func (v *Version) Overlapping(level int, smallest, largest []byte) []*FileMeta {
	var out []*FileMeta
	ukS := encoding.UserKey(smallest)
	ukL := encoding.UserKey(largest)
	for _, f := range v.Files[level] {
		if f.OverlapsUser(ukS, ukL) {
			out = append(out, f)
		}
	}
	return out
}

func applyEdit(base *Version, e *VersionEdit) *Version {
	nv := base.Clone()
	deleted := map[uint64]struct{}{}
	for _, d := range e.Deleted {
		deleted[d.Number] = struct{}{}
	}
	for lv := 0; lv < config.NumLevels; lv++ {
		kept := nv.Files[lv][:0]
		for _, f := range nv.Files[lv] {
			if _, ok := deleted[f.Number]; !ok {
				kept = append(kept, f)
			}
		}
		nv.Files[lv] = kept
	}
	for _, f := range e.Added {
		m := cloneMeta(f)
		lv := m.Level
		if lv < 0 || lv >= config.NumLevels {
			continue
		}
		nv.Files[lv] = append(nv.Files[lv], m)
		if lv > 0 {
			sort.Slice(nv.Files[lv], func(i, j int) bool {
				return encoding.CompareInternal(nv.Files[lv][i].Smallest, nv.Files[lv][j].Smallest) < 0
			})
		} else {
			sort.Slice(nv.Files[lv], func(i, j int) bool {
				return nv.Files[lv][i].Number < nv.Files[lv][j].Number
			})
		}
	}
	return nv
}
