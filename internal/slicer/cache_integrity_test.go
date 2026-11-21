package slicer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestCacheKeyIncludesEverySettingsField(t *testing.T) {
	c := NewResultCache(time.Hour, 10)
	models := []ModelInput{{Filename: "a.stl", Data: []byte("abc")}}
	base := c.makeModelsKey(models, Settings{})
	typ := reflect.TypeOf(Settings{})
	for i := 0; i < typ.NumField(); i++ {
		t.Run(typ.Field(i).Name, func(t *testing.T) {
			var settings Settings
			field := reflect.ValueOf(&settings).Elem().Field(i)
			switch field.Kind() {
			case reflect.String:
				field.SetString("changed")
			case reflect.Bool:
				field.SetBool(true)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				field.SetInt(1)
			case reflect.Float32, reflect.Float64:
				field.SetFloat(1)
			case reflect.Pointer:
				value := reflect.New(field.Type().Elem())
				switch value.Elem().Kind() {
				case reflect.Bool:
					value.Elem().SetBool(true)
				case reflect.Float32, reflect.Float64:
					value.Elem().SetFloat(1)
				default:
					t.Fatalf("uncovered pointer type %v", field.Type())
				}
				field.Set(value)
			case reflect.Map:
				if field.Type() == reflect.TypeOf(map[string]any{}) {
					field.Set(reflect.ValueOf(map[string]any{"x": 1}))
				} else {
					field.Set(reflect.ValueOf(map[string]map[string]any{"preset": {"x": 1}}))
				}
			case reflect.Slice:
				field.Set(reflect.ValueOf([]map[string]any{{"x": 1}}))
			default:
				t.Fatalf("uncovered settings type %v", field.Type())
			}
			if c.makeModelsKey(models, settings) == base {
				t.Fatal("field omitted from key")
			}
		})
	}
	if c.makeModelsKey([]ModelInput{{Filename: "bc", Data: []byte("a")}}, Settings{}) == c.makeModelsKey([]ModelInput{{Filename: "c", Data: []byte("ab")}}, Settings{}) {
		t.Fatal("ambiguous model framing")
	}
	if c.makeModelsKey(models, Settings{EnableSupport: boolPointer(false)}) == base {
		t.Fatal("explicit false must differ from profile default")
	}
	if c.makeModelsKey(models, Settings{PresetProfile: map[string]any{"a": 1, "b": 2}}) != c.makeModelsKey(models, Settings{PresetProfile: map[string]any{"b": 2, "a": 1}}) {
		t.Fatal("map insertion order must not affect key")
	}
	if c.makeModelsKey(models, Settings{PresetProfile: map[string]any{"bad": make(chan int)}}) != "" {
		t.Fatal("unserializable input must bypass cache")
	}
}

func TestCacheCopiesSurviveConcurrentReplacementAndCleanup(t *testing.T) {
	c := NewPersistentResultCache(time.Nanosecond, 1, t.TempDir())
	sourceDir := t.TempDir()
	source := filepath.Join(sourceDir, "result.gcode")
	if err := os.WriteFile(source, []byte("original"), 0600); err != nil {
		t.Fatal(err)
	}
	c.ttl = time.Hour
	c.Put("key", []string{source}, sourceDir)
	entry, ok := c.Get("key")
	if !ok {
		t.Fatal("miss")
	}
	defer os.RemoveAll(entry.Workdir)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				c.Put("key", []string{source}, sourceDir)
				c.Cleanup()
				c.EvictOldFiles()
				if e, ok := c.Get("key"); ok {
					os.RemoveAll(e.Workdir)
				}
			}
		}()
	}
	wg.Wait()
	c.Put("other-key", []string{source}, sourceDir) // Capacity eviction.
	if _, ok := c.Get("key"); ok {
		t.Fatal("expected old key to be evicted")
	}
	c.mu.Lock()
	c.ttl = time.Nanosecond
	c.mu.Unlock()
	c.Cleanup()
	c.EvictOldFiles()
	for _, path := range []string{source, entry.Files[0]} {
		data, err := os.ReadFile(path)
		if err != nil || string(data) != "original" {
			t.Fatalf("caller result damaged: %q %v", data, err)
		}
	}
}

func TestCacheCopyFailureNeverOwnsCallerDirectory(t *testing.T) {
	base := t.TempDir()
	blocked := filepath.Join(base, "file")
	os.WriteFile(blocked, []byte("block"), 0600)
	c := NewPersistentResultCache(time.Nanosecond, 1, blocked)
	source := filepath.Join(base, "result.gcode")
	os.WriteFile(source, []byte("keep"), 0600)
	c.Put("key", []string{source}, base)
	c.Cleanup()
	if _, err := os.Stat(source); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.Get("key"); ok {
		t.Fatal("failed snapshot must not be cached")
	}
}

