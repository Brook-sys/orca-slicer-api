package slicer

type Settings struct {
	Printer            string                    `json:"printer"`
	Preset             string                    `json:"preset"`
	Filament           string                    `json:"filament"`
	BedType            string                    `json:"bedType"`
	Plate              string                    `json:"plate"`
	Arrange            bool                      `json:"arrange"`
	Orient             bool                      `json:"orient"`
	ExportType         string                    `json:"exportType"`
	MulticolorOnePlate bool                      `json:"multicolorOnePlate"`
	Overrides          map[string]map[string]any `json:"overrides"`
}

type Metadata struct {
	PrintTimeSeconds float64 `json:"printTime"`
	FilamentUsedG    float64 `json:"filamentUsedG"`
	FilamentUsedMM   float64 `json:"filamentUsedMm"`
}

type Result struct {
	Files    []string
	Workdir  string
	Metadata Metadata
}
