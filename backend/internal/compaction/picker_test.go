package compaction

import (
	"testing"

	"gorocksdb/internal/config"
	"gorocksdb/internal/version"
)

func TestPickL0(t *testing.T) {
	v := version.NewVersion()
	p := config.Test()
	for i := 0; i < p.L0CompactionTrigger; i++ {
		v.Files[0] = append(v.Files[0], &version.FileMeta{
			Number: uint64(i + 1), Level: 0, Size: 100,
			Smallest: []byte("a"), Largest: []byte("z"),
		})
	}
	job := Pick(v, p)
	if job == nil || job.Level != 0 {
		t.Fatalf("%+v", job)
	}
}

func TestNoPickWhenEmpty(t *testing.T) {
	if Pick(version.NewVersion(), config.Test()) != nil {
		t.Fatal("empty should not compact")
	}
}
