package version

import (
	"encoding/binary"
	"fmt"

	"gorocksdb/internal/encoding"
)

const (
	tagComparator  = 1
	tagLogNumber   = 2
	tagNextFile    = 3
	tagLastSeq     = 4
	tagCompactPtr  = 5
	tagDeletedFile = 6
	tagNewFile     = 7
	tagPrevLog     = 9
)

type VersionEdit struct {
	HasLogNumber bool
	LogNumber    uint64
	HasNextFile  bool
	NextFile     uint64
	HasLastSeq   bool
	LastSeq      uint64
	Deleted      []DeletedFile
	Added        []*FileMeta
}

type DeletedFile struct {
	Level  int
	Number uint64
}

func (e *VersionEdit) SetLogNumber(n uint64) { e.HasLogNumber = true; e.LogNumber = n }
func (e *VersionEdit) SetNextFile(n uint64)  { e.HasNextFile = true; e.NextFile = n }
func (e *VersionEdit) SetLastSeq(n uint64)   { e.HasLastSeq = true; e.LastSeq = n }
func (e *VersionEdit) DeleteFile(level int, num uint64) {
	e.Deleted = append(e.Deleted, DeletedFile{Level: level, Number: num})
}
func (e *VersionEdit) AddFile(m *FileMeta) { e.Added = append(e.Added, cloneMeta(m)) }

func (e *VersionEdit) Encode() []byte {
	var buf []byte
	if e.HasLogNumber {
		buf = append(buf, tagLogNumber)
		buf = encoding.AppendUvarint(buf, e.LogNumber)
	}
	if e.HasNextFile {
		buf = append(buf, tagNextFile)
		buf = encoding.AppendUvarint(buf, e.NextFile)
	}
	if e.HasLastSeq {
		buf = append(buf, tagLastSeq)
		buf = encoding.AppendUvarint(buf, e.LastSeq)
	}
	for _, d := range e.Deleted {
		buf = append(buf, tagDeletedFile)
		buf = encoding.AppendUvarint(buf, uint64(d.Level))
		buf = encoding.AppendUvarint(buf, d.Number)
	}
	for _, f := range e.Added {
		buf = append(buf, tagNewFile)
		buf = encoding.AppendUvarint(buf, uint64(f.Level))
		buf = encoding.AppendUvarint(buf, f.Number)
		buf = encoding.AppendUvarint(buf, f.Size)
		buf = encoding.AppendUvarint(buf, f.Entries)
		buf = encoding.AppendUvarint(buf, uint64(len(f.Smallest)))
		buf = append(buf, f.Smallest...)
		buf = encoding.AppendUvarint(buf, uint64(len(f.Largest)))
		buf = append(buf, f.Largest...)
	}
	return buf
}

func DecodeEdit(buf []byte) (*VersionEdit, error) {
	e := &VersionEdit{}
	for len(buf) > 0 {
		tag := buf[0]
		buf = buf[1:]
		switch tag {
		case tagLogNumber:
			v, n := encoding.Uvarint(buf)
			if n <= 0 {
				return nil, encoding.ErrShortRecord
			}
			e.SetLogNumber(v)
			buf = buf[n:]
		case tagNextFile:
			v, n := encoding.Uvarint(buf)
			if n <= 0 {
				return nil, encoding.ErrShortRecord
			}
			e.SetNextFile(v)
			buf = buf[n:]
		case tagLastSeq:
			v, n := encoding.Uvarint(buf)
			if n <= 0 {
				return nil, encoding.ErrShortRecord
			}
			e.SetLastSeq(v)
			buf = buf[n:]
		case tagDeletedFile:
			lv, n1 := encoding.Uvarint(buf)
			if n1 <= 0 {
				return nil, encoding.ErrShortRecord
			}
			num, n2 := encoding.Uvarint(buf[n1:])
			if n2 <= 0 {
				return nil, encoding.ErrShortRecord
			}
			e.DeleteFile(int(lv), num)
			buf = buf[n1+n2:]
		case tagNewFile:
			lv, n := encoding.Uvarint(buf)
			if n <= 0 {
				return nil, encoding.ErrShortRecord
			}
			buf = buf[n:]
			num, n := encoding.Uvarint(buf)
			if n <= 0 {
				return nil, encoding.ErrShortRecord
			}
			buf = buf[n:]
			sz, n := encoding.Uvarint(buf)
			if n <= 0 {
				return nil, encoding.ErrShortRecord
			}
			buf = buf[n:]
			ent, n := encoding.Uvarint(buf)
			if n <= 0 {
				return nil, encoding.ErrShortRecord
			}
			buf = buf[n:]
			sl, n := encoding.Uvarint(buf)
			if n <= 0 || uint64(len(buf)-n) < sl {
				return nil, encoding.ErrShortRecord
			}
			buf = buf[n:]
			small := append([]byte(nil), buf[:sl]...)
			buf = buf[sl:]
			ll, n := encoding.Uvarint(buf)
			if n <= 0 || uint64(len(buf)-n) < ll {
				return nil, encoding.ErrShortRecord
			}
			buf = buf[n:]
			large := append([]byte(nil), buf[:ll]...)
			buf = buf[ll:]
			e.AddFile(&FileMeta{
				Level: int(lv), Number: num, Size: sz, Entries: ent,
				Smallest: small, Largest: large,
			})
		case tagComparator, tagCompactPtr, tagPrevLog:
			// skip unknown-but-known tags with a single varint if possible
			_, n := encoding.Uvarint(buf)
			if n <= 0 {
				return nil, fmt.Errorf("version: skip tag %d: %w", tag, encoding.ErrShortRecord)
			}
			buf = buf[n:]
		default:
			return nil, fmt.Errorf("version: unknown tag %d", tag)
		}
	}
	return e, nil
}

func EncodeRecord(payload []byte) []byte {
	out := make([]byte, 8+len(payload))
	binary.LittleEndian.PutUint32(out[4:], uint32(len(payload)))
	copy(out[8:], payload)
	crc := encoding.CRC32C(out[4:])
	binary.LittleEndian.PutUint32(out[0:4], encoding.MaskCRC(crc))
	return out
}

func DecodeRecord(buf []byte) (payload []byte, n int, err error) {
	if len(buf) < 8 {
		return nil, 0, encoding.ErrShortRecord
	}
	length := binary.LittleEndian.Uint32(buf[4:8])
	need := 8 + int(length)
	if len(buf) < need {
		return nil, 0, encoding.ErrShortRecord
	}
	want := encoding.UnmaskCRC(binary.LittleEndian.Uint32(buf[0:4]))
	got := encoding.CRC32C(buf[4:need])
	if want != got {
		return nil, 0, encoding.ErrChecksum
	}
	return buf[8:need], need, nil
}
