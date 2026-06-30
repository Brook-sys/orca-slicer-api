package profiles

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHandlerUploadListGetDelete(t *testing.T) {
	store := NewStore(t.TempDir())
	handler := Handler{Store: store}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("name", "standard")
	part, err := writer.CreateFormFile("file", "preset.json")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = part.Write([]byte(`{"layer_height":"0.20"}`))
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/profiles/presets/upload", body)
	req.SetPathValue("category", "presets")
	req.Header.Set("Content-Type", writer.FormDataContentType())
	res := httptest.NewRecorder()
	handler.Upload(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", res.Code, res.Body.String())
	}

	listReq := httptest.NewRequest(http.MethodGet, "/profiles/presets", nil)
	listReq.SetPathValue("category", "presets")
	listRes := httptest.NewRecorder()
	handler.List(listRes, listReq)
	if listRes.Code != http.StatusOK {
		t.Fatalf("expected 200 list, got %d", listRes.Code)
	}
	var items []ProfileInfo
	if err := json.Unmarshal(listRes.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Name != "standard" {
		t.Fatalf("unexpected list: %+v", items)
	}

	getReq := httptest.NewRequest(http.MethodGet, "/profiles/presets/standard", nil)
	getReq.SetPathValue("category", "presets")
	getReq.SetPathValue("name", "standard")
	getRes := httptest.NewRecorder()
	handler.Get(getRes, getReq)
	if getRes.Code != http.StatusOK {
		t.Fatalf("expected 200 get, got %d", getRes.Code)
	}

	deleteReq := httptest.NewRequest(http.MethodDelete, "/profiles/presets/standard", nil)
	deleteReq.SetPathValue("category", "presets")
	deleteReq.SetPathValue("name", "standard")
	deleteRes := httptest.NewRecorder()
	handler.Delete(deleteRes, deleteReq)
	if deleteRes.Code != http.StatusNoContent {
		t.Fatalf("expected 204 delete, got %d", deleteRes.Code)
	}
}
