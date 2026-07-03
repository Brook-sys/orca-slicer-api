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

func TestResultCacheUsesModelsAndSettings(t *testing.T) {
	cache := NewResultCache(time.Hour, 10)
	models := []ModelInput{{Filename: "a.stl", Data: []byte("solid a")}}
	key := cache.makeModelsKey(models, Settings{ExportType: "gcode"})
	cache.Put(key, []string{"/tmp/result.gcode"}, "/tmp/work")

	entry, ok := cache.Get(key)
	if !ok {
		t.Fatalf("expected cache hit")
	}
	if entry.Files[0] != "/tmp/result.gcode" || entry.Workdir != "/tmp/work" {
		t.Fatalf("unexpected cache entry %#v", entry)
	}

	otherKey := cache.makeModelsKey(models, Settings{ExportType: "3mf"})
	if _, ok := cache.Get(otherKey); ok {
		t.Fatalf("expected different settings to miss cache")
	}
}
