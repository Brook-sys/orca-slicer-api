package slicer

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestParseSettingsUsesContainerDefaults(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/slice", nil)
	settings, err := parseSettings(req, true, true, true)
	if err != nil {
		t.Fatal(err)
	}
	if !settings.GenerateImage || !settings.ResolveProfiles || !settings.SanitizeProfiles {
		t.Fatalf("expected container defaults to be applied")
	}
	if settings.Quantity != 1 || settings.Plate != "1" || settings.ExportType != "gcode" {
		t.Fatalf("unexpected strict defaults: %#v", settings)
	}
}

func TestParseSettingsQuantityAndTransforms(t *testing.T) {
	form := url.Values{
		"quantity":       {"3"},
		"arrange":        {"true"},
		"rotate":         {"30"},
		"rotateX":        {"0"},
		"rotateY":        {"-12.5"},
		"arrangeSpacing": {"0"},
	}
	req := httptest.NewRequest(http.MethodPost, "/slice", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	settings, err := parseSettings(req, false, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if settings.Quantity != 3 || settings.Rotate == nil || *settings.Rotate != 30 || settings.RotateX == nil || *settings.RotateX != 0 || settings.RotateY == nil || *settings.RotateY != -12.5 {
		t.Fatalf("quantity/rotations not preserved: %#v", settings)
	}
}

func TestParseSettingsRejectsInvalidStrictFields(t *testing.T) {
	tests := []struct {
		name string
		form url.Values
	}{
		{name: "quantity zero", form: url.Values{"quantity": {"0"}}},
		{name: "quantity negative", form: url.Values{"quantity": {"-1"}}},
		{name: "quantity fractional", form: url.Values{"quantity": {"1.5"}}},
		{name: "quantity above Orca limit", form: url.Values{"quantity": {"513"}, "arrange": {"true"}}},
		{name: "multiple without arrangement", form: url.Values{"quantity": {"2"}}},
		{name: "3MF export", form: url.Values{"exportType": {"3mf"}}},
		{name: "additional plate", form: url.Values{"plate": {"2"}}},
		{name: "automatic orient replaces rotation", form: url.Values{"orient": {"true"}, "rotate": {"0"}}},
		{name: "non finite rotation", form: url.Values{"rotateX": {"NaN"}}},
		{name: "unsupported spacing", form: url.Values{"arrangeSpacing": {"6"}}},
		{name: "negative spacing", form: url.Values{"arrangeSpacing": {"-1"}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/slice", strings.NewReader(test.form.Encode()))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
			if _, err := parseSettings(req, false, false, false); err == nil {
				t.Fatal("expected request to be rejected")
			}
		})
	}
}

func TestValidModelFile(t *testing.T) {
	for _, name := range []string{"model.stl", "MODEL.STL"} {
		if !validModelFile(name) {
			t.Fatalf("expected %s to be valid", name)
		}
	}
	for _, name := range []string{"model.step", "model.stp", "model.3mf", "model.obj"} {
		if validModelFile(name) {
			t.Fatalf("expected %s to be invalid", name)
		}
	}
}

func TestModelsFromMultipartRequiresExactlyOneSTL(t *testing.T) {
	for _, test := range []struct {
		name   string
		fields []string
	}{
		{name: "two file parts", fields: []string{"file", "file"}},
		{name: "file and files are not prioritized", fields: []string{"file", "files"}},
		{name: "repeated files", fields: []string{"files", "files"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var body bytes.Buffer
			writer := multipart.NewWriter(&body)
			for _, field := range test.fields {
				part, err := writer.CreateFormFile(field, "model.stl")
				if err != nil {
					t.Fatal(err)
				}
				_, _ = part.Write([]byte("solid test\nendsolid test\n"))
			}
			_ = writer.Close()
			req := httptest.NewRequest(http.MethodPost, "/slice", &body)
			req.Header.Set("Content-Type", writer.FormDataContentType())
			if err := req.ParseMultipartForm(maxModelSize); err != nil {
				t.Fatal(err)
			}
			if _, err := modelsFromMultipart(req); err == nil {
				t.Fatal("expected multiple inputs to be rejected")
			}
		})
	}
}
