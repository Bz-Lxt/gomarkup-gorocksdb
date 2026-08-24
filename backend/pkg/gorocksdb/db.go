package gorocksdb

import (
	"gorocksdb/internal/encoding"
	"gorocksdb/internal/engine"
	"gorocksdb/internal/events"
	"gorocksdb/internal/metrics"
	"gorocksdb/internal/wal"
)

// DB is the embedded LSM engine (形态 A).
type DB struct {
	inner *engine.DB
}

func Open(opts Options) (*DB, error) {
	in, err := engine.Open(opts.toEngine())
	if err != nil {
		return nil, err
	}
	return &DB{inner: in}, nil
}

func (db *DB) Put(key, value []byte) error { return db.inner.Put(key, value) }
func (db *DB) Delete(key []byte) error     { return db.inner.Delete(key) }
func (db *DB) Get(key []byte) ([]byte, error) {
	return db.inner.Get(key)
}

func (db *DB) GetAt(key []byte, snap *Snapshot) ([]byte, error) {
	if snap == nil {
		return db.inner.Get(key)
	}
	return db.inner.GetSnapshot(key, snap.seq)
}

func (db *DB) Scan(start, end []byte, limit int) ([]engine.KV, error) {
	return db.inner.Scan(start, end, limit)
}

func (db *DB) Write(batch *WriteBatch) error {
	items := make([]wal.BatchItem, 0, len(batch.items))
	for _, it := range batch.items {
		typ := encoding.TypeValue
		if it.Delete {
			typ = encoding.TypeDeletion
		}
		items = append(items, wal.BatchItem{Type: typ, Key: it.Key, Value: it.Value})
	}
	return db.inner.WriteBatch(items)
}

func (db *DB) Snapshot() *Snapshot {
	return &Snapshot{seq: db.inner.NewSnapshot(), db: db}
}

func (db *DB) Flush() error   { return db.inner.Flush() }
func (db *DB) Compact() error { return db.inner.Compact() }
func (db *DB) Close() error   { return db.inner.Close() }

func (db *DB) State() engine.LSMState          { return db.inner.SnapshotState() }
func (db *DB) Metrics() map[string]any         { return db.inner.Met.Snapshot() }
func (db *DB) Events() *events.Bus             { return db.inner.Events() }
func (db *DB) SetProfile(name string)          { db.inner.SetProfile(name) }
func (db *DB) ProfileName() string             { return db.inner.Profile().Name }
func (db *DB) SetSync(sync bool)               { db.inner.SetSync(sync) }
func (db *DB) SyncWrites() bool                { return db.inner.Sync() }
func (db *DB) Raw() *engine.DB                 { return db.inner }
func (db *DB) MetricsPtr() *metrics.Metrics    { return &db.inner.Met }
func (db *DB) CacheHitRate() float64           { return db.inner.Cache().HitRate() }
func (db *DB) CacheStats() (h, m, ins, used, cap int64) {
	return db.inner.Cache().Stats()
}
