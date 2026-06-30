package slicer

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Brook-sys/orca-slicer-api/internal/httpx"
)

const maxModelSize = 100_000_000

type Handler struct {
	Service *Service
}

func (h Handler) Status(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, h.Service.Status())
}

func (h Handler) ResolveProfile(w http.ResponseWriter, r *http.Request) {
	var req ResolveProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, "Invalid JSON body"))
		return
	}
	if req.Overrides == nil {
		req.Overrides = map[string]any{}
	}

	res, err := ResolveProfile(h.Service.DataPath, h.Service.OrcaProfilesPath, req.Category, req.Name, req.Overrides)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h Handler) ResolveProfiles(w http.ResponseWriter, r *http.Request) {
	var settings Settings
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, "Invalid JSON body"))
		return
	}
	if settings.Overrides == nil {
		settings.Overrides = map[string]map[string]any{}
	}

	res := ResolveProfilesResponse{}
	if settings.Printer != "" {
		resolved, err := ResolveProfile(h.Service.DataPath, h.Service.OrcaProfilesPath, "printers", settings.Printer, settings.Overrides["printer"])
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		res.Printer = &resolved
	}
	if settings.Preset != "" {
		resolved, err := ResolveProfile(h.Service.DataPath, h.Service.OrcaProfilesPath, "presets", settings.Preset, settings.Overrides["preset"])
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		res.Preset = &resolved
	}
	if settings.Filament != "" {
		resolved, err := ResolveProfile(h.Service.DataPath, h.Service.OrcaProfilesPath, "filaments", settings.Filament, settings.Overrides["filament"])
		if err != nil {
			httpx.WriteError(w, err)
			return
		}
		res.Filament = &resolved
	}

	httpx.WriteJSON(w, http.StatusOK, res)
}

func (h Handler) Slice(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxModelSize); err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, "Invalid multipart form"))
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, "Model file is required"))
		return
	}
	defer file.Close()

	if !validModelFile(header.Filename) {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, "Invalid file type. Only STL, STEP and 3MF files are allowed"))
		return
	}

	data, err := io.ReadAll(io.LimitReader(file, maxModelSize+1))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if len(data) > maxModelSize {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, "Model file is too large"))
		return
	}

	settings, err := parseSettings(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	result, err := h.Service.Slice(r.Context(), header.Filename, data, settings)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	defer os.RemoveAll(result.Workdir)

	if err := WriteResult(w, r, result); err != nil {
		httpx.WriteError(w, err)
		return
	}
}

func parseSettings(r *http.Request) (Settings, error) {
	settings := Settings{
		Printer:            r.FormValue("printer"),
		Preset:             r.FormValue("preset"),
		Filament:           r.FormValue("filament"),
		BedType:            r.FormValue("bedType"),
		Plate:              r.FormValue("plate"),
		Arrange:            parseBool(r.FormValue("arrange")),
		Orient:             parseBool(r.FormValue("orient")),
		ExportType:         r.FormValue("exportType"),
		MulticolorOnePlate: parseBool(r.FormValue("multicolorOnePlate")),
		Overrides:          map[string]map[string]any{},
	}

	if settings.ExportType != "" && settings.ExportType != "gcode" && settings.ExportType != "3mf" {
		return Settings{}, httpx.NewError(http.StatusBadRequest, "exportType must be gcode or 3mf")
	}

	if raw := r.FormValue("overrides"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &settings.Overrides); err != nil {
			return Settings{}, httpx.NewError(http.StatusBadRequest, "Invalid overrides JSON")
		}
	}

	return settings, nil
}

func validModelFile(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".stl" || ext == ".step" || ext == ".stp" || ext == ".3mf"
}

func parseBool(value string) bool {
	return value == "true" || value == "1" || value == "on"
}
