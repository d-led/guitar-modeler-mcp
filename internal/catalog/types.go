// Package catalog is the single source of truth for the models available on
// the HeadRush Gigboard: amps, cabs, mics and effect types. The data is derived
// from the device backup (Blocks/) and the community-maintained Gigboard Hints
// translation table, so the MCP server can describe exactly which tones are
// available and what real-world hardware each model emulates.
package catalog

// Amp is one HeadRush amp model together with the real amplifier it emulates.
type Amp struct {
	// Model is the exact model name as it appears on the device, e.g.
	// "82 Lead 800 100W". This string is written verbatim into rig files.
	Model string
	// Brand is the real manufacturer the model emulates, e.g. "Marshall".
	// Empty when the emulated amp is not publicly documented.
	Brand string
	// RealModel is the real amplifier, e.g. "JCM800 2203".
	RealModel string
	// Channel is the channel or variant when an amp is split across several
	// models, e.g. "Crunch" for "11 EPB II Crunch".
	Channel string
	// Wattage is a human readable wattage hint, e.g. "100W".
	Wattage string
	// Style lists the genres the amp is typically associated with.
	Style []string
	// Bass marks bass-oriented models.
	Bass bool
	// Description is a one sentence, human readable summary.
	Description string
}

// Cab is one HeadRush cabinet model.
type Cab struct {
	Model       string // exact device name, e.g. "4x12 Green 25W"
	Speakers    string // speaker configuration, e.g. "4x12"
	SpeakersRef string // emulated speaker, e.g. "Celestion Greenback"
	Description string
}

// Mic is one HeadRush microphone model.
type Mic struct {
	Model       string // exact device name, e.g. "Dyn 57"
	Kind        string // "dynamic", "condenser" or "ribbon"
	RealModel   string // e.g. "Shure SM57"
	Description string
}

// FX is one effect (module) type that can be placed in a rig chain.
type FX struct {
	Name        string // exact device module name, e.g. "Tape Echo"
	Category    string // "drive", "modulation", "delay/reverb", "dynamics/eq", "filter", "utility"
	Description string
}
