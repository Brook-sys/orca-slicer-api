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
cat > "$out/result.gcode" <<'EOF'
; total estimated time: 1h 2m 3s
; filament used [mm] = 1234.5
; filament used [g] = 12.3
G28
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

	if len(result.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(result.Files))
	}
	if result.Metadata.PrintTimeSeconds != 3723 {
		t.Fatalf("expected metadata print time, got %v", result.Metadata.PrintTimeSeconds)
	}
	if service.Status().Status != StatusCompleted {
		t.Fatalf("expected completed status")
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
