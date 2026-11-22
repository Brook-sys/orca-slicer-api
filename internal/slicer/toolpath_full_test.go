package slicer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/Brook-sys/orca-slicer-api/internal/httpx"
)

func TestFullToolpathPreviewMetadataAndGeometryCompleteness(t *testing.T) {
	path := writeToolpathFixture(t, `G90
M83
;LAYER_CHANGE
;Z:0.2
;TYPE:Bridge
G1 X0 Y0 Z0.2 F600
G1 X1 Y0 E0.1
;HEIGHT:0.2
;WIDTH:0.42
G1 X2 Y0 E0.1
;TYPE:Support interface
G1 Z0.4 E0.02
;WIDTH:0.5
G1 X2 Y1 E0.03
`)
	legacy, err := extractToolpathPreviewMode(context.Background(), path, toolpathModeLegacy)
	if err != nil {
		t.Fatal(err)
	}
	if legacy.Version != 1 || legacy.Mode != "" || legacy.GeometryComplete || legacy.TotalSegments != 3 || legacy.Truncated {
		t.Fatalf("legacy selection changed: %#v", legacy)
	}
	for _, layer := range legacy.Layers {
		for _, segment := range layer.Segments {
			if segment.WidthMM != nil || segment.HeightMM != nil || segment.fullMode {
				t.Fatalf("legacy segment acquired full metadata: %#v", segment)
			}
		}
	}

	preview, err := extractToolpathPreviewMode(context.Background(), path, toolpathModeFull)
	if err != nil {
		t.Fatal(err)
	}
	if preview.Version != 2 || preview.Mode != toolpathModeFull || !preview.SourceComplete || !preview.GeometryComplete || preview.Truncated {
		t.Fatalf("full preview is not explicitly complete: %#v", preview)
	}
	if preview.TotalSegments != 4 || preview.ReturnedSegments != 4 || preview.OmittedSegments != 0 || preview.SupportSegments != 2 {
		t.Fatalf("unexpected full XYZ/support accounting: %#v", preview)
	}
	if preview.Limits.Segments != maxFullToolpathSegments || preview.Limits.Layers != maxFullToolpathLayers {
		t.Fatalf("wrong full limits: %#v", preview.Limits)
	}
	segments := preview.Layers[0].Segments
	if segments[0].Feature != "Bridge" || segments[0].WidthMM != nil || segments[0].HeightMM != nil {
		t.Fatalf("missing metadata must stay unknown: %#v", segments[0])
	}
	assertDimension(t, segments[1].WidthMM, 0.42)
	assertDimension(t, segments[1].HeightMM, 0.2)
	if segments[2].From != [3]float64{2, 0, 0.2} || segments[2].To != [3]float64{2, 0, 0.4} || segments[2].Feature != "Support interface" {
		t.Fatalf("positive-extrusion Z geometry/category was lost: %#v", segments[2])
	}
	assertDimension(t, segments[2].WidthMM, 0.42)
	assertDimension(t, segments[2].HeightMM, 0.2)
	assertDimension(t, segments[3].WidthMM, 0.5)
	assertDimension(t, segments[3].HeightMM, 0.2)

	encoded, err := json.Marshal(preview)
	if err != nil {
		t.Fatal(err)
	}
	var wire struct {
		Version          int    `json:"version"`
		Mode             string `json:"mode"`
		GeometryComplete bool   `json:"geometryComplete"`
		Layers           []struct {
			Segments []map[string]any `json:"segments"`
		} `json:"layers"`
	}
	if err := json.Unmarshal(encoded, &wire); err != nil {
		t.Fatal(err)
	}
	if wire.Version != 2 || wire.Mode != "full" || !wire.GeometryComplete {
		t.Fatalf("unexpected full wire envelope: %s", encoded)
	}
	first := wire.Layers[0].Segments[0]
	if value, found := first["widthMm"]; !found || value != nil {
		t.Fatalf("unknown width must be explicit null: %#v", first)
	}
	if value, found := first["heightMm"]; !found || value != nil {
		t.Fatalf("unknown height must be explicit null: %#v", first)
	}
	if got := wire.Layers[0].Segments[1]["widthMm"]; got != 0.42 {
		t.Fatalf("widthMm = %#v", got)
	}
	if got := wire.Layers[0].Segments[1]["heightMm"]; got != 0.2 {
		t.Fatalf("heightMm = %#v", got)
	}
	legacyJSON, err := json.Marshal(legacy)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{`"mode"`, `"geometryComplete"`, `"widthMm"`, `"heightMm"`} {
		if strings.Contains(string(legacyJSON), key) {
			t.Fatalf("legacy JSON gained %s: %s", key, legacyJSON)
		}
	}
}

