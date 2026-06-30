package profiles

import (
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/Brook-sys/orca-slicer-api/internal/httpx"
)

type Handler struct {
	Store *Store
}

func (h Handler) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.Store.List(r.PathValue("category"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, items)
}

func (h Handler) Get(w http.ResponseWriter, r *http.Request) {
	data, err := h.Store.Get(r.PathValue("category"), r.PathValue("name"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}

func (h Handler) Upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxProfileSize); err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, "Invalid multipart form"))
		return
	}

	name := r.FormValue("name")
	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, "File is required"))
		return
	}
	defer file.Close()

	if !validProfileFile(header.Filename) {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, "Invalid file type. Only JSON files are allowed"))
		return
	}

	data, err := io.ReadAll(io.LimitReader(file, maxProfileSize+1))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	checksum, err := h.Store.Save(r.PathValue("category"), name, data)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, ImportResponse{Name: name, Checksum: checksum})
}

func (h Handler) ImportURL(w http.ResponseWriter, r *http.Request) {
	var req ImportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, "Invalid JSON body"))
		return
	}

	res, err := h.Store.ImportURL(r.Context(), r.PathValue("category"), req)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusCreated, res)
}

func (h Handler) UpdateFromSource(w http.ResponseWriter, r *http.Request) {
	res, err := h.Store.UpdateFromSource(r.Context(), r.PathValue("category"), r.PathValue("name"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.Store.Delete(r.PathValue("category"), r.PathValue("name")); err != nil {
		httpx.WriteError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func validProfileFile(name string) bool {
	return strings.ToLower(filepath.Ext(name)) == ".json"
}
