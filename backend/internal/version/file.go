package version

import (
	"gorocksdb/internal/encoding"
	"gorocksdb/internal/sstable"
)

type FileMeta = sstable.FileMeta

func cloneMeta(m *FileMeta) *FileMeta {
	if m == nil {
		return nil
	}
	cp := *m
	cp.Smallest = append([]byte(nil), m.Smallest...)
	cp.Largest = append([]byte(nil), m.Largest...)
	return &cp
}

func filesOverlapUser(files []*FileMeta, uk []byte) []*FileMeta {
	var out []*FileMeta
	for _, f := range files {
		if encoding.CompareUser(encoding.UserKey(f.Smallest), uk) <= 0 &&
			encoding.CompareUser(encoding.UserKey(f.Largest), uk) >= 0 {
			out = append(out, f)
		}
	}
	return out
}
