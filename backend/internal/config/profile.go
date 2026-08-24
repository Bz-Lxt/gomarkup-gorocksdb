package config

import "strings"

const (
	NumLevels = 7
	WALBlockSize = 32 * 1024
	FooterSize = 48
	MagicNumber uint64 = 0x474F524B53444231 // "GORKSDB1"
)

type Profile struct {
	Name                 string
	WriteBufferSize      int64
	L0CompactionTrigger  int
	L0SlowdownTrigger    int
	L0StopTrigger        int
	MaxLevels            int
	MaxBytesForLevelBase int64
	TargetFileSize       int64
	BlockSize            int
	BlockCacheBytes      int64
	BloomBitsPerKey      int
	RestartInterval      int
	MaxImmutable         int
	EventBuffer          int
	MetricsIntervalMS    int
}

func Production() Profile {
	return Profile{
		Name:                 "production",
		WriteBufferSize:      64 << 20,
		L0CompactionTrigger:  4,
		L0SlowdownTrigger:    8,
		L0StopTrigger:        12,
		MaxLevels:            NumLevels,
		MaxBytesForLevelBase: 10 << 20,
		TargetFileSize:       2 << 20,
		BlockSize:            4096,
		BlockCacheBytes:      8 << 20,
		BloomBitsPerKey:      10,
		RestartInterval:      16,
		MaxImmutable:         2,
		EventBuffer:          256,
		MetricsIntervalMS:    100,
	}
}

func Demo() Profile {
	return Profile{
		Name:                 "demo",
		WriteBufferSize:      256 << 10,
		L0CompactionTrigger:  3,
		L0SlowdownTrigger:    6,
		L0StopTrigger:        8,
		MaxLevels:            NumLevels,
		MaxBytesForLevelBase: 1 << 20,
		TargetFileSize:       256 << 10,
		BlockSize:            4096,
		BlockCacheBytes:      8 << 20,
		BloomBitsPerKey:      10,
		RestartInterval:      16,
		MaxImmutable:         2,
		EventBuffer:          256,
		MetricsIntervalMS:    100,
	}
}

func Test() Profile {
	return Profile{
		Name:                 "test",
		WriteBufferSize:      4 << 10,
		L0CompactionTrigger:  2,
		L0SlowdownTrigger:    4,
		L0StopTrigger:        6,
		MaxLevels:            NumLevels,
		MaxBytesForLevelBase: 16 << 10,
		TargetFileSize:       8 << 10,
		BlockSize:            1024,
		BlockCacheBytes:      1 << 20,
		BloomBitsPerKey:      10,
		RestartInterval:      16,
		MaxImmutable:         2,
		EventBuffer:          64,
		MetricsIntervalMS:    50,
	}
}

func Lookup(name string) Profile {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "production", "prod":
		return Production()
	case "test":
		return Test()
	default:
		return Demo()
	}
}

func (p Profile) MaxBytesForLevel(level int) int64 {
	if level <= 1 {
		return p.MaxBytesForLevelBase
	}
	n := p.MaxBytesForLevelBase
	for i := 1; i < level; i++ {
		if n > (1 << 60) {
			return 1 << 60
		}
		n *= 10
	}
	return n
}
