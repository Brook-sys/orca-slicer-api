package slicer

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/Brook-sys/orca-slicer-api/internal/jobs"
	"log/slog"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Brook-sys/orca-slicer-api/internal/httpx"
)

const maxSlicerOutputBytes = 1 << 20

type Service struct {
	DataPath         string
	OrcaSlicerPath   string
	OrcaProfilesPath string
	Timeout          time.Duration
	GenerateImage    bool
	ResolveProfiles  bool
	SanitizeProfiles bool
	State            *StateStore
	mu               sync.Mutex
	JobStore         *jobs.Store
	ResultCache      *ResultCache
	CacheTTL         time.Duration
}

func (s *Service) Slice(ctx context.Context, filename string, data []byte, settings Settings) (Result, error) {
	return s.SliceModels(ctx, []ModelInput{{Filename: filename, Data: data}}, settings)
}

func (s *Service) SliceAsync(ctx context.Context, jobID string, filename string, data []byte, settings Settings) (Result, error) {
	return s.SliceModelsAsync(ctx, jobID, []ModelInput{{Filename: filename, Data: data}}, settings)
}

func (s *Service) SliceModels(ctx context.Context, models []ModelInput, settings Settings) (Result, error) {
	return s.sliceInternal(ctx, "", models, settings)
}

func (s *Service) SliceModelsAsync(ctx context.Context, jobID string, models []ModelInput, settings Settings) (Result, error) {
	return s.sliceInternal(ctx, jobID, models, settings)
}

func (s *Service) sliceInternal(ctx context.Context, jobID string, models []ModelInput, settings Settings) (Result, error) {
	settings, err := normalizeSliceSettings(settings)
	if err != nil {
		return Result{}, err
	}
	if err := validateSingleSTL(models); err != nil {
		return Result{}, err
	}
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
	keepWorkdir := false
	defer func() {
		if !keepWorkdir {
			_ = os.RemoveAll(workdir)
		}
	}()

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

	// Use a private fixed name so an upload can never collide with a temporary
	// profile or with another input. The original filename remains in the cache key.
	inputPath := filepath.Join(inputDir, "model.stl")
	if err := os.WriteFile(inputPath, models[0].Data, 0o644); err != nil {
		s.setFailed(startedAt, err.Error())
		return Result{}, err
	}
	inputPaths := []string{inputPath}

	debug := SliceDebug{StartedAt: startedAt, Workdir: workdir, InputPath: inputPath, OutputDir: outputDir, Command: s.OrcaSlicerPath, Settings: settings}
	args, err := s.buildArgsForInputs(inputPaths, inputDir, outputDir, settings, &debug)
	debug.Args = args
	if err != nil {
		debug.FinishedAt = nowString()
		debug.ErrorMessage = err.Error()
		s.setDebug(debug)
		s.setFailed(startedAt, err.Error())
		return Result{}, err
	}
	// Resolve profiles before lookup so edits to named profiles or their ancestors
	// cannot reuse stale results. Hash the executable contents, not just its path.
	cacheKey := ""
	if s.ResultCache != nil {
		cacheKey = s.resultCacheKey(models, settings, debug)
		if cacheKey != "" {
			if entry, ok := s.ResultCache.Get(cacheKey); ok {
				files, statistics, validationErr := validateSliceOutput(entry.Workdir, settings.Quantity)
				if entry.Quantity == settings.Quantity && validationErr == nil {
					metadata := AggregateMetadata(files)
					s.setState(JobState{Status: StatusCompleted, StartedAt: startedAt, FinishedAt: nowString(), Files: files, Metadata: metadata})
					return Result{Files: files, Workdir: entry.Workdir, Metadata: metadata, Quantity: settings.Quantity, Statistics: statistics}, nil
				}
				// Cached bytes and evidence must pass the same validation as a fresh
				// slice. A damaged snapshot is a miss, never a partial success.
				_ = os.RemoveAll(entry.Workdir)
			}
		}
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
	if jobID != "" && s.JobStore != nil {
		s.JobStore.SetCancelFunc(jobID, cancel)
	}
	defer cancel()

	exe := s.OrcaSlicerPath
	cmd := exec.CommandContext(cmdCtx, exe, args...)
	slog.Info("slicing started",
		"file", filepath.Base(models[0].Filename),
		"export_type", settings.ExportType,
		"generate_image", settings.GenerateImage,
		"enable_support", optionalBoolLogValue(settings.EnableSupport),
		"brim_type", optionalBrimLogValue(settings.BrimType),
		"print_sequence_by_object", settings.PrintSequenceByObject,
		"resolve_profiles", settings.ResolveProfiles,
		"sanitize_profiles", settings.SanitizeProfiles,
		"arrange", settings.Arrange,
		"orient", settings.Orient,
		"quantity", settings.Quantity,
		"rotate_x", settings.RotateX,
		"rotate_y", settings.RotateY,
		"rotate_z", settings.Rotate,
	)
	started := time.Now()

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		debug.ErrorMessage = err.Error()
		s.setFailed(startedAt, err.Error())
		return Result{}, httpx.NewError(http.StatusInternalServerError, "Failed to start slicer: "+err.Error())
	}

	var outputBuilder strings.Builder
	var outputMu sync.Mutex
	var outputWG sync.WaitGroup
	outputTruncated := false
	readOutput := func(pipe io.Reader, trackProgress bool) {
		defer outputWG.Done()
		scanner := bufio.NewScanner(pipe)
		for scanner.Scan() {
			line := scanner.Text() + "\n"
			outputMu.Lock()
			remaining := maxSlicerOutputBytes - outputBuilder.Len()
			if remaining > 0 {
				if len(line) > remaining {
					line = line[:remaining]
					outputTruncated = true
				}
				outputBuilder.WriteString(line)
			} else {
				outputTruncated = true
			}
			outputMu.Unlock()
			if trackProgress && jobID != "" && s.JobStore != nil {
				if progress, ok := jobs.ExtractProgress(line); ok {
					s.JobStore.UpdateProgress(jobID, progress)
				}
			}
		}
	}

	outputWG.Add(2)
	go readOutput(stdout, true)
	go readOutput(stderr, true)

	err = cmd.Wait()
	outputWG.Wait()
	debug.Output = strings.TrimSpace(outputBuilder.String())
	if outputTruncated {
		debug.Output += "\n[slicer output truncated at 1 MiB]"
	}

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
		message := debug.Output
		debug.ErrorMessage = message
		s.setFailed(startedAt, message)
		return Result{}, httpx.NewError(http.StatusInternalServerError, "Slicing failed: "+message)
	}

	files, statistics, err := validateSliceOutput(outputDir, settings.Quantity)
	if err != nil {
		debug.ErrorMessage = err.Error()
		s.setFailed(startedAt, err.Error())
		return Result{}, err
	}

	if settings.GenerateImage {
		for _, file := range files {
			if err := addNeptune4ThumbnailsToGCode(file, models[0].Data); err != nil {
				debug.ErrorMessage = err.Error()
				s.setFailed(startedAt, err.Error())
				return Result{}, httpx.NewError(http.StatusInternalServerError, "Image generation failed: "+err.Error())
			}
		}
	}

	metadata := AggregateMetadata(files)
	result := Result{Files: files, Workdir: workdir, Metadata: metadata, Quantity: settings.Quantity, Statistics: statistics}
	if s.ResultCache != nil && cacheKey != "" {
		s.ResultCache.PutResult(cacheKey, result)
	}
	s.setState(JobState{Status: StatusCompleted, StartedAt: startedAt, FinishedAt: nowString(), Files: files, Metadata: metadata})
	slog.Info("slicing completed", "duration_ms", time.Since(started).Milliseconds(), "files", len(files), "quantity", settings.Quantity, "print_time_seconds", metadata.PrintTimeSeconds)
	keepWorkdir = true
	return result, nil
}

