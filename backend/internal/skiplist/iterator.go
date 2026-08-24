package skiplist

import "gorocksdb/internal/encoding"

type Iterator struct {
	list *SkipList
	node *node
}

func (s *SkipList) NewIterator() *Iterator {
	return &Iterator{list: s}
}

func (it *Iterator) SeekToFirst() {
	it.node = it.list.Front()
}

func (it *Iterator) SeekToLast() {
	var last *node
	for n := it.list.Front(); n != nil; n = n.Next() {
		last = n
	}
	it.node = last
}

func (it *Iterator) Seek(key []byte) {
	it.node = it.list.findGreaterOrEqual(key, nil)
}

func (it *Iterator) Next() {
	if it.node != nil {
		it.node = it.node.Next()
	}
}

func (it *Iterator) Valid() bool { return it.node != nil }

func (it *Iterator) Key() []byte {
	if it.node == nil {
		return nil
	}
	return it.node.key
}

func (it *Iterator) Value() []byte {
	if it.node == nil {
		return nil
	}
	return it.node.value
}

func (it *Iterator) UserKey() []byte {
	return encoding.UserKey(it.Key())
}
