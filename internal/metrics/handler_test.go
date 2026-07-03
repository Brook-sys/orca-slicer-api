package metrics

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Brook-sys/orca-slicer-api/internal/jobs"
	"github.com/Brook-sys/orca-slicer-api/internal/slicer"
)

func TestGetMetrics(t *testing.T) {
	jobStore := jobs.NewStore()
	jobStore.Create() // Creates a pending job

	cache := slicer.NewResultCache(time.Hour, 10)
	handler := Handler{
		JobStore: jobStore,
		Cache:    cache,
		Startup:  time.Now().Add(-10 * time.Second),
	}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/metrics", nil)
	handler.GetMetrics(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var stats map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &stats); err != nil {
		t.Fatal(err)
	}

	if stats["uptime_seconds"].(float64) < 9 {
		t.Fatalf("expected uptime >= 9s")
	}
	if stats["cache_enabled"] != true {
		t.Fatalf("expected cache_enabled=true")
	}
	
	jobStats := stats["jobs"].(map[string]any)
	if jobStats["total"].(float64) != 1 {
		t.Fatalf("expected 1 total job, got %v", jobStats["total"])
	}
}
