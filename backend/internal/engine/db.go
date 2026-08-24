package engine

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"gorocksdb/internal/cache"
	"gorocksdb/internal/clock"
	"gorocksdb/internal/config"
	"gorocksdb/internal/encoding"
	"gorocksdb/internal/events"
	"gorocksdb/internal/logger"
	"gorocksdb/internal/memtable"
	"gorocksdb/internal/metrics"
	"gorocksdb/internal/sstable"
	"gorocksdb/internal/version"
	"gorocksdb/internal/wal"
)

var (
	ErrClosed    = errors.New("engine: closed")
	ErrNotFound  = errors.New("engine: not found")
	ErrWriteStall = errors.New("engine: write stall")
)

type DB struct {
	opts Options
	mu   sync.Mutex

	mem *memtable.MemTable
	imm []*memtable.MemTable
	log *wal.Writer

	vs    *version.VersionSet
	cache *cache.LRU
	bus   *events.Bus
	Met   metrics.Metrics

	nextMemID uint64
	seq       atomic.Uint64

	closed     atomic.Bool
	stalling   bool
	compacting bool
	flushing   bool

	bgCh     chan struct{}
	stopCh   chan struct{}
	bgWg     sync.WaitGroup
	readers  sync.Map // file number -> *sstable.Reader
	obsolete []uint64
	snapMu   sync.Mutex
	snaps    []uint64
}

func Open(opts Options) (*DB, error) {
	if opts.Dir == "" {
		return nil, fmt.Errorf("engine: empty dir")
	}
	if opts.Profile.Name == "" {
		opts.Profile = config.Demo()
	}
	if err := os.MkdirAll(opts.Dir, 0o755); err != nil {
		return nil, err
	}
	vs, err := version.Open(opts.Dir)
	if err != nil {
		return nil, err
	}
	db := &DB{
		opts:  opts,
		vs:    vs,
		cache: cache.NewLRU(opts.Profile.BlockCacheBytes),
		bus:   events.New(opts.Profile.EventBuffer),
		bgCh:  make(chan struct{}, 2),
		stopCh: make(chan struct{}),
	}
	db.seq.Store(vs.LastSequence())
	if err := db.recover(); err != nil {
		vs.Close()
		return nil, err
	}
	db.bgWg.Add(2)
	go db.bgFlushLoop()
	go db.bgCompactLoop()
	logger.L().Info("engine opened",
		"dir", opts.Dir,
		"profile", opts.Profile.Name,
		"seq", db.seq.Load(),
		"time", clock.FormatNow(),
	)
	return db, nil
}

func (db *DB) Events() *events.Bus { return db.bus }
func (db *DB) Cache() *cache.LRU   { return db.cache }
func (db *DB) Profile() config.Profile {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.opts.Profile
}

func (db *DB) SetProfile(name string) {
	db.mu.Lock()
	db.opts.Profile = config.Lookup(name)
	db.mu.Unlock()
	db.kickBG()
}

func (db *DB) SetSync(sync bool) {
	db.mu.Lock()
	db.opts.Sync = sync
	db.mu.Unlock()
}

func (db *DB) Sync() bool {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.opts.Sync
}

func (db *DB) Put(key, value []byte) error {
	return db.write([]wal.BatchItem{{Type: encoding.TypeValue, Key: key, Value: value}})
}

func (db *DB) Delete(key []byte) error {
	return db.write([]wal.BatchItem{{Type: encoding.TypeDeletion, Key: key}})
}

func (db *DB) WriteBatch(items []wal.BatchItem) error {
	return db.write(items)
}

func (db *DB) write(items []wal.BatchItem) error {
	if db.closed.Load() {
		return ErrClosed
	}
	if len(items) == 0 {
		return nil
	}
	start := time.Now()
	db.mu.Lock()
	if err := db.maybeStallLocked(); err != nil {
		db.mu.Unlock()
		return err
	}
	if err := db.maybeRotateLocked(); err != nil {
		db.mu.Unlock()
		return err
	}
	seq := db.seq.Add(uint64(len(items)))
	first := seq - uint64(len(items)) + 1
	payload := wal.EncodeBatch(first, items)
	sync := db.opts.Sync
	log := db.log
	log.Ref()
	mem := db.mem
	mem.Ref()
	db.mu.Unlock()

	if err := log.Append(payload, sync); err != nil {
		mem.Unref()
		_ = log.Unref()
		return err
	}
	_ = log.Unref()
	if sync {
		db.Met.WALSyncs.Add(1)
	}
	for i, it := range items {
		mem.Add(first+uint64(i), it.Type, it.Key, it.Value)
		db.Met.BytesWritten.Add(int64(len(it.Key) + len(it.Value)))
		if it.Type == encoding.TypeDeletion {
			db.Met.Deletes.Add(1)
		} else {
			db.Met.Puts.Add(1)
		}
	}
	mem.Unref()
	db.Met.ObserveWrite(time.Since(start).Nanoseconds())

	if mem.ApproximateMemory() >= db.Profile().WriteBufferSize {
		db.mu.Lock()
		_ = db.maybeRotateLocked()
		db.mu.Unlock()
	}
	return nil
}

func (db *DB) maybeStallLocked() error {
	p := db.opts.Profile
	deadline := time.Now().Add(3 * time.Second)
	for {
		v := db.vs.Current()
		l0 := v.NumFiles(0)
		if l0 < p.L0StopTrigger && len(db.imm) < p.MaxImmutable {
			if l0 >= p.L0SlowdownTrigger {
				db.stalling = true
				db.Met.WriteStalls.Add(1)
				db.mu.Unlock()
				time.Sleep(2 * time.Millisecond)
				db.mu.Lock()
			} else {
				db.stalling = false
			}
			return nil
		}
		db.stalling = true
		db.Met.WriteStalls.Add(1)
		if time.Now().After(deadline) {
			return ErrWriteStall
		}
		db.kickBG()
		db.mu.Unlock()
		time.Sleep(5 * time.Millisecond)
		db.mu.Lock()
		if db.closed.Load() {
			return ErrClosed
		}
	}
}

