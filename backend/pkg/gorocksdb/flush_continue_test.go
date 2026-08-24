package gorocksdb_test

import (
	"testing"

	"gorocksdb/pkg/gorocksdb"
)

func TestWriteContinuesAfterFlush(t *testing.T) {
	db, err := gorocksdb.Open(gorocksdb.Options{
		Dir:     t.TempDir(),
		Profile: "production",
		Sync:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := db.Put([]byte("import/0001"), []byte("first")); err != nil {
		t.Fatalf("initial put: %v", err)
	}
	if err := db.Flush(); err != nil {
		t.Fatalf("flush: %v", err)
	}
	if got, err := db.Get([]byte("import/0001")); err != nil || string(got) != "first" {
		t.Fatalf("read after flush: got %q, err %v", got, err)
	}

	if err := db.Put([]byte("import/0002"), []byte("second")); err != nil {
		t.Fatalf("put after flush: %v", err)
	}
	if got, err := db.Get([]byte("import/0002")); err != nil || string(got) != "second" {
		t.Fatalf("read value written after flush: got %q, err %v", got, err)
	}
}
