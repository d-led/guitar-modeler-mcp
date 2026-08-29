// Package waza is the device-specific backend for the Boss Waza Air wireless
// headphone amp. It reads and writes the BOSS TONE STUDIO liveset format
// (.tsl) and renders a printable setup card. The model data is drawn from the
// owner's manual and its "Effect Parameter List".
package waza

// Item is one selectable model: the on-device name and the real hardware it
// emulates (empty when not documented).
type Item struct {
	Name       string `json:"name"`
	InspiredBy string `json:"inspired_by,omitempty"`
}

// Device describes the Waza Air and its selectable models.
type Device struct {
	Name         string
	Display      string
	FileExchange bool
	FileExt      string
	// Chain is the signal-chain order of the effect blocks.
	Chain []string
	// Amps is the five amp types.
	Amps []Item
	// Boosters is the BOOSTER block effect list.
	Boosters []Item
	// ModFX is the shared MOD/FX block effect list.
	ModFX []Item
	// Delays is the DELAY block effect list (the hardware knob quick-selects;
	// the full list may be extended).
	Delays []Item
	// Reverbs is the REVERB block effect list (hardware knob quick-selects).
	Reverbs []Item
	// CabResonance is the cabinet resonance type.
	CabResonance []string
	// Ambience is the ambience type.
	Ambience []string
	// Position is the positional audio type.
	Position []string
	// Mode is the DELAY/FX assignment mode.
	Mode []string
}

// Default returns the Waza Air device model.
func Default() Device {
	return Device{
		Name:         "wazaair",
		Display:      "BOSS Waza Air",
		FileExchange: true,
		FileExt:      ".tsl",
		Chain:        []string{"BOOSTER", "AMP", "MOD", "FX", "DELAY", "REVERB"},
		Amps: items(
			[2]string{"CLEAN", "Roland JC-120 / Fender Twin Reverb"},
			[2]string{"CRUNCH", "Marshall Plexi 1959 / Fender Tweed"},
			[2]string{"LEAD", "EVH 5150 / Peavey 5150 lead channel"},
			[2]string{"BROWN", "Soldano SLO-100 (EVH brown sound)"},
			[2]string{"FLAT", "Studio DI / Acoustic Preamp"},
		),
		Boosters: items(
			[2]string{"CLEAN BOOST", ""},
			[2]string{"TREBLE BOOST", ""},
			[2]string{"MID BOOST", ""},
			[2]string{"CRUNCH OD", ""},
			[2]string{"BLUES DRIVE", "BOSS BD-2 Blues Driver"},
			[2]string{"OVERDRIVE", "BOSS OD-1"},
			[2]string{"NATURAL OD", ""},
			[2]string{"WARM OD", ""},
			[2]string{"TURBO OD", "BOSS OD-2"},
			[2]string{"T-SCREAM", "Ibanez TS-808"},
			[2]string{"DISTORTION", ""},
			[2]string{"FAT DS", ""},
			[2]string{"DST+", "MXR Distortion+"},
			[2]string{"GUV DS", "Marshall Guv'nor"},
			[2]string{"RAT", "ProCo Rat"},
			[2]string{"METAL ZONE", "BOSS MT-2"},
			[2]string{"METAL DS", ""},
			[2]string{"'60S FUZZ", "Dunlop Fuzz Face"},
			[2]string{"MUFF FUZZ", "EHX Big Muff Pi"},
			[2]string{"OCT FUZZ", ""},
		),
		ModFX: items(
			[2]string{"CHORUS", ""},
			[2]string{"FLANGER", ""},
			[2]string{"PHASER", ""},
			[2]string{"UNI-V", "Shin-ei Uni-Vibe"},
			[2]string{"TREMOLO", ""},
			[2]string{"VIBRATO", ""},
			[2]string{"ROTARY", ""},
			[2]string{"RING MOD", ""},
			[2]string{"SLOW GEAR", ""},
			[2]string{"SLICER", ""},
			[2]string{"COMP", ""},
			[2]string{"LIMITER", ""},
			[2]string{"T.WAH", ""},
			[2]string{"AUTO WAH", ""},
			[2]string{"PEDAL WAH", ""},
			[2]string{"GRAPHIC EQ", ""},
		),
		Delays: items(
			[2]string{"DIGITAL DELAY", ""},
			[2]string{"ANALOG DELAY", ""},
			[2]string{"TAPE ECHO", "Maestro Echoplex EP-3 / Roland RE-201 Space Echo"},
		),
		Reverbs: items(
			[2]string{"PLATE REVERB", ""},
			[2]string{"SPRING REVERB", ""},
			[2]string{"HALL REVERB", ""},
		),
		CabResonance: []string{"VINTAGE", "MODERN", "DEEP"},
		Ambience:     []string{"STUDIO", "STAGE"},
		Position:     []string{"SURROUND", "STATIC", "STAGE"},
		Mode:         []string{"DELAY", "DLY+REV", "REVERB"},
	}
}

func items(pairs ...[2]string) []Item {
	out := make([]Item, len(pairs))
	for i, p := range pairs {
		out[i] = Item{Name: p[0], InspiredBy: p[1]}
	}
	return out
}
