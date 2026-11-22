package slicer

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

type toolpathCountingWriter struct {
	bytes int64
}

func (w *toolpathCountingWriter) Write(data []byte) (int, error) {
	w.bytes += int64(len(data))
	return len(data), nil
}

// Run explicitly with -bench BenchmarkFullToolpathPreview250KSegments
// -benchtime=1x -benchmem. This is intentionally not a normal unit test: it
// allocates a production-scale result to measure memory and JSON response costs.
func BenchmarkFullToolpathPreview250KSegments(b *testing.B) {
	benchmarkFullToolpathPreviewSegments(b, maxFullToolpathSegments)
}

func benchmarkFullToolpathPreviewSegments(b *testing.B, segmentCount int) {
	var gcode strings.Builder
	gcode.Grow(17*segmentCount + 64)
	gcode.WriteString("G91\n;HEIGHT:0.2\n;TYPE:Bridge\n;WIDTH:0.42\n")
	for index := 0; index < segmentCount; index++ {
		gcode.WriteString("G1 X0.001 E0.001\n")
	}
	path := writeToolpathFixture(b, gcode.String())
	b.ReportAllocs()
	b.ResetTimer()
	preview, err := extractToolpathPreviewMode(context.Background(), path, toolpathModeFull)
	if err != nil {
		b.Fatal(err)
	}
	if !preview.GeometryComplete || preview.ReturnedSegments != segmentCount || preview.Truncated {
		b.Fatalf("incomplete %d-segment preview: %#v", segmentCount, preview)
	}
	writer := &toolpathCountingWriter{}
	if err := json.NewEncoder(writer).Encode(preview); err != nil {
		b.Fatal(err)
	}
	b.ReportMetric(float64(writer.bytes)/1_000_000, "response-MB")
	b.StopTimer()
	b.Logf("synthetic response: %d bytes; source: %d bytes", writer.bytes, preview.SourceBytes)
}
