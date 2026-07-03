package slicer

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestListBuiltinProfiles(t *testing.T) {
	dir := t.TempDir()
	printersDir := filepath.Join(dir, "printers")
	if err := os.MkdirAll(printersDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(printersDir, "A.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(printersDir, "B.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	service := &Service{OrcaProfilesPath: dir}
	handler := Handler{Service: service}

	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/profiles/builtins/printers", nil)
	r.SetPathValue("category", "printers")

	handler.ListBuiltinProfiles(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var list []ProfileListItem
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatal(err)
	}

	if len(list) != 2 || list[0].Name != "A" || list[1].Name != "B" {
		t.Fatalf("expected sorted list A, B, got %#v", list)
	}
}

func TestListBuiltinProfilesInvalidCategory(t *testing.T) {
	handler := Handler{}
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/profiles/builtins/invalid", nil)
	r.SetPathValue("category", "invalid")
	handler.ListBuiltinProfiles(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for invalid category, got %d", w.Code)
	}
}