func TestPreviewPreservesSupportAndCacheResults(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "slicer")
	script := `#!/bin/sh
while [ "$#" -gt 0 ]; do
 if [ "$1" = "--outputdir" ]; then shift; out="$1"; fi
 shift
done
printf '; total estimated time: 1m\nG28\n' > "$out/plate_1.gcode"
printf '{"return_code":0,"plate_index":1,"sliced_plates":[{"id":1}]}\n' > "$out/result.json"
printf '{"version":1,"generator":"OrcaSlicer 2.5.0","objects":[{"name":"model.stl_1","instance":0}]}\n' > "$out/plate_1.statistics.json"
`
	if err := os.WriteFile(exe, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	c := NewPersistentResultCache(time.Hour, 1, filepath.Join(dir, "cache"))
	s := Service{OrcaSlicerPath: exe, ResultCache: c, State: NewStateStore(dir)}
	for _, support := range []*bool{nil, boolPointer(false), boolPointer(true)} {
		settings := Settings{PrinterProfile: map[string]any{"name": "test"}, PresetProfile: map[string]any{"enable_support": "profile"}, EnableSupport: support}
		for i := 0; i < 2; i++ {
			preview, err := s.Preview(context.Background(), "model.stl", testASCIISTL(), settings)
			if err != nil {
				t.Fatal(err)
			}
			if preview.Quantity != 1 || !preview.Toolpaths.SourceComplete {
				t.Fatalf("cache lost validated preview evidence: %#v", preview)
			}
			// The first execution records the actual preset passed to the CLI.
			if i == 0 {
				got := s.Debug().Preset["enable_support"]
				want := any("profile")
				if support != nil {
					want = boolArg(*support)
				}
				if got != want {
					t.Fatalf("support changed: got %v want %v", got, want)
				}
			}
		}
		settings.GenerateImage = true
		settings.ExportType = "gcode"
		result, err := s.Slice(context.Background(), "model.stl", testASCIISTL(), settings)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := os.Stat(result.Files[0]); err != nil {
			t.Fatal(err)
		}
		if result.Quantity != 1 || len(result.Statistics.Objects) != 1 {
			t.Fatalf("cache lost slice statistics: %#v", result)
		}
		os.RemoveAll(result.Workdir)
	}
}

func TestCacheHitRevalidatesGCodeAndEvidence(t *testing.T) {
	for _, test := range []struct {
		name   string
		damage func(t *testing.T, dir string)
	}{
		{name: "missing manifest", damage: func(t *testing.T, dir string) {
			if err := os.Remove(filepath.Join(dir, "result.json")); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "missing statistics", damage: func(t *testing.T, dir string) {
			if err := os.Remove(filepath.Join(dir, "plate_1.statistics.json")); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "wrong plate", damage: func(t *testing.T, dir string) {
			if err := os.Rename(filepath.Join(dir, "plate_1.gcode"), filepath.Join(dir, "plate_2.gcode")); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "duplicate identities", damage: func(t *testing.T, dir string) {
			statistics := `{"version":1,"generator":"OrcaSlicer 2.5.0","objects":[{"name":"model.stl_1","instance":0},{"name":"model.stl_1","instance":0}]}`
			if err := os.WriteFile(filepath.Join(dir, "plate_1.statistics.json"), []byte(statistics), 0o600); err != nil {
				t.Fatal(err)
			}
		}},
		{name: "empty G-code", damage: func(t *testing.T, dir string) {
			if err := os.WriteFile(filepath.Join(dir, "plate_1.gcode"), nil, 0o600); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			calls := filepath.Join(dir, "calls")
			exe := filepath.Join(dir, "slicer")
			script := fmt.Sprintf(`#!/bin/sh
printf 'call\n' >> %q
while [ "$#" -gt 0 ]; do
 if [ "$1" = "--outputdir" ]; then shift; out="$1"; fi
 shift
done
printf '; total estimated time: 1m\nG28\n' > "$out/plate_1.gcode"
printf '{"return_code":0,"plate_index":1,"sliced_plates":[{"id":1}]}\n' > "$out/result.json"
printf '{"version":1,"generator":"OrcaSlicer 2.5.0","objects":[{"name":"model.stl_1","instance":0}]}\n' > "$out/plate_1.statistics.json"
`, calls)
			if err := os.WriteFile(exe, []byte(script), 0o700); err != nil {
				t.Fatal(err)
			}
			cache := NewPersistentResultCache(time.Hour, 1, filepath.Join(dir, "cache"))
			service := Service{OrcaSlicerPath: exe, ResultCache: cache, State: NewStateStore(dir)}
			result, err := service.Slice(context.Background(), "model.stl", testASCIISTL(), Settings{})
			if err != nil {
				t.Fatal(err)
			}
			os.RemoveAll(result.Workdir)

			cache.mu.RLock()
			if len(cache.entries) != 1 {
				cache.mu.RUnlock()
				t.Fatal("validated result was not cached")
			}
			var snapshot string
			for _, entry := range cache.entries {
				snapshot = entry.Workdir
			}
			cache.mu.RUnlock()
			test.damage(t, snapshot)

			result, err = service.Slice(context.Background(), "model.stl", testASCIISTL(), Settings{})
			if err != nil {
				t.Fatalf("invalid cache should fall back to a fresh slice: %v", err)
			}
			defer os.RemoveAll(result.Workdir)
			data, err := os.ReadFile(calls)
			if err != nil {
				t.Fatal(err)
			}
			if got := strings.Count(string(data), "call\n"); got != 2 {
				t.Fatalf("damaged cache was served instead of resliced; executions=%d", got)
			}
		})
	}
}

func boolPointer(v bool) *bool { return &v }

func TestServiceCacheKeyTracksResolvedProfilesAndExecutable(t *testing.T) {
	exe := filepath.Join(t.TempDir(), "slicer")
	os.WriteFile(exe, []byte("v1"), 0600)
	s := Service{OrcaSlicerPath: exe, ResultCache: NewResultCache(time.Hour, 1)}
	models := []ModelInput{{Filename: "x.stl", Data: []byte("x")}}
	debug := SliceDebug{Preset: map[string]any{"layer_height": "0.2"}}
	key := s.resultCacheKey(models, Settings{}, debug)
	debug.Preset["layer_height"] = "0.3"
	if s.resultCacheKey(models, Settings{}, debug) == key {
		t.Fatal("resolved profile omitted")
	}
	key = s.resultCacheKey(models, Settings{}, debug)
	os.WriteFile(exe, []byte("v2"), 0600)
	if s.resultCacheKey(models, Settings{}, debug) == key {
		t.Fatal("executable omitted")
	}
}
