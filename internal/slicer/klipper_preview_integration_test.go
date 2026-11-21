package slicer

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// Exercise HTTP -> Slice -> generated artifact -> preview parser, not only parseLine.
// The executable is synthetic: this does not validate Orca or firmware behavior.
func TestHTTPPreviewKlipperConfiguration(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell fixture")
	}
	for _, tc := range []struct {
		name, commands string
		status         int
	}{
		{"retraction", "SET_RETRACTION RETRACT_LENGTH=0.3 RETRACT_SPEED=40 UNRETRACT_EXTRA_LENGTH=0 UNRETRACT_SPEED=30", 200},
		{"pressure", "SET_PRESSURE_ADVANCE ADVANCE=0.02725 SMOOTH_TIME=0.02", 200},
		{"both", "SET_RETRACTION RETRACT_LENGTH=0.3\nSET_PRESSURE_ADVANCE ADVANCE=0.02725", 200},
		{"unknown_macro", "MOVE_CUSTOM X10", 500},
		{"unknown_coordinate_transform", "SET_GCODE_OFFSET X=10", 500},
		{"malformed_move", "G1 Xbroken", 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			t.Setenv("TMPDIR", dir)
			executable := filepath.Join(dir, "slicer.sh")
			script := `#!/bin/sh
out=""
while [ "$#" -gt 0 ]; do
 if [ "$1" = "--outputdir" ]; then shift; out="$1"; fi
 shift
done
mkdir -p "$out"
cat > "$out/plate_1.gcode" <<'GCODE'
G21
G90
M83
` + tc.commands + `
;LAYER_CHANGE
;Z:0.2
;TYPE:Support
G1 X0 Y0 Z0.2 F600
G1 X1 Y0 E0.1
GCODE
printf '%s' '{"return_code":0,"plate_index":1,"sliced_plates":[{"id":1}]}' > "$out/result.json"
printf '%s' '{"version":1,"generator":"OrcaSlicer 2.5.0","objects":[{"name":"model.stl_1","instance":0}]}' > "$out/plate_1.statistics.json"
`
			if err := os.WriteFile(executable, []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			service := &Service{DataPath: dir, OrcaSlicerPath: executable, Timeout: 5 * time.Second, State: NewStateStore(dir)}
			handler := Handler{Service: service}
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			part, err := writer.CreateFormFile("file", "model.stl")
			if err != nil {
				t.Fatal(err)
			}
			if _, err = part.Write(testASCIISTL()); err != nil {
				t.Fatal(err)
			}
			if err = writer.Close(); err != nil {
				t.Fatal(err)
			}
			req := httptest.NewRequest(http.MethodPost, "/slice/preview", &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			res := httptest.NewRecorder()
			handler.Preview(res, req)
			if res.Code != tc.status {
				t.Fatalf("status %d want %d: %s", res.Code, tc.status, res.Body.String())
			}
			if tc.status == 200 {
				var result struct {
					Quantity        int             `json:"quantity"`
					UsesSupport     bool            `json:"usesSupport"`
					ThumbnailBase64 string          `json:"thumbnail"`
					Toolpaths       ToolpathPreview `json:"toolpaths"`
				}
				if err := json.Unmarshal(res.Body.Bytes(), &result); err != nil {
					t.Fatal(err)
				}
				if result.Quantity != 1 || !result.UsesSupport || result.ThumbnailBase64 == "" || result.Toolpaths.TotalSegments != 1 || !result.Toolpaths.SourceComplete {
					t.Fatalf("unexpected preview: %#v", result)
				}
				if !strings.Contains(strings.Join(result.Toolpaths.Warnings, " "), "not") {
					t.Fatal("expected firmware simulation warning")
				}
			}
			matches, err := filepath.Glob(filepath.Join(dir, "slice-*"))
			if err != nil {
				t.Fatal(err)
			}
			for _, match := range matches {
				info, err := os.Stat(match)
				if err != nil {
					t.Fatal(err)
				}
				if info.IsDir() {
					t.Fatalf("preview left slice directory: %s", match)
				}
			}
		})
	}
}