func TestFullToolpathPreviewPreservesFeatureAndObjectCategories(t *testing.T) {
	path := writeToolpathFixture(t, `G90
M83
;LAYER_CHANGE
;Z:0.2
;printing object model id:7 copy 0
;TYPE:Support
G1 X1 E0.01
;TYPE:Brim
G1 X2 E0.01
;TYPE:Bridge
G1 X3 E0.01
;TYPE:Sparse infill
G1 X4 E0.01
;WIPE_START
G1 X5 E0.01
;WIPE_END
;TYPE:Custom infill category
G1 X6 E0.01
`)
	preview, err := extractToolpathPreviewMode(context.Background(), path, toolpathModeFull)
	if err != nil {
		t.Fatal(err)
	}
	segments := preview.Layers[0].Segments
	wantFeatures := []string{"Support", "Brim", "Bridge", "Sparse infill", "Wipe", "Custom infill category"}
	if len(segments) != len(wantFeatures) {
		t.Fatalf("segments = %d, want %d", len(segments), len(wantFeatures))
	}
	for index, want := range wantFeatures {
		if segments[index].Feature != want || segments[index].Object != "model" {
			t.Errorf("segment %d feature/object = %q/%q, want %q/model", index, segments[index].Feature, segments[index].Object, want)
		}
	}
	if preview.SupportSegments != 1 || preview.BrimSegments != 1 {
		t.Fatalf("category counters changed: support=%d brim=%d", preview.SupportSegments, preview.BrimSegments)
	}
}

