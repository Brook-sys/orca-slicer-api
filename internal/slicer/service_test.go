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
