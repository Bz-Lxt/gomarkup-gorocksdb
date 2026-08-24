package sstable

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"

	"gorocksdb/internal/bloom"
	"gorocksdb/internal/config"
	"gorocksdb/internal/encoding"
)

type Writer struct {
	f          *os.File
	path       string
	number     uint64
	offset     uint64
	data       *blockBuilder
	index      *blockBuilder
	blockSize  int
	restartN   int
	lastKey    []byte
	smallest   []byte
	largest    []byte
	pending    bool
	pendHandle encoding.BlockHandle
	filter     *bloom.Filter
	keys       int
	entries    uint64
	dataBytes  uint64
}

func Create(dir string, number uint64, p config.Profile, estimateKeys int) (*Writer, error) {
	path := filepath.Join(dir, fmt.Sprintf("%06d.sst", number))
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, err
	}
	if estimateKeys < 16 {
		estimateKeys = 16
	}
	return &Writer{
		f:         f,
		path:      path,
		number:    number,
		data:      newBlockBuilder(p.RestartInterval),
		index:     newBlockBuilder(p.RestartInterval),
		blockSize: p.BlockSize,
		restartN:  p.RestartInterval,
		filter:    bloom.New(estimateKeys, p.BloomBitsPerKey),
	}, nil
}

func (w *Writer) Add(key, value []byte) error {
	if w.pending {
		w.index.Add(w.lastKey, encodeIndexValue(w.pendHandle))
		w.pending = false
	}
	if w.smallest == nil {
		w.smallest = bytesClone(key)
	}
	w.largest = bytesClone(key)
	w.filter.Add(encoding.UserKey(key))
	w.data.Add(key, value)
	w.entries++
	w.keys++
	if w.data.Size() >= w.blockSize {
		return w.flushData()
	}
	w.lastKey = append(w.lastKey[:0], key...)
	return nil
}

func (w *Writer) flushData() error {
	if w.data.Empty() {
		return nil
	}
	raw := wrapBlock(w.data.Finish())
	n, err := w.f.Write(raw)
	if err != nil {
		return err
	}
	w.pendHandle = encoding.BlockHandle{Offset: w.offset, Size: uint64(n - blockTrailerLen)}
	w.offset += uint64(n)
	w.dataBytes += uint64(n)
	w.pending = true
	w.lastKey = append(w.lastKey[:0], w.data.lastKey...)
	w.data.Reset()
	return nil
}

func (w *Writer) Finish() (*FileMeta, error) {
	if err := w.flushData(); err != nil {
		return nil, err
	}
	if w.pending {
		w.index.Add(w.lastKey, encodeIndexValue(w.pendHandle))
		w.pending = false
	}

	filterRaw := wrapBlock(w.filter.Encode())
	filterOff := w.offset
	if _, err := w.f.Write(filterRaw); err != nil {
		return nil, err
	}
	filterHandle := encoding.BlockHandle{Offset: filterOff, Size: uint64(len(filterRaw) - blockTrailerLen)}
	w.offset += uint64(len(filterRaw))

	meta := newBlockBuilder(w.restartN)
	meta.Add([]byte("filter.bloom"), encodeIndexValue(filterHandle))
	metaRaw := wrapBlock(meta.Finish())
	metaOff := w.offset
	if _, err := w.f.Write(metaRaw); err != nil {
		return nil, err
	}
	metaHandle := encoding.BlockHandle{Offset: metaOff, Size: uint64(len(metaRaw) - blockTrailerLen)}
	w.offset += uint64(len(metaRaw))

	idxRaw := wrapBlock(w.index.Finish())
	idxOff := w.offset
	if _, err := w.f.Write(idxRaw); err != nil {
		return nil, err
	}
	idxHandle := encoding.BlockHandle{Offset: idxOff, Size: uint64(len(idxRaw) - blockTrailerLen)}
	w.offset += uint64(len(idxRaw))

	footer := make([]byte, config.FooterSize)
	mh := encoding.EncodeHandle(metaHandle)
	ih := encoding.EncodeHandle(idxHandle)
	copy(footer[0:], mh)
	copy(footer[20:], ih)
	binary.LittleEndian.PutUint64(footer[config.FooterSize-8:], config.MagicNumber)
	if _, err := w.f.Write(footer); err != nil {
		return nil, err
	}
	if err := w.f.Sync(); err != nil {
		return nil, err
	}
	if err := w.f.Close(); err != nil {
		return nil, err
	}
	info, err := os.Stat(w.path)
	if err != nil {
		return nil, err
	}
	return &FileMeta{
		Number:   w.number,
		Size:     uint64(info.Size()),
		Smallest: bytesClone(w.smallest),
		Largest:  bytesClone(w.largest),
		Entries:  w.entries,
		Path:     w.path,
	}, nil
}

func (w *Writer) ApproxSize() int64 {
	return int64(w.offset) + int64(w.data.Size())
}

func (w *Writer) Abort() {
	_ = w.f.Close()
	_ = os.Remove(w.path)
}

type FileMeta struct {
	Number   uint64 `json:"number"`
	Level    int    `json:"level"`
	Size     uint64 `json:"size"`
	Smallest []byte `json:"-"`
	Largest  []byte `json:"-"`
	Entries  uint64 `json:"entries"`
	Path     string `json:"path,omitempty"`
}

func (m *FileMeta) SmallestUser() string {
	return string(encoding.UserKey(m.Smallest))
}

func (m *FileMeta) LargestUser() string {
	return string(encoding.UserKey(m.Largest))
}

func (m *FileMeta) Overlaps(smallest, largest []byte) bool {
	if encoding.CompareInternal(m.Largest, smallest) < 0 {
		return false
	}
	if encoding.CompareInternal(m.Smallest, largest) > 0 {
		return false
	}
	return true
}

func (m *FileMeta) OverlapsUser(ukSmall, ukLarge []byte) bool {
	if encoding.CompareUser(encoding.UserKey(m.Largest), ukSmall) < 0 {
		return false
	}
	if encoding.CompareUser(encoding.UserKey(m.Smallest), ukLarge) > 0 {
		return false
	}
	return true
}

func SSTPath(dir string, number uint64) string {
	return filepath.Join(dir, fmt.Sprintf("%06d.sst", number))
}
