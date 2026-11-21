package slicer

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Brook-sys/orca-slicer-api/internal/httpx"
)

func TestNormalizeSliceSettingsDefaults(t *testing.T) {
	settings, err := normalizeSliceSettings(Settings{})
	if err != nil {
		t.Fatal(err)
	}
	if settings.Quantity != 1 || settings.Plate != "1" || settings.ExportType != "gcode" || settings.ArrangeSpacing != 0 {
		t.Fatalf("unexpected defaults: %#v", settings)
	}
}

func TestValidateSliceOutputRequiresExactObjectCount(t *testing.T) {
	for _, test := range []struct {
		name       string
		statistics string
		manifest   string
		extraGCode bool
		wantStatus int
	}{
		{
			name:       "reduced quantity is unprocessable",
			statistics: `{"version":1,"generator":"OrcaSlicer 2.5.0","objects":[{"name":"model_1","instance":0}]}`,
			manifest:   `{"return_code":0,"plate_index":1,"sliced_plates":[{"id":1}]}`,
			wantStatus: 422,
		},
		{
			name:       "statistics missing",
			statistics: "",
			manifest:   `{"return_code":0,"plate_index":1,"sliced_plates":[{"id":1}]}`,
			wantStatus: 500,
		},
		{
			name:       "statistics malformed",
			statistics: `{`,
			manifest:   `{"return_code":0,"plate_index":1,"sliced_plates":[{"id":1}]}`,
			wantStatus: 500,
		},
		{
			name:       "duplicate identities",
			statistics: `{"version":1,"generator":"OrcaSlicer 2.5.0","objects":[{"name":"model_1","instance":0},{"name":"model_1","instance":0}]}`,
			manifest:   `{"return_code":0,"plate_index":1,"sliced_plates":[{"id":1}]}`,
			wantStatus: 500,
		},
		{
			name:       "additional sliced plate",
			statistics: `{"version":1,"generator":"OrcaSlicer 2.5.0","objects":[{"name":"model_1","instance":0},{"name":"model_2","instance":0}]}`,
			manifest:   `{"return_code":0,"plate_index":1,"sliced_plates":[{"id":1},{"id":2}]}`,
			wantStatus: 422,
		},
		{
			name:       "additional G-code",
			statistics: `{"version":1,"generator":"OrcaSlicer 2.5.0","objects":[{"name":"model_1","instance":0},{"name":"model_2","instance":0}]}`,
			manifest:   `{"return_code":0,"plate_index":1,"sliced_plates":[{"id":1}]}`,
			extraGCode: true,
			wantStatus: 422,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "plate_1.gcode"), []byte("G28\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			if test.extraGCode {
				_ = os.WriteFile(filepath.Join(dir, "plate_2.gcode"), []byte("G28\n"), 0o600)
			}
			if test.manifest != "" {
				_ = os.WriteFile(filepath.Join(dir, "result.json"), []byte(test.manifest), 0o600)
			}
			if test.statistics != "" {
				_ = os.WriteFile(filepath.Join(dir, "plate_1.statistics.json"), []byte(test.statistics), 0o600)
			}
			_, _, err := validateSliceOutput(dir, 2)
			if err == nil {
				t.Fatal("expected output validation to fail")
			}
			var httpError *httpx.Error
			if !errors.As(err, &httpError) || httpError.Status != test.wantStatus {
				t.Fatalf("expected HTTP %d, got %#v", test.wantStatus, err)
			}
		})
	}
}

func TestValidateSliceOutputRejectsNonCanonicalGCode(t *testing.T) {
	for _, name := range []string{"result.gcode", "plate_2.gcode"} {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			_ = os.WriteFile(filepath.Join(dir, name), []byte("G28\n"), 0o600)
			_ = os.WriteFile(filepath.Join(dir, "result.json"), []byte(`{"return_code":0,"plate_index":1,"sliced_plates":[{"id":1}]}`), 0o600)
			_ = os.WriteFile(filepath.Join(dir, "plate_1.statistics.json"), []byte(`{"version":1,"generator":"OrcaSlicer 2.5.0","objects":[{"name":"model_1","instance":0}]}`), 0o600)
			_, _, err := validateSliceOutput(dir, 1)
			if err == nil {
				t.Fatalf("%s must not be bound to plate 1 by sidecar files alone", name)
			}
		})
	}
}

func TestValidateSliceOutputAcceptsExactSinglePlate(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "plate_1.gcode"), []byte("G28\n"), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "result.json"), []byte(`{"return_code":0,"plate_index":1,"sliced_plates":[{"id":1}]}`), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "plate_1.statistics.json"), []byte(`{"version":1,"generator":"OrcaSlicer 2.5.0","objects":[{"name":"model_1","instance":0},{"name":"model_2","instance":0}]}`), 0o600)
	files, statistics, err := validateSliceOutput(dir, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || len(statistics.Objects) != 2 {
		t.Fatalf("unexpected valid result: files=%#v statistics=%#v", files, statistics)
	}
}
