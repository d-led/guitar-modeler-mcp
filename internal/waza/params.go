package waza

// The Waza Air .tsl backup stores each patch as a 2335-byte record in the
// BOSS Katana dense patch layout, so the parameter offsets below match
// lib-katana's data/tsl-map.csv. The offsets and the type indices were
// verified against real Waza Air backups and the Katana MIDI implementation
// (katana_sysex.txt by Steven Hirsch). All single-byte values are 0-100
// unless noted.
const (
	// Amp (preamp A).
	offPreampType     = 81
	offPreampGain     = 82
	offPreampBass     = 84
	offPreampMiddle   = 85
	offPreampTreble   = 86
	offPreampPresence = 87
	offPreampLevel    = 88

	// Booster (OD/DS).
	offBoosterType  = 49
	offBoosterDrive = 50
	offBoosterTone  = 52
	offBoosterLevel = 55

	// Mod (FX1) and FX (FX2).
	offFX1OnOff = 192
	offFX1Type  = 193
	offFX2OnOff = 460
	offFX2Type  = 461

	// Delay. Time is two bytes: high 4 bits + low 7 bits, in milliseconds.
	// The block also carries two independent tap lines (Dual Delay 1 & 2);
	// they share the same 11-bit time encoding.
	offDelayOnOff    = 736
	offDelayType     = 737
	offDelayTimeHi   = 738
	offDelayTimeLo   = 739
	offDelayFeedback = 740
	offDelayLevel    = 742
	offDelayD1TimeHi = 745
	offDelayD1TimeLo = 746
	offDelayD1Level  = 749
	offDelayD2TimeHi = 750
	offDelayD2TimeLo = 751
	offDelayD2Level  = 754

	// Reverb.
	offReverbOnOff = 784
	offReverbType  = 785
	offReverbTime  = 786
	offReverbLevel = 792
)

// ampTypeIndex maps the Waza Air amp name to its Katana preamp_a_type index.
// The Waza Air exposes the Katana panel's five amp positions.
var ampTypeIndex = map[string]byte{
	"FLAT":   1,  // FULL RANGE
	"CLEAN":  8,  // JC-120
	"CRUNCH": 11, // TWEED
	"LEAD":   24, // 5150 DRIVE
	"BROWN":  23, // SLDN (Soldano SLO-100)
}

// boosterTypeIndex maps the Waza Air booster name to its Katana od_ds_type
// index.
var boosterTypeIndex = map[string]byte{
	"MID BOOST": 0, "CLEAN BOOST": 1, "TREBLE BOOST": 2, "CRUNCH OD": 3,
	"NATURAL OD": 4, "WARM OD": 5, "FAT DS": 6, "METAL DS": 8, "OCT FUZZ": 9,
	"BLUES DRIVE": 10, "OVERDRIVE": 11, "T-SCREAM": 12, "TURBO OD": 13,
	"DISTORTION": 14, "RAT": 15, "GUV DS": 16, "DST+": 17, "METAL ZONE": 18,
	"'60S FUZZ": 19, "MUFF FUZZ": 20,
}

// modFXTypeIndex maps a MOD/FX effect name to its Katana fx1/fx2 type index.
// The Waza Air catalog exposes a subset of these; the rest are readable here
// because factory presets use them (e.g. the template's Pitch Shifter).
var modFXTypeIndex = map[string]byte{
	"T.WAH": 0, "AUTO WAH": 1, "PEDAL WAH": 2, "COMP": 3, "LIMITER": 4,
	"GRAPHIC EQ": 6, "PARAMETRIC EQ": 7, "GUITAR SIM": 9, "SLOW GEAR": 10,
	"WAVE SYNTH": 12, "OCTAVE": 14, "PITCH SHIFTER": 15, "HARMONIST": 16,
	"AC PROCESSOR": 18, "PHASER": 19, "FLANGER": 20, "TREMOLO": 21,
	"ROTARY": 22, "UNI-V": 23, "SLICER": 25, "VIBRATO": 26, "RING MOD": 27,
	"HUMANIZER": 28, "CHORUS": 29, "AC GUITAR SIM": 31,
}

// delayTypeIndex maps a delay name to its Katana delay_type index. The
// hardware knob quick-selects the first three; the rest are read-only here.
var delayTypeIndex = map[string]byte{
	"DIGITAL DELAY": 0,
	"REVERSE DELAY": 6,
	"ANALOG DELAY":  7,
	"TAPE ECHO":     8,
	"MODULATE":      9,
}

// reverbTypeIndex maps the Waza Air reverb name to its Katana reverb_type
// index.
var reverbTypeIndex = map[string]byte{
	"PLATE REVERB":  4,
	"SPRING REVERB": 5,
	"HALL REVERB":   3,
}

func invert(m map[string]byte) map[byte]string {
	out := make(map[byte]string, len(m))
	for name, idx := range m {
		out[idx] = name
	}
	return out
}

var (
	ampTypeName     = invert(ampTypeIndex)
	boosterTypeName = invert(boosterTypeIndex)
	modFXTypeName   = invert(modFXTypeIndex)
	delayTypeName   = invert(delayTypeIndex)
	reverbTypeName  = invert(reverbTypeIndex)
)

