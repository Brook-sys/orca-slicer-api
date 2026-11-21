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
		Files:    []string{file},
		Quantity: 1,
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
	if res.Header().Get("X-Slice-Quantity") != "1" || res.Header().Get("X-Slice-Plate") != "1" {
		t.Fatalf("expected exact quantity and plate headers, got %#v", res.Header())
	}
	if res.Header().Get("Content-Type") != "application/x-gcode" || res.Body.String() != "G28" {
		t.Fatalf("expected direct G-code response, got content-type=%q body=%q", res.Header().Get("Content-Type"), res.Body.String())
	}
}

func TestWriteResultRejectsMultipleFilesInsteadOfZIP(t *testing.T) {
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/slice", nil)
	err := WriteResult(res, req, Result{Files: []string{"plate_1.gcode", "plate_2.gcode"}, Quantity: 2})
	if err == nil {
		t.Fatal("expected multiple results to be rejected")
	}
	if res.Header().Get("Content-Type") == "application/zip" {
		t.Fatal("ZIP fallback is forbidden")
	}
}
