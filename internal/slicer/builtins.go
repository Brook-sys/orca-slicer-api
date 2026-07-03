package slicer

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Brook-sys/orca-slicer-api/internal/httpx"
)

type ProfileListItem struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func (h Handler) ListBuiltinProfiles(w http.ResponseWriter, r *http.Request) {
	category := r.PathValue("category")
	if category == "" {
		category = r.URL.Query().Get("category")
	}
	if category == "" {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, "category is required (printers|presets|filaments)"))
		return
	}
	if category != "printers" && category != "presets" && category != "filaments" {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, "invalid category"))
		return
	}

	base := h.Service.OrcaProfilesPath
	if base == "" {
		base = "/app/squashfs-root/resources/profiles"
	}

	catPath := filepath.Join(base, category)
	entries, err := os.ReadDir(catPath)
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusInternalServerError, "failed to read profiles: "+err.Error()))
		return
	}

	items := make([]ProfileListItem, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".json") {
			continue
		}
		items = append(items, ProfileListItem{
			Name: strings.TrimSuffix(name, ".json"),
			Path: filepath.Join(catPath, name),
		})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(items)
}
