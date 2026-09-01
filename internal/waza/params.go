package waza

import "math"

// The Waza Air .tsl backup stores each patch as a 2335-byte record in the
// BOSS Katana dense patch layout. The offsets, encodings and type indices
// below are taken from the authoritative reverse-engineering reference
// (the `waza-tsl` Python package: known_indexes.py / waza_tsl.py), which maps
// offset -> [name, case, limits] where case is one of:
//
//	"listed"  an enum of the given allowed values
//	"minmax"  a plain byte clamped to [lo, hi]
//	"scaled"  stored = clamp(round(base + slope*input), lo, hi)
//	"2bytes"  11-bit value split as [value/128, value%128] over two bytes
//
// All single-byte values are 0-100 unless noted.
const (
	// Amp (preamp).
	offPreampType     = 81
	offPreampGain     = 82 // scaled: stored = 20 + 0.8*gain
	offPreampBass     = 84
	offPreampMiddle   = 85
	offPreampTreble   = 86
	offPreampPresence = 87
	offPreampLevel    = 88

	// Booster (OD/DS).
	offBoosterOnOff  = 48
	offBoosterType   = 49
	offBoosterDrive  = 50 // 0-120
	offBoosterBottom = 51 // scaled: stored = 50 + bottom (bottom -50..+50)
	offBoosterTone   = 52 // scaled: stored = 50 + tone (tone -50..+50)
	offBoosterSoloSW = 53
	offBoosterSoloLv = 54
	offBoosterLevel  = 55
	offBoosterMix    = 56

	// Mod (FX1) and FX (FX2).
	offFX1OnOff = 192
	offFX1Type  = 193
	offFX2OnOff = 460
	offFX2Type  = 461

	// Delay (single-tap). Time is two bytes in milliseconds: high byte is
	// value/128, low byte is value%128.
	offDelayOnOff     = 736
	offDelayType      = 737
	offDelayTimeHi    = 738
	offDelayTimeLo    = 739
	offDelayFeedback  = 740
	offDelayHighCut   = 741 // 0-14
	offDelayLevel     = 742 // 0-120
	offDelayDirectMix = 743 // 0-100

	// Delay 2 is the second delay block the Waza Air reaches in DLY+REV mode.
	// A single requested delay turns this off so no second repeat leaks
	// through from the template.
	offDelay2OnOff = 2126

	// Reverb.
	offReverbOnOff     = 784
	offReverbType      = 785
	offReverbTime      = 786 // scaled: stored = -1 + 10*seconds (0.1-10.0 s)
	offReverbPreDelay  = 787 // 2 bytes, 0-500 ms
	offReverbLevel     = 792
	offReverbDirectMix = 793 // 0-100

	// Gyro (positional audio) and ambience.
	offGyroType = 854 // OFF/SURROUND/STATIC/STAGE
	offGyroPos  = 855 // guitar position -180..+180, stored = 60 + pos/3
	offAmbType  = 853 // STUDIO/STAGE
	offAmbLevel = 856

	// The DELAY / DLY+REV / REVERB knob, stored once per color slot; a tone
	// written here sets all three slots to the same mode.
	offModeGreen  = 2332
	offModeRed    = 2333
	offModeYellow = 2334

	// Noise suppressor.
	offNSOn        = 867
	offNSThreshold = 868
	offNSRelease   = 869
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

// delayTypeIndex maps a delay name to its delay type index.
var delayTypeIndex = map[string]byte{
	"DIGITAL DELAY": 0,
	"REVERSE DELAY": 6,
	"ANALOG DELAY":  7,
	"TAPE ECHO":     8,
	"MODULATE":      9,
	"SDE-3000":      10,
}

// reverbTypeIndex maps the Waza Air reverb name to its reverb type index.
var reverbTypeIndex = map[string]byte{
	"ROOM REVERB":     1,
	"HALL REVERB":     3,
	"PLATE REVERB":    4,
	"SPRING REVERB":   5,
	"MODULATE REVERB": 6,
}

// gyroTypeIndex maps the POSITION knob to the gyro type index.
var gyroTypeIndex = map[string]byte{
	"OFF":      0,
	"SURROUND": 1,
	"STATIC":   2,
	"STAGE":    3,
}

// ambienceTypeIndex maps the AMBIENCE knob to its type index.
var ambienceTypeIndex = map[string]byte{
	"STUDIO": 0,
	"STAGE":  1,
}

// modeIndex maps the DELAY / DLY+REV / REVERB knob to its reverb-mode index.
var modeIndex = map[string]byte{
	"DELAY":   0,
	"DLY+REV": 1,
	"REVERB":  2,
}

func invert(m map[string]byte) map[byte]string {
	out := make(map[byte]string, len(m))
	for name, idx := range m {
		out[idx] = name
	}
	return out
}

var (
	ampTypeName      = invert(ampTypeIndex)
	boosterTypeName  = invert(boosterTypeIndex)
	modFXTypeName    = invert(modFXTypeIndex)
	delayTypeName    = invert(delayTypeIndex)
	reverbTypeName   = invert(reverbTypeIndex)
	gyroTypeName     = invert(gyroTypeIndex)
	ambienceTypeName = invert(ambienceTypeIndex)
	modeName         = invert(modeIndex)
)

// Params is the decoded, human-facing parameter set of one patch. A zero
// numeric field or empty type name means the value is unset/unknown; on
// write, those fields are left untouched (the template's bytes are kept).
type Params struct {
	AmpType     string
	AmpGain     int
	AmpVolume   int
	AmpBass     int
	AmpMiddle   int
	AmpTreble   int
	AmpPresence int

	BoosterType      string
	BoosterDrive     int
	BoosterBottom    int // -50..+50
	BoosterTone      int // 0-100, 50 = neutral
	BoosterSolo      bool
	BoosterSoloLevel int
	BoosterLevel     int
	BoosterDirectMix int

	ModType   string
	ModParams map[string]float64 // canonical knob -> value, e.g. rate/depth
	FXType    string
	FXParams  map[string]float64

	DelayType      string
	DelayTime      int // milliseconds
	DelayFeedback  int
	DelayHighCut   int // 0-14
	DelayLevel     int
	DelayDirectMix int // 0-100

	ReverbType      string
	ReverbTime      float64 // seconds, 0.1-10.0
	ReverbPreDelay  int     // milliseconds, 0-500
	ReverbLevel     int
	ReverbDirectMix int // 0-100

	Position       string // SURROUND / STATIC / STAGE / OFF
	GuitarPosition int    // degrees, -180..+180 (SURROUND only)
	Ambience       string // STUDIO / STAGE
	AmbienceLevel  int    // 0-100
	Mode           string // DELAY / DLY+REV / REVERB

	NSOn        *bool // nil = keep template; true = on; false = off
	NSThreshold int
	NSRelease   int
}

// ReadParams decodes the patch's active parameters into names and values. An
// effect type is only reported when its block is on, so an off block reads as
// empty rather than echoing its remembered type.
func (p Patch) ReadParams() Params {
	raw := p.Raw
	pr := Params{
		AmpType:        ampTypeName[raw[offPreampType]],
		AmpGain:        ampGainDecode(raw[offPreampGain]),
		AmpVolume:      int(raw[offPreampLevel]),
		AmpBass:        int(raw[offPreampBass]),
		AmpMiddle:      int(raw[offPreampMiddle]),
		AmpTreble:      int(raw[offPreampTreble]),
		AmpPresence:    int(raw[offPreampPresence]),
		Position:       gyroTypeName[raw[offGyroType]],
		GuitarPosition: gyroPositionDecode(raw[offGyroPos]),
		Ambience:       ambienceTypeName[raw[offAmbType]],
		AmbienceLevel:  int(raw[offAmbLevel]),
		Mode:           modeName[raw[offModeGreen]],
		NSOn:           boolPtr(raw[offNSOn] != 0),
		NSThreshold:    int(raw[offNSThreshold]),
		NSRelease:      int(raw[offNSRelease]),
	}
	if raw[offBoosterOnOff] != 0 {
		pr.BoosterType = boosterTypeName[raw[offBoosterType]]
		pr.BoosterDrive = int(raw[offBoosterDrive])
		pr.BoosterBottom = int(raw[offBoosterBottom]) - 50
		pr.BoosterTone = int(raw[offBoosterTone])
		pr.BoosterSolo = raw[offBoosterSoloSW] != 0
		pr.BoosterSoloLevel = int(raw[offBoosterSoloLv])
		pr.BoosterLevel = int(raw[offBoosterLevel])
		pr.BoosterDirectMix = int(raw[offBoosterMix])
	}
	if raw[offFX1OnOff] != 0 {
		pr.ModType = modFXTypeName[raw[offFX1Type]]
		pr.ModParams = readKnobs(raw, pr.ModType, false)
	}
	if raw[offFX2OnOff] != 0 {
		pr.FXType = modFXTypeName[raw[offFX2Type]]
		pr.FXParams = readKnobs(raw, pr.FXType, true)
	}
	if raw[offDelayOnOff] != 0 {
		pr.DelayType = delayTypeName[raw[offDelayType]]
		pr.DelayTime = int(raw[offDelayTimeHi])<<7 | int(raw[offDelayTimeLo])
		pr.DelayFeedback = int(raw[offDelayFeedback])
		pr.DelayHighCut = int(raw[offDelayHighCut])
		pr.DelayLevel = int(raw[offDelayLevel])
		pr.DelayDirectMix = int(raw[offDelayDirectMix])
	}
	if raw[offReverbOnOff] != 0 {
		pr.ReverbType = reverbTypeName[raw[offReverbType]]
		pr.ReverbTime = reverbTimeDecode(raw[offReverbTime])
		pr.ReverbPreDelay = int(raw[offReverbPreDelay])<<7 | int(raw[offReverbPreDelay+1])
		pr.ReverbLevel = int(raw[offReverbLevel])
		pr.ReverbDirectMix = int(raw[offReverbDirectMix])
	}
	return pr
}

// WriteParams applies the given parameters to a copy of the patch and returns
// it. A zero numeric field or empty type name leaves the template's byte
// unchanged, so only the values that were specified are touched.
func (p Patch) WriteParams(pr Params) Patch {
	out := NewPatch(p.Raw)
	writeAmpParams(out.Raw, pr)
	writeBoosterParams(out.Raw, pr)
	writeModFXParams(out.Raw, pr)
	writeDelayParams(out.Raw, pr)
	writeReverbParams(out.Raw, pr)
	writeSpatialParams(out.Raw, pr)
	return out
}

func setByte(raw []byte, off, v int) {
	if v > 0 {
		raw[off] = byte(v) // #nosec G115 -- v is a validated 0..100 knob value
	}
}

func writeAmpParams(raw []byte, pr Params) {
	if idx, ok := ampTypeIndex[pr.AmpType]; ok {
		raw[offPreampType] = idx
	}
	if pr.AmpGain > 0 {
		raw[offPreampGain] = ampGainEncode(pr.AmpGain)
	}
	setByte(raw, offPreampLevel, pr.AmpVolume)
	setByte(raw, offPreampBass, pr.AmpBass)
	setByte(raw, offPreampMiddle, pr.AmpMiddle)
	setByte(raw, offPreampTreble, pr.AmpTreble)
	setByte(raw, offPreampPresence, pr.AmpPresence)
}

func writeBoosterParams(raw []byte, pr Params) {
	if pr.BoosterType == "" {
		raw[offBoosterOnOff] = 0
		return
	}
	raw[offBoosterOnOff] = 1
	raw[offBoosterType] = boosterTypeIndex[pr.BoosterType]
	setByte(raw, offBoosterDrive, pr.BoosterDrive)
	if pr.BoosterBottom != 0 {
		raw[offBoosterBottom] = byte(50 + pr.BoosterBottom) // #nosec G115 -- bottom is -50..50, so 50+bottom is 0..100
	}
	setByte(raw, offBoosterTone, pr.BoosterTone)
	if pr.BoosterSolo {
		raw[offBoosterSoloSW] = 1
		setByte(raw, offBoosterSoloLv, pr.BoosterSoloLevel)
	} else {
		raw[offBoosterSoloSW] = 0
	}
	setByte(raw, offBoosterLevel, pr.BoosterLevel)
	setByte(raw, offBoosterMix, pr.BoosterDirectMix)
}

func writeModFXParams(raw []byte, pr Params) {
	writeModOrFX(raw, offFX1OnOff, offFX1Type, pr.ModType, pr.ModParams, false)
	writeModOrFX(raw, offFX2OnOff, offFX2Type, pr.FXType, pr.FXParams, true)
}

func writeModOrFX(raw []byte, onOff, typ int, name string, params map[string]float64, fx bool) {
	if name == "" {
		raw[onOff] = 0
		return
	}
	raw[onOff] = 1
	raw[typ] = modFXTypeIndex[name]
	applyKnobs(raw, name, fx, params)
}

func writeDelayParams(raw []byte, pr Params) {
	if pr.DelayType == "" {
		raw[offDelayOnOff] = 0
		raw[offDelay2OnOff] = 0
		return
	}
	raw[offDelayOnOff] = 1
	raw[offDelayType] = delayTypeIndex[pr.DelayType]
	if pr.DelayTime > 0 {
		raw[offDelayTimeHi] = byte(pr.DelayTime / 128) // #nosec G115 -- high byte of a two-byte ms value
		raw[offDelayTimeLo] = byte(pr.DelayTime % 128)
	}
	setByte(raw, offDelayFeedback, pr.DelayFeedback)
	setByte(raw, offDelayHighCut, pr.DelayHighCut)
	setByte(raw, offDelayLevel, pr.DelayLevel)
	setByte(raw, offDelayDirectMix, pr.DelayDirectMix)
	// One requested delay means the second delay block stays off.
	raw[offDelay2OnOff] = 0
}

func writeReverbParams(raw []byte, pr Params) {
	if pr.ReverbType == "" {
		raw[offReverbOnOff] = 0
	} else {
		raw[offReverbOnOff] = 1
		raw[offReverbType] = reverbTypeIndex[pr.ReverbType]
	}
	if pr.ReverbTime > 0 {
		raw[offReverbTime] = reverbTimeEncode(pr.ReverbTime)
	}
	if pr.ReverbPreDelay > 0 {
		raw[offReverbPreDelay] = byte(pr.ReverbPreDelay / 128) // #nosec G115 -- high byte of a two-byte ms value
		raw[offReverbPreDelay+1] = byte(pr.ReverbPreDelay % 128)
	}
	setByte(raw, offReverbLevel, pr.ReverbLevel)
	setByte(raw, offReverbDirectMix, pr.ReverbDirectMix)
}

func writeSpatialParams(raw []byte, pr Params) {
	if idx, ok := gyroTypeIndex[pr.Position]; ok {
		raw[offGyroType] = idx
	}
	if pr.GuitarPosition != 0 {
		raw[offGyroPos] = gyroPositionEncode(pr.GuitarPosition)
	}
	if idx, ok := ambienceTypeIndex[pr.Ambience]; ok {
		raw[offAmbType] = idx
	}
	setByte(raw, offAmbLevel, pr.AmbienceLevel)
	if idx, ok := modeIndex[pr.Mode]; ok {
		raw[offModeGreen] = idx
		raw[offModeRed] = idx
		raw[offModeYellow] = idx
	}
	if pr.NSOn != nil {
		if *pr.NSOn {
			raw[offNSOn] = 1
		} else {
			raw[offNSOn] = 0
		}
	}
	setByte(raw, offNSThreshold, pr.NSThreshold)
	setByte(raw, offNSRelease, pr.NSRelease)
}

// ampGainEncode stores the amp gain knob (0-100) as the Katana gain byte:
// stored = round(20 + 0.8*gain), clamped to [20, 100].
func ampGainEncode(gain int) byte {
	return byte(clamp(int(math.Round(20+0.8*float64(gain))), 20, 100)) // #nosec G115 -- clamped to 20..100
}

// ampGainDecode recovers the amp gain knob (0-100) from a stored gain byte:
// gain = round((stored - 20) / 0.8).
func ampGainDecode(stored byte) int {
	return int(math.Round((float64(stored) - 20) / 0.8))
}

// gyroPositionEncode stores the guitar position in degrees (-180..+180) as
// the gyro byte: stored = 60 + position/3.
func gyroPositionEncode(position int) byte {
	return byte(clamp(int(math.Round(60+float64(position)/3)), 0, 120)) // #nosec G115 -- clamped to 0..120
}

// gyroPositionDecode recovers the guitar position in degrees from the stored
// gyro byte: position = (stored - 60) * 3.
func gyroPositionDecode(stored byte) int {
	return (int(stored) - 60) * 3
}

// reverbTimeEncode stores the reverb time in seconds (0.1-10.0) as the reverb
// time byte: stored = -1 + 10*seconds.
func reverbTimeEncode(seconds float64) byte {
	return byte(clamp(int(math.Round(-1+10*seconds)), 0, 99)) // #nosec G115 -- clamped to 0..99
}

// reverbTimeDecode recovers the reverb time in seconds from the stored byte:
// seconds = (stored + 1) / 10.
func reverbTimeDecode(stored byte) float64 {
	return (float64(stored) + 1) / 10
}

// boolPtr returns a pointer to b, for optional on/off fields where nil means
// "keep the template's value".
func boolPtr(b bool) *bool {
	return &b
}

func clamp(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
