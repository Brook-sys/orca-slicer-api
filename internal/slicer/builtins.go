package slicer

import (
	"encoding/json"
	"log/slog"
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

// builtInCategoryDirName devolve o nome do diretorio que o Orca usa na arvore
// vendor. A API usa "printers|presets|filaments"; o AppImage usa
// "machine|process|filament".
func builtInCategoryDirName(category string) string {
	if mapped := map[string]string{
		"printers":  "machine",
		"presets":   "process",
		"filaments": "filament",
	}[category]; mapped != "" {
		return mapped
	}
	return category
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

	// Aceita os dois layouts: diretorio com o nome da categoria (perfis de
	// usuario espelhados) e o nome que a arvore vendor do AppImage usa.
	candidates := []string{
		filepath.Join(base, category),
		filepath.Join(base, builtInCategoryDirName(category)),
	}

	catPath := ""
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			catPath = candidate
			break
		}
	}
	if catPath == "" {
		httpx.WriteError(w, httpx.NewError(http.StatusNotFound, "no built-in "+category+" profiles directory found"))
		return
	}

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
	if err := json.NewEncoder(w).Encode(items); err != nil {
		slog.Error("failed to write built-in profiles response", "error", err)
	}
}
