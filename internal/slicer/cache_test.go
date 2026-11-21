package slicer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBuildArgsUsesStrictSingleSTLContract(t *testing.T) {
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

	model := filepath.Join(inputDir, "a.stl")
	zero := 0.0
	rotation := 30.0
	args, err := service.buildArgsForInputs([]string{model}, inputDir, outputDir, Settings{Quantity: 3, Arrange: true, RotateX: &zero, Rotate: &rotation}, &SliceDebug{})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	for _, expected := range []string{"--slice 1", "--clone-objects 3", "--export-statistics", "--rotate-x 0 --rotate 30 --ensure-on-bed --arrange 1 --orient 0"} {
		if !strings.Contains(joined, expected) {
			t.Fatalf("expected %q in args, got %#v", expected, args)
		}
	}
	if args[len(args)-1] != model {
		t.Fatalf("expected one trailing STL path, got %#v", args)
	}
	if strings.Contains(joined, "--export-3mf") || strings.Contains(joined, "--repetitions") {
		t.Fatalf("unexpected legacy arguments: %#v", args)
	}
	plainArgs, err := service.buildArgsForInputs([]string{model}, inputDir, outputDir, Settings{}, &SliceDebug{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(strings.Join(plainArgs, " "), "--ensure-on-bed") {
		t.Fatalf("unrotated input must preserve its original Z placement: %#v", plainArgs)
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
	defer os.RemoveAll(entry.Workdir)
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
	defer os.RemoveAll(entry.Workdir)
	if entry.Files[0] == resultPath || entry.Workdir == workdir {
		t.Fatalf("cache must return an independent copy: %#v", entry)
	}

	otherKey := cache.makeModelsKey(models, Settings{ExportType: "gcode", Quantity: 2, Arrange: true})
	if _, ok := cache.Get(otherKey); ok {
		t.Fatalf("expected different settings to miss cache")
	}
}
