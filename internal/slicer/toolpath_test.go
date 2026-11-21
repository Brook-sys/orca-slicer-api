package slicer

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
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

func TestToolpathPreviewThumbnailPayloadIsNotEmbeddedCommand(t *testing.T) {
	// VAAA is base64 data, not Orca's embedded ;V command syntax. Outer
	// metadata is not required, and multiple independent thumbnails may occur.
	block := "; thumbnail begin 160x160 4\n; VAAA\n; thumbnail end\n"
	path := writeToolpathFixture(t, "; THUMBNAIL_BLOCK_START\n;\n"+block+block+"; THUMBNAIL_BLOCK_END\nM83\nG1 X1 E1\n")
	preview, err := extractToolpathPreview(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if !preview.SourceComplete || preview.TotalSegments != 1 || preview.Bounds.Max[0] != 1 {
		t.Fatalf("thumbnail affected motion: %#v", preview)
	}
	for _, suffix := range []string{";V1\n", "; V motion\n"} {
		_, err := extractToolpathPreview(context.Background(), writeToolpathFixture(t, block+suffix))
		if err == nil || !strings.Contains(err.Error(), "embedded comment commands") {
			t.Fatalf("embedded command outside thumbnail accepted: %q, %v", suffix, err)
		}
	}
}

func TestToolpathPreviewThumbnailBoundariesFailClosed(t *testing.T) {
	begin := "; thumbnail begin 160x160 4\n"
	for name, content := range map[string]string{
		"empty":                 begin + "; thumbnail end\n",
		"unclosed":              begin + "; VAAA\n",
		"unclosed incomplete":   begin,
		"nested":                begin + begin,
		"stray end":             "; thumbnail end\n",
		"bad dimensions":        "; thumbnail begin 160 4\n; VAAA\n; thumbnail end\n",
		"zero dimensions":       "; thumbnail begin 0x160 4\n",
		"missing length":        "; thumbnail begin 160x160\n",
		"zero length":           "; thumbnail begin 160x160 0\n",
		"huge length":           "; thumbnail begin 160x160 99999999999999999999\n",
		"invalid length":        "; thumbnail begin 160x160 3\n",
		"empty payload comment": begin + ";\n; VAAA\n; thumbnail end\n",
		"blank payload line":    begin + "\n; VAAA\n; thumbnail end\n",
		"short payload":         begin + "; VA\n; thumbnail end\n",
		"long payload":          begin + "; VAAAAAAA\n; thumbnail end\n",
		"invalid alphabet":      begin + "; V@AA\n; thumbnail end\n",
		"invalid padding":       begin + "; =AAA\n; thumbnail end\n",
		"data after padding":    begin + "; VA=A\n; thumbnail end\n",
		"motion":                begin + "G1 X10 E1\n; VAAA\n; thumbnail end\n",
		"inline begin":          "G1 X1 E1 ; thumbnail begin 160x160 4\n;VAAA\n",
		"motion end":            begin + "; VAAA\nG1 X10 E1 ; thumbnail end\n",
		"comment motion":        begin + "; G1 X10 E1\n; thumbnail end\n",
		"outer marker only":     "; THUMBNAIL_BLOCK_START\n;VAAA\n; THUMBNAIL_BLOCK_END\n",
		"premature outer end":   begin + "; THUMBNAIL_BLOCK_END\n",
	} {
		t.Run(name, func(t *testing.T) {
			if preview, err := extractToolpathPreview(context.Background(), writeToolpathFixture(t, content)); err == nil {
				t.Fatalf("malformed thumbnail accepted: %#v", preview)
			}
		})
	}
	for _, payload := range []string{"VAAA", "VAA=", "VA==", "VA\n; AA"} {
		if _, err := extractToolpathPreview(context.Background(), writeToolpathFixture(t, begin+"; "+payload+"\n; thumbnail end\n")); err != nil {
			t.Fatalf("valid base64 payload %q rejected: %v", payload, err)
		}
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

func TestToolpathArcP1HelicalGeometryAndState(t *testing.T) {
	for _, command := range []string{"G2", "G02", "G3", "G03"} {
		for _, fullCircle := range []bool{false, true} {
			for _, extruding := range []bool{false, true} {
				name := fmt.Sprintf("%s/full=%t/extruding=%t", command, fullCircle, extruding)
				t.Run(name, func(t *testing.T) {
					p := toolpathParser{ctx: context.Background(), unitsScale: 1, currentLayer: -1, retainedLayer: -1, fallbackZ: math.NaN()}
					for _, line := range []string{"G90", "M82", ";LAYER_CHANGE", "G1 X1 Y0 Z0.2", "G92 E5"} {
						if err := p.parseLine(line); err != nil {
							t.Fatal(err)
						}
					}
					clockwise := command == "G2" || command == "G02"
					end := [3]float64{0, 1, 1.2}
					xy := "X0 Y1 "
					if clockwise {
						end[1], xy = -1, "X0 Y-1 "
					}
					if fullCircle {
						end[0], end[1], xy = 1, 0, "" // Orca omits XY on a spiral lift.
					}
					eWord := ""
					if extruding {
						eWord = " E7"
					}
					if err := p.parseLine(command + " " + xy + "Z1.2 I-1 J0 P1 F600" + eWord); err != nil {
						t.Fatal(err)
					}
					wantE := 0.0
					if extruding {
						wantE = 2
					}
					if p.position != end || p.extruderPosition != wantE || p.feedrate != 10 {
						t.Fatalf("incorrect arc endpoint/state: position=%v E=%v F=%v", p.position, p.extruderPosition, p.feedrate)
					}
					segments := p.preview.Layers[0].Segments
					if !extruding && len(segments) != 0 {
						t.Fatal("travel arc must not create extrusion geometry")
					}
					if extruding {
						if len(segments) < 2 || segments[0].From != [3]float64{1, 0, 0.2} || segments[len(segments)-1].To != end {
							t.Fatalf("incorrect tessellation endpoints: %v", segments)
						}
						var sweep, sumE float64
						for i, segment := range segments {
							if i > 0 && segment.From != segments[i-1].To {
								t.Fatal("discontinuous arc")
							}
							wantZ := 0.2 + float64(i+1)/float64(len(segments))
							if math.Abs(segment.To[2]-wantZ) > 1e-12 || math.Abs(math.Hypot(segment.To[0], segment.To[1])-1) > 1e-12 {
								t.Fatalf("incorrect helix interpolation: %v", segment)
							}
							a, b := segment.From, segment.To
							sweep += math.Atan2(a[0]*b[1]-a[1]*b[0], a[0]*b[0]+a[1]*b[1])
							sumE += segment.Extrusion
						}
						wantSweep := math.Pi / 2
						if fullCircle {
							wantSweep = 2 * math.Pi
						}
						if clockwise {
							wantSweep = -wantSweep
						}
						if math.Abs(sweep-wantSweep) > 1e-12 || math.Abs(sumE-2) > 1e-12 {
							t.Fatalf("P1 must not add a turn: sweep=%v extrusion=%v", sweep, sumE)
						}
					}
					// The next extrusion starts at the helix endpoint, also after travel,
					// and uses the unchanged absolute extrusion coordinate origin.
					nextE := 6.0 + wantE
					if err := p.parseLine(fmt.Sprintf("G1 X2 E%g", nextE)); err != nil {
						t.Fatal(err)
					}
					segments = p.preview.Layers[0].Segments
					last := segments[len(segments)-1]
					if last.From != end || last.To != [3]float64{2, end[1], end[2]} || last.Extrusion != 1 {
						t.Fatalf("incorrect continuation after arc: %v", last)
					}
				})
			}
		}
	}
}

func TestToolpathArcPStrictRejections(t *testing.T) {
	for _, command := range []string{"G2", "G02", "G3", "G03"} {
		for _, words := range []string{"P0", "P-1", "P0.5", "P1.5", "P2", "PNaN", "PInf", "P1 P1", "P1 P2", "P1 Q1", "P1 K1", "P1 G1", "P1 X1 X2"} {
			t.Run(command+"/"+words, func(t *testing.T) {
				_, err := extractToolpathPreview(context.Background(), writeToolpathFixture(t, "G1 X1\n"+command+" I-1 J0 "+words+"\n"))
				if err == nil {
					t.Fatalf("must reject ambiguous/invalid arc words: %s", words)
				}
			})
		}
	}
	for _, command := range []string{"G0", "G00", "G1", "G01", "G92", "G17", "G20", "G21", "G90", "G91", "M82", "M83", "T0", "G28", "G999", "UNKNOWN"} {
		t.Run(command+"/P1", func(t *testing.T) {
			_, err := extractToolpathPreview(context.Background(), writeToolpathFixture(t, command+" P1\n"))
			if err == nil {
				t.Fatalf("P compatibility must be limited to G2/G3, accepted %s", command)
			}
		})
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

func TestToolpathPreviewKlipperExtrusionConfiguration(t *testing.T) {
	commands := []string{
		"SET_RETRACTION RETRACT_LENGTH=0.3",
		"SET_PRESSURE_ADVANCE ADVANCE=0.035 SMOOTH_TIME=0.04",
	}
	preamble := "PRINT_START EXTRUDER=210 BED=60\nG21\nG90\nM83\nSET_VELOCITY_LIMIT ACCEL=500\n"
	motion := ";LAYER_CHANGE\n;Z:0.2\n;TYPE:Outer wall\nG1 X0 Y0 Z0.2 F1200\nG1 X10 Y0 E0.5\n"
	for _, command := range append(append([]string{}, commands...), strings.Join(commands, "\n")) {
		t.Run(command, func(t *testing.T) {
			baseline, err := extractToolpathPreview(context.Background(), writeToolpathFixture(t, preamble+motion))
			if err != nil {
				t.Fatal(err)
			}
			preview, err := extractToolpathPreview(context.Background(), writeToolpathFixture(t, preamble+command+"\n"+motion+command+"\n"))
			if err != nil {
				t.Fatal(err)
			}
			wantWarnings := append([]string{}, baseline.Warnings...)
			for _, line := range strings.Split(command, "\n") {
				name := strings.Fields(line)[0]
				wantWarnings = append(wantWarnings, name+" firmware effects are not simulated; only explicit G-code motion is represented")
			}
			if !reflect.DeepEqual(preview.Warnings, wantWarnings) {
				t.Fatalf("warnings must be explicit and deduplicated per command: got %q want %q", preview.Warnings, wantWarnings)
			}
			if !preview.SourceComplete || preview.TotalSegments != 1 || preview.SourceLines != baseline.SourceLines+2*len(strings.Split(command, "\n")) {
				t.Fatalf("unexpected completeness/counts: %#v", preview)
			}
			// Only source accounting and warnings should change, not geometry,
			// bounds, layers, feedrates, extrusion, attribution, or truncation.
			preview.SourceBytes = baseline.SourceBytes
			preview.SourceLines = baseline.SourceLines
			preview.Warnings = baseline.Warnings
			if !reflect.DeepEqual(preview, baseline) {
				t.Fatalf("configuration changed preview: got %#v want %#v", preview, baseline)
			}
		})
	}
}

func TestToolpathKlipperExtrusionConfigurationPreservesParserState(t *testing.T) {
	for _, command := range []string{"SET_RETRACTION RETRACT_LENGTH=0.3", "SET_PRESSURE_ADVANCE ADVANCE=0.04"} {
		t.Run(command, func(t *testing.T) {
			p := toolpathParser{ctx: context.Background(), unitsScale: 1, currentLayer: -1, retainedLayer: -1, fallbackZ: math.NaN()}
			// Exercise non-default units, origins, relative modes, tool and
			// attribution, including wiping and pending layer metadata.
			lines := []string{
				"G20", "G90", "M82", "T2", "G1 X1 Y2 Z0.1 E3 F60", "G92 X0 Y0 Z0 E0",
				"G91", "M83", ";TYPE:Support", "; printing object synthetic id:0 copy 0", ";WIPE_START", ";LAYER_CHANGE",
				"G1 X0.1 Y0 E0.01", "G90", "M82", "G1 X0.2 Y0 E0.02", ";WIPE_END",
			}
			for _, line := range lines {
				before := p
				fallbackZ := p.fallbackZ
				// fallbackZ begins as NaN, which reflect.DeepEqual cannot compare.
				before.fallbackZ = 0
				if err := p.parseLine(command); err != nil {
					t.Fatal(err)
				}
				after := p
				after.fallbackZ = 0
				after.preview.Warnings = before.preview.Warnings
				if !reflect.DeepEqual(after, before) || (p.fallbackZ != fallbackZ && !(math.IsNaN(p.fallbackZ) && math.IsNaN(fallbackZ))) {
					t.Fatalf("configuration changed parser state before %q", line)
				}
				if err := p.parseLine(line); err != nil {
					t.Fatalf("%s: %v", line, err)
				}
			}
			if len(p.preview.Warnings) != 1 || p.preview.TotalSegments != 3 {
				t.Fatalf("unexpected warnings or geometry: %#v", p.preview)
			}
		})
	}
}

func TestToolpathPreviewKlipperConfigurationLineSyntax(t *testing.T) {
	for _, command := range []string{"SET_RETRACTION RETRACT_LENGTH=0.3", "SET_PRESSURE_ADVANCE ADVANCE=0.04"} {
		t.Run(command, func(t *testing.T) {
			content := "; synthetic preamble\r\nN10 " + command + " *123 ; config\r\n\tN20 " + strings.ToLower(command) + " ; repeated\r\nG90\nM83\nG1 X1 E0.1\n"
			preview, err := extractToolpathPreview(context.Background(), writeToolpathFixture(t, content))
			if err != nil {
				t.Fatal(err)
			}
			if preview.SourceLines != 6 || preview.TotalSegments != 1 || len(preview.Warnings) != 1 {
				t.Fatalf("line numbers/comments changed configuration handling: %#v", preview)
			}
		})
	}
}

func TestToolpathPreviewKlipperConfigurationArgumentsAreNotValidated(t *testing.T) {
	// This preview is not a firmware validator. Match the existing exact-name
	// SET_VELOCITY_LIMIT skip, including absent and uninterpreted arguments.
	for _, command := range []string{"SET_RETRACTION", "SET_PRESSURE_ADVANCE"} {
		for _, args := range []string{"", " VALUE=firmware_specific", " ADVANCE=NaN RETRACT_LENGTH=bad", " X999 E999 F999"} {
			t.Run(command+args, func(t *testing.T) {
				preview, err := extractToolpathPreview(context.Background(), writeToolpathFixture(t, command+args+"\n"))
				if err != nil {
					t.Fatal(err)
				}
				if preview.TotalSegments != 0 || preview.Bounds != nil || len(preview.Warnings) != 1 {
					t.Fatalf("configuration arguments became motion: %#v", preview)
				}
			})
		}
	}
}

func TestToolpathPreviewKlipperConfigurationDoesNotRelaxStrictErrors(t *testing.T) {
	for _, invalid := range []struct{ line, message string }{
		{"UNKNOWN_MACRO", "unsupported command UNKNOWN_MACRO"},
		{"SET_RETRACTION_CUSTOM", "unsupported command SET_RETRACTION_CUSTOM"},
		{"SET_PRESSURE_ADVANCE_CUSTOM", "unsupported command SET_PRESSURE_ADVANCE_CUSTOM"},
		{"SET_GCODE_OFFSET", "unsupported command SET_GCODE_OFFSET"},
		{"G10", "unsupported G-code command G10"},
		{"G11", "unsupported G-code command G11"},
		{"G5 X1 E1", "unsupported G-code command G5"},
		{"M402", "unsupported M-code command M402"},
		{"G1 Xbad E1", "invalid numeric word"},
		{"G1 XNaN E1", "invalid numeric word"},
		{"G1 X1 X2 E1", "duplicate X word"},
		{"G1 X1 ADVANCE=0.1", "unsupported command word"},
		{"G2 X1 Y1 E1", "arc requires R or I/J"},
	} {
		t.Run(invalid.line, func(t *testing.T) {
			content := "SET_RETRACTION RETRACT_LENGTH=0.3\nSET_PRESSURE_ADVANCE ADVANCE=0.04\n; synthetic comment\nN40 " + invalid.line + " ; still invalid\n"
			_, err := extractToolpathPreview(context.Background(), writeToolpathFixture(t, content))
			if err == nil || !strings.Contains(err.Error(), "line 4: "+invalid.message) {
				t.Fatalf("expected strict line-4 error %q, got %v", invalid.message, err)
			}
		})
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
