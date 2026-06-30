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

	cmd := exec.CommandContext(cmdCtx, s.OrcaSlicerPath, args...)
	slog.Info("slicing started", "file", filepath.Base(filename), "export_type", settings.ExportType)
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

	metadata := AggregateMetadata(files)
	result := Result{Files: files, Workdir: workdir, Metadata: metadata}
	s.setState(JobState{Status: StatusCompleted, StartedAt: startedAt, FinishedAt: nowString(), Files: files, Metadata: metadata})
	slog.Info("slicing completed", "duration_ms", time.Since(started).Milliseconds(), "files", len(files), "print_time_seconds", metadata.PrintTimeSeconds)
	return result, nil
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
		profile := merge(copyMap(settings.PrinterProfile), settings.Overrides["printer"])
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
		if debug != nil {
			debug.Printer = profile
		}
		printerPath = filepath.Join(inputDir, "printer.json")
		if err := writeProfile(printerPath, profile); err != nil {
			return nil, fmt.Errorf("printer profile: %w", err)
		}
	}

	if settings.PresetProfile != nil {
		profile := merge(copyMap(settings.PresetProfile), settings.Overrides["preset"])
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
		if debug != nil {
			debug.Preset = profile
		}
		presetPath = filepath.Join(inputDir, "preset.json")
		if err := writeProfile(presetPath, profile); err != nil {
			return nil, fmt.Errorf("preset profile: %w", err)
		}
	}

	if len(settings.FilamentProfiles) > 0 {
		if debug != nil {
			debug.Filament = settings.FilamentProfiles[0]
			debug.Filaments = settings.FilamentProfiles
		}
		for index, uploaded := range settings.FilamentProfiles {
			profile := merge(copyMap(uploaded), settings.Overrides["filament"])
			filamentPath := filepath.Join(inputDir, fmt.Sprintf("filament_%d.json", index+1))
			if err := writeProfile(filamentPath, profile); err != nil {
				return nil, fmt.Errorf("filament profile: %w", err)
			}
			filamentPaths = append(filamentPaths, filamentPath)
		}
	} else if settings.Filament != "" {
		profile, err := s.loadSelectedProfile("filaments", settings.Filament, settings.Overrides["filament"], settings.ResolveProfiles)
		if err != nil {
			return nil, fmt.Errorf("filament profile: %w", err)
		}
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
