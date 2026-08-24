package version

import (
	"testing"

	"gorocksdb/internal/encoding"
)

func TestEditRoundTrip(t *testing.T) {
	e := &VersionEdit{}
	e.SetLogNumber(3)
	e.SetNextFile(9)
	e.SetLastSeq(100)
	e.AddFile(&FileMeta{
		Level: 0, Number: 4, Size: 12, Entries: 2,
		Smallest: encoding.EncodeInternalKey([]byte("a"), 1, encoding.TypeValue),
		Largest:  encoding.EncodeInternalKey([]byte("z"), 2, encoding.TypeValue),
	})
	e.DeleteFile(1, 2)
	raw := e.Encode()
	got, err := DecodeEdit(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !got.HasLogNumber || got.LogNumber != 3 || len(got.Added) != 1 || len(got.Deleted) != 1 {
		t.Fatalf("%+v", got)
	}
}

func TestApplyEdit(t *testing.T) {
	v := NewVersion()
	e := &VersionEdit{}
	e.AddFile(&FileMeta{Level: 1, Number: 1, Size: 10, Smallest: []byte("a\x00\x00\x00\x00\x00\x00\x00\x01"), Largest: []byte("m\x00\x00\x00\x00\x00\x00\x00\x01")})
	nv := applyEdit(v, e)
	if nv.NumFiles(1) != 1 {
		t.Fatal(nv.NumFiles(1))
	}
	e2 := &VersionEdit{}
	e2.DeleteFile(1, 1)
	nv2 := applyEdit(nv, e2)
	if nv2.NumFiles(1) != 0 {
		t.Fatal("delete failed")
	}
}
