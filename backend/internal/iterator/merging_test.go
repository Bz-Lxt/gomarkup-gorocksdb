package iterator

import (
	"testing"

	"gorocksdb/internal/encoding"
	"gorocksdb/internal/skiplist"
)

func TestMergeOrder(t *testing.T) {
	a := skiplist.New()
	b := skiplist.New()
	a.Insert(encoding.EncodeInternalKey([]byte("a"), 2, encoding.TypeValue), []byte("a2"))
	b.Insert(encoding.EncodeInternalKey([]byte("a"), 1, encoding.TypeValue), []byte("a1"))
	b.Insert(encoding.EncodeInternalKey([]byte("b"), 1, encoding.TypeValue), []byte("b1"))
	m := NewMerging([]Iterator{WrapMem(a.NewIterator()), WrapMem(b.NewIterator())})
	m.SeekToFirst()
	var keys []string
	for m.Valid() {
		keys = append(keys, string(encoding.UserKey(m.Key()))+string(m.Value()))
		m.Next()
	}
	if len(keys) != 3 || keys[0] != "aa2" {
		t.Fatalf("%v", keys)
	}
}
