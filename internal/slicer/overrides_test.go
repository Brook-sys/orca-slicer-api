package slicer

import "testing"

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
