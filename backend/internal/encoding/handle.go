package encoding

import "encoding/binary"

type BlockHandle struct {
	Offset uint64
	Size   uint64
}

func EncodeHandle(h BlockHandle) []byte {
	var buf [20]byte
	n := PutUvarint(buf[:], h.Offset)
	n += PutUvarint(buf[n:], h.Size)
	return append([]byte(nil), buf[:n]...)
}

func DecodeHandle(buf []byte) (BlockHandle, int) {
	off, n1 := Uvarint(buf)
	if n1 <= 0 {
		return BlockHandle{}, -1
	}
	sz, n2 := Uvarint(buf[n1:])
	if n2 <= 0 {
		return BlockHandle{}, -1
	}
	return BlockHandle{Offset: off, Size: sz}, n1 + n2
}

func PutFixed64(buf []byte, v uint64) {
	binary.LittleEndian.PutUint64(buf, v)
}

func GetFixed64(buf []byte) uint64 {
	return binary.LittleEndian.Uint64(buf)
}

func PutFixed32(buf []byte, v uint32) {
	binary.LittleEndian.PutUint32(buf, v)
}

func GetFixed32(buf []byte) uint32 {
	return binary.LittleEndian.Uint32(buf)
}
