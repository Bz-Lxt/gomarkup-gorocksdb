package api

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"gorocksdb/pkg/gorocksdb"
)

type kvBody struct {
	Value string `json:"value"`
	Sync  *bool  `json:"sync"`
}

func (s *Server) handleGet(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	val, err := s.db.Get([]byte(key))
	if errors.Is(err, gorocksdb.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "NOT_FOUND", "key not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	writeOK(w, map[string]any{"key": key, "value": string(val)})
}

func (s *Server) handlePut(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	var body kvBody
	_ = json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&body)
	if body.Value == "" && r.URL.Query().Get("value") != "" {
		body.Value = r.URL.Query().Get("value")
	}
	if body.Sync != nil {
		s.db.SetSync(*body.Sync)
	}
	if err := s.db.Put([]byte(key), []byte(body.Value)); err != nil {
		if errors.Is(err, gorocksdb.ErrWriteStall) {
			writeErr(w, http.StatusTooManyRequests, "WRITE_STALL", err.Error())
			return
		}
		writeErr(w, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	writeOK(w, map[string]any{"key": key, "value": body.Value})
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if err := s.db.Delete([]byte(key)); err != nil {
		writeErr(w, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	writeOK(w, map[string]any{"key": key, "deleted": true})
}

type batchReq struct {
	Ops []struct {
		Op    string `json:"op"`
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"ops"`
}

func (s *Server) handleBatch(w http.ResponseWriter, r *http.Request) {
	var req batchReq
	if err := json.NewDecoder(io.LimitReader(r.Body, 4<<20)).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}
	b := &gorocksdb.WriteBatch{}
	for _, op := range req.Ops {
		if op.Key == "" {
			writeErr(w, http.StatusBadRequest, "VALIDATION", "empty key")
			return
		}
		switch op.Op {
		case "put", "PUT":
			b.Put([]byte(op.Key), []byte(op.Value))
		case "delete", "DELETE", "del":
			b.Delete([]byte(op.Key))
		default:
			writeErr(w, http.StatusBadRequest, "VALIDATION", "unknown op "+op.Op)
			return
		}
	}
	if err := s.db.Write(b); err != nil {
		writeErr(w, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	writeOK(w, map[string]any{"count": len(req.Ops)})
}

func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	kvs, err := s.db.Scan([]byte(q.Get("start")), []byte(q.Get("end")), limit)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	items := make([]map[string]string, 0, len(kvs))
	for _, kv := range kvs {
		items = append(items, map[string]string{"key": string(kv.Key), "value": string(kv.Value)})
	}
	writeOK(w, map[string]any{"items": items, "count": len(items)})
}

func (s *Server) handleState(w http.ResponseWriter, r *http.Request) {
	writeOK(w, s.db.State())
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	m := s.db.Metrics()
	h, miss, _, used, cap := s.db.CacheStats()
	m["cache_hits"] = h
	m["cache_misses"] = miss
	m["cache_used"] = used
	m["cache_cap"] = cap
	m["cache_hit_rate"] = s.db.CacheHitRate()
	m["event_dropped"] = s.db.Events().Dropped()
	m["bench"] = s.bench.Stats()
	writeOK(w, m)
}

func (s *Server) handleFlush(w http.ResponseWriter, r *http.Request) {
	if err := s.db.Flush(); err != nil {
		writeErr(w, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	writeOK(w, map[string]any{"flushed": true})
}

func (s *Server) handleCompact(w http.ResponseWriter, r *http.Request) {
	if err := s.db.Compact(); err != nil {
		writeErr(w, http.StatusInternalServerError, "INTERNAL", err.Error())
		return
	}
	writeOK(w, map[string]any{"compacted": true})
}

type profileReq struct {
	Profile string `json:"profile"`
	Sync    *bool  `json:"sync"`
}

func (s *Server) handleProfile(w http.ResponseWriter, r *http.Request) {
	var req profileReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}
	if req.Profile != "" {
		s.db.SetProfile(req.Profile)
	}
	if req.Sync != nil {
		s.db.SetSync(*req.Sync)
	}
	writeOK(w, map[string]any{"profile": s.db.ProfileName(), "sync": s.db.SyncWrites()})
}
