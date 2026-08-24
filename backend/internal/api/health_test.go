package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gorocksdb/pkg/gorocksdb"
)

func TestHealthAndKV(t *testing.T) {
	db, err := gorocksdb.Open(gorocksdb.Options{Dir: t.TempDir(), Profile: "test"})
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := New(db, "http://localhost:28741")
	req := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	rr := httptest.NewRecorder()
	s.Handler().ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("health %d %s", rr.Code, rr.Body.String())
	}
	var wrap map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &wrap); err != nil || wrap["ok"] != true {
		t.Fatalf("%s", rr.Body.String())
	}

	put := httptest.NewRequest(http.MethodPut, "/api/kv/foo?value=bar", nil)
	pr := httptest.NewRecorder()
	s.Handler().ServeHTTP(pr, put)
	if pr.Code != 200 {
		t.Fatalf("put %d %s", pr.Code, pr.Body.String())
	}
	get := httptest.NewRequest(http.MethodGet, "/api/kv/foo", nil)
	gr := httptest.NewRecorder()
	s.Handler().ServeHTTP(gr, get)
	if gr.Code != 200 {
		t.Fatalf("get %d %s", gr.Code, gr.Body.String())
	}
}