func (db *DB) maybeRotateLocked() error {
	if db.mem == nil {
		return db.newMemLocked()
	}
	if db.mem.ApproximateMemory() < db.opts.Profile.WriteBufferSize {
		return nil
	}
	if len(db.imm) >= db.opts.Profile.MaxImmutable {
		return nil
	}
	old := db.mem
	old.MarkImmutable()
	db.imm = append(db.imm, old)
	oldLog := db.log
	if err := db.newMemLocked(); err != nil {
		return err
	}
	if oldLog != nil {
		_ = oldLog.Unref()
	}
	db.Met.RotateCount.Add(1)
	db.bus.Publish("memtable.rotate", map[string]any{
		"old_id": old.ID(), "new_id": db.mem.ID(),
		"bytes": old.ApproximateMemory(), "entries": old.Len(),
	})
	db.kickBG()
	return nil
}

func (db *DB) newMemLocked() error {
	num := db.vs.NewFileNumber()
	w, err := wal.Create(db.opts.Dir, num)
	if err != nil {
		return err
	}
	db.nextMemID++
	db.mem = memtable.New(db.nextMemID, num)
	db.log = w
	return nil
}

func (db *DB) Get(key []byte) ([]byte, error) {
	return db.GetSnapshot(key, encoding.MaxSequence)
}

func (db *DB) GetSnapshot(key []byte, snapshot uint64) ([]byte, error) {
	if db.closed.Load() {
		return nil, ErrClosed
	}
	db.Met.Gets.Add(1)
	db.mu.Lock()
	mem := db.mem
	mem.Ref()
	imm := make([]*memtable.MemTable, len(db.imm))
	copy(imm, db.imm)
	for _, m := range imm {
		m.Ref()
	}
	db.mu.Unlock()

	defer func() {
		mem.Unref()
		for _, m := range imm {
			m.Unref()
		}
	}()

	if val, found, alive := mem.Get(key, snapshot); found {
		if !alive {
			db.Met.GetMisses.Add(1)
			return nil, ErrNotFound
		}
		db.Met.GetHits.Add(1)
		return append([]byte(nil), val...), nil
	}
	for i := len(imm) - 1; i >= 0; i-- {
		if val, found, alive := imm[i].Get(key, snapshot); found {
			if !alive {
				db.Met.GetMisses.Add(1)
				return nil, ErrNotFound
			}
			db.Met.GetHits.Add(1)
			return append([]byte(nil), val...), nil
		}
	}

	v := db.vs.Current()
	v.Ref()
	defer v.Unref()
	cands := v.GetCandidates(key)
	for _, f := range cands {
		r, err := db.reader(f)
		if err != nil {
			return nil, err
		}
		if !r.MayContain(key) {
			db.Met.BloomRejects.Add(1)
			continue
		}
		db.Met.SSTTouched.Add(1)
		val, found, err := r.Get(key, snapshot)
		if err != nil {
			return nil, err
		}
		if found {
			if val == nil {
				db.Met.GetMisses.Add(1)
				return nil, ErrNotFound
			}
			db.Met.GetHits.Add(1)
			return val, nil
		}
	}
	db.Met.GetMisses.Add(1)
	return nil, ErrNotFound
}

func (db *DB) reader(f *sstable.FileMeta) (*sstable.Reader, error) {
	if v, ok := db.readers.Load(f.Number); ok {
		return v.(*sstable.Reader), nil
	}
	path := f.Path
	if path == "" {
		path = sstable.SSTPath(db.opts.Dir, f.Number)
	}
	r, err := sstable.Open(path, f.Number, db.cache)
	if err != nil {
		return nil, err
	}
	actual, loaded := db.readers.LoadOrStore(f.Number, r)
	if loaded {
		r.Close()
		return actual.(*sstable.Reader), nil
	}
	return r, nil
}

func (db *DB) kickBG() {
	select {
	case db.bgCh <- struct{}{}:
	default:
	}
}

func (db *DB) Close() error {
	if !db.closed.CompareAndSwap(false, true) {
		return nil
	}
	close(db.stopCh)
	db.bgWg.Wait()
	db.mu.Lock()
	if db.log != nil {
		_ = db.log.Unref()
	}
	db.mu.Unlock()
	db.readers.Range(func(_, v any) bool {
		v.(*sstable.Reader).Close()
		return true
	})
	_ = db.vs.Close()
	db.bus.Close()
	return nil
}

func (db *DB) Dir() string { return db.opts.Dir }

func (db *DB) NewSnapshot() uint64 {
	s := db.seq.Load()
	db.snapMu.Lock()
	db.snaps = append(db.snaps, s)
	db.snapMu.Unlock()
	return s
}

func (db *DB) ReleaseSnapshot(s uint64) {
	db.snapMu.Lock()
	defer db.snapMu.Unlock()
	out := db.snaps[:0]
	for _, x := range db.snaps {
		if x != s {
			out = append(out, x)
		}
	}
	db.snaps = out
}

func (db *DB) oldestSnapshot() uint64 {
	db.snapMu.Lock()
	defer db.snapMu.Unlock()
	if len(db.snaps) == 0 {
		return encoding.MaxSequence
	}
	min := db.snaps[0]
	for _, s := range db.snaps[1:] {
		if s < min {
			min = s
		}
	}
	return min
}

func WALPath(dir string, n uint64) string {
	return filepath.Join(dir, fmt.Sprintf("%06d.log", n))
}
