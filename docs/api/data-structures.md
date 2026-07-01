# Estrutura de Dados

## Settings (corpo do slice)

```go
type Settings struct {
    Printer               string
    Preset                string
    Filament              string
    BedType               string
    Plate                 string
    Arrange               bool
    Orient                bool
    ExportType            string
    MulticolorOnePlate    bool
    ResolveProfiles       bool
    SanitizeProfiles      bool
    GenerateImage         bool
    EnableSupport         *bool
    BrimType              *bool
    PrintSequenceByObject bool
    Overrides             map[string]map[string]any
    PrinterProfile        map[string]any
    PresetProfile         map[string]any
    FilamentProfiles      []map[string]any
}
```

## Metadata

```go
type Metadata struct {
    PrintTimeSeconds float64
    FilamentUsedG    float64
    FilamentUsedMM   float64
}
```

## Result

```go
type Result struct {
    Files    []string
    Workdir  string
    Metadata Metadata
}
```

## PreviewResult

```go
type PreviewResult struct {
    UsesSupport      bool
    PrintTimeSeconds float64
    FilamentUsedG    float64
    FilamentUsedMm   float64
    ThumbnailBase64  string
    Workdir          string
}
```
