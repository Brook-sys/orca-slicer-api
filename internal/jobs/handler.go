package jobs

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Brook-sys/orca-slicer-api/internal/httpx"
)

type Handler struct {
	Store *Store
}

func (h Handler) GetJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, "Missing job ID"))
		return
	}

	job, ok := h.Store.Get(id)
	if !ok {
		httpx.WriteError(w, httpx.NewError(http.StatusNotFound, "Job not found"))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(job); err != nil {
		slog.Error("failed to write job response", "error", err)
	}
}

func (h Handler) DownloadJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	job, ok := h.Store.Get(id)
	if !ok {
		httpx.WriteError(w, httpx.NewError(http.StatusNotFound, "Job not found"))
		return
	}

	if job.Status != StatusCompleted || len(job.ResultFiles) == 0 {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, "Job is not completed yet"))
		return
	}
	if len(job.ResultFiles) != 1 || !strings.EqualFold(filepath.Ext(job.ResultFiles[0]), ".gcode") {
		httpx.WriteError(w, httpx.NewError(http.StatusInternalServerError, "Job result must contain exactly one G-code file"))
		return
	}

	file, err := os.Open(job.ResultFiles[0])
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusInternalServerError, "Failed to open result file"))
		return
	}
	defer file.Close()

	w.Header().Set("Content-Disposition", `attachment; filename="`+filepath.Base(file.Name())+`"`)
	w.Header().Set("Content-Type", "application/x-gcode")
	http.ServeContent(w, r, file.Name(), job.UpdatedAt, file)
}

func (h Handler) CancelJob(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.Store.Cancel(id); err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusNotFound, err.Error()))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) ListJobs(w http.ResponseWriter, r *http.Request) {
	type jobView struct {
		ID        string    `json:"id"`
		Status    Status    `json:"status"`
		Progress  int       `json:"progress"`
		Error     string    `json:"error,omitempty"`
		CreatedAt time.Time `json:"createdAt"`
		UpdatedAt time.Time `json:"updatedAt"`
	}

	h.Store.mu.RLock()
	defer h.Store.mu.RUnlock()

	list := make([]jobView, 0, len(h.Store.jobs))
	for _, job := range h.Store.jobs {
		list = append(list, jobView{
			ID:        job.ID,
			Status:    job.Status,
			Progress:  job.Progress,
			Error:     job.Error,
			CreatedAt: job.CreatedAt,
			UpdatedAt: job.UpdatedAt,
		})
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(list); err != nil {
		slog.Error("failed to write jobs response", "error", err)
	}
}
