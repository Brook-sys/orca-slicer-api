package slicer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectSupportInGCode(t *testing.T) {
	dir := t.TempDir()
	gcodePath := filepath.Join(dir, "support.gcode")
	if err := os.WriteFile(gcodePath, []byte(";TYPE:SUPPORT\nG1 X0 Y0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	usesSupport, err := detectSupportInGCode(gcodePath)
	if err != nil {
		t.Fatal(err)
	}
	if !usesSupport {
		t.Fatal("expected support to be detected")
	}
}

func TestDetectSupportInGCodeReturnsFalseWithoutSupport(t *testing.T) {
	dir := t.TempDir()
	gcodePath := filepath.Join(dir, "no-support.gcode")
	if err := os.WriteFile(gcodePath, []byte(";TYPE:WALL-OUTER\nG1 X0 Y0\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	usesSupport, err := detectSupportInGCode(gcodePath)
	if err != nil {
		t.Fatal(err)
	}
	if usesSupport {
		t.Fatal("expected support not to be detected")
	}
}
