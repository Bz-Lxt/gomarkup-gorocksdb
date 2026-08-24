package sstable

import (
	"fmt"
	"testing"

	"gorocksdb/internal/config"
	"gorocksdb/internal/encoding"
)

func TestSSTableRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := config.Test()
	w, err := Create(dir, 1, p, 200)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 200; i++ {
		k := encoding.EncodeInternalKey([]byte(fmt.Sprintf("k%03d", i)), uint64(i+1), encoding.TypeValue)
		if err := w.Add(k, []byte(fmt.Sprintf("v%03d", i))); err != nil {
			t.Fatal(err)
		}
	}
	meta, err := w.Finish()
	if err != nil {
		t.Fatal(err)
	}
	if meta.Entries != 200 {
		t.Fatalf("entries %d", meta.Entries)
	}
	r, err := Open(meta.Path, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	val, found, err := r.Get([]byte("k042"), encoding.MaxSequence)
	if err != nil || !found || string(val) != "v042" {
		t.Fatalf("get %s found=%v err=%v", val, found, err)
	}
	if r.MayContain([]byte("no-such-key-zzzz")) && false {
		t.Log("bloom may false-positive")
	}
}

func TestCorruptCRC(t *testing.T) {
	raw := wrapBlock([]byte("hello"))
	raw[0] ^= 0xff
	if _, err := unwrapBlock(raw); err == nil {
		t.Fatal("expected checksum error")
	}
}
