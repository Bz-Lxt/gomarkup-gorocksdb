package events

import "testing"

func TestDropWhenFull(t *testing.T) {
	b := New(2)
	b.Publish("a", nil)
	b.Publish("b", nil)
	b.Publish("c", nil)
	if b.Dropped() == 0 {
		t.Fatal("expected drop")
	}
	CloseSafe(b)
}

func CloseSafe(b *Bus) { b.Close() }