func TestToolpathSegmentLegacyJSONCompatibility(t *testing.T) {
	segment := ToolpathSegment{From: [3]float64{0, 0, 0}, To: [3]float64{1, 0, 0}, Extrusion: 0.1}
	encoded, err := json.Marshal(segment)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"from":[0,0,0],"to":[1,0,0],"extrusion":0.1,"tool":0}`
	if string(encoded) != want {
		t.Fatalf("legacy JSON = %s, want %s", encoded, want)
	}
}

func TestFullToolpathPreviewRetainsBeyondLegacyLimits(t *testing.T) {
	var gcode strings.Builder
	gcode.WriteString("G90\nM83\n")
	for layer := 0; layer <= maxToolpathLayers; layer++ {
		fmt.Fprintf(&gcode, ";LAYER_CHANGE\n;Z:%0.1f\n;HEIGHT:0.2\n;TYPE:Bridge\n;WIDTH:0.42\n", float64(layer)*0.2+0.2)
		for index := 0; index < 40; index++ {
			fmt.Fprintf(&gcode, "G1 X%d Y%d Z%0.1f E0.01\n", index+1, layer, float64(layer)*0.2+0.2)
		}
	}
	preview, err := extractToolpathPreviewMode(context.Background(), writeToolpathFixture(t, gcode.String()), toolpathModeFull)
	if err != nil {
		t.Fatal(err)
	}
	wantSegments := (maxToolpathLayers + 1) * 40
	if !preview.GeometryComplete || preview.Truncated || preview.ReturnedSegments != wantSegments || preview.TotalSegments != wantSegments || preview.ReturnedLayers != maxToolpathLayers+1 {
		t.Fatalf("full mode did not retain beyond both legacy limits: %#v", preview)
	}
}

func TestFullToolpathPreviewBudgetFailuresAreExplicit(t *testing.T) {
	t.Run("segments", func(t *testing.T) {
		parser := newTestFullToolpathParser(1, 2)
		for _, line := range []string{"G90", "M83", "G1 X1 E0.1", "G1 X2 E0.1", "G1 X3 E0.1"} {
			err := parser.parseLine(line)
			if line == "G1 X3 E0.1" {
				assertFullBudgetError(t, err, "2-segment retention limit")
			} else if err != nil {
				t.Fatal(err)
			}
		}
		if parser.preview.GeometryComplete {
			t.Fatal("failed full parse claimed complete geometry")
		}
	})
	t.Run("layers", func(t *testing.T) {
		parser := newTestFullToolpathParser(1, 2)
		if err := parser.parseLine(";LAYER_CHANGE"); err != nil {
			t.Fatal(err)
		}
		assertFullBudgetError(t, parser.parseLine(";LAYER_CHANGE"), "1-layer retention limit")
		if parser.preview.GeometryComplete {
			t.Fatal("failed full parse claimed complete geometry")
		}
	})
}

func TestFullToolpathPreviewConfiguredLimitsReturnNoPartialResult(t *testing.T) {
	for _, test := range []struct {
		name    string
		gcode   string
		message string
	}{
		{"layers", strings.Repeat(";LAYER_CHANGE\n", maxFullToolpathLayers+1), fmt.Sprintf("%d-layer retention limit", maxFullToolpathLayers)},
		{"segments", "G91\n" + strings.Repeat("G1 X0.001 E0.001\n", maxFullToolpathSegments+1), fmt.Sprintf("%d-segment retention limit", maxFullToolpathSegments)},
	} {
		t.Run(test.name, func(t *testing.T) {
			preview, err := extractToolpathPreviewMode(context.Background(), writeToolpathFixture(t, test.gcode), toolpathModeFull)
			assertFullBudgetError(t, err, test.message)
			if preview.Version != 0 || preview.GeometryComplete || preview.SourceComplete || len(preview.Layers) != 0 {
				t.Fatal("budget failure returned a partial preview")
			}
		})
	}
}

func TestFullToolpathPreviewRejectsInvalidDimensionsWithoutChangingLegacy(t *testing.T) {
	for _, comment := range []string{";WIDTH:0", ";WIDTH:NaN", ";WIDTH:nope", ";HEIGHT:-0.2", ";HEIGHT:Inf"} {
		t.Run(comment, func(t *testing.T) {
			path := writeToolpathFixture(t, "G90\nM83\n"+comment+"\nG1 X1 E0.1\n")
			if _, err := extractToolpathPreviewMode(context.Background(), path, toolpathModeFull); err == nil || !strings.Contains(err.Error(), "metadata") {
				t.Fatalf("expected full metadata error, got %v", err)
			}
			legacy, err := extractToolpathPreviewMode(context.Background(), path, toolpathModeLegacy)
			if err != nil || legacy.TotalSegments != 1 {
				t.Fatalf("legacy behavior changed for %q: %#v, %v", comment, legacy, err)
			}
		})
	}
}

func TestToolpathModeValidation(t *testing.T) {
	for _, mode := range []string{"", "legacy", "full"} {
		if _, err := normalizeToolpathMode(mode); err != nil {
			t.Fatalf("valid mode %q rejected: %v", mode, err)
		}
	}
	for _, mode := range []string{"FULL", "auto", " full", "1"} {
		err := func() error {
			_, err := normalizeToolpathMode(mode)
			return err
		}()
		var httpError *httpx.Error
		if !errors.As(err, &httpError) || httpError.Status != http.StatusBadRequest {
			t.Fatalf("invalid mode %q error = %#v", mode, err)
		}
	}
}

func newTestFullToolpathParser(layerLimit, segmentLimit int) *toolpathParser {
	return &toolpathParser{
		ctx:           context.Background(),
		unitsScale:    1,
		currentLayer:  -1,
		retainedLayer: -1,
		fullMode:      true,
		layerLimit:    layerLimit,
		segmentLimit:  segmentLimit,
		fallbackZ:     0,
	}
}

func assertFullBudgetError(t *testing.T, err error, message string) {
	t.Helper()
	var httpError *httpx.Error
	if !errors.As(err, &httpError) || httpError.Status != http.StatusUnprocessableEntity || !strings.Contains(err.Error(), message) || !strings.Contains(err.Error(), "no partial geometry") {
		t.Fatalf("expected explicit 422 %q error, got %#v", message, err)
	}
}

func assertDimension(t *testing.T, value *float64, want float64) {
	t.Helper()
	if value == nil || *value != want {
		t.Fatalf("dimension = %v, want %v", value, want)
	}
}
