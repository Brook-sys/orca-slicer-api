package slicer

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Brook-sys/orca-slicer-api/internal/httpx"
	"github.com/Brook-sys/orca-slicer-api/internal/jobs"
)

const maxModelSize = 100_000_000

type Handler struct {
	Service  *Service
	JobStore *jobs.Store
}

func (h Handler) Status(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, h.Service.Status())
}

func (h Handler) Debug(w http.ResponseWriter, r *http.Request) {
	httpx.WriteJSON(w, http.StatusOK, h.Service.Debug())
}

func (h Handler) ListAliases(w http.ResponseWriter, r *http.Request) {
	aliases, err := h.Service.ListAliases()
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, aliases)
}

func (h Handler) SaveAlias(w http.ResponseWriter, r *http.Request) {
	var alias ProfileAlias
	if err := json.NewDecoder(r.Body).Decode(&alias); err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, "Invalid JSON body"))
		return
	}
	aliases, err := h.Service.SaveAlias(alias)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, aliases)
}

func (h Handler) DeleteAlias(w http.ResponseWriter, r *http.Request) {
	aliases, err := h.Service.DeleteAlias(r.PathValue("category"), r.PathValue("from"))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	httpx.WriteJSON(w, http.StatusOK, aliases)
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

	models, err := modelsFromMultipart(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	settings, err := parseSettings(r, serviceGenerateImage(h.Service), serviceResolveProfiles(h.Service), serviceSanitizeProfiles(h.Service))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := parseUploadedProfiles(r, &settings); err != nil {
		httpx.WriteError(w, err)
		return
	}

	result, err := h.Service.SliceModels(r.Context(), models, settings)
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

func (h Handler) SliceAsync(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxModelSize); err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, "Invalid multipart form"))
		return
	}

	models, err := modelsFromMultipart(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	settings, err := parseSettings(r, serviceGenerateImage(h.Service), serviceResolveProfiles(h.Service), serviceSanitizeProfiles(h.Service))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := parseUploadedProfiles(r, &settings); err != nil {
		httpx.WriteError(w, err)
		return
	}

	job, err := h.JobStore.Create()
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	go func() {
		result, err := h.Service.SliceModelsAsync(context.Background(), job.ID, models, settings)
		if err != nil {
			h.JobStore.UpdateStatus(job.ID, jobs.StatusFailed, err)
			return
		}
		h.JobStore.SetResult(job.ID, result.Files, result.Workdir)
		h.JobStore.UpdateStatus(job.ID, jobs.StatusCompleted, nil)
	}()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	if err := json.NewEncoder(w).Encode(job); err != nil {
		slog.Error("failed to write async job response", "error", err)
	}
}

func (h Handler) Preview(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(maxModelSize); err != nil {
		httpx.WriteError(w, httpx.NewError(http.StatusBadRequest, "Invalid multipart form"))
		return
	}

	models, err := modelsFromMultipart(r)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}

	settings, err := parseSettings(r, false, serviceResolveProfiles(h.Service), serviceSanitizeProfiles(h.Service))
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	if err := parseUploadedProfiles(r, &settings); err != nil {
		httpx.WriteError(w, err)
		return
	}

	result, err := h.Service.Preview(r.Context(), models[0].Filename, models[0].Data, settings)
	if err != nil {
		httpx.WriteError(w, err)
		return
	}
	defer os.RemoveAll(result.Workdir)

	httpx.WriteJSON(w, http.StatusOK, map[string]any{
		"usesSupport":    result.UsesSupport,
		"printTime":      result.PrintTimeSeconds,
		"filamentUsedG":  result.FilamentUsedG,
		"filamentUsedMm": result.FilamentUsedMm,
		"thumbnail":      result.ThumbnailBase64,
		"quantity":       result.Quantity,
		"toolpaths":      result.Toolpaths,
	})
}

func parseSettings(r *http.Request, defaultGenerateImage bool, defaultResolveProfiles bool, defaultSanitizeProfiles bool) (Settings, error) {
	settings := Settings{
		Printer:               r.FormValue("printer"),
		Preset:                r.FormValue("preset"),
		Filament:              r.FormValue("filament"),
		BedType:               r.FormValue("bedType"),
		Plate:                 r.FormValue("plate"),
		Quantity:              1,
		Arrange:               parseBool(r.FormValue("arrange")),
		Orient:                parseBool(r.FormValue("orient")),
		ExportType:            r.FormValue("exportType"),
		MulticolorOnePlate:    parseBool(r.FormValue("multicolorOnePlate")),
		ResolveProfiles:       parseBool(r.FormValue("resolveProfiles")) || defaultResolveProfiles,
		SanitizeProfiles:      parseBool(r.FormValue("sanitizeProfiles")) || defaultSanitizeProfiles,
		GenerateImage:         parseBool(r.FormValue("generateImage")) || defaultGenerateImage,
		EnableSupport:         parseBoolPtr(r.FormValue("enableSupport")),
		BrimType:              parseBoolPtr(r.FormValue("brimType")),
		PrintSequenceByObject: parseBool(r.FormValue("printSequenceByObject")),
		Overrides:             map[string]map[string]any{},
	}

	if raw := r.FormValue("quantity"); raw != "" {
		quantity, err := strconv.Atoi(raw)
		if err != nil {
			return Settings{}, httpx.NewError(http.StatusBadRequest, "quantity must be an integer")
		}
		if quantity < 1 || quantity > maxSliceQuantity {
			return Settings{}, httpx.NewError(http.StatusBadRequest, "quantity must be an integer between 1 and 512")
		}
		settings.Quantity = quantity
	}
	for name, target := range map[string]**float64{"rotate": &settings.Rotate, "rotateX": &settings.RotateX, "rotateY": &settings.RotateY} {
		value, err := parseOptionalFloat(r.FormValue(name), name)
		if err != nil {
			return Settings{}, err
		}
		*target = value
	}
	arrangeSpacing, err := parseOptionalFloat(r.FormValue("arrangeSpacing"), "arrangeSpacing")
	if err != nil {
		return Settings{}, err
	}
	if arrangeSpacing != nil {
		settings.ArrangeSpacing = *arrangeSpacing
	}

	if raw := r.FormValue("overrides"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &settings.Overrides); err != nil {
			return Settings{}, httpx.NewError(http.StatusBadRequest, "Invalid overrides JSON")
		}
	}

	return normalizeSliceSettings(settings)
}

