package api

import (
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"gorocksdb/internal/clock"
	"gorocksdb/pkg/gorocksdb"
)

type Bench struct {
	db      *gorocksdb.DB
	mu      sync.Mutex
	running atomic.Bool
	stop    chan struct{}
	wg      sync.WaitGroup
	ops     atomic.Int64
	errs    atomic.Int64
	started time.Time
	workers int
	qps     int
	valSize int
}

func NewBench(db *gorocksdb.DB) *Bench { return &Bench{db: db} }

type benchReq struct {
	Workers int `json:"workers"`
	QPS     int `json:"qps"`
	ValSize int `json:"value_size"`
}

func (s *Server) handleBenchStart(w http.ResponseWriter, r *http.Request) {
	var req benchReq
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.Workers <= 0 {
		req.Workers = 8
	}
	if req.Workers > 64 {
		req.Workers = 64
	}
	if req.ValSize <= 0 {
		req.ValSize = 100
	}
	if req.ValSize > 4096 {
		req.ValSize = 4096
	}
	s.bench.Start(req.Workers, req.QPS, req.ValSize)
	writeOK(w, s.bench.Stats())
}

func (s *Server) handleBenchStop(w http.ResponseWriter, r *http.Request) {
	s.bench.Stop()
	writeOK(w, s.bench.Stats())
}

func (b *Bench) Start(workers, qps, valSize int) {
	b.Stop()
	b.mu.Lock()
	b.stop = make(chan struct{})
	b.workers = workers
	b.qps = qps
	b.valSize = valSize
	b.started = clock.Now()
	b.ops.Store(0)
	b.errs.Store(0)
	b.running.Store(true)
	stop := b.stop
	b.mu.Unlock()

	b.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go b.worker(i, qps, valSize, stop)
	}
}

func (b *Bench) worker(id, qps, valSize int, stop <-chan struct{}) {
	defer b.wg.Done()
	val := make([]byte, valSize)
	for i := range val {
		val[i] = byte('a' + i%26)
	}
	var ticker *time.Ticker
	if qps > 0 && b.workers > 0 {
		per := qps / b.workers
		if per < 1 {
			per = 1
		}
		ticker = time.NewTicker(time.Second / time.Duration(per))
		defer ticker.Stop()
	}
	n := 0
	for {
		if ticker != nil {
			select {
			case <-stop:
				return
			case <-ticker.C:
			}
		} else {
			select {
			case <-stop:
				return
			default:
			}
		}
		key := []byte(fmt.Sprintf("k-%d-%d-%d", id, n, rand.IntN(1<<20)))
		if err := b.db.Put(key, val); err != nil {
			b.errs.Add(1)
		} else {
			b.ops.Add(1)
		}
		n++
	}
}

func (b *Bench) Stop() {
	b.mu.Lock()
	if b.stop != nil {
		// Closing the channel wakes every worker simultaneously.
		// A single send would only notify one worker, leaving the rest
		// running indefinitely and leaking writes into the business DB.
		close(b.stop)
		b.stop = nil
		b.running.Store(false)
		b.mu.Unlock()
		// Block until every worker has actually exited, so no Put
		// can land after this returns. This also makes repeated
		// start/stop cycles safe: Start() calls Stop() and must not
		// return before the previous round's workers are gone.
		b.wg.Wait()
		return
	}
	b.running.Store(false)
	b.mu.Unlock()
	b.wg.Wait()
}

func (b *Bench) Stats() map[string]any {
	elapsed := 0.0
	if b.running.Load() {
		elapsed = clock.Now().Sub(b.started).Seconds()
	}
	ops := b.ops.Load()
	qps := 0.0
	if elapsed > 0 {
		qps = float64(ops) / elapsed
	}
	return map[string]any{
		"running":    b.running.Load(),
		"ops":        ops,
		"errors":     b.errs.Load(),
		"qps":        qps,
		"workers":    b.workers,
		"target_qps": b.qps,
		"value_size": b.valSize,
		"elapsed_s":  elapsed,
	}
}
