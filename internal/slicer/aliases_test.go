package slicer

import "testing"

func TestKnownProfileAliasNeptune4ToN4(t *testing.T) {
	got := applyKnownProfileAliases("0.20mm Standard @Elegoo Neptune4 (0.4 nozzle)")
	want := "0.20mm Standard @Elegoo N4 (0.4 nozzle)"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestProfileAliasStore(t *testing.T) {
	service := &Service{DataPath: t.TempDir()}
	aliases, err := service.SaveAlias(ProfileAlias{Category: "presets", From: "a", To: "b"})
	if err != nil {
		t.Fatal(err)
	}
	if len(aliases) != 1 {
		t.Fatalf("expected alias")
	}
	loaded, err := service.ListAliases()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 1 || loaded[0].To != "b" {
		t.Fatalf("expected persisted alias")
	}
	loaded, err = service.DeleteAlias("presets", "a")
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 0 {
		t.Fatalf("expected alias deleted")
	}
}
