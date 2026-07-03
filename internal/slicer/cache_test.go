package slicer

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildArgsForInputsPassesAllModels(t *testing.T) {
	service := Service{DataPath: t.TempDir()}
	workdir := t.TempDir()
	inputDir := filepath.Join(workdir, "input")
	outputDir := filepath.Join(workdir, "output")
	if err := os.MkdirAll(inputDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatal(err)
	}

	modelA := filepath.Join(inputDir, "a.stl")
	modelB := filepath.Join(inputDir, "b.stl")
	args, err := service.buildArgsForInputs([]string{modelA, modelB}, inputDir, outputDir, Settings{Arrange: true}, &SliceDebug{})
	if err != nil {
		t.Fatal(err)
	}

	if args[len(args)-2] != modelA || args[len(args)-1] != modelB {
		t.Fatalf("expected both models at end of args, got %#v", args)
	}
}

func TestPersistentResultCacheCopiesResultFiles(t *testing.T) {
	cacheDir := filepath.Join(t.TempDir(), "cache")
	workdir := t.TempDir()
	resultPath := filepath.Join(workdir, "result.gcode")
	if err := os.WriteFile(resultPath, []byte("G28"), 0o644); err != nil {
		t.Fatal(err)
	}

	cache := NewPersistentResultCache(time.Hour, 10, cacheDir)
	models := []ModelInput{{Filename: "a.stl", Data: []byte("solid a")}}
	key := cache.makeModelsKey(models, Settings{})
	cache.Put(key, []string{resultPath}, workdir)

	if err := os.RemoveAll(workdir); err != nil {
		t.Fatal(err)
	}

	entry, ok := cache.Get(key)
	if !ok {
		t.Fatalf("expected persistent cache hit")
	}
	if entry.Workdir == workdir {
		t.Fatalf("expected cache workdir to be copied")
	}
	if _, err := os.Stat(entry.Files[0]); err != nil {
		t.Fatalf("expected cached file to exist: %v", err)
	}
}

func TestResultCacheUsesModelsAndSettings(t *testing.T) {
	cache := NewResultCache(time.Hour, 10)
	workdir := t.TempDir()
	resultPath := filepath.Join(workdir, "result.gcode")
	if err := os.WriteFile(resultPath, []byte("G28"), 0o644); err != nil {
		t.Fatal(err)
	}
	models := []ModelInput{{Filename: "a.stl", Data: []byte("solid a")}}
	key := cache.makeModelsKey(models, Settings{ExportType: "gcode"})
	cache.Put(key, []string{resultPath}, workdir)

	entry, ok := cache.Get(key)
	if !ok {
		t.Fatalf("expected cache hit")
	}
	if entry.Files[0] != resultPath || entry.Workdir != workdir {
		t.Fatalf("unexpected cache entry %#v", entry)
	}

	otherKey := cache.makeModelsKey(models, Settings{ExportType: "3mf"})
	if _, ok := cache.Get(otherKey); ok {
		t.Fatalf("expected different settings to miss cache")
	}
}
