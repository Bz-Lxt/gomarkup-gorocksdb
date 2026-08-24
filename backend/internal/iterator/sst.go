package iterator

import "gorocksdb/internal/sstable"

type SSTIter struct {
	inner *sstable.Iterator
}

func WrapSST(it *sstable.Iterator) *SSTIter {
	return &SSTIter{inner: it}
}

func (s *SSTIter) SeekToFirst() { s.inner.SeekToFirst() }
func (s *SSTIter) Seek(key []byte) { s.inner.Seek(key) }
func (s *SSTIter) Next()           { s.inner.Next() }
func (s *SSTIter) Valid() bool     { return s.inner.Valid() }
func (s *SSTIter) Key() []byte     { return s.inner.Key() }
func (s *SSTIter) Value() []byte   { return s.inner.Value() }
func (s *SSTIter) Close()          { s.inner.Close() }
