package metrics

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime"
	"time"

	"github.com/Brook-sys/orca-slicer-api/internal/jobs"
	"github.com/Brook-sys/orca-slicer-api/internal/slicer"
)

type Handler struct {
	JobStore *jobs.Store
	Cache    *slicer.ResultCache
	Startup  time.Time
}

func (h Handler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	stats := map[string]any{
		"uptime_seconds":  time.Since(h.Startup).Seconds(),
		"goroutines":      runtime.NumGoroutine(),
		"memory_alloc_mb": float64(m.Alloc) / 1024 / 1024,
	}

	if h.JobStore != nil {
		stats["jobs"] = h.JobStore.Stats()
	}

	if h.Cache != nil {
		stats["cache_enabled"] = true
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(stats); err != nil {
		slog.Error("failed to write metrics response", "error", err)
	}
}
