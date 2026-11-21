package slicer

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestServiceSliceWithFakeSlicer(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test")
	}

	dir := t.TempDir()
	fake := filepath.Join(dir, "fake-slicer.sh")
	if err := os.WriteFile(fake, []byte(`#!/bin/sh
out=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--outputdir" ]; then
    shift
    out="$1"
  fi
  shift
done
mkdir -p "$out"
cat > "$out/plate_1.gcode" <<'EOF'
; total estimated time: 1h 2m 3s
; filament used [mm] = 1234.5
; filament used [g] = 12.3
G28
EOF
cat > "$out/result.json" <<'EOF'
{"return_code":0,"plate_index":1,"sliced_plates":[{"id":1}]}
EOF
cat > "$out/plate_1.statistics.json" <<'EOF'
{"version":1,"generator":"OrcaSlicer 2.5.0","objects":[{"name":"model.stl_1","instance":0}]}
EOF
`), 0o755); err != nil {
		t.Fatal(err)
	}

	state := NewStateStore(dir)
	service := &Service{DataPath: dir, OrcaSlicerPath: fake, Timeout: 5 * time.Second, State: state}
	result, err := service.Slice(context.Background(), "model.stl", []byte("solid model"), Settings{})
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(result.Workdir)

	if len(result.Files) != 1 || filepath.Ext(result.Files[0]) != ".gcode" {
		t.Fatalf("expected one G-code file, got %#v", result.Files)
	}
	if result.Quantity != 1 || len(result.Statistics.Objects) != 1 {
		t.Fatalf("expected validated quantity evidence, got quantity=%d statistics=%#v", result.Quantity, result.Statistics)
	}
	if result.Metadata.PrintTimeSeconds != 3723 {
		t.Fatalf("expected metadata print time, got %v", result.Metadata.PrintTimeSeconds)
	}
	if service.Status().Status != StatusCompleted {
		t.Fatalf("expected completed status")
	}
}

func TestServicePreviewWithFakeSlicerDetectsSupportAndThumbnail(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test")
	}

	dir := t.TempDir()
	fake := filepath.Join(dir, "fake-preview-slicer.sh")
	if err := os.WriteFile(fake, []byte(`#!/bin/sh
out=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--outputdir" ]; then
    shift
    out="$1"
  fi
  shift
done
mkdir -p "$out"
cat > "$out/plate_1.gcode" <<'EOF'
; total estimated time: 10m 1s
; filament used [mm] = 500
; filament used [g] = 5
G90
M83
;LAYER_CHANGE
;Z:0.3
;TYPE:Support
G1 X0 Y0 Z0.3 F600
G1 X1 Y0 E0.1
EOF
cat > "$out/result.json" <<'EOF'
{"return_code":0,"plate_index":1,"sliced_plates":[{"id":1}]}
EOF
cat > "$out/plate_1.statistics.json" <<'EOF'
{"version":1,"generator":"OrcaSlicer 2.5.0","objects":[{"name":"model.stl_1","instance":0}]}
EOF
`), 0o755); err != nil {
		t.Fatal(err)
	}

	service := &Service{DataPath: dir, OrcaSlicerPath: fake, Timeout: 5 * time.Second, State: NewStateStore(dir)}
	result, err := service.Preview(context.Background(), "model.stl", testASCIISTL(), Settings{})
	if err != nil {
		t.Fatal(err)
	}
	if !result.UsesSupport {
		t.Fatal("expected preview to detect support")
	}
	if result.ThumbnailBase64 == "" {
		t.Fatal("expected preview thumbnail")
	}
	if result.PrintTimeSeconds != 601 {
		t.Fatalf("expected preview print time 601, got %v", result.PrintTimeSeconds)
	}
	if result.FilamentUsedG != 5 || result.FilamentUsedMm != 500 {
		t.Fatalf("expected filament metadata, got g=%v mm=%v", result.FilamentUsedG, result.FilamentUsedMm)
	}
	if result.Quantity != 1 || !result.Toolpaths.SourceComplete || result.Toolpaths.Truncated || result.Toolpaths.TotalSegments != 1 {
		t.Fatalf("expected one complete toolpath segment, got %#v", result.Toolpaths)
	}
	if len(result.Toolpaths.Layers) != 1 || len(result.Toolpaths.Layers[0].Segments) != 1 || result.Toolpaths.Layers[0].Segments[0].Feature != "Support" {
		t.Fatalf("expected actual support toolpath, got %#v", result.Toolpaths.Layers)
	}
}

func TestServicePreviewWithFakeSlicerWithoutSupport(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test")
	}

	dir := t.TempDir()
	fake := filepath.Join(dir, "fake-preview-no-support.sh")
	if err := os.WriteFile(fake, []byte(`#!/bin/sh
out=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--outputdir" ]; then
    shift
    out="$1"
  fi
  shift
done
mkdir -p "$out"
cat > "$out/plate_1.gcode" <<'EOF'
; total estimated time: 1m
G90
M83
;LAYER_CHANGE
;Z:0.2
;TYPE:Brim
G1 X0 Y0 Z0.2 F600
G1 X2 Y0 E0.2
EOF
cat > "$out/result.json" <<'EOF'
{"return_code":0,"plate_index":1,"sliced_plates":[{"id":1}]}
EOF
cat > "$out/plate_1.statistics.json" <<'EOF'
{"version":1,"generator":"OrcaSlicer 2.5.0","objects":[{"name":"model.stl_1","instance":0}]}
EOF
`), 0o755); err != nil {
		t.Fatal(err)
	}

	service := &Service{DataPath: dir, OrcaSlicerPath: fake, Timeout: 5 * time.Second, State: NewStateStore(dir)}
	result, err := service.Preview(context.Background(), "model.stl", testASCIISTL(), Settings{})
	if err != nil {
		t.Fatal(err)
	}
	if result.UsesSupport {
		t.Fatal("expected preview not to detect support")
	}
	if result.ThumbnailBase64 == "" {
		t.Fatal("expected preview thumbnail")
	}
	if result.Toolpaths.TotalSegments != 1 || result.Toolpaths.Layers[0].Segments[0].Feature != "Brim" {
		t.Fatalf("expected actual brim toolpath, got %#v", result.Toolpaths)
	}
}

func TestServiceSliceTimeoutWithFakeSlicer(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script test")
	}

	dir := t.TempDir()
	fake := filepath.Join(dir, "slow-slicer.sh")
	if err := os.WriteFile(fake, []byte("#!/bin/sh\nsleep 2\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	service := &Service{DataPath: dir, OrcaSlicerPath: fake, Timeout: 10 * time.Millisecond, State: NewStateStore(dir)}
	_, err := service.Slice(context.Background(), "model.stl", []byte("solid model"), Settings{})
	if err == nil {
		t.Fatalf("expected timeout error")
	}
	if service.Status().Status != StatusCancelled {
		t.Fatalf("expected cancelled status, got %s", service.Status().Status)
	}
}
