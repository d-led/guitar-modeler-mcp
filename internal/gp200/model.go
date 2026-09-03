package gp200

import "github.com/d-led/guitar-modeler-mcp/internal/device"

// Model describes one GP-200 family device. The whole family (GP-200, -R, -X,
// -JR and -LT) writes the same .prst file and shares one effect catalog, so
// the only thing that varies between models is the display name; the effect
// tables in catalog_data.go are common to all of them.
type Model struct {
	// Name is the stable identifier, e.g. "gp200".
	Name string `json:"name"`
	// Display is the human-readable name, e.g. "Valeton GP-200".
	Display string `json:"display"`
	// FileExchange reports whether presets can be transferred via a file.
	// The GP-200 always can; this field mirrors the other device backends.
	FileExchange bool `json:"file_exchange"`
	// FileExt is the preset file extension.
	FileExt string `json:"file_ext,omitempty"`
}

// models is the registry of supported GP-200 devices, in display order.
var models = []Model{
	{Name: "gp200", Display: "Valeton GP-200", FileExchange: true, FileExt: ".prst"},
	{Name: "gp200lt", Display: "Valeton GP-200 LT", FileExchange: true, FileExt: ".prst"},
}

// Models returns the supported GP-200 devices.
func Models() []Model {
	return append([]Model(nil), models...)
}

// ModelByName returns the model with the given identifier, matching
// case-insensitively against both the stable name and the display name.
func ModelByName(name string) (Model, bool) {
	return device.FindByName(models, name, func(m Model) (string, string) { return m.Name, m.Display })
}

// Default returns the canonical GP-200 model used for cross-device mapping.
func Default() Model {
	m, _ := ModelByName("gp200")
	return m
}
