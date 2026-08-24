package encoding

import "testing"

func TestInternalKeyRoundTrip(t *testing.T) {
	ikey := EncodeInternalKey([]byte("alpha"), 99, TypeValue)
	uk, seq, typ, ok := SplitInternalKey(ikey)
	if !ok || string(uk) != "alpha" || seq != 99 || typ != TypeValue {
		t.Fatalf("got %q %d %d", uk, seq, typ)
	}
}

func TestCompareNewerFirst(t *testing.T) {
	a := EncodeInternalKey([]byte("k"), 2, TypeValue)
	b := EncodeInternalKey([]byte("k"), 1, TypeValue)
	if CompareInternal(a, b) >= 0 {
		t.Fatal("seq 2 should sort before seq 1")
	}
	if CompareInternal(EncodeInternalKey([]byte("a"), 1, TypeValue), EncodeInternalKey([]byte("b"), 9, TypeValue)) >= 0 {
		t.Fatal("user key order")
	}
}

func TestVarint(t *testing.T) {
	for _, v := range []uint64{0, 1, 127, 128, 300, 1 << 21, 1 << 35} {
		var buf [10]byte
		n := PutUvarint(buf[:], v)
		got, m := Uvarint(buf[:n])
		if got != v || m != n {
			t.Fatalf("%d -> %d n=%d m=%d", v, got, n, m)
		}
	}
}
