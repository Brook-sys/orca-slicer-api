package slicer

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSliceReturnsBusyWhenLocked(t *testing.T) {
	service := &Service{}
	service.mu.Lock()
	defer service.mu.Unlock()

	_, err := service.Slice(context.Background(), "model.stl", []byte("model"), Settings{})
	if err == nil {
		t.Fatalf("expected busy error")
	}
	if err.Error() != "Slicer is busy" {
		t.Fatalf("expected busy error, got %v", err)
	}
}

func TestSliceStatusDefaultsIdle(t *testing.T) {
	service := &Service{}
	if service.Status().Status != StatusIdle {
		t.Fatalf("expected idle status")
	}
}

func TestPrepareProfileForSlicingSanitizesKnownProblemFields(t *testing.T) {
	printer := prepareProfileForSlicing("printers", map[string]any{"from": "User", "name": "Printer"}, true)
	if printer["from"] != "system" {
		t.Fatalf("expected printer from to be set to system")
	}
	preset := prepareProfileForSlicing("presets", map[string]any{"from": "User", "small_perimeter_speed": "0", "name": "Preset"}, true)
	if preset["from"] != "system" {
		t.Fatalf("expected preset from to be set to system")
	}
	if _, ok := preset["small_perimeter_speed"]; ok {
		t.Fatalf("expected preset small_perimeter_speed to be removed")
	}
}

func TestBuildArgsCanResolveNamedProfiles(t *testing.T) {
	dataDir := t.TempDir()
	builtInDir := filepath.Join(dataDir, "builtins", "Elegoo", "process")
	if err := os.MkdirAll(filepath.Join(dataDir, "printers"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "presets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(builtInDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "printers", "printer.json"), []byte(`{"name":"Printer"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(builtInDir, "base.json"), []byte(`{"name":"Base","layer_height":"0.2"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "presets", "preset.json"), []byte(`{"name":"Preset","inherits":"Base","speed":"100"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	service := Service{DataPath: dataDir, OrcaProfilesPath: filepath.Join(dataDir, "builtins")}
	workdir := t.TempDir()
	inputDir := filepath.Join(workdir, "input")
	outputDir := filepath.Join(workdir, "output")
	if err := os.MkdirAll(inputDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}

	debug := SliceDebug{}
	_, err := service.buildArgs(filepath.Join(inputDir, "model.stl"), inputDir, outputDir, Settings{Printer: "printer", Preset: "preset", ResolveProfiles: true}, &debug)
	if err != nil {
		t.Fatal(err)
	}
	if debug.Preset["layer_height"] != "0.2" {
		t.Fatalf("expected resolved inherited preset")
	}
	if _, ok := debug.Preset["inherits"]; ok {
		t.Fatalf("expected inherits to be removed")
	}
}

func TestBuildArgsUsesUploadedProfilesBeforeSavedNames(t *testing.T) {
	service := Service{DataPath: t.TempDir()}
	workdir := t.TempDir()
	inputDir := filepath.Join(workdir, "input")
	outputDir := filepath.Join(workdir, "output")
	if err := os.MkdirAll(inputDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}

	debug := SliceDebug{}
	args, err := service.buildArgs(filepath.Join(inputDir, "model.stl"), inputDir, outputDir, Settings{
		Printer:        "saved-printer",
		Preset:         "saved-preset",
		Filament:       "saved-filament",
		PrinterProfile: map[string]any{"name": "uploaded-printer"},
		PresetProfile:  map[string]any{"name": "uploaded-preset"},
		FilamentProfiles: []map[string]any{
			{"name": "uploaded-filament-1"},
			{"name": "uploaded-filament-2"},
		},
	}, &debug)
	if err != nil {
		t.Fatal(err)
	}

	joined := strings.Join(args, " ")
	if !strings.Contains(joined, filepath.Join(inputDir, "printer.json")+";"+filepath.Join(inputDir, "preset.json")) {
		t.Fatalf("expected uploaded printer/preset load-settings, got %v", args)
	}
	if !strings.Contains(joined, "filament_1.json;"+filepath.Join(inputDir, "filament_2.json")) {
		t.Fatalf("expected multiple uploaded filaments, got %v", args)
	}
	if debug.Printer["name"] != "uploaded-printer" || debug.Preset["name"] != "uploaded-preset" {
		t.Fatalf("expected debug profiles from uploaded files")
	}
	if len(debug.Filaments) != 2 {
		t.Fatalf("expected two debug filaments")
	}
}
