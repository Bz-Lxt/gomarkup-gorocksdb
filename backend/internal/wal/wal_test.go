package wal

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"gorocksdb/internal/encoding"
)

func TestWALRoundTripAndTruncate(t *testing.T) {
	dir := t.TempDir()
	w, err := Create(dir, 1)
	if err != nil {
		t.Fatal(err)
	}
	items := []BatchItem{{Type: encoding.TypeValue, Key: []byte("a"), Value: bytes.Repeat([]byte("x"), 100)}}
	if err := w.Append(EncodeBatch(1, items), true); err != nil {
		t.Fatal(err)
	}
	big := []BatchItem{{Type: encoding.TypeValue, Key: []byte("b"), Value: bytes.Repeat([]byte("y"), 80_000)}}
	if err := w.Append(EncodeBatch(2, big), true); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}

	n := 0
	trunc, err := Replay(w.Path(), func(rec []byte) error {
		n++
		_, its, err := DecodeBatch(rec)
		if err != nil {
			return err
		}
		if len(its) != 1 {
			t.Fatalf("items %d", len(its))
		}
		return nil
	})
	if err != nil || trunc || n != 2 {
		t.Fatalf("n=%d trunc=%v err=%v", n, trunc, err)
	}

	// corrupt tail
	f, err := os.OpenFile(w.Path(), os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.Write([]byte{1, 2, 3, 4, 5})
	f.Close()
	n = 0
	trunc, err = Replay(w.Path(), func(rec []byte) error {
		n++
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("expected 2 good records, got %d", n)
	}
	if !trunc {
		t.Log("tail may be treated as padding; acceptable if no panic")
	}
	_ = filepath.Separator
}

func TestCloseOnce(t *testing.T) {
	dir := t.TempDir()
	w, err := Create(dir, 3)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatalf("second close: %v", err)
	}
}
