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
	Model string `json:"model"`
	// Brand is the real manufacturer the model emulates, e.g. "Marshall".
	Brand string `json:"brand,omitempty"`
	// RealModel is the real amplifier, e.g. "JCM800 2203".
	RealModel string `json:"real_model,omitempty"`
	// Channel is the channel or variant when an amp is split across several
	// models, e.g. "Crunch" for "11 EPB II Crunch".
	Channel string `json:"channel,omitempty"`
	// Wattage is a human readable wattage hint, e.g. "100W".
	Wattage string `json:"wattage,omitempty"`
	// Style lists the genres the amp is typically associated with.
	Style []string `json:"style,omitempty"`
	// Bass marks bass-oriented models.
	Bass bool `json:"bass,omitempty"`
	// Gain classifies the amp's drive character: "bass", "clean",
	// "edge of breakup", "crunch" or "high gain".
	Gain string `json:"gain,omitempty"`
	// Description is a one sentence, human readable summary.
	Description string `json:"description"`
	// ModeledAfter is the real amplifier the model emulates, e.g.
	// "Marshall JCM800 2203 Head", taken from the Gigboard Hints.
	ModeledAfter string `json:"modeled_after,omitempty"`
	// Confirmed marks whether the community has confirmed the emulation target.
	Confirmed bool `json:"confirmed"`
}

// Cab is one HeadRush cabinet model.
type Cab struct {
	Model        string `json:"model"`                  // exact device name, e.g. "4x12 Green 25W"
	Speakers     string `json:"speakers,omitempty"`     // speaker configuration, e.g. "4x12"
	SpeakersRef  string `json:"speakers_ref,omitempty"` // emulated speaker, e.g. "Celestion Greenback"
	Description  string `json:"description"`
	ModeledAfter string `json:"modeled_after,omitempty"`
	Confirmed    bool   `json:"confirmed"`
}

// Mic is one HeadRush microphone model.
type Mic struct {
	Model        string `json:"model"` // exact device name, e.g. "Dyn 57"
	Kind         string `json:"kind,omitempty"`
	RealModel    string `json:"real_model,omitempty"` // e.g. "Shure SM57"
	Description  string `json:"description"`
	ModeledAfter string `json:"modeled_after,omitempty"`
	Confirmed    bool   `json:"confirmed"`
}

// FX is one effect (module) type that can be placed in a rig chain.
type FX struct {
	Name         string `json:"name"` // exact device module name, e.g. "Tape Echo"
	Category     string `json:"category,omitempty"`
	Description  string `json:"description"`
	ModeledAfter string `json:"modeled_after,omitempty"`
	Confirmed    bool   `json:"confirmed"`
	// Gain classifies a distortion effect's drive strength — "boost",
	// "overdrive", "distortion", "fuzz", "bass" or "bitcrusher" — so an agent
	// can match the pedal to the part (mirrors Amp.Gain). Empty for non-drive
	// effects.
	Gain string `json:"gain,omitempty"`
}

// Category is one effect category, e.g. "delay" or "reverb". Categories mirror
// the subdivision used by the Gigboard Hints reference (distortion, dynamics,
// eq, expression, modulation, delay, reverb) plus a utility group for
// IR/FX-loop/acoustic simulation modules.
type Category struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}
