package cache

import (
	"container/list"
	"sync"
	"sync/atomic"
)

type entry struct {
	key  string
	val  []byte
	size int64
}

type LRU struct {
	mu      sync.Mutex
	cap     int64
	used    int64
	ll      *list.List
	tab     map[string]*list.Element
	hits    atomic.Int64
	misses  atomic.Int64
	inserts atomic.Int64
}

func NewLRU(capacity int64) *LRU {
	if capacity <= 0 {
		capacity = 8 << 20
	}
	return &LRU{
		cap: capacity,
		ll:  list.New(),
		tab: make(map[string]*list.Element),
	}
}

func (c *LRU) Get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.tab[key]; ok {
		c.ll.MoveToFront(el)
		c.hits.Add(1)
		v := el.Value.(*entry).val
		out := make([]byte, len(v))
		copy(out, v)
		return out, true
	}
	c.misses.Add(1)
	return nil, false
}

func (c *LRU) Put(key string, val []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	sz := int64(len(key) + len(val))
	if el, ok := c.tab[key]; ok {
		ent := el.Value.(*entry)
		c.used -= ent.size
		ent.val = append([]byte(nil), val...)
		ent.size = sz
		c.used += sz
		c.ll.MoveToFront(el)
	} else {
		ent := &entry{key: key, val: append([]byte(nil), val...), size: sz}
		c.tab[key] = c.ll.PushFront(ent)
		c.used += sz
		c.inserts.Add(1)
	}
	for c.used > c.cap && c.ll.Len() > 0 {
		back := c.ll.Back()
		ent := back.Value.(*entry)
		c.ll.Remove(back)
		delete(c.tab, ent.key)
		c.used -= ent.size
	}
}

func (c *LRU) Stats() (hits, misses, inserts, used, cap int64) {
	return c.hits.Load(), c.misses.Load(), c.inserts.Load(), c.used, c.cap
}

func (c *LRU) HitRate() float64 {
	h, m := c.hits.Load(), c.misses.Load()
	if h+m == 0 {
		return 0
	}
	return float64(h) / float64(h+m)
}
