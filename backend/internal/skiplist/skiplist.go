package skiplist

import (
	"math/rand/v2"
	"sync"
	"sync/atomic"

	"gorocksdb/internal/encoding"
)

const (
	MaxHeight = 12
	Branching = 4
)

type node struct {
	key    []byte
	value  []byte
	tower  [MaxHeight]atomic.Pointer[node]
	height uint32
}

type SkipList struct {
	head   *node
	height atomic.Int32
	mu     sync.Mutex
	bytes  atomic.Int64
	count  atomic.Int64
}

func New() *SkipList {
	h := &node{height: MaxHeight}
	s := &SkipList{head: h}
	s.height.Store(1)
	return s
}

func randomHeight() int {
	h := 1
	for h < MaxHeight && rand.IntN(Branching) == 0 {
		h++
	}
	return h
}

func (s *SkipList) ApproximateBytes() int64 { return s.bytes.Load() }
func (s *SkipList) Len() int64              { return s.count.Load() }

func (s *SkipList) findGreaterOrEqual(key []byte, preds *[MaxHeight]*node) *node {
	x := s.head
	h := int(s.height.Load())
	for level := h - 1; level >= 0; level-- {
		for {
			n := x.tower[level].Load()
			if n == nil || encoding.CompareInternal(n.key, key) >= 0 {
				break
			}
			x = n
		}
		if preds != nil {
			preds[level] = x
		}
	}
	return x.tower[0].Load()
}

func (s *SkipList) Get(key []byte) (value []byte, ok bool) {
	n := s.findGreaterOrEqual(key, nil)
	if n == nil || encoding.CompareInternal(n.key, key) != 0 {
		return nil, false
	}
	return n.value, true
}

func (s *SkipList) Seek(key []byte) (k, v []byte, ok bool) {
	n := s.findGreaterOrEqual(key, nil)
	if n == nil {
		return nil, nil, false
	}
	return n.key, n.value, true
}

func (s *SkipList) Insert(key, value []byte) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var preds [MaxHeight]*node
	n := s.findGreaterOrEqual(key, &preds)
	if n != nil && encoding.CompareInternal(n.key, key) == 0 {
		old := int64(len(n.value))
		n.value = append([]byte(nil), value...)
		s.bytes.Add(int64(len(value)) - old)
		return
	}

	h := randomHeight()
	curH := int(s.height.Load())
	if h > curH {
		for level := curH; level < h; level++ {
			preds[level] = s.head
		}
		s.height.Store(int32(h))
	}

	nn := &node{
		key:    append([]byte(nil), key...),
		value:  append([]byte(nil), value...),
		height: uint32(h),
	}
	for level := 0; level < h; level++ {
		pred := preds[level]
		if pred == nil {
			pred = s.head
		}
		nn.tower[level].Store(pred.tower[level].Load())
		pred.tower[level].Store(nn)
	}
	s.bytes.Add(int64(len(key) + len(value) + 24 + h*8))
	s.count.Add(1)
}

func (s *SkipList) Front() *node {
	return s.head.tower[0].Load()
}

func (n *node) Next() *node {
	if n == nil {
		return nil
	}
	return n.tower[0].Load()
}

func (n *node) Key() []byte {
	if n == nil {
		return nil
	}
	return n.key
}

func (n *node) Value() []byte {
	if n == nil {
		return nil
	}
	return n.value
}
