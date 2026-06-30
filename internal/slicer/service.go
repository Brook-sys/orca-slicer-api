package slicer

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Brook-sys/orca-slicer-api/internal/httpx"
)

type Service struct {
	DataPath       string
	OrcaSlicerPath string
	Timeout        time.Duration
}

func (s Service) Slice(ctx context.Context, filename string, data []byte, settings Settings) (Result, error) {
	if s.OrcaSlicerPath == "" {
		return Result{}, httpx.NewError(http.StatusInternalServerError, "ORCASLICER_PATH is not configured")
	}

	workdir, err := os.MkdirTemp("", "slice-*")
	if err != nil {
		return Result{}, err
	}

	inputDir := filepath.Join(workdir, "input")
	outputDir := filepath.Join(workdir, "output")
	if err := os.MkdirAll(inputDir, 0o755); err != nil {
		return Result{}, err
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return Result{}, err
	}

	inputPath := filepath.Join(inputDir, filepath.Base(filename))
	if err := os.WriteFile(inputPath, data, 0o644); err != nil {
		return Result{}, err
	}

	args, err := s.buildArgs(inputPath, inputDir, outputDir, settings)
	if err != nil {
		return Result{}, err
	}

	cmdCtx := ctx
	cancel := func() {}
	if s.Timeout > 0 {
		cmdCtx, cancel = context.WithTimeout(ctx, s.Timeout)
	}
	defer cancel()

	cmd := exec.CommandContext(cmdCtx, s.OrcaSlicerPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		slicerError := readSlicerError(outputDir)
		if slicerError != "" {
			return Result{}, httpx.NewError(http.StatusInternalServerError, "Slicing failed: "+slicerError)
		}
		return Result{}, httpx.NewError(http.StatusInternalServerError, "Slicing failed: "+strings.TrimSpace(string(output)))
	}

	files, err := resultFiles(outputDir, settings.ExportType)
	if err != nil {
		return Result{}, err
	}
	if len(files) == 0 {
		return Result{}, httpx.NewError(http.StatusInternalServerError, "No files generated during slicing")
	}

	return Result{Files: files, Workdir: workdir, Metadata: AggregateMetadata(files)}, nil
}

func (s Service) buildArgs(inputPath string, inputDir string, outputDir string, settings Settings) ([]string, error) {
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
	filamentPath := ""

	if settings.Printer != "" {
		printerPath = filepath.Join(inputDir, "printer.json")
		if err := writeResolvedProfile(s.DataPath, "printers", settings.Printer, settings.Overrides["printer"], printerPath); err != nil {
			return nil, fmt.Errorf("printer profile: %w", err)
		}
	}

	if settings.Preset != "" {
		presetPath = filepath.Join(inputDir, "preset.json")
		if err := writeResolvedProfile(s.DataPath, "presets", settings.Preset, settings.Overrides["preset"], presetPath); err != nil {
			return nil, fmt.Errorf("preset profile: %w", err)
		}
	}

	if settings.Filament != "" {
		filamentPath = filepath.Join(inputDir, "filament.json")
		if err := writeResolvedProfile(s.DataPath, "filaments", settings.Filament, settings.Overrides["filament"], filamentPath); err != nil {
			return nil, fmt.Errorf("filament profile: %w", err)
		}
	}

	if printerPath != "" && presetPath != "" {
		args = append(args, "--load-settings", printerPath+";"+presetPath)
	}
	if filamentPath != "" {
		args = append(args, "--load-filaments", filamentPath)
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

func boolArg(value bool) string {
	if value {
		return "1"
	}
	return "0"
}
