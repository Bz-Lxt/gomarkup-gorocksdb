package engine

import "gorocksdb/internal/config"

type Options struct {
	Dir     string
	Profile config.Profile
	Sync    bool
}

func DefaultOptions(dir string) Options {
	return Options{Dir: dir, Profile: config.Demo(), Sync: false}
}
