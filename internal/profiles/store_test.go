package profiles

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStoreListProfileInfo(t *testing.T) {
	store := NewStore(t.TempDir())
	if _, err := store.Save("presets", "standard", []byte(`{"layer_height":"0.20"}`)); err != nil {
		t.Fatal(err)
	}

	items, err := store.List("presets")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Name != "standard" || items[0].Checksum == "" || items[0].Size == 0 || items[0].UpdatedAt == "" {
		t.Fatalf("invalid profile info: %+v", items[0])
	}
}

func TestImportURLAndUpdateFromSource(t *testing.T) {
	content := `{"layer_height":"0.20"}`
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, content)
	}))
	defer server.Close()

	store := NewStore(t.TempDir())
	store.Client = server.Client()

	res, err := store.ImportURL(context.Background(), "presets", ImportRequest{Name: "standard", URL: server.URL, Overwrite: true})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Updated || res.SourceURL != server.URL {
		t.Fatalf("invalid import response: %+v", res)
	}

	unchanged, err := store.UpdateFromSource(context.Background(), "presets", "standard")
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Updated {
		t.Fatalf("expected unchanged update")
	}

	content = `{"layer_height":"0.16"}`
	updated, err := store.UpdateFromSource(context.Background(), "presets", "standard")
	if err != nil {
		t.Fatal(err)
	}
	if !updated.Updated {
		t.Fatalf("expected updated response")
	}
}

func TestImportURLWithoutOverwriteConflicts(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, `{"layer_height":"0.20"}`)
	}))
	defer server.Close()

	store := NewStore(t.TempDir())
	store.Client = server.Client()
	if _, err := store.Save("presets", "standard", []byte(`{"layer_height":"0.20"}`)); err != nil {
		t.Fatal(err)
	}

	_, err := store.ImportURL(context.Background(), "presets", ImportRequest{Name: "standard", URL: server.URL})
	if err == nil {
		t.Fatalf("expected conflict")
	}
}
