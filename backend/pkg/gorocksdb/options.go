package gorocksdb

import (
	"gorocksdb/internal/config"
	"gorocksdb/internal/engine"
)

type Options struct {
	Dir     string
	Profile string
	Sync    bool
}

func (o Options) toEngine() engine.Options {
	p := config.Lookup(o.Profile)
	return engine.Options{Dir: o.Dir, Profile: p, Sync: o.Sync}
}

type WriteBatch struct {
	items []BatchItem
}

type BatchItem struct {
	Delete bool
	Key    []byte
	Value  []byte
}

func (b *WriteBatch) Put(key, value []byte) {
	b.items = append(b.items, BatchItem{Key: append([]byte(nil), key...), Value: append([]byte(nil), value...)})
}

func (b *WriteBatch) Delete(key []byte) {
	b.items = append(b.items, BatchItem{Delete: true, Key: append([]byte(nil), key...)})
}

type Snapshot struct {
	seq uint64
	db  *DB
}

func (s *Snapshot) Release() {
	if s.db != nil {
		s.db.inner.ReleaseSnapshot(s.seq)
	}
}
