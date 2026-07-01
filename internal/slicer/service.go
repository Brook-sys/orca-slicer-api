package slicer

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/Brook-sys/orca-slicer-api/internal/httpx"
)

type Service struct {
	DataPath         string
	OrcaSlicerPath   string
	OrcaProfilesPath string
	Timeout          time.Duration
	GenerateImage    bool
	State            *StateStore
	mu               sync.Mutex
}

func (s *Service) Slice(ctx context.Context, filename string, data []byte, settings Settings) (Result, error) {
	if !s.mu.TryLock() {
		return Result{}, httpx.NewError(http.StatusConflict, "Slicer is busy")
	}
	defer s.mu.Unlock()

	startedAt := nowString()
	if s.State != nil {
		_ = s.State.Set(JobState{Status: StatusProcessing, StartedAt: startedAt})
	}

	if s.OrcaSlicerPath == "" {
		err := httpx.NewError(http.StatusInternalServerError, "ORCASLICER_PATH is not configured")
		s.setFailed(startedAt, err.Error())
		return Result{}, err
	}

	workdir, err := os.MkdirTemp("", "slice-*")
	if err != nil {
		s.setFailed(startedAt, err.Error())
		return Result{}, err
	}

	inputDir := filepath.Join(workdir, "input")
	outputDir := filepath.Join(workdir, "output")
	if err := os.MkdirAll(inputDir, 0o755); err != nil {
		s.setFailed(startedAt, err.Error())
		return Result{}, err
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		s.setFailed(startedAt, err.Error())
		return Result{}, err
	}

	inputPath := filepath.Join(inputDir, filepath.Base(filename))
	if err := os.WriteFile(inputPath, data, 0o644); err != nil {
		s.setFailed(startedAt, err.Error())
		return Result{}, err
	}

	debug := SliceDebug{StartedAt: startedAt, Workdir: workdir, InputPath: inputPath, OutputDir: outputDir, Command: s.OrcaSlicerPath, Settings: settings}
	args, err := s.buildArgs(inputPath, inputDir, outputDir, settings, &debug)
	debug.Args = args
	if err != nil {
		debug.FinishedAt = nowString()
		debug.ErrorMessage = err.Error()
		s.setDebug(debug)
		s.setFailed(startedAt, err.Error())
		return Result{}, err
	}
	defer func() {
		debug.ResultJSON = readResultJSON(outputDir)
		debug.SlicerError = readSlicerError(outputDir)
		debug.Files, _ = resultFiles(outputDir, settings.ExportType)
		if debug.FinishedAt == "" {
			debug.FinishedAt = nowString()
		}
		s.setDebug(debug)
	}()

	cmdCtx := ctx
	cancel := func() {}
	if s.Timeout > 0 {
		cmdCtx, cancel = context.WithTimeout(ctx, s.Timeout)
	}
	defer cancel()

	exe := s.OrcaSlicerPath
	cmd := exec.CommandContext(cmdCtx, exe, args...)
	slog.Info("slicing started", "file", filepath.Base(filename), "export_type", settings.ExportType, "generate_image", settings.GenerateImage)
	started := time.Now()
	output, err := cmd.CombinedOutput()
	debug.Output = strings.TrimSpace(string(output))
	if err != nil {
		if cmdCtx.Err() != nil {
			debug.ErrorMessage = cmdCtx.Err().Error()
			s.setState(JobState{Status: StatusCancelled, StartedAt: startedAt, FinishedAt: nowString(), ErrorMessage: cmdCtx.Err().Error()})
			return Result{}, httpx.NewError(http.StatusRequestTimeout, "Slicing cancelled or timed out")
		}
		slicerError := readSlicerError(outputDir)
		if slicerError != "" {
			debug.ErrorMessage = slicerError
			s.setFailed(startedAt, slicerError)
			return Result{}, httpx.NewError(http.StatusInternalServerError, "Slicing failed: "+slicerError)
		}
		message := strings.TrimSpace(string(output))
		debug.ErrorMessage = message
		s.setFailed(startedAt, message)
		return Result{}, httpx.NewError(http.StatusInternalServerError, "Slicing failed: "+message)
	}

	files, err := resultFiles(outputDir, settings.ExportType)
	if err != nil {
		debug.ErrorMessage = err.Error()
		s.setFailed(startedAt, err.Error())
		return Result{}, err
	}
	if len(files) == 0 {
		err := httpx.NewError(http.StatusInternalServerError, "No files generated during slicing")
		debug.ErrorMessage = err.Error()
		s.setFailed(startedAt, err.Error())
		return Result{}, err
	}

	if settings.GenerateImage && settings.ExportType != "3mf" {
		for _, file := range files {
			if err := addNeptune4ThumbnailsToGCode(file, data); err != nil {
				debug.ErrorMessage = err.Error()
				s.setFailed(startedAt, err.Error())
				return Result{}, httpx.NewError(http.StatusInternalServerError, "Image generation failed: "+err.Error())
			}
		}
	}

	metadata := AggregateMetadata(files)
	result := Result{Files: files, Workdir: workdir, Metadata: metadata}
	s.setState(JobState{Status: StatusCompleted, StartedAt: startedAt, FinishedAt: nowString(), Files: files, Metadata: metadata})
	slog.Info("slicing completed", "duration_ms", time.Since(started).Milliseconds(), "files", len(files), "print_time_seconds", metadata.PrintTimeSeconds)
	return result, nil
}

