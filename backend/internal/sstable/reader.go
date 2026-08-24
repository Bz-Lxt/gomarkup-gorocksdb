package sstable

import (
	"encoding/binary"
	"fmt"
	"os"

	"gorocksdb/internal/bloom"
	"gorocksdb/internal/cache"
	"gorocksdb/internal/config"
	"gorocksdb/internal/encoding"
)

type Reader struct {
	f          *os.File
	path       string
	size       int64
	index      []byte
	filter     *bloom.Filter
	indexH     encoding.BlockHandle
	cache      *cache.LRU
	fileNum    uint64
	bloomHits  uint64
	bloomMiss  uint64
}

func Open(path string, fileNum uint64, c *cache.LRU) (*Reader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if st.Size() < int64(config.FooterSize) {
		f.Close()
		return nil, encoding.ErrShortRecord
	}
	footer := make([]byte, config.FooterSize)
	if _, err := f.ReadAt(footer, st.Size()-int64(config.FooterSize)); err != nil {
		f.Close()
		return nil, err
	}
	magic := binary.LittleEndian.Uint64(footer[config.FooterSize-8:])
	if magic != config.MagicNumber {
		f.Close()
		return nil, fmt.Errorf("%w: bad magic", encoding.ErrCorrupt)
	}
	metaH, n := encoding.DecodeHandle(footer[0:])
	if n <= 0 {
		f.Close()
		return nil, encoding.ErrCorrupt
	}
	idxH, n := encoding.DecodeHandle(footer[20:])
	if n <= 0 {
		f.Close()
		return nil, encoding.ErrCorrupt
	}
	r := &Reader{f: f, path: path, size: st.Size(), indexH: idxH, cache: c, fileNum: fileNum}
	idx, err := r.readBlock(idxH)
	if err != nil {
		f.Close()
		return nil, err
	}
	r.index = idx
	meta, err := r.readBlock(metaH)
	if err != nil {
		f.Close()
		return nil, err
	}
	mit, err := newBlockIter(meta)
	if err != nil {
		f.Close()
		return nil, err
	}
	mit.Seek([]byte("filter.bloom"))
	if mit.Valid() && string(mit.Key()) == "filter.bloom" {
		fh, ok := decodeIndexValue(mit.Value())
		if ok {
			fb, err := r.readBlock(fh)
			if err == nil {
				r.filter = bloom.Decode(fb)
			}
		}
	}
	return r, nil
}

func (r *Reader) Close() error { return r.f.Close() }

func (r *Reader) readBlock(h encoding.BlockHandle) ([]byte, error) {
	ck := fmt.Sprintf("%d:%d:%d", r.fileNum, h.Offset, h.Size)
	if r.cache != nil {
		if v, ok := r.cache.Get(ck); ok {
			return v, nil
		}
	}
	raw := make([]byte, h.Size+uint64(blockTrailerLen))
	if _, err := r.f.ReadAt(raw, int64(h.Offset)); err != nil {
		return nil, err
	}
	data, err := unwrapBlock(raw)
	if err != nil {
		return nil, err
	}
	if r.cache != nil {
		r.cache.Put(ck, data)
	}
	return data, nil
}

func (r *Reader) MayContain(userKey []byte) bool {
	if r.filter == nil {
		return true
	}
	if r.filter.MayContain(userKey) {
		r.bloomHits++
		return true
	}
	r.bloomMiss++
	return false
}

func (r *Reader) Get(userKey []byte, snapshot uint64) ([]byte, bool, error) {
	if !r.MayContain(userKey) {
		return nil, false, nil
	}
	it, err := r.NewIterator()
	if err != nil {
		return nil, false, err
	}
	defer it.Close()
	it.Seek(encoding.SeekKey(userKey))
	for it.Valid() {
		uk, seq, typ, ok := encoding.SplitInternalKey(it.Key())
		if !ok || encoding.CompareUser(uk, userKey) != 0 {
			return nil, false, nil
		}
		if seq > snapshot {
			it.Next()
			continue
		}
		if typ == encoding.TypeDeletion {
			return nil, true, nil
		}
		return bytesClone(it.Value()), true, nil
	}
	return nil, false, nil
}

type Iterator struct {
	r       *Reader
	index   *blockIter
	data    *blockIter
	dataBlk []byte
	valid   bool
	err     error
}

func (r *Reader) NewIterator() (*Iterator, error) {
	ix, err := newBlockIter(r.index)
	if err != nil {
		return nil, err
	}
	return &Iterator{r: r, index: ix}, nil
}

func (it *Iterator) loadData() bool {
	if !it.index.Valid() {
		it.valid = false
		return false
	}
	h, ok := decodeIndexValue(it.index.Value())
	if !ok {
		it.err = encoding.ErrCorrupt
		it.valid = false
		return false
	}
	blk, err := it.r.readBlock(h)
	if err != nil {
		it.err = err
		it.valid = false
		return false
	}
	dit, err := newBlockIter(blk)
	if err != nil {
		it.err = err
		it.valid = false
		return false
	}
	it.data = dit
	it.dataBlk = blk
	return true
}

func (it *Iterator) SeekToFirst() {
	it.index.SeekToFirst()
	if !it.loadData() {
		return
	}
	it.data.SeekToFirst()
	it.valid = it.data.Valid()
}

func (it *Iterator) Seek(key []byte) {
	it.index.Seek(key)
	if !it.index.Valid() {
		// last block may still contain
		// index keys are last key of each block; seek past end means miss
		it.valid = false
		return
	}
	if !it.loadData() {
		return
	}
	it.data.Seek(key)
	if !it.data.Valid() {
		it.index.Next()
		if !it.loadData() {
			return
		}
		it.data.SeekToFirst()
	}
	it.valid = it.data.Valid()
}

func (it *Iterator) Next() {
	if it.data != nil {
		it.data.Next()
		if it.data.Valid() {
			it.valid = true
			return
		}
	}
	it.index.Next()
	if !it.loadData() {
		return
	}
	it.data.SeekToFirst()
	it.valid = it.data.Valid()
}

func (it *Iterator) Valid() bool  { return it.valid }
func (it *Iterator) Key() []byte   { return it.data.Key() }
func (it *Iterator) Value() []byte { return it.data.Value() }
func (it *Iterator) Err() error    { return it.err }
func (it *Iterator) Close()        {}

func (r *Reader) BloomStats() (pass, reject uint64) {
	return r.bloomHits, r.bloomMiss
}
