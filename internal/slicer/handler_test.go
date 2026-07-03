package slicer

import (
	"net/http/httptest"
	"testing"
)

func TestParseSettingsUsesContainerDefaults(t *testing.T) {
	req := httptest.NewRequest("POST", "/slice", nil)
	settings, err := parseSettings(req, true, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if !settings.GenerateImage || !settings.ResolveProfiles || !settings.SanitizeProfiles {
		t.Fatalf("expected container defaults to be applied")
	}
}

func TestValidModelFile(t *testing.T) {
	valid := []string{"model.stl", "model.step", "model.stp", "model.3mf"}
	for _, name := range valid {
		if !validModelFile(name) {
			t.Fatalf("expected %s to be valid", name)
		}
	}
	if validModelFile("model.obj") {
		t.Fatalf("expected obj to be invalid")
	}
}
