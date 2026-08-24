package cache

import "testing"

func TestLRUEvict(t *testing.T) {
	c := NewLRU(20)
	c.Put("a", []byte("12345678"))
	c.Put("b", []byte("12345678"))
	c.Put("c", []byte("12345678"))
	if _, ok := c.Get("a"); ok {
		t.Fatal("a should be evicted")
	}
	if _, ok := c.Get("c"); !ok {
		t.Fatal("c missing")
	}
}

func TestHitRate(t *testing.T) {
	c := NewLRU(1024)
	c.Put("k", []byte("v"))
	c.Get("k")
	c.Get("miss")
	if c.HitRate() < 0.4 {
		t.Fatalf("hit rate %f", c.HitRate())
	}
}
