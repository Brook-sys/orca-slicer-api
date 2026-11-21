package slicer

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeToolpathFixture(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "plate_1.gcode")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestToolpathPreviewIncludesActualSupportBrimAndObjects(t *testing.T) {
	path := writeToolpathFixture(t, `M104 S210
G21
G90
M83
;LAYER_CHANGE
;Z:0.3
;TYPE:Brim
G1 X10 Y10 Z0.3 F600
G1 X11 Y10 E0.1
; printing object model id:2 copy 0
;TYPE:Support interface
G1 X11 Y11 E0.2
; stop printing object model id:2 copy 0
`)
	preview, err := extractToolpathPreview(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.SourceComplete || preview.Truncated || preview.TotalSegments != 2 || preview.ReturnedSegments != 2 || preview.TotalLayers != 1 || preview.SupportSegments != 1 || preview.BrimSegments != 1 {
		t.Fatalf("unexpected preview totals: %#v", preview)
	}
	segments := preview.Layers[0].Segments
	if segments[0].Feature != "Brim" || segments[0].Object != "" || segments[1].Feature != "Support interface" || segments[1].Object != "model" {
		t.Fatalf("feature/object attribution failed: %#v", segments)
	}
	if segments[0].Feedrate != 10 || math.Abs(segments[1].Extrusion-0.2) > 1e-12 || preview.Bounds == nil || preview.Bounds.Max[0] != 11 || preview.Bounds.Max[1] != 11 {
		t.Fatalf("unexpected coordinates/feedrate: %#v bounds=%#v", segments, preview.Bounds)
	}
}

func TestToolpathPreviewDoesNotTreatFeatureCommentsAsExtrusion(t *testing.T) {
	path := writeToolpathFixture(t, ";TYPE:Support\n; support_material = 1\n;TYPE:Brim\nG1 X1 Y1\n")
	preview, err := extractToolpathPreview(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if preview.SupportSegments != 0 || preview.BrimSegments != 0 || preview.TotalSegments != 0 {
		t.Fatalf("comments without positive-E XY motion must not count as features: %#v", preview)
	}
}

func TestToolpathPreviewPositioningUnitsAndG92(t *testing.T) {
	path := writeToolpathFixture(t, `G20
G90
M82
;LAYER_CHANGE
;Z:0.1
G1 X1 Y0 Z0.1 E0 F60
G92 E0
G1 X2 Y0 E0.1
G91
M83
G1 X0.5 Y0 E0.1
`)
	preview, err := extractToolpathPreview(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if preview.TotalSegments != 2 {
		t.Fatalf("expected two extrusions, got %#v", preview)
	}
	segments := preview.Layers[0].Segments
	if segments[0].To[0] != 50.8 || segments[0].Extrusion != 2.54 || segments[0].Feedrate != 25.4 {
		t.Fatalf("inch conversion failed: %#v", segments[0])
	}
	if segments[1].From[0] != 50.8 || segments[1].To[0] != 63.5 || segments[1].Extrusion != 2.54 {
		t.Fatalf("relative mode failed: %#v", segments[1])
	}
}

func TestToolpathPreviewG92DoesNotMovePhysicalExtruder(t *testing.T) {
	path := writeToolpathFixture(t, `G21
G90
M82
;LAYER_CHANGE
;Z:0.2
G1 X0 Y0 Z0.2
G1 X1 Y0 E10
G92
G92 E5
G1 X2 Y0 E6
`)
	preview, err := extractToolpathPreview(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if preview.TotalSegments != 2 || preview.Layers[0].Segments[1].Extrusion != 1 {
		t.Fatalf("G92 must preserve physical E position and change only its origin: %#v", preview)
	}
}

func TestToolpathPreviewLinearizesArcToExactEndpoint(t *testing.T) {
	path := writeToolpathFixture(t, `G21
G90
M83
;LAYER_CHANGE
;Z:0.2
;TYPE:Outer wall
G1 X1 Y0 Z0.2
G3 X0 Y1 I-1 J0 E1
`)
	preview, err := extractToolpathPreview(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if preview.TotalSegments < 2 {
		t.Fatalf("expected tessellated arc, got %#v", preview)
	}
	segments := preview.Layers[0].Segments
	last := segments[len(segments)-1]
	if last.To[0] != 0 || last.To[1] != 1 || last.Feature != "Outer wall" {
		t.Fatalf("arc did not reach exact target: %#v", last)
	}
	sum := 0.0
	for _, segment := range segments {
		sum += segment.Extrusion
	}
	if sum != 1 {
		t.Fatalf("arc extrusion was not conserved: %v", sum)
	}
}

func TestToolpathPreviewBoundsTotalArcWork(t *testing.T) {
	// Each full circle stays below the per-move cap, but their sum crosses the
	// independent total-work cap after the retained segment budget is exhausted.
	radius := toolpathArcToleranceMM / (1 - math.Cos(math.Pi/maxToolpathArcSegments)) * 0.99
	var gcode strings.Builder
	gcode.WriteString("G21\nG90\nM83\n;LAYER_CHANGE\n;Z:0.2\n")
	for index := 0; index < maxToolpathArcWork/4000+2; index++ {
		fmt.Fprintf(&gcode, "G1 X%.6f Y0 Z0.2\n", radius)
		fmt.Fprintf(&gcode, "G3 X%.6f Y0 I%.6f J0 E1\n", radius, -radius)
	}
	_, err := extractToolpathPreview(context.Background(), writeToolpathFixture(t, gcode.String()))
	if err == nil || !strings.Contains(err.Error(), "total preview work limit") {
		t.Fatalf("expected total arc work limit, got %v", err)
	}
}

func TestToolpathPreviewMarksSegmentAndLayerTruncation(t *testing.T) {
	var gcode strings.Builder
	gcode.WriteString("G21\nG90\nM83\n;LAYER_CHANGE\n;Z:0.2\n;TYPE:Outer wall\n")
	for index := 0; index < maxToolpathSegments+2; index++ {
		if index == maxToolpathSegments {
			gcode.WriteString(";TYPE:Support\n")
		}
		fmt.Fprintf(&gcode, "G1 X%d Y0 E0.01\n", index+1)
	}
	for layer := 1; layer < maxToolpathLayers+1; layer++ {
		fmt.Fprintf(&gcode, ";LAYER_CHANGE\n;Z:%0.1f\n", float64(layer)+0.2)
	}
	preview, err := extractToolpathPreview(context.Background(), writeToolpathFixture(t, gcode.String()))
	if err != nil {
		t.Fatal(err)
	}
	if !preview.SourceComplete || !preview.Truncated || preview.ReturnedSegments != maxToolpathSegments || preview.OmittedSegments != 2 || preview.ReturnedLayers != maxToolpathLayers || preview.TotalLayers != maxToolpathLayers+1 {
		t.Fatalf("truncation was not explicit and bounded: %#v", preview)
	}
	if preview.SupportSegments != 2 || preview.Bounds == nil || preview.Bounds.Max[0] != maxToolpathSegments+2 {
		t.Fatalf("source-wide counters/bounds must include omitted geometry: %#v", preview)
	}
	for _, layer := range preview.Layers {
		for _, segment := range layer.Segments {
			if segment.Feature == "Support" {
				t.Fatal("omitted support unexpectedly appeared in retained prefix")
			}
		}
	}
}

func TestToolpathPreviewHandlesRealKlipperControlCommands(t *testing.T) {
	path := writeToolpathFixture(t, `PRINT_START EXTRUDER=220 BED=35
EXCLUDE_OBJECT_DEFINE NAME=model_id_0_copy_0 CENTER=1,1 POLYGON=[[0,0],[1,1]]
G90
M83
;LAYER_CHANGE
;Z:0.2
; printing object model id:0 copy 0
EXCLUDE_OBJECT_START NAME=model_id_0_copy_0
SET_VELOCITY_LIMIT ACCEL=500
;TYPE:Outer wall
G1 X0 Y0 Z0.2
G1 X1 Y0 E0.1
EXCLUDE_OBJECT_END NAME=model_id_0_copy_0
; stop printing object model id:0 copy 0
PRINT_END
`)
	preview, err := extractToolpathPreview(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if preview.TotalSegments != 1 || len(preview.Warnings) != 2 {
		t.Fatalf("expected explicit motion and macro warnings, got %#v", preview)
	}
}

func TestToolpathPreviewRejectsUnsupportedOrInvalidGeometry(t *testing.T) {
	for _, content := range []string{
		"G5 X1 E1\n",
		";V G1 X1 E1\n",
		"G1 XNaN E1\n",
		"G1 X1 X2 E1\n",
		"G2 X1 Y1 E1\n",
		"G2 X1 Y1 R0 E1\n",
		"G18\n",
		"M402\n",
		"UNKNOWN_MACRO X1\n",
	} {
		if _, err := extractToolpathPreview(context.Background(), writeToolpathFixture(t, content)); err == nil {
			t.Fatalf("expected invalid input %q to fail", content)
		}
	}
}

func TestToolpathPreviewRejectsOversizedSource(t *testing.T) {
	path := writeToolpathFixture(t, "G28\n")
	if err := os.Truncate(path, maxToolpathSourceBytes+1); err != nil {
		t.Fatal(err)
	}
	if _, err := extractToolpathPreview(context.Background(), path); err == nil {
		t.Fatal("expected source byte limit to fail closed")
	}
}
