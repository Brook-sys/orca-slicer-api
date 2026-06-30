package slicer

import "testing"

func TestParseMetadata(t *testing.T) {
	metadata := ParseMetadata(`; total estimated time: 1h 2m 3s
; filament used [mm] = 1234.5
; filament used [g] = 12.3`)

	if metadata.PrintTimeSeconds != 3723 {
		t.Fatalf("expected 3723 seconds, got %v", metadata.PrintTimeSeconds)
	}
	if metadata.FilamentUsedMM != 1234.5 {
		t.Fatalf("expected 1234.5 mm, got %v", metadata.FilamentUsedMM)
	}
	if metadata.FilamentUsedG != 12.3 {
		t.Fatalf("expected 12.3 g, got %v", metadata.FilamentUsedG)
	}
}
