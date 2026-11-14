package slicer

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// O vendor tree real do Orca contem arquivos com BOM UTF-8. encoding/json
// rejeita o BOM, e uma varredura que aborta no primeiro arquivo invalido faz a
// resolucao de heranca falhar por um arquivo que nao tem nada a ver com o
// perfil pedido (acontecia com BenFab/process/*.json na 2.5.0).
func TestResolveProfileToleratesInvalidSiblings(t *testing.T) {
	data := t.TempDir()
	orca := t.TempDir()

	writeFile(t, filepath.Join(data, "printers", "user.json"),
		`{"name":"My Printer","inherits":"Elegoo Neptune 4 0.4 nozzle"}`)
	writeFile(t, filepath.Join(orca, "Elegoo", "machine", "EN4SERIES", "Elegoo Neptune 4 0.4 nozzle.json"),
		`{"name":"Elegoo Neptune 4 0.4 nozzle","from":"system","nozzle_diameter":[0.4]}`)
	// BOM UTF-8 (EF BB BF) antes do JSON: ilegivel para encoding/json.
	writeFile(t, filepath.Join(orca, "BenFab", "process", "fdm_process_benfab_04020.json"),
		"\xef\xbb\xbf"+`{"name":"Broken With Bom"}`)
	writeFile(t, filepath.Join(orca, "BenFab", "process", "not_json_at_all.json"), "{{{ isto nao e json")

	res, err := ResolveProfile(data, orca, "printers", "My Printer", nil)
	if err != nil {
		t.Fatalf("expected resolution to succeed, got %v", err)
	}
	if res.Resolved["nozzle_diameter"] == nil {
		t.Fatalf("expected parent nozzle_diameter to be inherited, got %#v", res.Resolved)
	}
	if _, ok := res.Resolved["inherits"]; ok {
		t.Fatalf("expected inherits to be flattened away, got %#v", res.Resolved["inherits"])
	}
}

// O Orca renomeia perfis entre versoes. Quando o nome novo nao bate com o
// "inherits" de um perfil de usuario e o renamed_from tambem nao cobre o caso,
// o ultimo recurso e aceitar um nome com o mesmo conjunto de tokens
// ("Generic PLA @Elegoo" vs "Elegoo Generic PLA"), avisando na resposta.
func TestResolveProfileFallsBackToTokenEquivalentParent(t *testing.T) {
	data := t.TempDir()
	orca := t.TempDir()

	writeFile(t, filepath.Join(data, "filaments", "user.json"),
		`{"name":"Elegoo Generic TPU-OpenNept4une","inherits":"Elegoo Generic PLA","from":"User","filament_type":["TPU"]}`)
	writeFile(t, filepath.Join(orca, "Elegoo", "filament", "Generic", "Generic PLA @Elegoo.json"),
		`{"name":"Generic PLA @Elegoo","from":"system","filament_type":["PLA"],"nozzle_temperature":210}`)

	res, err := ResolveProfile(data, orca, "filaments", "Elegoo Generic TPU-OpenNept4une", nil)
	if err != nil {
		t.Fatalf("expected fallback resolution, got %v", err)
	}
	if res.Resolved["nozzle_temperature"] != float64(210) {
		t.Fatalf("expected parent nozzle_temperature to be inherited, got %#v", res.Resolved)
	}
	warnings := 0
	for _, warning := range res.Warnings {
		if len(warning) > 0 {
			warnings++
		}
	}
	if warnings != 1 {
		t.Fatalf("expected one resolution warning, got %#v", res.Warnings)
	}
}

// Um pai que nao existe em lugar nenhum continua sendo erro explicito.
func TestReadProfileWithBOM(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bom.json")
	writeFile(t, path, "\xef\xbb\xbf"+`{"name":"BOM parent","temperature":210}`)
	profile, err := readProfileFile(path, "filaments", "BOM parent")
	if err != nil || profile["temperature"] != float64(210) {
		t.Fatalf("BOM profile: profile=%v err=%v", profile, err)
	}
}

func TestTokenEquivalentParentRejectsAmbiguity(t *testing.T) {
	data, orca := t.TempDir(), t.TempDir()
	writeFile(t, filepath.Join(data, "filaments", "user.json"), `{"name":"User filament","inherits":"Elegoo Generic PLA"}`)
	writeFile(t, filepath.Join(orca, "A", "filament", "one.json"), `{"name":"Generic PLA @Elegoo","temperature":210}`)
	writeFile(t, filepath.Join(orca, "B", "filament", "two.json"), `{"name":"PLA Generic @Elegoo","temperature":240}`)
	if _, err := ResolveProfile(data, orca, "filaments", "User filament", nil); err == nil {
		t.Fatal("ambiguous fallback must not silently select a temperature")
	}
}

func TestNamedProfileAlwaysResolvesForSlicing(t *testing.T) {
	data := t.TempDir()
	writeFile(t, filepath.Join(data, "printers", "parent.json"), `{"name":"Parent","nozzle_diameter":[0.4]}`)
	writeFile(t, filepath.Join(data, "printers", "child.json"), `{"name":"Child","inherits":"Parent"}`)
	s := &Service{DataPath: data}
	profile, err := s.loadSelectedProfile("printers", "Child", nil, false)
	if err != nil || profile["nozzle_diameter"] == nil || profile["inherits"] != nil {
		t.Fatalf("raw flag must still produce flat profile: profile=%v err=%v", profile, err)
	}
}

func TestResolveProfileMissingParentStillFails(t *testing.T) {
	data := t.TempDir()
	orca := t.TempDir()

	writeFile(t, filepath.Join(data, "printers", "user.json"),
		`{"name":"My Printer","inherits":"Printer That Does Not Exist"}`)
	writeFile(t, filepath.Join(orca, "Elegoo", "machine", "EN4SERIES", "Other.json"),
		`{"name":"Other","from":"system"}`)

	_, err := ResolveProfile(data, orca, "printers", "My Printer", nil)
	if err == nil {
		t.Fatal("expected error for missing parent profile")
	}
}