func (s *Service) Preview(ctx context.Context, filename string, data []byte, settings Settings) (PreviewResult, error) {
	trueVal := true
	settings.EnableSupport = &trueVal
	settings.GenerateImage = true

	result, err := s.Slice(ctx, filename, data, settings)
	if err != nil {
		return PreviewResult{}, err
	}
	defer os.RemoveAll(result.Workdir)

	if len(result.Files) == 0 {
		return PreviewResult{}, httpx.NewError(http.StatusInternalServerError, "No files generated")
	}

	gcodePath := result.Files[0]
	usesSupport, _ := detectSupportInGCode(gcodePath)

	thumb := extractThumbnailFromGCode(gcodePath)

	return PreviewResult{
		UsesSupport:      usesSupport,
		PrintTimeSeconds: result.Metadata.PrintTimeSeconds,
		FilamentUsedG:    result.Metadata.FilamentUsedG,
		FilamentUsedMm:   result.Metadata.FilamentUsedMM,
		ThumbnailBase64:  thumb,
		Workdir:          result.Workdir,
	}, nil
}

func (s Service) buildArgs(inputPath string, inputDir string, outputDir string, settings Settings, debug *SliceDebug) ([]string, error) {
	args := make([]string, 0)

	if settings.ExportType == "3mf" {
		args = append(args, "--export-3mf", "result.3mf")
	}

	plate := settings.Plate
	if plate == "" {
		plate = "1"
	}
	args = append(args, "--slice", plate)
	args = append(args, "--arrange", boolArg(settings.Arrange))
	args = append(args, "--orient", boolArg(settings.Orient))

	printerPath := ""
	presetPath := ""
	filamentPaths := make([]string, 0)

	if settings.PrinterProfile != nil {
		profile := prepareProfileForSlicing("printers", merge(copyMap(settings.PrinterProfile), settings.Overrides["printer"]), settings.SanitizeProfiles)
		if debug != nil {
			debug.Printer = profile
		}
		printerPath = filepath.Join(inputDir, "printer.json")
		if err := writeProfile(printerPath, profile); err != nil {
			return nil, fmt.Errorf("printer profile: %w", err)
		}
	} else if settings.Printer != "" {
		profile, err := s.loadSelectedProfile("printers", settings.Printer, settings.Overrides["printer"], settings.ResolveProfiles)
		if err != nil {
			return nil, fmt.Errorf("printer profile: %w", err)
		}
		profile = prepareProfileForSlicing("printers", profile, settings.SanitizeProfiles)
		if debug != nil {
			debug.Printer = profile
		}
		printerPath = filepath.Join(inputDir, "printer.json")
		if err := writeProfile(printerPath, profile); err != nil {
			return nil, fmt.Errorf("printer profile: %w", err)
		}
	}

	if settings.PresetProfile != nil {
		profile := prepareProfileForSlicing("presets", merge(copyMap(settings.PresetProfile), settings.Overrides["preset"]), settings.SanitizeProfiles)
		if settings.EnableSupport != nil {
			profile["enable_support"] = *settings.EnableSupport
		}
		if settings.BrimType != nil {
			if *settings.BrimType {
				profile["brim_type"] = "auto_brim"
			} else {
				profile["brim_type"] = "no_brim"
			}
		}
		if settings.PrintSequenceByObject {
			profile["print_sequence"] = "by object"
		} else {
			profile["print_sequence"] = "by layer"
		}
		if debug != nil {
			debug.Preset = profile
		}
		presetPath = filepath.Join(inputDir, "preset.json")
		if err := writeProfile(presetPath, profile); err != nil {
			return nil, fmt.Errorf("preset profile: %w", err)
		}
	} else if settings.Preset != "" {
		profile, err := s.loadSelectedProfile("presets", settings.Preset, settings.Overrides["preset"], settings.ResolveProfiles)
		if err != nil {
			return nil, fmt.Errorf("preset profile: %w", err)
		}
		profile = prepareProfileForSlicing("presets", profile, settings.SanitizeProfiles)
		if settings.EnableSupport != nil {
			profile["enable_support"] = *settings.EnableSupport
		}
		if settings.BrimType != nil {
			if *settings.BrimType {
				profile["brim_type"] = "auto_brim"
			} else {
				profile["brim_type"] = "no_brim"
			}
		}
		if settings.PrintSequenceByObject {
			profile["print_sequence"] = "by object"
		} else {
			profile["print_sequence"] = "by layer"
		}
		if debug != nil {
			debug.Preset = profile
		}
		presetPath = filepath.Join(inputDir, "preset.json")
		if err := writeProfile(presetPath, profile); err != nil {
			return nil, fmt.Errorf("preset profile: %w", err)
		}
	}

	if len(settings.FilamentProfiles) > 0 {
		debugFilaments := make([]map[string]any, 0, len(settings.FilamentProfiles))
		for index, uploaded := range settings.FilamentProfiles {
			profile := prepareProfileForSlicing("filaments", merge(copyMap(uploaded), settings.Overrides["filament"]), settings.SanitizeProfiles)
			debugFilaments = append(debugFilaments, profile)
			filamentPath := filepath.Join(inputDir, fmt.Sprintf("filament_%d.json", index+1))
			if err := writeProfile(filamentPath, profile); err != nil {
				return nil, fmt.Errorf("filament profile: %w", err)
			}
			filamentPaths = append(filamentPaths, filamentPath)
		}
		if debug != nil && len(debugFilaments) > 0 {
			debug.Filament = debugFilaments[0]
			debug.Filaments = debugFilaments
		}
	} else if settings.Filament != "" {
		profile, err := s.loadSelectedProfile("filaments", settings.Filament, settings.Overrides["filament"], settings.ResolveProfiles)
		if err != nil {
			return nil, fmt.Errorf("filament profile: %w", err)
		}
		profile = prepareProfileForSlicing("filaments", profile, settings.SanitizeProfiles)
		if debug != nil {
			debug.Filament = profile
			debug.Filaments = []map[string]any{profile}
		}
		filamentPath := filepath.Join(inputDir, "filament.json")
		if err := writeProfile(filamentPath, profile); err != nil {
			return nil, fmt.Errorf("filament profile: %w", err)
		}
		filamentPaths = append(filamentPaths, filamentPath)
	}

	if printerPath != "" && presetPath != "" {
		args = append(args, "--load-settings", printerPath+";"+presetPath)
	}
	if len(filamentPaths) > 0 {
		args = append(args, "--load-filaments", strings.Join(filamentPaths, ";"))
	}
	if settings.BedType != "" {
		args = append(args, "--curr-bed-type", settings.BedType)
	}
	if settings.MulticolorOnePlate {
		args = append(args, "--allow-multicolor-oneplate")
	}

	args = append(args, "--allow-newer-file", "--outputdir", outputDir, inputPath)
	return args, nil
}