func (s *Service) Preview(ctx context.Context, filename string, data []byte, settings Settings) (PreviewResult, error) {
	return s.PreviewWithOptions(ctx, filename, data, settings, PreviewOptions{})
}

func (s *Service) PreviewWithOptions(ctx context.Context, filename string, data []byte, settings Settings, options PreviewOptions) (PreviewResult, error) {
	mode, err := normalizeToolpathMode(options.ToolpathMode)
	if err != nil {
		return PreviewResult{}, err
	}
	// Preview changes only the representation, never the support decision. The
	// thumbnail is returned separately so the parser reads the original G-code
	// without an in-memory thumbnail-injection copy.
	settings.GenerateImage = false
	settings.ExportType = "gcode"

	result, err := s.Slice(ctx, filename, data, settings)
	if err != nil {
		return PreviewResult{}, err
	}
	defer os.RemoveAll(result.Workdir)

	if len(result.Files) != 1 {
		return PreviewResult{}, httpx.NewError(http.StatusInternalServerError, "Preview requires exactly one G-code file")
	}

	gcodePath := result.Files[0]
	toolpaths, err := extractToolpathPreviewMode(ctx, gcodePath, mode)
	if err != nil {
		var httpError *httpx.Error
		if errors.As(err, &httpError) {
			return PreviewResult{}, err
		}
		return PreviewResult{}, httpx.NewError(http.StatusInternalServerError, "Failed to build toolpath preview: "+err.Error())
	}
	thumb, err := neptune4ThumbnailBase64(data)
	if err != nil {
		return PreviewResult{}, httpx.NewError(http.StatusInternalServerError, "Image generation failed: "+err.Error())
	}

	return PreviewResult{
		UsesSupport:      toolpaths.SupportSegments > 0,
		PrintTimeSeconds: result.Metadata.PrintTimeSeconds,
		FilamentUsedG:    result.Metadata.FilamentUsedG,
		FilamentUsedMm:   result.Metadata.FilamentUsedMM,
		ThumbnailBase64:  thumb,
		Toolpaths:        toolpaths,
		Quantity:         result.Quantity,
		Workdir:          result.Workdir,
	}, nil
}

