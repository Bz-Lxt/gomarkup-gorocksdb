package wal

import "gorocksdb/internal/config"

const (
	headerSize = 9 // crc32 + length32 + type

	RecFull   = 1
	RecFirst  = 2
	RecMiddle = 3
	RecLast   = 4
)

const BlockSize = config.WALBlockSize
