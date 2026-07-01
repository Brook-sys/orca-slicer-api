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
	ResolveProfiles    bool                      `json:"resolveProfiles"`
	SanitizeProfiles   bool                      `json:"sanitizeProfiles"`
	GenerateImage      bool                      `json:"generateImage"`
	EnableSupport      *bool                     `json:"enableSupport"`
	BrimType           *bool                     `json:"brimType"`
	Overrides          map[string]map[string]any `json:"overrides"`
	PrinterProfile     map[string]any            `json:"-"`
	PresetProfile      map[string]any            `json:"-"`
	FilamentProfiles   []map[string]any          `json:"-"`
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
