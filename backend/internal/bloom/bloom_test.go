package bloom

import (
	"fmt"
	"testing"
)

func TestFalsePositiveRate(t *testing.T) {
	n := 100_000
	f := New(n, 10)
	for i := 0; i < n; i++ {
		f.Add([]byte(fmt.Sprintf("key-%d", i)))
	}
	fp := 0
	trials := 100_000
	for i := 0; i < trials; i++ {
		k := []byte(fmt.Sprintf("miss-%d", i))
		if f.MayContain(k) {
			fp++
		}
	}
	rate := float64(fp) / float64(trials)
	if rate > 0.01 {
		t.Fatalf("FPR %.4f > 1%%", rate)
	}
	for i := 0; i < 1000; i++ {
		if !f.MayContain([]byte(fmt.Sprintf("key-%d", i))) {
			t.Fatalf("false negative at %d", i)
		}
	}
}

func TestRoundTrip(t *testing.T) {
	f := New(16, 10)
	f.Add([]byte("hello"))
	raw := f.Encode()
	g := Decode(raw)
	if !g.MayContain([]byte("hello")) {
		t.Fatal("decoded miss")
	}
}
