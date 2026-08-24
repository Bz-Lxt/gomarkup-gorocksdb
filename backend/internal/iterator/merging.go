package iterator

import (
	"container/heap"

	"gorocksdb/internal/encoding"
)

type heapItem struct {
	it    Iterator
	index int
}

type mergeHeap []*heapItem

func (h mergeHeap) Len() int { return len(h) }
func (h mergeHeap) Less(i, j int) bool {
	c := encoding.CompareInternal(h[i].it.Key(), h[j].it.Key())
	if c != 0 {
		return c < 0
	}
	return h[i].index < h[j].index
}
func (h mergeHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *mergeHeap) Push(x interface{}) { *h = append(*h, x.(*heapItem)) }
func (h *mergeHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[:n-1]
	return x
}

type MergingIterator struct {
	children []Iterator
	h        mergeHeap
	current  *heapItem
	valid    bool
}

func NewMerging(children []Iterator) *MergingIterator {
	return &MergingIterator{children: children}
}

func (m *MergingIterator) rebuild(seekFirst bool, seekKey []byte) {
	m.h = m.h[:0]
	for i, it := range m.children {
		if seekFirst {
			it.SeekToFirst()
		} else {
			it.Seek(seekKey)
		}
		if it.Valid() {
			heap.Push(&m.h, &heapItem{it: it, index: i})
		}
	}
	m.advance()
}

func (m *MergingIterator) advance() {
	if m.h.Len() == 0 {
		m.valid = false
		m.current = nil
		return
	}
	m.current = heap.Pop(&m.h).(*heapItem)
	m.valid = m.current.it.Valid()
}

func (m *MergingIterator) SeekToFirst() { m.rebuild(true, nil) }
func (m *MergingIterator) Seek(key []byte) { m.rebuild(false, key) }

func (m *MergingIterator) Next() {
	if m.current == nil {
		return
	}
	m.current.it.Next()
	if m.current.it.Valid() {
		heap.Push(&m.h, m.current)
	}
	m.advance()
}

func (m *MergingIterator) Valid() bool { return m.valid }
func (m *MergingIterator) Key() []byte {
	if m.current == nil {
		return nil
	}
	return m.current.it.Key()
}
func (m *MergingIterator) Value() []byte {
	if m.current == nil {
		return nil
	}
	return m.current.it.Value()
}

func (m *MergingIterator) Close() {
	for _, it := range m.children {
		it.Close()
	}
}