func parseOptionalFloat(raw string, name string) (*float64, error) {
	if raw == "" {
		return nil, nil
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return nil, httpx.NewError(http.StatusBadRequest, name+" must be a number")
	}
	return &value, nil
}

func parseUploadedProfiles(r *http.Request, settings *Settings) error {
	printer, err := readJSONProfilePart(r, "printerProfile")
	if err != nil {
		return err
	}
	preset, err := readJSONProfilePart(r, "presetProfile")
	if err != nil {
		return err
	}
	filaments, err := readJSONProfileParts(r, "filamentProfile")
	if err != nil {
		return err
	}

	settings.PrinterProfile = printer
	settings.PresetProfile = preset
	settings.FilamentProfiles = filaments
	return nil
}

func readJSONProfilePart(r *http.Request, field string) (map[string]any, error) {
	profiles, err := readJSONProfileParts(r, field)
	if err != nil {
		return nil, err
	}
	if len(profiles) == 0 {
		return nil, nil
	}
	return profiles[0], nil
}

func readJSONProfileParts(r *http.Request, field string) ([]map[string]any, error) {
	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		return nil, nil
	}
	files := r.MultipartForm.File[field]
	profiles := make([]map[string]any, 0, len(files))
	for _, header := range files {
		file, err := header.Open()
		if err != nil {
			return nil, err
		}
		data, readErr := io.ReadAll(io.LimitReader(file, maxModelSize+1))
		closeErr := file.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if len(data) > maxModelSize {
			return nil, httpx.NewError(http.StatusBadRequest, field+" is too large")
		}
		var profile map[string]any
		if err := json.Unmarshal(data, &profile); err != nil {
			return nil, httpx.NewError(http.StatusBadRequest, field+" must be valid JSON")
		}
		profiles = append(profiles, profile)
	}
	return profiles, nil
}

func modelsFromMultipart(r *http.Request) ([]ModelInput, error) {
	if r.MultipartForm == nil {
		return nil, httpx.NewError(http.StatusBadRequest, "Model file is required")
	}
	files := append([]*multipart.FileHeader(nil), r.MultipartForm.File["file"]...)
	files = append(files, r.MultipartForm.File["files"]...)
	if len(files) == 0 {
		return nil, httpx.NewError(http.StatusBadRequest, "Model file is required")
	}
	if len(files) != 1 {
		return nil, httpx.NewError(http.StatusBadRequest, "Exactly one STL file is required; multiple models are not supported")
	}

	models := make([]ModelInput, 0, len(files))
	for _, header := range files {
		if !validModelFile(header.Filename) {
			return nil, httpx.NewError(http.StatusBadRequest, "Invalid file type. Exactly one STL file is required; STEP and 3MF are not supported")
		}
		file, err := header.Open()
		if err != nil {
			return nil, err
		}
		data, readErr := io.ReadAll(io.LimitReader(file, maxModelSize+1))
		closeErr := file.Close()
		if readErr != nil {
			return nil, readErr
		}
		if closeErr != nil {
			return nil, closeErr
		}
		if len(data) > maxModelSize {
			return nil, httpx.NewError(http.StatusBadRequest, "Model file is too large")
		}
		models = append(models, ModelInput{Filename: header.Filename, Data: data})
	}
	return models, nil
}

func validModelFile(name string) bool {
	return strings.EqualFold(filepath.Ext(name), ".stl")
}

func serviceGenerateImage(service *Service) bool {
	return service != nil && service.GenerateImage
}

func serviceResolveProfiles(service *Service) bool {
	return service != nil && service.ResolveProfiles
}

func serviceSanitizeProfiles(service *Service) bool {
	return service != nil && service.SanitizeProfiles
}

func parseBool(value string) bool {
	return value == "true" || value == "1" || value == "on"
}

func parseBoolPtr(value string) *bool {
	if value == "" {
		return nil
	}
	b := parseBool(value)
	return &b
}
