package jobs

import "testing"

func TestExtractProgress(t *testing.T) {
	cases := map[string]int{
		"[10%] slicing": 10,
		"progress 55%":  55,
		"100% done":     100,
	}
	for input, expected := range cases {
		got, ok := ExtractProgress(input)
		if !ok || got != expected {
			t.Fatalf("expected %d from %q, got %d ok=%v", expected, input, got, ok)
		}
	}
}

func TestExtractProgressNoMatch(t *testing.T) {
	if _, ok := ExtractProgress("no progress here"); ok {
		t.Fatalf("expected no progress match")
	}
}
