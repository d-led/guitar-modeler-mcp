// Package mooer is the device-specific backend for the Mooer GE150 Pro Li /
// GE150 Max. It models the device's fixed nine-module signal chain and the
// .mo single-preset file format, and is intentionally independent of every
// other device backend (see internal/catalog and internal/rig for the HeadRush
// Gigboard).
//
// The device numbers each module's effect by an 8-bit index (effect_type) into
// that module's own list; the lists below are the ground truth for those
// indices. The vocabulary follows the owner's manual: the nine modules are
// FX, DS, AMP, CAB, NS, EQ, MOD, DELAY and REVERB.
package mooer

// Amps is every amp model, in effect_type index order (index = wire value).
var Amps = []string{
	"Deluxe Vib", "Deluxe Tweed", "Brit 800", "Brit 2000",
	"US Hi-Gain", "SLO 100", "Fireman", "Dual Rect",
	"Die VH4", "PV 5150", "BE 100", "Recto Verb",
	"Jazz 120", "AC 15", "AC 30", "Match DC30",
	"Tiny Terror", "Blues Jr", "Plexi 50W", "JTM 45",
	"Super Reverb", "Twin Reverb", "Bassman", "Champ",
	"Princeton", "Hiwatt DR103", "Fender 57", "Orange AD30",
	"Marshall JVM", "Mesa MarkV", "Bogner Ecstasy", "ENGL Savage",
	"Diezel Herbert", "Friedman BE", "Soldano SLO", "EVH 5150III",
	"Peavey 6505", "Randall RG", "Laney IRT", "Blackstar HT",
	"Hughes & Kettner", "Koch", "Egnater", "Rivera",
	"Dr. Z", "BadCat", "Budda", "Vox Night Train",
	"Fender Mustang", "Acoustic", "Clean DI", "Crunch DI",
	"Hi-Gain DI", "Lead DI", "Bass",
}

// Cabs is every cabinet model, in effect_type index order.
var Cabs = []string{
	"1x8 Champ", "1x10 Princeton", "1x12 Deluxe", "1x12 AC15",
	"2x10 Twin", "2x12 AC30", "2x12 Jazz", "2x12 Blue",
	"2x12 Match", "2x12 Recto", "4x10 Bassman", "4x12 1960A",
	"4x12 1960B", "4x12 Recto", "4x12 5150", "4x12 SLO",
	"4x12 Uber", "4x12 V30", "4x12 Green", "4x12 Orange",
	"IR Slot 1", "IR Slot 2", "IR Slot 3", "IR Slot 4",
	"IR Slot 5", "IR Slot 6",
}

// Effects is each module's effect list, in effect_type index order. Modules
// that carry a fixed single effect (AMP, CAB, NS, EQ) are omitted: their
// effect_type selects among the dedicated Amps/Cabs lists or is unused.
var Effects = map[string][]string{
	"fx": {
		"Comp", "Red Comp", "T-Comp", "Limiter", "Graphic EQ", "Wah",
		"Auto Wah", "Touch Wah", "Vol Pedal", "Tremolo", "Uni-Vibe",
		"Octave", "Pitch",
	},
	"od": {
		"Blues OD", "TS808", "TS9", "SD-1", "OCD", "Klon", "Rat",
		"Metal Zone", "DS-1", "Fuzz Face", "Big Muff", "Tube Screamer",
	},
	"mod": {
		"Chorus", "Flanger", "Phaser", "Vibrato", "Rotary", "Tremolo",
		"Ring Mod", "Uni-Vibe", "Auto Wah", "Envelope", "Pitch Shift",
		"Detune", "Harmonizer",
	},
	"delay": {
		"Digital", "Analog", "Tape", "Mod Delay", "Reverse", "Ping Pong",
		"Sweep", "Filter", "Crystal",
	},
	"reverb": {
		"Room", "Hall", "Plate", "Spring", "Mod Reverb", "Shimmer",
		"Ambient", "Church", "Arena",
	},
}

// The index↔name and "inspired by" accessors were folded into the Model
// methods (model.go); ModuleOrder moved there too. This file now holds only
// the GE150 Pro Li / GE150 Max model tables (Amps/Cabs/Effects).
