package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"gorocksdb/internal/api"
	"gorocksdb/pkg/gorocksdb"
)

type benchMetricsResponse struct {
	OK   bool `json:"ok"`
	Data struct {
		Puts  int64 `json:"puts"`
		Bench struct {
			Running bool  `json:"running"`
			Ops     int64 `json:"ops"`
		} `json:"bench"`
	} `json:"data"`
}

func TestBenchStopQuiescesWorkers(t *testing.T) {
	db, err := gorocksdb.Open(gorocksdb.Options{
		Dir:     t.TempDir(),
		Profile: "production",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	handler := api.New(db, "").Handler()
	start := serveBenchRequest(t, handler, http.MethodPost, "/api/bench/start",
		`{"workers":2,"qps":200,"value_size":32}`)
	if start.Code != http.StatusOK {
		t.Fatalf("start returned %d: %s", start.Code, start.Body.String())
	}

	deadline := time.Now().Add(2 * time.Second)
	for {
		metrics := readBenchMetrics(t, handler)
		if metrics.Data.Bench.Ops >= 4 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("benchmark did not start producing writes: %+v", metrics.Data.Bench)
		}
		time.Sleep(10 * time.Millisecond)
	}

	stop := serveBenchRequest(t, handler, http.MethodPost, "/api/bench/stop", "")
	if stop.Code != http.StatusOK {
		t.Fatalf("stop returned %d: %s", stop.Code, stop.Body.String())
	}
	var stopped struct {
		OK   bool `json:"ok"`
		Data struct {
			Running bool `json:"running"`
		} `json:"data"`
	}
	if err := json.Unmarshal(stop.Body.Bytes(), &stopped); err != nil {
		t.Fatalf("decode stop response: %v", err)
	}
	if !stopped.OK || stopped.Data.Running {
		t.Fatalf("stop response did not report a stopped benchmark: %s", stop.Body.String())
	}

	// Allow writes already in progress when Stop returned to finish before
	// checking that the stopped run remains quiescent.
	time.Sleep(250 * time.Millisecond)
	before := readBenchMetrics(t, handler)
	time.Sleep(200 * time.Millisecond)
	after := readBenchMetrics(t, handler)

	if after.Data.Bench.Running {
		t.Fatal("benchmark reported running after it was stopped")
	}
	if after.Data.Bench.Ops != before.Data.Bench.Ops || after.Data.Puts != before.Data.Puts {
		t.Fatalf("writes continued after stop: ops %d -> %d, puts %d -> %d",
			before.Data.Bench.Ops, after.Data.Bench.Ops, before.Data.Puts, after.Data.Puts)
	}
}

func serveBenchRequest(t *testing.T, handler http.Handler, method, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder
}

func readBenchMetrics(t *testing.T, handler http.Handler) benchMetricsResponse {
	t.Helper()
	recorder := serveBenchRequest(t, handler, http.MethodGet, "/api/metrics", "")
	if recorder.Code != http.StatusOK {
		t.Fatalf("metrics returned %d: %s", recorder.Code, recorder.Body.String())
	}
	var response benchMetricsResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode metrics response: %v", err)
	}
	if !response.OK {
		t.Fatalf("metrics response was not successful: %s", recorder.Body.String())
	}
	return response
}
