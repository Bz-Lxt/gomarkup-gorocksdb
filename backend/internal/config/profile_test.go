package config

import "testing"

func TestLookupAndLevelBytes(t *testing.T) {
	if Lookup("production").WriteBufferSize != 64<<20 {
		t.Fatal("prod 64MB")
	}
	if Lookup("demo").L0CompactionTrigger != 3 {
		t.Fatal("demo L0")
	}
	p := Test()
	if p.MaxBytesForLevel(1) != p.MaxBytesForLevelBase {
		t.Fatal("L1 base")
	}
	if p.MaxBytesForLevel(2) != p.MaxBytesForLevelBase*10 {
		t.Fatal("L2 x10")
	}
}
