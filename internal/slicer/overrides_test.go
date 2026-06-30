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
	_, err := ResolveProfile(t.TempDir(), "", "presets", "missing", nil)
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

func TestResolveProfileInheritsByInternalName(t *testing.T) {
	dir := t.TempDir()
	profileDir := filepath.Join(dir, "presets")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profileDir, "base_file.json"), []byte(`{"name":"Base Profile @Vendor","layer_height":"0.20","speed":"100"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profileDir, "child.json"), []byte(`{"name":"Child","inherits":"Base Profile @Vendor","speed":"120"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	resolved, err := ResolveProfile(dir, "", "presets", "child", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Resolved["layer_height"] != "0.20" {
		t.Fatalf("expected inherited layer height")
	}
	if resolved.Resolved["speed"] != "120" {
		t.Fatalf("expected child override")
	}
	if _, ok := resolved.Resolved["inherits"]; ok {
		t.Fatalf("expected inherits to be removed")
	}
}

func TestResolveProfileInheritsFromBuiltInProfiles(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	builtInDir := filepath.Join(dir, "resources", "profiles", "Elegoo", "machine", "EN4SERIES")
	if err := os.MkdirAll(filepath.Join(dataDir, "printers"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(builtInDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(builtInDir, "base.json"), []byte(`{"name":"Elegoo Neptune 4 0.4 nozzle","printable_height":"265"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "printers", "custom.json"), []byte(`{"name":"Custom","inherits":"Elegoo Neptune 4 0.4 nozzle","printable_area":"0x0,235x0,235x235,0x235"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	resolved, err := ResolveProfile(dataDir, filepath.Join(dir, "resources", "profiles"), "printers", "custom", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Resolved["printable_height"] != "265" {
		t.Fatalf("expected built-in inherited value")
	}
	if resolved.Resolved["printable_area"] == "" {
		t.Fatalf("expected custom value")
	}
}

func TestResolveProfileFindsBuiltInProcessInArbitrarySubfolder(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	builtInDir := filepath.Join(dir, "resources", "profiles", "SomeVendor", "deep", "process", "EN4SERIES")
	if err := os.MkdirAll(filepath.Join(dataDir, "presets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(builtInDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(builtInDir, "base.json"), []byte(`{"name":"0.20mm Standard @Elegoo Neptune4 (0.4 nozzle)","layer_height":"0.2"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "presets", "custom.json"), []byte(`{"name":"Custom","inherits":"0.20mm Standard @Elegoo Neptune4 (0.4 nozzle)","speed":"120"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	resolved, err := ResolveProfile(dataDir, filepath.Join(dir, "resources", "profiles"), "presets", "custom", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Resolved["layer_height"] != "0.2" {
		t.Fatalf("expected inherited process profile from arbitrary subfolder")
	}
}

func TestResolveProfileFindsBuiltInByPrintSettingsID(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	builtInDir := filepath.Join(dir, "resources", "profiles", "Elegoo", "process", "EN4SERIES")
	if err := os.MkdirAll(filepath.Join(dataDir, "presets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(builtInDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(builtInDir, "some-file.json"), []byte(`{"print_settings_id":"0.20mm Standard @Elegoo Neptune4 (0.4 nozzle)","layer_height":"0.2"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "presets", "custom.json"), []byte(`{"name":"Custom","inherits":"0.20mm Standard @Elegoo Neptune4 (0.4 nozzle)"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	resolved, err := ResolveProfile(dataDir, filepath.Join(dir, "resources", "profiles"), "presets", "custom", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Resolved["layer_height"] != "0.2" {
		t.Fatalf("expected inherited value from print_settings_id")
	}
}

func TestResolveProfileUsesCustomAlias(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	builtInDir := filepath.Join(dir, "resources", "profiles", "Elegoo", "process")
	if err := os.MkdirAll(filepath.Join(dataDir, "presets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(builtInDir, 0o755); err != nil {
		t.Fatal(err)
	}
	service := &Service{DataPath: dataDir}
	if _, err := service.SaveAlias(ProfileAlias{Category: "presets", From: "Strange Old Name", To: "Real New Name"}); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(builtInDir, "real.json"), []byte(`{"name":"Real New Name","layer_height":"0.2"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "presets", "custom.json"), []byte(`{"name":"Custom","inherits":"Strange Old Name"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	resolved, err := ResolveProfile(dataDir, filepath.Join(dir, "resources", "profiles"), "presets", "custom", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Resolved["layer_height"] != "0.2" {
		t.Fatalf("expected inherited value through custom alias")
	}
}

func TestEnsureCompatibleProfileAddsPrinterNames(t *testing.T) {
	profile := map[string]any{
		"compatible_printers": []any{"Base Printer"},
		"compatible_printers_condition": "some condition",
	}
	printer := map[string]any{
		"name": "Custom Printer",
		"printer_settings_id": "Custom Printer Settings",
	}

	ensureCompatibleProfile(profile, printer)

	printers := profile["compatible_printers"].([]any)
	if len(printers) != 3 {
		t.Fatalf("expected three compatible printers, got %d", len(printers))
	}
	if printers[1] != "Custom Printer" || printers[2] != "Custom Printer Settings" {
		t.Fatalf("expected custom printer names to be added")
	}
	if profile["compatible_printers_condition"] != "" {
		t.Fatalf("expected compatibility condition to be cleared")
	}
}

func TestEnsureCompatibleProfileAppendsToStringList(t *testing.T) {
	profile := map[string]any{"compatible_printers": "Base Printer;Other Printer"}
	printer := map[string]any{"name": "Custom Printer"}

	ensureCompatibleProfile(profile, printer)

	if profile["compatible_printers"] != "Base Printer;Other Printer;Custom Printer" {
		t.Fatalf("expected semicolon string list to be preserved and extended")
	}
}

func TestResolveProfileMissingParentReturnsClearError(t *testing.T) {
	dir := t.TempDir()
	profileDir := filepath.Join(dir, "presets")
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(profileDir, "child.json"), []byte(`{"name":"Child","inherits":"Missing Parent"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ResolveProfile(dir, "", "presets", "child", nil)
	if err == nil {
		t.Fatalf("expected missing parent error")
	}
	var httpErr *httpx.Error
	if !errors.As(err, &httpErr) {
		t.Fatalf("expected http error")
	}
	if httpErr.Status != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", httpErr.Status)
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

	resolved, err := ResolveProfile(dir, "", "presets", "standard", map[string]any{"layer_height": "0.16", "new_key": "value"})
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
