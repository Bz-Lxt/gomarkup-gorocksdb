package wal

import (
	"encoding/binary"

	"gorocksdb/internal/encoding"
)

// Record layout: seq(8) | count(4) | repeated {type(1) keylen varint | key | vallen varint | val}
func EncodeBatch(seq uint64, items []BatchItem) []byte {
	buf := make([]byte, 12)
	binary.LittleEndian.PutUint64(buf[0:8], seq)
	binary.LittleEndian.PutUint32(buf[8:12], uint32(len(items)))
	for _, it := range items {
		buf = append(buf, byte(it.Type))
		buf = encoding.AppendUvarint(buf, uint64(len(it.Key)))
		buf = append(buf, it.Key...)
		buf = encoding.AppendUvarint(buf, uint64(len(it.Value)))
		buf = append(buf, it.Value...)
	}
	return buf
}

type BatchItem struct {
	Type  encoding.ValueType
	Key   []byte
	Value []byte
}

func DecodeBatch(rec []byte) (seq uint64, items []BatchItem, err error) {
	if len(rec) < 12 {
		return 0, nil, encoding.ErrShortRecord
	}
	seq = binary.LittleEndian.Uint64(rec[0:8])
	n := binary.LittleEndian.Uint32(rec[8:12])
	p := rec[12:]
	items = make([]BatchItem, 0, n)
	for i := uint32(0); i < n; i++ {
		if len(p) < 1 {
			return 0, nil, encoding.ErrShortRecord
		}
		typ := encoding.ValueType(p[0])
		p = p[1:]
		klen, ksz := encoding.Uvarint(p)
		if ksz <= 0 || uint64(len(p)-ksz) < klen {
			return 0, nil, encoding.ErrShortRecord
		}
		p = p[ksz:]
		key := append([]byte(nil), p[:klen]...)
		p = p[klen:]
		vlen, vsz := encoding.Uvarint(p)
		if vsz <= 0 || uint64(len(p)-vsz) < vlen {
			return 0, nil, encoding.ErrShortRecord
		}
		p = p[vsz:]
		val := append([]byte(nil), p[:vlen]...)
		p = p[vlen:]
		items = append(items, BatchItem{Type: typ, Key: key, Value: val})
	}
	return seq, items, nil
}
