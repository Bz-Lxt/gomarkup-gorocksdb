package api

import (
	"net/http"
	"time"

	"github.com/gorilla/websocket"

	"gorocksdb/internal/logger"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 4096,
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(req *http.Request) bool {
		return allowOrigin(req, s.whitelist)
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.L().Warn("ws upgrade", "err", err)
		return
	}
	defer conn.Close()

	snap := map[string]any{"type": "lsm.snapshot", "payload": s.db.State()}
	if err := conn.WriteJSON(snap); err != nil {
		return
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	evCh := s.hub.Subscribe()
	defer s.hub.Unsubscribe(evCh)
	tick := time.NewTicker(100 * time.Millisecond)
	defer tick.Stop()

	for {
		select {
		case <-done:
			return
		case ev, ok := <-evCh:
			if !ok {
				return
			}
			if err := conn.WriteJSON(ev); err != nil {
				return
			}
		case <-tick.C:
			payload := s.db.Metrics()
			payload["lsm"] = s.db.State()
			payload["bench"] = s.bench.Stats()
			msg := map[string]any{"type": "metrics.tick", "payload": payload}
			if err := conn.WriteJSON(msg); err != nil {
				return
			}
		}
	}
}
