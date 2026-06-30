package slicer

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteResultHeaders(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "result.gcode")
	if err := os.WriteFile(file, []byte("G28"), 0o644); err != nil {
		t.Fatal(err)
	}

	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/slice", nil)
	err := WriteResult(res, req, Result{
		Files: []string{file},
		Metadata: Metadata{
			PrintTimeSeconds: 10,
			FilamentUsedG:    1.2,
			FilamentUsedMM:   100,
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if res.Header().Get("X-Print-Time-Seconds") != "10" {
		t.Fatalf("expected print time header")
	}
	if res.Header().Get("X-Filament-Used-g") != "1.2" {
		t.Fatalf("expected filament g header")
	}
	if res.Header().Get("X-Filament-Used-mm") != "100" {
		t.Fatalf("expected filament mm header")
	}
}
