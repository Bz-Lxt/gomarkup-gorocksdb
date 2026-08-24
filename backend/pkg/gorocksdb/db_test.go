package gorocksdb

import (
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func openTest(t *testing.T) *DB {
	t.Helper()
	dir := t.TempDir()
	db, err := Open(Options{Dir: dir, Profile: "test", Sync: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func TestPutGetDelete(t *testing.T) {
	db := openTest(t)
	if err := db.Put([]byte("hello"), []byte("world")); err != nil {
		t.Fatal(err)
	}
	v, err := db.Get([]byte("hello"))
	if err != nil || string(v) != "world" {
		t.Fatalf("got %s %v", v, err)
	}
	if err := db.Delete([]byte("hello")); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Get([]byte("hello")); !IsNotFound(err) {
		t.Fatalf("want not found, got %v", err)
	}
}

func TestOverwriteKeepsLatest(t *testing.T) {
	db := openTest(t)
	_ = db.Put([]byte("k"), []byte("v1"))
	_ = db.Put([]byte("k"), []byte("v2"))
	v, err := db.Get([]byte("k"))
	if err != nil || string(v) != "v2" {
		t.Fatalf("got %s %v", v, err)
	}
}

func TestFlushAndRecover(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(Options{Dir: dir, Profile: "test", Sync: true})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 200; i++ {
		k := []byte(fmt.Sprintf("k%04d", i))
		if err := db.Put(k, []byte("x")); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db2, err := Open(Options{Dir: dir, Profile: "test", Sync: true})
	if err != nil {
		t.Fatal(err)
	}
	defer db2.Close()
	v, err := db2.Get([]byte("k0007"))
	if err != nil || string(v) != "x" {
		t.Fatalf("recover miss %s %v", v, err)
	}
}

func TestDifferentialMap(t *testing.T) {
	db := openTest(t)
	ref := map[string]string{}
	rng := rand.New(rand.NewPCG(1, 2))
	const N = 5000
	for i := 0; i < N; i++ {
		k := fmt.Sprintf("k%04d", rng.IntN(800))
		switch rng.IntN(10) {
		case 0:
			delete(ref, k)
			_ = db.Delete([]byte(k))
		default:
			v := fmt.Sprintf("v%d", i)
			ref[k] = v
			if err := db.Put([]byte(k), []byte(v)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := db.Flush(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	_ = db.Compact()
	for k, want := range ref {
		got, err := db.Get([]byte(k))
		if err != nil || string(got) != want {
			t.Fatalf("key %s want %s got %s err %v", k, want, got, err)
		}
	}
	// random misses
	for i := 0; i < 200; i++ {
		k := fmt.Sprintf("miss-%d", i)
		if _, ok := ref[k]; ok {
			continue
		}
		if _, err := db.Get([]byte(k)); !IsNotFound(err) {
			t.Fatalf("expected miss %s: %v", k, err)
		}
	}
}

func TestScan(t *testing.T) {
	db := openTest(t)
	for i := 0; i < 20; i++ {
		_ = db.Put([]byte(fmt.Sprintf("a%02d", i)), []byte("v"))
	}
	kvs, err := db.Scan([]byte("a05"), []byte("a10"), 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(kvs) != 5 {
		t.Fatalf("scan %d", len(kvs))
	}
}

func TestBatchAndSnapshot(t *testing.T) {
	db := openTest(t)
	_ = db.Put([]byte("s"), []byte("1"))
	snap := db.Snapshot()
	defer snap.Release()
	_ = db.Put([]byte("s"), []byte("2"))
	v, err := db.GetAt([]byte("s"), snap)
	if err != nil || string(v) != "1" {
		t.Fatalf("snap %s %v", v, err)
	}
	b := &WriteBatch{}
	b.Put([]byte("x"), []byte("y"))
	b.Delete([]byte("s"))
	if err := db.Write(b); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Get([]byte("s")); !IsNotFound(err) {
		t.Fatal(err)
	}
}

func TestStateJSONShape(t *testing.T) {
	db := openTest(t)
	st := db.State()
	if st.Profile != "test" || st.Levels == nil {
		t.Fatalf("%+v", st)
	}
}

func TestNoDataDirLeak(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(Options{Dir: filepath.Join(dir, "n"), Profile: "test"})
	if err != nil {
		t.Fatal(err)
	}
	_ = db.Put([]byte("a"), []byte("b"))
	_ = db.Close()
	if _, err := os.Stat(filepath.Join(dir, "n", "CURRENT")); err != nil {
		t.Fatal(err)
	}
}
