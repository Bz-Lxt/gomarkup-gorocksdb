package memtable

import (
	"sync/atomic"

	"gorocksdb/internal/encoding"
	"gorocksdb/internal/skiplist"
)

type MemTable struct {
	list     *skiplist.SkipList
	id       uint64
	logNum   uint64
	refs     atomic.Int32
	imm      atomic.Bool
	approx   atomic.Int64
}

func New(id, logNum uint64) *MemTable {
	m := &MemTable{list: skiplist.New(), id: id, logNum: logNum}
	m.refs.Store(1)
	return m
}

func (m *MemTable) ID() uint64     { return m.id }
func (m *MemTable) LogNum() uint64 { return m.logNum }
func (m *MemTable) Ref()           { m.refs.Add(1) }
func (m *MemTable) Unref() int32   { return m.refs.Add(-1) }
func (m *MemTable) MarkImmutable() { m.imm.Store(true) }
func (m *MemTable) Immutable() bool { return m.imm.Load() }
func (m *MemTable) ApproximateMemory() int64 { return m.list.ApproximateBytes() }
func (m *MemTable) Len() int64               { return m.list.Len() }

func (m *MemTable) Add(seq uint64, typ encoding.ValueType, key, value []byte) {
	ikey := encoding.EncodeInternalKey(key, seq, typ)
	m.list.Insert(ikey, value)
}

func (m *MemTable) Get(key []byte, snapshot uint64) ([]byte, bool, bool) {
	seek := encoding.SeekKey(key)
	nkey, nval, ok := m.list.Seek(seek)
	if !ok {
		return nil, false, false
	}
	uk, seq, typ, valid := encoding.SplitInternalKey(nkey)
	if !valid || encoding.CompareUser(uk, key) != 0 {
		return nil, false, false
	}
	if seq > snapshot {
		it := m.list.NewIterator()
		it.Seek(seek)
		for it.Valid() {
			uk2, seq2, typ2, ok2 := encoding.SplitInternalKey(it.Key())
			if !ok2 || encoding.CompareUser(uk2, key) != 0 {
				return nil, false, false
			}
			if seq2 <= snapshot {
				if typ2 == encoding.TypeDeletion {
					return nil, true, false
				}
				return it.Value(), true, true
			}
			it.Next()
		}
		return nil, false, false
	}
	if typ == encoding.TypeDeletion {
		return nil, true, false
	}
	return nval, true, true
}

func (m *MemTable) NewIterator() *skiplist.Iterator {
	return m.list.NewIterator()
}
