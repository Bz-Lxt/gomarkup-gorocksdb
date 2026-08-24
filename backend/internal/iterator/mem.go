package iterator

import "gorocksdb/internal/skiplist"

type MemIter struct {
	inner *skiplist.Iterator
}

func WrapMem(it *skiplist.Iterator) *MemIter {
	return &MemIter{inner: it}
}

func (m *MemIter) SeekToFirst() { m.inner.SeekToFirst() }
func (m *MemIter) Seek(key []byte) { m.inner.Seek(key) }
func (m *MemIter) Next()           { m.inner.Next() }
func (m *MemIter) Valid() bool     { return m.inner.Valid() }
func (m *MemIter) Key() []byte     { return m.inner.Key() }
func (m *MemIter) Value() []byte   { return m.inner.Value() }
func (m *MemIter) Close()          {}
