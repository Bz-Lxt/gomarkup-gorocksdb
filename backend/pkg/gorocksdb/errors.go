package gorocksdb

import (
	"errors"

	"gorocksdb/internal/engine"
)

var (
	ErrNotFound   = engine.ErrNotFound
	ErrClosed     = engine.ErrClosed
	ErrWriteStall = engine.ErrWriteStall
)

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
