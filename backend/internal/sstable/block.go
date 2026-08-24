package sstable

import (
	"bytes"
	"encoding/binary"

	"gorocksdb/internal/encoding"
)

const (
	compressionNone = 0
	blockTrailerLen = 5 // type + crc32
)

func sharedPrefix(a, b []byte) int {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	return i
}

type blockBuilder struct {
	buf       []byte
	restarts  []uint32
	counter   int
	restartN  int
	lastKey   []byte
	finished  bool
}

func newBlockBuilder(restartInterval int) *blockBuilder {
	if restartInterval <= 0 {
		restartInterval = 16
	}
	return &blockBuilder{restartN: restartInterval, restarts: []uint32{0}}
}

func (b *blockBuilder) Add(key, value []byte) {
	var shared int
	if b.counter < b.restartN {
		shared = sharedPrefix(b.lastKey, key)
	} else {
		b.restarts = append(b.restarts, uint32(len(b.buf)))
		b.counter = 0
		shared = 0
	}
	unshared := len(key) - shared
	b.buf = encoding.AppendUvarint(b.buf, uint64(shared))
	b.buf = encoding.AppendUvarint(b.buf, uint64(unshared))
	b.buf = encoding.AppendUvarint(b.buf, uint64(len(value)))
	b.buf = append(b.buf, key[shared:]...)
	b.buf = append(b.buf, value...)
	b.lastKey = append(b.lastKey[:0], key...)
	b.counter++
}

func (b *blockBuilder) Finish() []byte {
	for _, off := range b.restarts {
		var tmp [4]byte
		binary.LittleEndian.PutUint32(tmp[:], off)
		b.buf = append(b.buf, tmp[:]...)
	}
	var n [4]byte
	binary.LittleEndian.PutUint32(n[:], uint32(len(b.restarts)))
	b.buf = append(b.buf, n[:]...)
	b.finished = true
	return b.buf
}

func (b *blockBuilder) Reset() {
	b.buf = b.buf[:0]
	b.restarts = b.restarts[:0]
	b.restarts = append(b.restarts, 0)
	b.counter = 0
	b.lastKey = b.lastKey[:0]
	b.finished = false
}

func (b *blockBuilder) Size() int { return len(b.buf) }
func (b *blockBuilder) Empty() bool { return len(b.buf) == 0 }

func wrapBlock(data []byte) []byte {
	out := make([]byte, len(data)+blockTrailerLen)
	copy(out, data)
	out[len(data)] = compressionNone
	crc := encoding.CRC32C(out[:len(data)+1])
	binary.LittleEndian.PutUint32(out[len(data)+1:], encoding.MaskCRC(crc))
	return out
}

func unwrapBlock(raw []byte) ([]byte, error) {
	if len(raw) < blockTrailerLen {
		return nil, encoding.ErrShortRecord
	}
	data := raw[:len(raw)-blockTrailerLen]
	typ := raw[len(raw)-blockTrailerLen]
	want := encoding.UnmaskCRC(binary.LittleEndian.Uint32(raw[len(raw)-4:]))
	got := encoding.CRC32C(raw[:len(raw)-4])
	if want != got {
		return nil, encoding.ErrChecksum
	}
	if typ != compressionNone {
		return nil, encoding.ErrCorrupt
	}
	return data, nil
}

type blockIter struct {
	data      []byte
	restarts  []uint32
	offset    int
	key       []byte
	value     []byte
	valid     bool
	restartAt int
}

func newBlockIter(data []byte) (*blockIter, error) {
	if len(data) < 4 {
		return nil, encoding.ErrShortRecord
	}
	n := int(binary.LittleEndian.Uint32(data[len(data)-4:]))
	if n <= 0 || 4+4*n > len(data) {
		return nil, encoding.ErrCorrupt
	}
	base := len(data) - 4 - 4*n
	restarts := make([]uint32, n)
	for i := 0; i < n; i++ {
		restarts[i] = binary.LittleEndian.Uint32(data[base+4*i : base+4*i+4])
	}
	return &blockIter{data: data[:base], restarts: restarts}, nil
}

func (it *blockIter) parseEntry(off int) (next int, shared, unshared, vlen int, ok bool) {
	if off >= len(it.data) {
		return 0, 0, 0, 0, false
	}
	s, n1 := encoding.Uvarint(it.data[off:])
	if n1 <= 0 {
		return 0, 0, 0, 0, false
	}
	u, n2 := encoding.Uvarint(it.data[off+n1:])
	if n2 <= 0 {
		return 0, 0, 0, 0, false
	}
	v, n3 := encoding.Uvarint(it.data[off+n1+n2:])
	if n3 <= 0 {
		return 0, 0, 0, 0, false
	}
	hdr := n1 + n2 + n3
	if off+hdr+int(u)+int(v) > len(it.data) {
		return 0, 0, 0, 0, false
	}
	return off + hdr + int(u) + int(v), int(s), int(u), int(v), true
}

func (it *blockIter) SeekToFirst() {
	it.offset = 0
	it.key = it.key[:0]
	it.decodeCurrent()
}

func (it *blockIter) decodeCurrent() {
	it.valid = false
	next, shared, unshared, vlen, ok := it.parseEntry(it.offset)
	if !ok {
		return
	}
	hdrSkip := next - unshared - vlen
	uk := it.data[hdrSkip : hdrSkip+unshared]
	if shared > len(it.key) {
		return
	}
	it.key = append(it.key[:shared], uk...)
	it.value = it.data[hdrSkip+unshared : hdrSkip+unshared+vlen]
	it.offset = next
	it.valid = true
}

func (it *blockIter) Next() {
	if !it.valid {
		return
	}
	if it.offset >= len(it.data) {
		it.valid = false
		return
	}
	it.decodeCurrent()
}

func (it *blockIter) Valid() bool  { return it.valid }
func (it *blockIter) Key() []byte   { return it.key }
func (it *blockIter) Value() []byte { return it.value }

func (it *blockIter) Seek(target []byte) {
	lo, hi := 0, len(it.restarts)-1
	for lo < hi {
		mid := (lo + hi + 1) / 2
		it.offset = int(it.restarts[mid])
		it.key = it.key[:0]
		it.decodeCurrent()
		if !it.valid {
			hi = mid - 1
			continue
		}
		if encoding.CompareInternal(it.key, target) <= 0 {
			lo = mid
		} else {
			hi = mid - 1
		}
	}
	it.offset = int(it.restarts[lo])
	it.key = it.key[:0]
	it.decodeCurrent()
	for it.valid && encoding.CompareInternal(it.key, target) < 0 {
		it.Next()
	}
}

func encodeIndexValue(h encoding.BlockHandle) []byte {
	return encoding.EncodeHandle(h)
}

func decodeIndexValue(v []byte) (encoding.BlockHandle, bool) {
	h, n := encoding.DecodeHandle(v)
	return h, n > 0
}

func bytesClone(b []byte) []byte {
	if b == nil {
		return nil
	}
	return bytes.Clone(b)
}
