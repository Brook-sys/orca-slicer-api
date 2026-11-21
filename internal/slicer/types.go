package slicer

type ModelInput struct {
	Filename string
	Data     []byte
}

type Settings struct {
	Printer               string                    `json:"printer"`
	Preset                string                    `json:"preset"`
	Filament              string                    `json:"filament"`
	BedType               string                    `json:"bedType"`
	Plate                 string                    `json:"plate"`
	Quantity              int                       `json:"quantity"`
	Arrange               bool                      `json:"arrange"`
	ArrangeSpacing        float64                   `json:"arrangeSpacing"`
	Orient                bool                      `json:"orient"`
	Rotate                *float64                  `json:"rotate"`
	RotateX               *float64                  `json:"rotateX"`
	RotateY               *float64                  `json:"rotateY"`
	ExportType            string                    `json:"exportType"`
	MulticolorOnePlate    bool                      `json:"multicolorOnePlate"`
	ResolveProfiles       bool                      `json:"resolveProfiles"`
	SanitizeProfiles      bool                      `json:"sanitizeProfiles"`
	GenerateImage         bool                      `json:"generateImage"`
	EnableSupport         *bool                     `json:"enableSupport"`
	BrimType              *bool                     `json:"brimType"`
	PrintSequenceByObject bool                      `json:"printSequenceByObject"`
	Overrides             map[string]map[string]any `json:"overrides"`
	PrinterProfile        map[string]any            `json:"-"`
	PresetProfile         map[string]any            `json:"-"`
	FilamentProfiles      []map[string]any          `json:"-"`
}

type Metadata struct {
	PrintTimeSeconds float64 `json:"printTime"`
	FilamentUsedG    float64 `json:"filamentUsedG"`
	FilamentUsedMM   float64 `json:"filamentUsedMm"`
}

type Result struct {
	Files      []string
	Workdir    string
	Metadata   Metadata
	Quantity   int
	Statistics SliceStatistics
}

type PreviewResult struct {
	UsesSupport      bool
	PrintTimeSeconds float64
	FilamentUsedG    float64
	FilamentUsedMm   float64
	ThumbnailBase64  string
	Toolpaths        ToolpathPreview
	Quantity         int
	Workdir          string
}
