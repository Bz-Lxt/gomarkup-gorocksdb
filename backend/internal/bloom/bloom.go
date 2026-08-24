package bloom

import (
	"encoding/binary"
	"hash/fnv"
	"math"
)

const DefaultBitsPerKey = 10
const DefaultHashes = 7

type Filter struct {
	bits []byte
	k    uint32
	nbit uint32
}

func New(nKeys, bitsPerKey int) *Filter {
	if bitsPerKey <= 0 {
		bitsPerKey = DefaultBitsPerKey
	}
	nbit := uint32(nKeys * bitsPerKey)
	if nbit < 64 {
		nbit = 64
	}
	nbytes := (nbit + 7) / 8
	nbit = nbytes * 8
	k := uint32(DefaultHashes)
	if bitsPerKey > 0 {
		// k ≈ ln(2) * bits/key
		kk := int(math.Round(0.69 * float64(bitsPerKey)))
		if kk < 1 {
			kk = 1
		}
		if kk > 30 {
			kk = 30
		}
		k = uint32(kk)
	}
	return &Filter{bits: make([]byte, nbytes), k: k, nbit: nbit}
}

func hashPair(key []byte) (uint32, uint32) {
	h := fnv.New64a()
	_, _ = h.Write(key)
	sum := h.Sum64()
	return uint32(sum), uint32(sum >> 32)
}

func (f *Filter) Add(key []byte) {
	h1, h2 := hashPair(key)
	for i := uint32(0); i < f.k; i++ {
		pos := (h1 + i*h2) % f.nbit
		f.bits[pos/8] |= 1 << (pos % 8)
	}
}

func (f *Filter) MayContain(key []byte) bool {
	if f == nil || f.nbit == 0 {
		return true
	}
	h1, h2 := hashPair(key)
	for i := uint32(0); i < f.k; i++ {
		pos := (h1 + i*h2) % f.nbit
		if f.bits[pos/8]&(1<<(pos%8)) == 0 {
			return false
		}
	}
	return true
}

func (f *Filter) Encode() []byte {
	out := make([]byte, 8+len(f.bits))
	binary.LittleEndian.PutUint32(out[0:4], f.k)
	binary.LittleEndian.PutUint32(out[4:8], f.nbit)
	copy(out[8:], f.bits)
	return out
}

func Decode(buf []byte) *Filter {
	if len(buf) < 8 {
		return &Filter{}
	}
	k := binary.LittleEndian.Uint32(buf[0:4])
	nbit := binary.LittleEndian.Uint32(buf[4:8])
	bits := append([]byte(nil), buf[8:]...)
	return &Filter{bits: bits, k: k, nbit: nbit}
}

func (f *Filter) BitsPerKey(nKeys int) float64 {
	if nKeys == 0 {
		return 0
	}
	return float64(f.nbit) / float64(nKeys)
}
