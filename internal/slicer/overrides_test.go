package slicer

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/Brook-sys/orca-slicer-api/internal/httpx"
)

func TestMerge(t *testing.T) {
	base := map[string]any{
		"layer_height": "0.20",
		"nested": map[string]any{
			"speed": "100",
			"temp":  "210",
		},
	}
	overrides := map[string]any{
		"layer_height": "0.16",
		"nested": map[string]any{
			"speed": "120",
		},
	}

	result := merge(base, overrides)
	if result["layer_height"] != "0.16" {
		t.Fatalf("expected override layer height")
	}
	nested := result["nested"].(map[string]any)
	if nested["speed"] != "120" {
		t.Fatalf("expected nested speed override")
	}
	if nested["temp"] != "210" {
		t.Fatalf("expected nested temp to remain")
	}
}

func TestMissingKeys(t *testing.T) {
	warnings := missingKeys(map[string]any{"known": "value"}, map[string]any{"unknown": "value"}, "")
	if len(warnings) != 1 {
		t.Fatalf("expected warning for unknown key")
	}
}

func TestResolveProfileMissingReturnsHTTPError(t *testing.T) {
	_, err := ResolveProfile(t.TempDir(), "presets", "missing", nil)
	if err == nil {
		t.Fatalf("expected error")
	}
	var httpErr *httpx.Error
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected http error, got %T", err)
	}
	if httpErr.Status != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", httpErr.Status)
	}
}

func TestResolveProfile(t *testing.T) {
	dir := t.TempDir()
	profileDir := filepath.Join(dir, "presets")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profileDir, "standard.json"), []byte(`{"layer_height":"0.20","speed":"100"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	resolved, err := ResolveProfile(dir, "presets", "standard", map[string]any{"layer_height": "0.16", "new_key": "value"})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Resolved["layer_height"] != "0.16" {
		t.Fatalf("expected layer height override")
	}
	if len(resolved.Warnings) != 1 {
		t.Fatalf("expected warning for new key")
	}
}
