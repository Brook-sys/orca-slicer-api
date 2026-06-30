package slicer

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHandlerResolveProfiles(t *testing.T) {
	dir := t.TempDir()
	for _, category := range []string{"printers", "presets", "filaments"} {
		if err := os.MkdirAll(filepath.Join(dir, category), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	_ = os.WriteFile(filepath.Join(dir, "printers", "printer.json"), []byte(`{"bed":"x1"}`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "presets", "preset.json"), []byte(`{"layer_height":"0.20"}`), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "filaments", "pla.json"), []byte(`{"temp":"210"}`), 0o644)

	handler := Handler{Service: &Service{DataPath: dir}}
	body, _ := json.Marshal(Settings{
		Printer:  "printer",
		Preset:   "preset",
		Filament: "pla",
		Overrides: map[string]map[string]any{
			"preset": {"layer_height": "0.16"},
		},
	})
	req := httptest.NewRequest(http.MethodPost, "/slice/resolve-profiles", bytes.NewReader(body))
	res := httptest.NewRecorder()

	handler.ResolveProfiles(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
	var resolved ResolveProfilesResponse
	if err := json.Unmarshal(res.Body.Bytes(), &resolved); err != nil {
		t.Fatal(err)
	}
	if resolved.Preset == nil || resolved.Preset.Resolved["layer_height"] != "0.16" {
		t.Fatalf("expected resolved preset override")
	}
}
