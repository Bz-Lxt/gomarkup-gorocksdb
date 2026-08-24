package engine

import (
	"fmt"
	"testing"
	"time"

	"gorocksdb/internal/config"
)

func TestCompactDropsTombstone(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(Options{Dir: dir, Profile: config.Test(), Sync: true})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for i := 0; i < 80; i++ {
		if err := db.Put([]byte(fmt.Sprintf("k%02d", i)), []byte("v")); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Delete([]byte("k01")); err != nil {
		t.Fatal(err)
	}
	if err := db.Flush(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(4 * time.Second)
	for time.Now().Before(deadline) {
		_ = db.Compact()
		st := db.SnapshotState()
		if st.Levels[0].Files != nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if _, err := db.Get([]byte("k01")); err != ErrNotFound {
		t.Fatalf("tombstone should hide key: %v", err)
	}
	v, err := db.Get([]byte("k02"))
	if err != nil || string(v) != "v" {
		t.Fatalf("alive key %s %v", v, err)
	}
}
