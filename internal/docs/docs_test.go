package docs

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestOpenAPIHandler(t *testing.T) {
	parsed := serveOpenAPI(t)
	if parsed["openapi"] == "" {
		t.Fatalf("expected openapi version")
	}
}

func TestOpenAPIStrictSlicingContract(t *testing.T) {
	document := serveOpenAPI(t)
	for _, path := range []string{"/slice", "/slice/preview", "/slice/async"} {
		if got := openAPIString(t, document, "paths", path, "post", "requestBody", "content", "multipart/form-data", "schema", "$ref"); got != "#/components/schemas/SliceMultipartRequest" {
			t.Errorf("%s request schema = %q", path, got)
		}
	}
	if got := openAPIString(t, document, "paths", "/slice", "post", "responses", "200", "$ref"); got != "#/components/responses/GCodeSliceResult" {
		t.Errorf("slice success response = %q", got)
	}
	for _, path := range []string{"/slice", "/slice/preview"} {
		if got := openAPIString(t, document, "paths", path, "post", "responses", "422", "$ref"); got != "#/components/responses/UnprocessableSlice" {
			t.Errorf("%s overflow response = %q", path, got)
		}
	}

	errorProperties := openAPIObject(t, document, "components", "schemas", "ErrorResponse", "properties")
	if _, found := errorProperties["message"]; !found {
		t.Error("ErrorResponse must expose message")
	}
	if _, found := errorProperties["error"]; found {
		t.Error("ErrorResponse must not advertise the obsolete error field")
	}

	requestProperties := openAPIObject(t, document, "components", "schemas", "SliceMultipartRequest", "properties")
	quantity := openAPIObject(t, requestProperties, "quantity")
	if quantity["minimum"] != float64(1) || quantity["maximum"] != float64(512) {
		t.Errorf("quantity bounds = %v..%v", quantity["minimum"], quantity["maximum"])
	}
	files := openAPIObject(t, requestProperties, "files")
	if files["maxItems"] != float64(1) {
		t.Errorf("files maxItems = %v, want 1", files["maxItems"])
	}
	assertSingleEnumValue(t, openAPIObject(t, requestProperties, "plate"), "1")
	assertSingleEnumValue(t, openAPIObject(t, requestProperties, "exportType"), "gcode")

	previewProperties := openAPIObject(t, document, "components", "schemas", "PreviewResponse", "properties")
	if got := openAPIString(t, previewProperties, "toolpaths", "$ref"); got != "#/components/schemas/ToolpathPreview" {
		t.Errorf("preview toolpaths schema = %q", got)
	}
	limitProperties := openAPIObject(t, document, "components", "schemas", "ToolpathLimits", "properties")
	for field, expected := range map[string]float64{
		"sourceBytes":         32 << 20,
		"lineBytes":           1 << 20,
		"lines":               4_000_000,
		"arcSegmentsPerMove":  4_096,
		"arcSegmentsTotal":    1_000_000,
		"arcChordToleranceMm": 0.05,
	} {
		assertSingleEnumValue(t, openAPIObject(t, limitProperties, field), expected)
	}
	assertEnumValues(t, openAPIObject(t, limitProperties, "layers"), float64(512), float64(4096))
	assertEnumValues(t, openAPIObject(t, limitProperties, "segments"), float64(20_000), float64(250_000))

	assertEnumValues(t, openAPIObject(t, requestProperties, "toolpathMode"), "legacy", "full")
	toolpathProperties := openAPIObject(t, document, "components", "schemas", "ToolpathPreview", "properties")
	assertEnumValues(t, openAPIObject(t, toolpathProperties, "version"), float64(1), float64(2))
	assertEnumValues(t, openAPIObject(t, toolpathProperties, "mode"), "full")
	assertEnumValues(t, openAPIObject(t, toolpathProperties, "geometryComplete"), true)
	segmentProperties := openAPIObject(t, document, "components", "schemas", "ToolpathSegment", "properties")
	for _, field := range []string{"widthMm", "heightMm"} {
		if nullable, ok := openAPIObject(t, segmentProperties, field)["nullable"].(bool); !ok || !nullable {
			t.Errorf("%s must be nullable for unknown Orca metadata", field)
		}
	}

	headers := openAPIObject(t, document, "components", "responses", "GCodeSliceResult", "headers")
	for _, header := range []string{"X-Slice-Quantity", "X-Slice-Plate", "X-Print-Time-Seconds", "X-Filament-Used-g", "X-Filament-Used-mm"} {
		if _, found := headers[header]; !found {
			t.Errorf("slice response is missing header %s", header)
		}
	}
}

func TestOpenAPIReferencesResolve(t *testing.T) {
	document := serveOpenAPI(t)
	var walk func(any, string)
	walk = func(value any, location string) {
		switch typed := value.(type) {
		case map[string]any:
			if reference, ok := typed["$ref"].(string); ok {
				if !strings.HasPrefix(reference, "#/") {
					t.Errorf("%s has non-local reference %q", location, reference)
				} else {
					resolved := any(document)
					for _, part := range strings.Split(strings.TrimPrefix(reference, "#/"), "/") {
						object, ok := resolved.(map[string]any)
						if !ok {
							t.Errorf("%s reference %q does not resolve at %q", location, reference, part)
							break
						}
						resolved, ok = object[part]
						if !ok {
							t.Errorf("%s reference %q does not resolve at %q", location, reference, part)
							break
						}
					}
				}
			}
			for key, child := range typed {
				walk(child, location+"/"+key)
			}
		case []any:
			for index, child := range typed {
				walk(child, location+"/"+strconv.Itoa(index))
			}
		}
	}
	walk(document, "")
}

func serveOpenAPI(t *testing.T) map[string]any {
	t.Helper()
	res := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/openapi.json", nil)
	OpenAPIHandler(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", res.Code)
	}
	var parsed map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &parsed); err != nil {
		t.Fatalf("expected valid json: %v", err)
	}
	return parsed
}

func openAPIAt(t *testing.T, value any, path ...string) any {
	t.Helper()
	current := value
	for _, part := range path {
		object, ok := current.(map[string]any)
		if !ok {
			t.Fatalf("OpenAPI path %v does not address an object", path)
		}
		current, ok = object[part]
		if !ok {
			t.Fatalf("OpenAPI path %v is missing %q", path, part)
		}
	}
	return current
}

func openAPIObject(t *testing.T, value any, path ...string) map[string]any {
	t.Helper()
	object, ok := openAPIAt(t, value, path...).(map[string]any)
	if !ok {
		t.Fatalf("OpenAPI path %v is not an object", path)
	}
	return object
}

func openAPIString(t *testing.T, value any, path ...string) string {
	t.Helper()
	text, ok := openAPIAt(t, value, path...).(string)
	if !ok {
		t.Fatalf("OpenAPI path %v is not a string", path)
	}
	return text
}

func assertSingleEnumValue(t *testing.T, schema map[string]any, expected any) {
	t.Helper()
	assertEnumValues(t, schema, expected)
}

func assertEnumValues(t *testing.T, schema map[string]any, expected ...any) {
	t.Helper()
	values, ok := schema["enum"].([]any)
	if !ok || len(values) != len(expected) {
		t.Errorf("enum = %#v, want %#v", schema["enum"], expected)
		return
	}
	for index := range expected {
		if values[index] != expected[index] {
			t.Errorf("enum = %#v, want %#v", values, expected)
			return
		}
	}
}
