package wal

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"

	"gorocksdb/internal/encoding"
)

var ErrCorruptTail = errors.New("wal: corrupt tail truncated")

type Reader struct {
	f      *os.File
	block  []byte
	off    int
	n      int
	eof    bool
	trunc  bool
}

func OpenReader(path string) (*Reader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	return &Reader{f: f, block: make([]byte, BlockSize)}, nil
}

func (r *Reader) Truncated() bool { return r.trunc }

func (r *Reader) Close() error { return r.f.Close() }

func (r *Reader) fill() error {
	if r.eof {
		return io.EOF
	}
	n, err := io.ReadFull(r.f, r.block)
	r.n = n
	r.off = 0
	if err == io.EOF || err == io.ErrUnexpectedEOF {
		r.eof = true
		if n == 0 {
			return io.EOF
		}
		return nil
	}
	return err
}

func (r *Reader) Next() ([]byte, error) {
	var assembled []byte
	for {
		if r.off+headerSize > r.n {
			if r.eof {
				if len(assembled) > 0 {
					r.trunc = true
					return nil, ErrCorruptTail
				}
				if r.n == 0 {
					return nil, io.EOF
				}
				// leftover pad
				if err := r.fill(); err != nil {
					if errors.Is(err, io.EOF) && len(assembled) == 0 {
						return nil, io.EOF
					}
					return nil, err
				}
				continue
			}
			if err := r.fill(); err != nil {
				if errors.Is(err, io.EOF) && len(assembled) == 0 {
					return nil, io.EOF
				}
				return nil, err
			}
			if r.off+headerSize > r.n {
				if len(assembled) > 0 {
					r.trunc = true
					return nil, ErrCorruptTail
				}
				return nil, io.EOF
			}
		}
		hdr := r.block[r.off:]
		if r.off+headerSize > r.n {
			r.trunc = true
			return nil, ErrCorruptTail
		}
		length := binary.LittleEndian.Uint32(hdr[4:8])
		typ := hdr[8]
		if typ == 0 && length == 0 {
			// padding to end of block
			r.off = r.n
			continue
		}
		need := headerSize + int(length)
		if r.off+need > r.n {
			r.trunc = true
			return nil, ErrCorruptTail
		}
		payload := r.block[r.off+headerSize : r.off+need]
		want := encoding.UnmaskCRC(binary.LittleEndian.Uint32(hdr[:4]))
		got := encoding.CRC32C(r.block[r.off+4 : r.off+need])
		if want != got {
			r.trunc = true
			return nil, fmt.Errorf("%w: crc mismatch", ErrCorruptTail)
		}
		r.off += need
		switch typ {
		case RecFull:
			if len(assembled) > 0 {
				r.trunc = true
				return nil, ErrCorruptTail
			}
			out := make([]byte, len(payload))
			copy(out, payload)
			return out, nil
		case RecFirst:
			assembled = append(assembled[:0], payload...)
		case RecMiddle:
			if assembled == nil {
				r.trunc = true
				return nil, ErrCorruptTail
			}
			assembled = append(assembled, payload...)
		case RecLast:
			if assembled == nil {
				r.trunc = true
				return nil, ErrCorruptTail
			}
			assembled = append(assembled, payload...)
			return assembled, nil
		default:
			r.trunc = true
			return nil, ErrCorruptTail
		}
	}
}

func Replay(path string, fn func([]byte) error) (truncated bool, err error) {
	r, err := OpenReader(path)
	if err != nil {
		return false, err
	}
	defer r.Close()
	for {
		rec, e := r.Next()
		if e != nil {
			if errors.Is(e, io.EOF) {
				return r.Truncated(), nil
			}
			if errors.Is(e, ErrCorruptTail) {
				return true, nil
			}
			return r.Truncated(), e
		}
		if err := fn(rec); err != nil {
			if errors.Is(err, encoding.ErrShortRecord) && r.eof {
				return true, nil
			}
			return r.Truncated(), err
		}
	}
}
