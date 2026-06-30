package slicer

import "testing"

func TestValidModelFile(t *testing.T) {
	valid := []string{"model.stl", "model.step", "model.stp", "model.3mf"}
	for _, name := range valid {
		if !validModelFile(name) {
			t.Fatalf("expected %s to be valid", name)
		}
	}
	if validModelFile("model.obj") {
		t.Fatalf("expected obj to be invalid")
	}
}
