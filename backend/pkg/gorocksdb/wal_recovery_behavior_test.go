package gorocksdb_test

import (
	"encoding/binary"
	"hash/crc32"
	"os"
	"path/filepath"
	"testing"

	"gorocksdb/pkg/gorocksdb"
)

func TestOpenRejectsMalformedLogicalWALRecord(t *testing.T) {
	dir := t.TempDir()
	db, err := gorocksdb.Open(gorocksdb.Options{Dir: dir, Profile: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"before", "malformed", "after"} {
		if err := db.Put([]byte(key), []byte("committed")); err != nil {
			t.Fatal(err)
		}
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	logs, err := filepath.Glob(filepath.Join(dir, "*.log"))
	if err != nil {
		t.Fatal(err)
	}
	if len(logs) != 1 {
		t.Fatalf("expected one WAL, got %d", len(logs))
	}
	makeSecondBatchLogicallyIncomplete(t, logs[0])

	reopened, err := gorocksdb.Open(gorocksdb.Options{Dir: dir, Profile: "demo"})
	if err == nil {
		if reopened != nil {
			_ = reopened.Close()
		}
		t.Fatal("Open accepted a checksum-valid WAL record with an incomplete logical batch")
	}
}

func makeSecondBatchLogicallyIncomplete(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	const headerSize = 9
	off := 0
	for record := 0; record < 3; record++ {
		if off+headerSize > len(data) {
			t.Fatalf("WAL ended before record %d", record+1)
		}
		length := int(binary.LittleEndian.Uint32(data[off+4 : off+8]))
		end := off + headerSize + length
		if end > len(data) {
			t.Fatalf("WAL record %d is physically incomplete", record+1)
		}
		if record == 1 {
			payload := data[off+headerSize : end]
			if len(payload) < 12 {
				t.Fatalf("WAL record %d has a short batch header", record+1)
			}
			count := binary.LittleEndian.Uint32(payload[8:12])
			binary.LittleEndian.PutUint32(payload[8:12], count+1)
			checksum := crc32.Checksum(data[off+4:end], crc32.MakeTable(crc32.Castagnoli))
			binary.LittleEndian.PutUint32(data[off:off+4], maskWALChecksum(checksum))
		}
		off = end
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func maskWALChecksum(checksum uint32) uint32 {
	return ((checksum >> 15) | (checksum << 17)) + 0xa282ead8
}
