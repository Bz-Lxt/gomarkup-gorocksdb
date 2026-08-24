package wal

import (
	"os"
	"sync"
)

func OpenAppend(path string, number uint64) (*Writer, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	// start a fresh block after existing bytes; leftover partial block is
	// already durable. new records begin at next write.
	_ = st
	return &Writer{
		f:         f,
		path:      path,
		block:     make([]byte, BlockSize),
		number:    number,
		closeOnce: sync.Once{},
		refs:      1,
	}, nil
}
