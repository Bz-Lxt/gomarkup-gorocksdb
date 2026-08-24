package engine

import (
	"gorocksdb/internal/encoding"
	"gorocksdb/internal/iterator"
	"gorocksdb/internal/memtable"
	"gorocksdb/internal/sstable"
)

type KV struct {
	Key   []byte
	Value []byte
}

func (db *DB) Scan(start, end []byte, limit int) ([]KV, error) {
	if db.closed.Load() {
		return nil, ErrClosed
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	db.Met.Scans.Add(1)

	db.mu.Lock()
	mem := db.mem
	mem.Ref()
	imm := make([]*memtable.MemTable, len(db.imm))
	copy(imm, db.imm)
	for _, m := range imm {
		m.Ref()
	}
	db.mu.Unlock()
	defer func() {
		mem.Unref()
		for _, m := range imm {
			m.Unref()
		}
	}()

	var kids []iterator.Iterator
	kids = append(kids, iterator.WrapMem(mem.NewIterator()))
	for i := len(imm) - 1; i >= 0; i-- {
		kids = append(kids, iterator.WrapMem(imm[i].NewIterator()))
	}
	v := db.vs.Current()
	v.Ref()
	defer v.Unref()
	var opened []*sstable.Reader
	_ = opened
	for _, f := range v.AllFiles() {
		r, err := db.reader(f)
		if err != nil {
			for _, k := range kids {
				k.Close()
			}
			return nil, err
		}
		it, err := r.NewIterator()
		if err != nil {
			for _, k := range kids {
				k.Close()
			}
			return nil, err
		}
		kids = append(kids, iterator.WrapSST(it))
	}
	merge := iterator.NewMerging(kids)
	defer merge.Close()

	if len(start) == 0 {
		merge.SeekToFirst()
	} else {
		merge.Seek(encoding.SeekKey(start))
	}

	var out []KV
	var lastUser []byte
	snap := encoding.MaxSequence
	for merge.Valid() && len(out) < limit {
		uk, seq, typ, ok := encoding.SplitInternalKey(merge.Key())
		if !ok {
			merge.Next()
			continue
		}
		if len(end) > 0 && encoding.CompareUser(uk, end) >= 0 {
			break
		}
		if lastUser != nil && encoding.CompareUser(uk, lastUser) == 0 {
			merge.Next()
			continue
		}
		if seq > snap {
			merge.Next()
			continue
		}
		lastUser = append(lastUser[:0], uk...)
		if typ == encoding.TypeDeletion {
			merge.Next()
			continue
		}
		out = append(out, KV{Key: append([]byte(nil), uk...), Value: append([]byte(nil), merge.Value()...)})
		merge.Next()
	}
	return out, nil
}
