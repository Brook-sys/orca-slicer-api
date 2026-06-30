package profiles

import "testing"

func TestValidProfileFile(t *testing.T) {
	if !validProfileFile("preset.json") {
		t.Fatalf("expected json profile to be valid")
	}
	if validProfileFile("preset.txt") {
		t.Fatalf("expected txt profile to be invalid")
	}
}
