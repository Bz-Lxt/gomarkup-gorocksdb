package memtable

import (
	"testing"

	"gorocksdb/internal/encoding"
)

func TestGetTombstone(t *testing.T) {
	m := New(1, 1)
	m.Add(1, encoding.TypeValue, []byte("k"), []byte("v"))
	m.Add(2, encoding.TypeDeletion, []byte("k"), nil)
	_, found, alive := m.Get([]byte("k"), encoding.MaxSequence)
	if !found || alive {
		t.Fatalf("found=%v alive=%v", found, alive)
	}
	val, found, alive := m.Get([]byte("k"), 1)
	if !found || !alive || string(val) != "v" {
		t.Fatalf("%s %v %v", val, found, alive)
	}
}
