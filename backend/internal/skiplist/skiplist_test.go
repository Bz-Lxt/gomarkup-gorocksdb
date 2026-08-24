package skiplist

import (
	"fmt"
	"sync"
	"testing"

	"gorocksdb/internal/encoding"
)

func TestInsertSeek(t *testing.T) {
	s := New()
	for i := 0; i < 100; i++ {
		k := encoding.EncodeInternalKey([]byte(fmt.Sprintf("k%03d", i)), uint64(i+1), encoding.TypeValue)
		s.Insert(k, []byte(fmt.Sprintf("v%d", i)))
	}
	if s.Len() != 100 {
		t.Fatalf("len %d", s.Len())
	}
	seek := encoding.SeekKey([]byte("k050"))
	k, v, ok := s.Seek(seek)
	if !ok || string(encoding.UserKey(k)) != "k050" || string(v) != "v50" {
		t.Fatalf("seek got %s %s %v", k, v, ok)
	}
}

func TestConcurrentReadWrite(t *testing.T) {
	s := New()
	var wg sync.WaitGroup
	for w := 0; w < 4; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				k := encoding.EncodeInternalKey([]byte(fmt.Sprintf("%d-%03d", id, i)), uint64(i+1), encoding.TypeValue)
				s.Insert(k, []byte("x"))
			}
		}(w)
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 400; i++ {
			_ = s.ApproximateBytes()
			it := s.NewIterator()
			it.SeekToFirst()
			for it.Valid() {
				it.Next()
			}
		}
	}()
	wg.Wait()
	if s.Len() != 800 {
		t.Fatalf("len %d", s.Len())
	}
}