func (s Service) loadSelectedProfile(category string, name string, overrides map[string]any, resolve bool) (map[string]any, error) {
	if resolve {
		resolved, err := ResolveProfile(s.DataPath, s.OrcaProfilesPath, category, name, overrides)
		if err != nil {
			return nil, err
		}
		return resolved.Resolved, nil
	}
	return loadRawUserProfile(s.DataPath, category, name, overrides)
}

func prepareProfileForSlicing(category string, profile map[string]any, sanitize bool) map[string]any {
	if sanitize {
		return sanitizeProfileForSlicing(category, profile)
	}
	return profile
}

func resultFiles(outputDir string, exportType string) ([]string, error) {
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return nil, err
	}

	ext := ".gcode"
	if exportType == "3mf" {
		ext = ".3mf"
	}

	files := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ext) {
			continue
		}
		files = append(files, filepath.Join(outputDir, entry.Name()))
	}
	return files, nil
}

func readSlicerError(outputDir string) string {
	data, err := os.ReadFile(filepath.Join(outputDir, "result.json"))
	if err != nil {
		return ""
	}
	var result struct {
		ErrorString string `json:"error_string"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return ""
	}
	return result.ErrorString
}

func readResultJSON(outputDir string) map[string]any {
	data, err := os.ReadFile(filepath.Join(outputDir, "result.json"))
	if err != nil {
		return nil
	}
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	return result
}

func (s *Service) Status() JobState {
	if s.State == nil {
		return JobState{Status: StatusIdle}
	}
	return s.State.Get()
}

func (s *Service) Debug() SliceDebug {
	if s.State == nil {
		return SliceDebug{}
	}
	return s.State.GetDebug()
}

func (s *Service) setFailed(startedAt string, message string) {
	s.setState(JobState{Status: StatusFailed, StartedAt: startedAt, FinishedAt: nowString(), ErrorMessage: message})
}

func (s *Service) setState(state JobState) {
	if s.State != nil {
		_ = s.State.Set(state)
	}
}

func (s *Service) setDebug(debug SliceDebug) {
	if s.State != nil {
		_ = s.State.SetDebug(debug)
	}
}

func boolArg(value bool) string {
	if value {
		return "1"
	}
	return "0"
}
