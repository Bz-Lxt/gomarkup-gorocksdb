package encoding

import (
	"bytes"
	"encoding/binary"
	"errors"
)

var (
	ErrShortRecord = errors.New("encoding: short record")
	ErrCorrupt     = errors.New("encoding: corrupt data")
	ErrChecksum    = errors.New("encoding: checksum mismatch")
)

type ValueType uint8

const (
	TypeDeletion ValueType = 0
	TypeValue    ValueType = 1
)

const (
	PackSize     = 8
	MaxSequence  = (uint64(1) << 56) - 1
	MaxKeyLength = 1 << 20
	MaxValLength = 4 << 20
)

func Pack(seq uint64, typ ValueType) uint64 {
	if seq > MaxSequence {
		seq = MaxSequence
	}
	return (seq << 8) | uint64(typ)
}

func Unpack(p uint64) (seq uint64, typ ValueType) {
	return p >> 8, ValueType(p & 0xff)
}

func EncodeInternalKey(userKey []byte, seq uint64, typ ValueType) []byte {
	buf := make([]byte, len(userKey)+PackSize)
	copy(buf, userKey)
	binary.LittleEndian.PutUint64(buf[len(userKey):], Pack(seq, typ))
	return buf
}

func AppendInternalKey(dst, userKey []byte, seq uint64, typ ValueType) []byte {
	dst = append(dst, userKey...)
	var pack [PackSize]byte
	binary.LittleEndian.PutUint64(pack[:], Pack(seq, typ))
	return append(dst, pack[:]...)
}

func SplitInternalKey(ikey []byte) (userKey []byte, seq uint64, typ ValueType, ok bool) {
	if len(ikey) < PackSize {
		return nil, 0, 0, false
	}
	userKey = ikey[:len(ikey)-PackSize]
	p := binary.LittleEndian.Uint64(ikey[len(ikey)-PackSize:])
	seq, typ = Unpack(p)
	return userKey, seq, typ, true
}

func UserKey(ikey []byte) []byte {
	if len(ikey) < PackSize {
		return ikey
	}
	return ikey[:len(ikey)-PackSize]
}

func CompareInternal(a, b []byte) int {
	ukA := UserKey(a)
	ukB := UserKey(b)
	if c := bytes.Compare(ukA, ukB); c != 0 {
		return c
	}
	if len(a) < PackSize || len(b) < PackSize {
		return len(a) - len(b)
	}
	pa := binary.LittleEndian.Uint64(a[len(a)-PackSize:])
	pb := binary.LittleEndian.Uint64(b[len(b)-PackSize:])
	sa, ta := Unpack(pa)
	sb, tb := Unpack(pb)
	if sa > sb {
		return -1
	}
	if sa < sb {
		return 1
	}
	if ta < tb {
		return -1
	}
	if ta > tb {
		return 1
	}
	return 0
}

func CompareUser(a, b []byte) int {
	return bytes.Compare(a, b)
}

func SeekKey(userKey []byte) []byte {
	return EncodeInternalKey(userKey, MaxSequence, TypeDeletion)
}
