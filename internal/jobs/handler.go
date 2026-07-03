package jobs

import (
	"encoding/json"
	"net/http"
	"os"

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
	json.NewEncoder(w).Encode(job)
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

	file, err := os.Open(job.ResultFiles[0])
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusInternalServerError, "Failed to open result file"))
		return
	}
	defer file.Close()

	w.Header().Set("Content-Disposition", `attachment; filename="`+file.Name()+`"`)
	w.Header().Set("Content-Type", "application/octet-stream")
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
