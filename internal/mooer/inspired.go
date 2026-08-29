package mooer

import "strings"

// The device's models, mapped to the real hardware they emulate. Only targets
// that are obvious from the name or the owner's manual are annotated; anything
// ambiguous is left out, matching the Gigboard catalog's "leave empty rather
// than guess" policy. These strings are the Mooer half of the cross-device
// "inspired by" lookup tables consumed by internal/presetmap.
var (
	ampInspiredBy = map[string]string{
		"Deluxe Vib":       "Fender Deluxe Reverb",
		"Deluxe Tweed":     "Fender Tweed Deluxe",
		"Brit 800":         "Marshall JCM800",
		"Brit 2000":        "Marshall JCM2000",
		"SLO 100":          "Soldano SLO-100",
		"Dual Rect":        "Mesa Boogie Dual Rectifier",
		"Die VH4":          "Diezel VH4",
		"PV 5150":          "Peavey 5150",
		"BE 100":           "Friedman BE-100",
		"Recto Verb":       "Mesa Boogie Recto-Verb",
		"Jazz 120":         "Roland JC-120",
		"AC 15":            "Vox AC15",
		"AC 30":            "Vox AC30",
		"Match DC30":       "Matchless DC-30",
		"Tiny Terror":      "Orange Tiny Terror",
		"Blues Jr":         "Fender Blues Junior",
		"Plexi 50W":        "Marshall Super Lead",
		"JTM 45":           "Marshall JTM45",
		"Super Reverb":     "Fender Super Reverb",
		"Twin Reverb":      "Fender Twin Reverb",
		"Bassman":          "Fender Bassman",
		"Champ":            "Fender Champ",
		"Princeton":        "Fender Princeton",
		"Hiwatt DR103":     "Hiwatt DR103",
		"Orange AD30":      "Orange AD30",
		"Marshall JVM":     "Marshall JVM",
		"Mesa MarkV":       "Mesa Boogie Mark V",
		"Bogner Ecstasy":   "Bogner Ecstasy",
		"ENGL Savage":      "ENGL Savage",
		"Diezel Herbert":   "Diezel Herbert",
		"Friedman BE":      "Friedman BE-100",
		"Soldano SLO":      "Soldano SLO-100",
		"EVH 5150III":      "EVH 5150 III",
		"Peavey 6505":      "Peavey 6505",
		"Laney IRT":        "Laney Ironheart",
		"Blackstar HT":     "Blackstar HT",
		"Vox Night Train":  "Vox Night Train",
		"Fender Mustang":   "Fender Mustang",
		"Acoustic":         "Acoustic",
		"Bass":             "Ampeg SVT",
	}

	cabInspiredBy = map[string]string{
		"1x8 Champ":     "Fender Champ 1x8",
		"1x10 Princeton": "Fender Princeton 1x10",
		"1x12 Deluxe":   "Fender Deluxe Reverb 1x12",
		"1x12 AC15":     "Vox AC15 1x12",
		"2x10 Twin":     "Fender Twin 2x10",
		"2x12 AC30":     "Vox AC30 2x12",
		"2x12 Jazz":     "Roland JC-120 2x12",
		"2x12 Blue":     "Vox AC30 2x12 Alnico Blue",
		"2x12 Match":    "Matchless 2x12",
		"2x12 Recto":    "Mesa Boogie Rectifier 2x12",
		"4x10 Bassman":  "Fender Bassman 4x10",
		"4x12 1960A":    "Marshall 1960A 4x12",
		"4x12 1960B":    "Marshall 1960B 4x12",
		"4x12 Recto":    "Mesa Boogie Rectifier 4x12",
		"4x12 5150":     "EVH 5150 4x12",
		"4x12 SLO":      "Soldano 4x12",
		"4x12 Uber":     "Bogner Uberkab 4x12",
		"4x12 V30":      "Celestion Vintage 30 4x12",
		"4x12 Green":    "Celestion Greenback 4x12",
		"4x12 Orange":   "Orange 4x12",
	}

	fxInspiredBy = map[string]map[string]string{
		"fx": {
			"Comp": "Studio Compressor", "Red Comp": "Ross Compressor",
			"Graphic EQ": "Graphic EQ", "Wah": "Wah",
			"Auto Wah": "Envelope Filter", "Touch Wah": "Envelope Filter",
			"Vol Pedal": "Volume Pedal", "Tremolo": "Tremolo",
			"Uni-Vibe": "Uni-Vibe", "Octave": "Octave",
		},
		"od": {
			"TS808": "Ibanez TS808", "TS9": "Ibanez TS9",
			"DS-1": "BOSS DS-1", "Big Muff": "EHX Big Muff",
			"Tube Screamer": "Ibanez Tube Screamer",
		},
		"mod": {
			"Chorus": "Chorus", "Flanger": "Flanger", "Phaser": "Phaser",
			"Vibrato": "Vibrato", "Rotary": "Rotary", "Tremolo": "Tremolo",
			"Ring Mod": "Ring Modulator", "Uni-Vibe": "Uni-Vibe",
			"Auto Wah": "Envelope Filter", "Detune": "Detune",
			"Harmonizer": "Harmonizer",
		},
		"delay": {
			"Digital": "Digital Delay", "Analog": "Analog Delay",
			"Tape": "Tape Echo", "Reverse": "Reverse Delay",
		},
		"reverb": {
			"Room": "Room Reverb", "Hall": "Hall Reverb",
			"Spring": "Spring Reverb", "Mod Reverb": "Modulated Reverb",
			"Shimmer": "Shimmer Reverb", "Ambient": "Ambient Reverb",
		},
	}
)

// AmpInspiredBy returns the real amplifier a Mooer amp model emulates.
func AmpInspiredBy(model string) (string, bool) {
	v, ok := ampInspiredBy[exact(ampInspiredBy, model)]
	return v, ok
}

// CabInspiredBy returns the real cabinet a Mooer cab model emulates.
func CabInspiredBy(model string) (string, bool) {
	v, ok := cabInspiredBy[exact(cabInspiredBy, model)]
	return v, ok
}

// FXInspiredBy returns the real effect a Mooer effect (by module and name)
// emulates.
func FXInspiredBy(module, name string) (string, bool) {
	table, ok := fxInspiredBy[strings.ToLower(module)]
	if !ok {
		return "", false
	}
	v, ok := table[exact(table, name)]
	return v, ok
}

// exact resolves name against a map case-insensitively, returning the exact
// map key that matched.
func exact[V any](m map[string]V, name string) string {
	if _, ok := m[name]; ok {
		return name
	}
	for k := range m {
		if strings.EqualFold(k, name) {
			return k
		}
	}
	return name
}