// Params is the decoded, human-facing parameter set of one patch. A zero
// numeric field or empty type name means the value is unset/unknown; on
// write, those fields are left untouched (the template's bytes are kept).
type Params struct {
	AmpType       string
	AmpGain       int
	AmpVolume     int
	AmpBass       int
	AmpMiddle     int
	AmpTreble     int
	AmpPresence   int
	BoosterType   string
	BoosterDrive  int
	BoosterTone   int
	BoosterLevel  int
	ModType       string
	FXType        string
	DelayType     string
	DelayTime     int // milliseconds
	DelayFeedback int
	DelayLevel    int
	ReverbType    string
	ReverbLevel   int
}

// ReadParams decodes the patch's active parameters into names and values. An
// effect type is only reported when its block is on, so an off block reads as
// empty rather than echoing its remembered type.
func (p Patch) ReadParams() Params {
	raw := p.Raw
	pr := Params{
		AmpType:      ampTypeName[raw[offPreampType]],
		AmpGain:      int(raw[offPreampGain]),
		AmpVolume:    int(raw[offPreampLevel]),
		AmpBass:      int(raw[offPreampBass]),
		AmpMiddle:    int(raw[offPreampMiddle]),
		AmpTreble:    int(raw[offPreampTreble]),
		AmpPresence:  int(raw[offPreampPresence]),
		BoosterType:  boosterTypeName[raw[offBoosterType]],
		BoosterDrive: int(raw[offBoosterDrive]),
		BoosterTone:  int(raw[offBoosterTone]),
		BoosterLevel: int(raw[offBoosterLevel]),
	}
	if raw[offFX1OnOff] != 0 {
		pr.ModType = modFXTypeName[raw[offFX1Type]]
	}
	if raw[offFX2OnOff] != 0 {
		pr.FXType = modFXTypeName[raw[offFX2Type]]
	}
	if raw[offDelayOnOff] != 0 {
		pr.DelayType = delayTypeName[raw[offDelayType]]
		pr.DelayTime = int(raw[offDelayTimeHi])<<7 | int(raw[offDelayTimeLo])
		pr.DelayFeedback = int(raw[offDelayFeedback])
		pr.DelayLevel = int(raw[offDelayLevel])
	}
	if raw[offReverbOnOff] != 0 {
		pr.ReverbType = reverbTypeName[raw[offReverbType]]
		pr.ReverbLevel = int(raw[offReverbLevel])
	}
	return pr
}

// WriteParams applies the given parameters to a copy of the patch and returns
// it. A zero numeric field or empty type name leaves the template's byte
// unchanged, so only the values that were specified are touched.
func (p Patch) WriteParams(pr Params) Patch {
	out := NewPatch(p.Raw)

	if idx, ok := ampTypeIndex[pr.AmpType]; ok {
		out.Raw[offPreampType] = idx
	}
	setByte := func(off int, v int) {
		if v > 0 {
			out.Raw[off] = byte(v)
		}
	}
	setByte(offPreampGain, pr.AmpGain)
	setByte(offPreampLevel, pr.AmpVolume)
	setByte(offPreampBass, pr.AmpBass)
	setByte(offPreampMiddle, pr.AmpMiddle)
	setByte(offPreampTreble, pr.AmpTreble)
	setByte(offPreampPresence, pr.AmpPresence)

	if idx, ok := boosterTypeIndex[pr.BoosterType]; ok {
		out.Raw[offBoosterType] = idx
	}
	setByte(offBoosterDrive, pr.BoosterDrive)
	setByte(offBoosterTone, pr.BoosterTone)
	setByte(offBoosterLevel, pr.BoosterLevel)

	if pr.ModType != "" {
		out.Raw[offFX1OnOff] = 1
		out.Raw[offFX1Type] = modFXTypeIndex[pr.ModType]
	} else {
		out.Raw[offFX1OnOff] = 0
	}
	if pr.FXType != "" {
		out.Raw[offFX2OnOff] = 1
		out.Raw[offFX2Type] = modFXTypeIndex[pr.FXType]
	} else {
		out.Raw[offFX2OnOff] = 0
	}

	if pr.DelayType != "" {
		out.Raw[offDelayOnOff] = 1
		out.Raw[offDelayType] = delayTypeIndex[pr.DelayType]
	} else {
		out.Raw[offDelayOnOff] = 0
	}
	if pr.DelayTime > 0 {
		out.Raw[offDelayTimeHi] = byte(pr.DelayTime >> 7)
		out.Raw[offDelayTimeLo] = byte(pr.DelayTime & 0x7F)
		// Align the two tap lines with the requested single-tap time, so a
		// preset written over the template does not keep the template's
		// double-delay taps (the Waza Air spreads them wide in headphones).
		out.Raw[offDelayD1TimeHi] = byte(pr.DelayTime >> 7)
		out.Raw[offDelayD1TimeLo] = byte(pr.DelayTime & 0x7F)
		out.Raw[offDelayD2TimeHi] = byte(pr.DelayTime >> 7)
		out.Raw[offDelayD2TimeLo] = byte(pr.DelayTime & 0x7F)
	}
	setByte(offDelayFeedback, pr.DelayFeedback)
	setByte(offDelayLevel, pr.DelayLevel)
	setByte(offDelayD1Level, pr.DelayLevel)
	setByte(offDelayD2Level, pr.DelayLevel)

	if pr.ReverbType != "" {
		out.Raw[offReverbOnOff] = 1
		out.Raw[offReverbType] = reverbTypeIndex[pr.ReverbType]
	} else {
		out.Raw[offReverbOnOff] = 0
	}
	setByte(offReverbLevel, pr.ReverbLevel)

	return out
}
