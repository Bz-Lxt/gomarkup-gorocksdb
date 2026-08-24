package api

import (
	"net/http"
	"strings"
	"time"

	"gorocksdb/internal/clock"
	"gorocksdb/internal/events"
	"gorocksdb/internal/logger"
	"gorocksdb/pkg/gorocksdb"
)

type Server struct {
	db        *gorocksdb.DB
	mux       *http.ServeMux
	whitelist []string
	bench     *Bench
	started   time.Time
	hub       *events.Hub
}

func New(db *gorocksdb.DB, cors string) *Server {
	var wl []string
	for _, p := range strings.Split(cors, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			wl = append(wl, p)
		}
	}
	s := &Server{
		db: db, mux: http.NewServeMux(), whitelist: wl,
		started: clock.Now(), bench: NewBench(db),
		hub: events.NewHub(db.Events()),
	}
	s.routes()
	return s
}

func (s *Server) routes() {
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
	s.mux.HandleFunc("GET /api/kv/{key}", s.handleGet)
	s.mux.HandleFunc("PUT /api/kv/{key}", s.handlePut)
	s.mux.HandleFunc("DELETE /api/kv/{key}", s.handleDelete)
	s.mux.HandleFunc("POST /api/batch", s.handleBatch)
	s.mux.HandleFunc("GET /api/scan", s.handleScan)
	s.mux.HandleFunc("GET /api/lsm/state", s.handleState)
	s.mux.HandleFunc("GET /api/metrics", s.handleMetrics)
	s.mux.HandleFunc("POST /api/bench/start", s.handleBenchStart)
	s.mux.HandleFunc("POST /api/bench/stop", s.handleBenchStop)
	s.mux.HandleFunc("POST /api/admin/flush", s.handleFlush)
	s.mux.HandleFunc("POST /api/admin/compact", s.handleCompact)
	s.mux.HandleFunc("POST /api/admin/profile", s.handleProfile)
	s.mux.HandleFunc("GET /ws/events", s.handleWS)
}

func (s *Server) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodOptions {
			s.cors(w, r)
			w.WriteHeader(http.StatusNoContent)
			return
		}
		s.cors(w, r)
		s.mux.ServeHTTP(w, r)
	})
}

func (s *Server) cors(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	if origin != "" && allowOrigin(r, s.whitelist) {
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Methods", "GET,PUT,POST,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	}
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeOK(w, map[string]any{
		"status":  "ok",
		"time":    clock.FormatNow(),
		"profile": s.db.ProfileName(),
		"uptime_s": int(clock.Now().Sub(s.started).Seconds()),
	})
}

func (s *Server) Listen(addr string) error {
	srv := &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	logger.L().Info("http listen", "addr", addr)
	return srv.ListenAndServe()
}
