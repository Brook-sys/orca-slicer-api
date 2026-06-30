package slicer

import (
	"os"
	"path/filepath"
	"testing"
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
