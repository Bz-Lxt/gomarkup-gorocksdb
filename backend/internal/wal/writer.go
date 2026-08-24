package wal

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gorocksdb/internal/encoding"
)

type Writer struct {
	mu        sync.Mutex
	f         *os.File
	path      string
	block     []byte
	off       int
	number    uint64
	closed    bool
	closeOnce sync.Once
	refs      int32
}

func Create(dir string, number uint64) (*Writer, error) {
	path := filepath.Join(dir, fmt.Sprintf("%06d.log", number))
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, err
	}
	w := &Writer{
		f:      f,
		path:   path,
		block:  make([]byte, BlockSize),
		number: number,
		refs:   1,
	}
	return w, nil
}

func (w *Writer) Ref() {
	w.mu.Lock()
	w.refs++
	w.mu.Unlock()
}

func (w *Writer) Unref() error {
	w.mu.Lock()
	w.refs--
	n := w.refs
	w.mu.Unlock()
	if n <= 0 {
		return w.Close()
	}
	return nil
}

func (w *Writer) Number() uint64 { return w.number }
func (w *Writer) Path() string   { return w.path }

func (w *Writer) Append(payload []byte, sync bool) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.closed {
		return fmt.Errorf("wal: writer closed")
	}
	left := payload
	first := true
	for len(left) > 0 {
		avail := BlockSize - w.off
		if avail < headerSize {
			if err := w.flushPad(); err != nil {
				return err
			}
			avail = BlockSize
		}
		frag := avail - headerSize
		if frag > len(left) {
			frag = len(left)
		}
		var typ byte
		switch {
		case first && frag == len(left):
			typ = RecFull
		case first:
			typ = RecFirst
		case frag == len(left):
			typ = RecLast
		default:
			typ = RecMiddle
		}
		if err := w.emit(typ, left[:frag]); err != nil {
			return err
		}
		left = left[frag:]
		first = false
	}
	if sync {
		if err := w.f.Sync(); err != nil {
			return err
		}
	}
	return nil
}

func (w *Writer) emit(typ byte, frag []byte) error {
	start := w.off
	binary.LittleEndian.PutUint32(w.block[start+4:], uint32(len(frag)))
	w.block[start+8] = typ
	copy(w.block[start+headerSize:], frag)
	crc := encoding.CRC32C(w.block[start+4 : start+headerSize+len(frag)])
	binary.LittleEndian.PutUint32(w.block[start:], encoding.MaskCRC(crc))
	w.off += headerSize + len(frag)
	if w.off == BlockSize {
		return w.flushFull()
	}
	return nil
}

func (w *Writer) flushPad() error {
	for i := w.off; i < BlockSize; i++ {
		w.block[i] = 0
	}
	if _, err := w.f.Write(w.block[:BlockSize]); err != nil {
		return err
	}
	w.off = 0
	return nil
}

func (w *Writer) flushFull() error {
	if _, err := w.f.Write(w.block[:BlockSize]); err != nil {
		return err
	}
	w.off = 0
	return nil
}

func (w *Writer) Sync() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.off > 0 {
		if _, err := w.f.Write(w.block[:w.off]); err != nil {
			return err
		}
		w.off = 0
	}
	return w.f.Sync()
}

func (w *Writer) Close() error {
	var err error
	w.closeOnce.Do(func() {
		w.mu.Lock()
		defer w.mu.Unlock()
		w.closed = true
		if w.off > 0 {
			if _, e := w.f.Write(w.block[:w.off]); e != nil {
				err = e
			}
		}
		if e := w.f.Sync(); e != nil && err == nil {
			err = e
		}
		if e := w.f.Close(); e != nil && err == nil {
			err = e
		}
	})
	return err
}