func (s *Service) buildArgs(inputPath string, inputDir string, outputDir string, settings Settings, debug *SliceDebug) ([]string, error) {
	return s.buildArgsForInputs([]string{inputPath}, inputDir, outputDir, settings, debug)
}

func (s *Service) buildArgsForInputs(inputPaths []string, inputDir string, outputDir string, settings Settings, debug *SliceDebug) ([]string, error) {
	settings, err := normalizeSliceSettings(settings)
	if err != nil {
		return nil, err
	}
	if len(inputPaths) != 1 || strings.ToLower(filepath.Ext(inputPaths[0])) != ".stl" {
		return nil, httpx.NewError(http.StatusBadRequest, "Exactly one STL input is supported")
	}
	args := make([]string, 0)
	args = append(args, "--slice", "1", "--clone-objects", strconv.Itoa(settings.Quantity), "--export-statistics")
	// Orca applies transforms in command-line order. Keep the explicit X/Y/Z
	// rotations before arrangement and before the mutually exclusive --orient.
	if settings.RotateX != nil {
		args = append(args, "--rotate-x", rotationArg(settings.RotateX))
	}
	if settings.RotateY != nil {
		args = append(args, "--rotate-y", rotationArg(settings.RotateY))
	}
	if settings.Rotate != nil {
		args = append(args, "--rotate", rotationArg(settings.Rotate))
	}
	if hasExplicitRotation(settings) {
		// Orca 2.5.0 centers volumes around the origin after every rotation.
		// Without this final transform, the lower half is left below Z=0 and
		// silently omitted from the sliced object.
		args = append(args, "--ensure-on-bed")
	}
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
			profile["enable_support"] = boolArg(*settings.EnableSupport)
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
			profile["enable_support"] = boolArg(*settings.EnableSupport)
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

	args = append(args, "--allow-newer-file", "--outputdir", outputDir)
	args = append(args, inputPaths...)
	return args, nil
}

// loadSelectedProfile carrega um perfil nomeado do DataPath ou da arvore vendor.
//
// A heranca e sempre resolvida aqui, e nao no CLI do Orca. Ate a 2.4.1 o
// --load-settings aceitava um preset cru com "inherits" e resolvia a cadeia
// usando a arvore vendor; na 2.5.0 o resolve_preset_config passou a exigir que o
// arquivo ja esteja carregado no bundle (ou venha de uma arvore vendor com
// manifesto), e um arquivo arbitrario em diretorio temporario e rejeitado com
// "The input preset file is invalid and can not be parsed". Resolver aqui e
// escrever o perfil ja achatado e a unica forma de o CLI 2.5.0 fatar.
//
// O parametro resolve existe apenas para compatibilidade de assinatura: a
// partir da 2.5.0 ele e ignorado, porque nao ha caminho viavel sem resolucao.
func (s *Service) loadSelectedProfile(category string, name string, overrides map[string]any, resolve bool) (map[string]any, error) {
	_ = resolve
	resolved, err := ResolveProfile(s.DataPath, s.OrcaProfilesPath, category, name, overrides)
	if err != nil {
		return nil, err
	}
	return resolved.Resolved, nil
}

func prepareProfileForSlicing(category string, profile map[string]any, sanitize bool) map[string]any {
	if sanitize {
		return sanitizeProfileForSlicing(category, profile)
	}
	return profile
}

func resultFiles(outputDir string, exportType string) ([]string, error) {
	if exportType != "" && exportType != "gcode" {
		return nil, fmt.Errorf("only G-code results are supported")
	}
	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return nil, err
	}

	files := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".gcode") {
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

func optionalBoolLogValue(value *bool) string {
	if value == nil {
		return "profile_default"
	}
	if *value {
		return "true"
	}
	return "false"
}

func optionalBrimLogValue(value *bool) string {
	if value == nil {
		return "profile_default"
	}
	if *value {
		return "auto_brim"
	}
	return "no_brim"
}

func (s *Service) resultCacheKey(models []ModelInput, settings Settings, debug SliceDebug) string {
	base := s.ResultCache.makeModelsKey(models, settings)
	if base == "" {
		return ""
	}
	executable, err := os.Open(s.OrcaSlicerPath)
	if err != nil {
		return ""
	}
	defer executable.Close()
	digest := sha256.New()
	if _, err := io.Copy(digest, executable); err != nil {
		return ""
	}
	return cacheDigest(struct {
		Version    int
		Inputs     string
		Executable string
		Printer    map[string]any
		Preset     map[string]any
		Filaments  []map[string]any
	}{2, base, hex.EncodeToString(digest.Sum(nil)), debug.Printer, debug.Preset, debug.Filaments})
}
