package waza

import (
	"math"

	"github.com/d-led/guitar-modeler-mcp/internal/device"
)

// knobCase discriminates how a knob value is encoded in the patch record.
type knobCase int

const (
	knobListed   knobCase = iota // one of the allowed integer values
	knobMinmax                   // plain byte clamped to [lo, hi]
	knobScaled                   // stored = round(base + slope*input), clamped
	knobTwoBytes                 // 11-bit value split [value/128, value%128]
)

// knob is one editable parameter at a fixed offset. `name` is the canonical
// key (lower-case, underscores) an agent passes in mod_params/fx_params; the
// reference names from waza-tsl are kept as comments next to each table.
type knob struct {
	name   string
	offset int
	kind   knobCase
	limits []float64
}

func minmax(name string, offset, lo, hi int) knob {
	return knob{name: name, offset: offset, kind: knobMinmax, limits: []float64{float64(lo), float64(hi)}}
}

func lst(name string, offset int, values ...int) knob {
	limits := make([]float64, len(values))
	for i, v := range values {
		limits[i] = float64(v)
	}
	return knob{name: name, offset: offset, kind: knobListed, limits: limits}
}

func scl(name string, offset int, base, slope float64, lo, hi int) knob {
	return knob{name: name, offset: offset, kind: knobScaled, limits: []float64{base, slope, float64(lo), float64(hi)}}
}

func two(name string, offset, lo, hi int) knob {
	return knob{name: name, offset: offset, kind: knobTwoBytes, limits: []float64{float64(lo), float64(hi)}}
}

// set writes a value into the record. A value of NaN leaves the byte(s)
// untouched, so absent knobs keep the template's value.
func (k knob) set(raw []byte, v float64) {
	if math.IsNaN(v) {
		return
	}
	switch k.kind {
	case knobListed:
		for _, allowed := range k.limits {
			if v == allowed {
				raw[k.offset] = byte(v)
				return
			}
		}
	case knobMinmax:
		raw[k.offset] = byte(clamp(int(math.Round(v)), int(k.limits[0]), int(k.limits[1]))) // #nosec G115 -- clamped to the knob's byte range
	case knobScaled:
		base, slope, lo, hi := k.limits[0], k.limits[1], k.limits[2], k.limits[3]
		raw[k.offset] = byte(clamp(int(math.Round(base+slope*v)), int(lo), int(hi))) // #nosec G115 -- clamped to the knob's byte range
	case knobTwoBytes:
		val := clamp(int(math.Round(v)), int(k.limits[0]), int(k.limits[1]))
		raw[k.offset] = byte(val / 128)   // #nosec G115 -- high byte of a clamped two-byte value
		raw[k.offset+1] = byte(val % 128) // #nosec G115 -- low byte of a clamped two-byte value
	}
}

// get reads the knob's value, decoding scaled and two-byte encodings back into
// their input units. Listed and minmax knobs return the raw byte.
func (k knob) get(raw []byte) float64 {
	switch k.kind {
	case knobScaled:
		base, slope := k.limits[0], k.limits[1]
		return (float64(raw[k.offset]) - base) / slope
	case knobTwoBytes:
		return float64(int(raw[k.offset])*128 + int(raw[k.offset+1]))
	default:
		return float64(raw[k.offset])
	}
}

// effectKnobs is the parameter set of one MOD/FX effect type. The MOD block
// uses the stored offsets directly; the FX block offsets are `delta` bytes
// further on (268 for the shared effects, fewer for the extras).
type effectKnobs struct {
	delta int
	knobs []knob
}

// fxEffectDelta is the constant offset between the MOD and FX parameter
// regions for the effects the two blocks share.
const fxEffectDelta = 268

// effectKnobTable maps an effect name (as in modFXTypeIndex) to its knobs at
// MOD-block offsets, plus the delta to the FX block. Offsets and encodings
// are transcribed from waza-tsl's known_indexes.py.
var effectKnobTable = map[string]effectKnobs{
	"T.WAH": {fxEffectDelta, []knob{
		lst("mode", 204, 0, 1), lst("polarity", 205, 0, 1), minmax("sens", 206, 0, 100),
		minmax("frequency", 207, 0, 100), minmax("peak", 208, 0, 100), minmax("direct_mix", 209, 0, 100),
		minmax("effect_level", 210, 0, 100),
	}},
	"AUTO WAH": {fxEffectDelta, []knob{
		lst("mode", 212, 0, 1), minmax("frequency", 213, 0, 100), minmax("peak", 214, 0, 100),
		minmax("rate", 215, 0, 100), minmax("depth", 216, 0, 100), minmax("direct_mix", 217, 0, 100),
		minmax("effect_level", 218, 0, 100),
	}},
	"PEDAL WAH": {fxEffectDelta, []knob{
		lst("type", 220, 0, 1, 2, 3, 4, 5), minmax("pedal_position", 221, 0, 100),
		minmax("pedal_min", 222, 0, 100), minmax("pedal_max", 223, 0, 100),
		minmax("effect_level", 224, 0, 100), minmax("direct_mix", 225, 0, 100),
	}},
	"COMP": {fxEffectDelta, []knob{
		lst("type", 227, 0, 1, 2, 3, 4, 5, 6), minmax("sustain", 228, 0, 100),
		minmax("attack", 229, 0, 100), scl("tone", 230, 50, 1, 0, 100), minmax("level", 231, 0, 100),
	}},
	"LIMITER": {fxEffectDelta, []knob{
		lst("type", 233, 0, 1, 2), minmax("attack", 234, 0, 100), minmax("threshold", 235, 0, 100),
		lst("ratio", 236, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17),
		minmax("release", 237, 0, 100), minmax("level", 238, 0, 100),
	}},
	"GRAPHIC EQ": {fxEffectDelta, []knob{
		scl("31hz", 240, 20, 1, 0, 40), scl("62hz", 241, 20, 1, 0, 40), scl("125hz", 242, 20, 1, 0, 40),
		scl("250hz", 243, 20, 1, 0, 40), scl("500hz", 244, 20, 1, 0, 40), scl("1khz", 245, 20, 1, 0, 40),
		scl("2khz", 246, 20, 1, 0, 40), scl("4khz", 247, 20, 1, 0, 40), scl("8khz", 248, 20, 1, 0, 40),
		scl("16khz", 249, 20, 1, 0, 40), scl("level", 250, 20, 1, 0, 40),
	}},
	"PARAMETRIC EQ": {fxEffectDelta, []knob{
		minmax("low_cut", 252, 0, 17), scl("low_gain", 253, 20, 1, 0, 40), minmax("low_mid_frequency", 254, 0, 27),
		lst("low_mid_q", 255, 0, 1, 2, 3, 4, 5), scl("low_mid_gain", 256, 20, 1, 0, 40),
		minmax("high_mid_frequency", 257, 0, 27), lst("high_mid_q", 258, 0, 1, 2, 3, 4, 5),
		scl("high_mid_gain", 259, 20, 1, 0, 40), scl("high_gain", 260, 20, 1, 0, 40),
		minmax("high_cut", 261, 0, 14), scl("level", 262, 20, 1, 0, 40),
	}},
	"GUITAR SIM": {fxEffectDelta, []knob{
		lst("type", 270, 0, 1, 2, 3, 4, 5, 6, 7), scl("low", 271, 50, 1, 0, 100),
		scl("high", 272, 50, 1, 0, 100), minmax("level", 273, 0, 100), minmax("body", 274, 0, 100),
	}},
	"SLOW GEAR": {fxEffectDelta, []knob{
		minmax("sens", 276, 0, 100), minmax("rise_time", 277, 0, 100), minmax("level", 278, 0, 100),
	}},
	"WAVE SYNTH": {fxEffectDelta, []knob{
		lst("wave", 288, 0, 1), minmax("cutoff", 289, 0, 100), minmax("resonance", 290, 0, 100),
		minmax("filter_sens", 291, 0, 100), minmax("filter_decay", 292, 0, 100), minmax("filter_depth", 293, 0, 100),
		minmax("synth_level", 294, 0, 100), minmax("direct_mix", 295, 0, 100),
	}},
	"OCTAVE": {fxEffectDelta, []knob{
		lst("range", 305, 0, 1, 2, 3), minmax("effect_level", 306, 0, 100), minmax("direct_mix", 307, 0, 100),
	}},
	"PITCH SHIFTER": {fxEffectDelta, []knob{
		lst("voice", 309, 0, 1), lst("ps1_mode", 310, 0, 1, 2, 3), scl("ps1_pitch", 311, 24, 1, 0, 48),
		scl("ps1_fine", 312, 50, 1, 0, 100), two("ps1_pre_delay", 313, 0, 300), minmax("ps1_level", 315, 0, 100),
		lst("ps2_mode", 316, 0, 1, 2, 3), scl("ps2_pitch", 317, 24, 1, 0, 48),
		scl("ps2_fine", 318, 50, 1, 0, 100), two("ps2_pre_delay", 319, 0, 300), minmax("ps2_level", 321, 0, 100),
		minmax("ps1_feedback", 322, 0, 100), minmax("direct_mix", 323, 0, 100),
	}},
	"HARMONIST": {fxEffectDelta, []knob{
		lst("voice", 325, 0, 1), lst("hr1_harmony", 326, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29),
		two("hr1_pre_delay", 327, 0, 300), minmax("hr1_level", 329, 0, 100),
		lst("hr2_harmony", 330, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29),
		two("hr2_pre_delay", 331, 0, 300), minmax("hr2_level", 333, 0, 100),
		minmax("hr1_feedback", 334, 0, 100), minmax("direct_mix", 335, 0, 100),
	}},
	"AC PROCESSOR": {fxEffectDelta, []knob{
		lst("type", 365, 0, 1, 2, 3), scl("bass", 366, 50, 1, 0, 100), scl("middle", 367, 50, 1, 0, 100),
		minmax("middle_frequency", 368, 0, 27), scl("treble", 369, 50, 1, 0, 100),
		scl("presence", 370, 50, 1, 0, 100), minmax("level", 371, 0, 100),
	}},
	"PHASER": {fxEffectDelta, []knob{
		lst("type", 373, 0, 1, 2, 3), minmax("rate", 374, 0, 100), minmax("depth", 375, 0, 100),
		minmax("manual", 376, 0, 100), minmax("resonance", 377, 0, 100), minmax("step_rate", 378, 0, 101),
		minmax("effect_level", 379, 0, 100), minmax("direct_mix", 380, 0, 100),
	}},
	"FLANGER": {fxEffectDelta, []knob{
		minmax("rate", 382, 0, 100), minmax("depth", 383, 0, 100), minmax("manual", 384, 0, 100),
		minmax("resonance", 385, 0, 100), lst("low_cut", 387, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10),
		minmax("effect_level", 388, 0, 100), minmax("direct_mix", 389, 0, 100),
	}},
	"TREMOLO": {fxEffectDelta, []knob{
		minmax("wave_shape", 391, 0, 100), minmax("rate", 392, 0, 100), minmax("depth", 393, 0, 100), minmax("level", 394, 0, 100),
	}},
	"ROTARY": {fxEffectDelta, []knob{
		minmax("rate", 398, 0, 100), minmax("depth", 401, 0, 100), minmax("level", 402, 0, 100),
	}},
	"UNI-V": {fxEffectDelta, []knob{
		minmax("rate", 404, 0, 100), minmax("depth", 405, 0, 100), minmax("level", 406, 0, 100),
	}},
	"SLICER": {fxEffectDelta, []knob{
		lst("pattern", 415, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19),
		minmax("rate", 416, 0, 100), minmax("trigger_sens", 417, 0, 100),
		minmax("effect_level", 418, 0, 100), minmax("direct_mix", 419, 0, 100),
	}},
	"VIBRATO": {fxEffectDelta, []knob{
		minmax("rate", 421, 0, 100), minmax("depth", 422, 0, 100), minmax("level", 425, 0, 100),
	}},
	"RING MOD": {fxEffectDelta, []knob{
		lst("mode", 427, 0, 1), minmax("frequency", 428, 0, 100),
		minmax("effect_level", 429, 0, 100), minmax("direct_mix", 430, 0, 100),
	}},
	"HUMANIZER": {fxEffectDelta, []knob{
		lst("mode", 432, 0, 1), lst("vowel1", 433, 0, 1, 2, 3, 4), lst("vowel2", 434, 0, 1, 2, 3, 4),
		minmax("sens", 435, 0, 100), minmax("rate", 436, 0, 100), minmax("depth", 437, 0, 100),
		minmax("manual", 438, 0, 100), minmax("level", 439, 0, 100),
	}},
	"CHORUS": {fxEffectDelta, []knob{
		lst("xover_frequency", 441, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16),
		minmax("low_rate", 442, 0, 100), minmax("low_depth", 443, 0, 100), scl("low_pre_delay", 444, 0, 2, 0, 80),
		minmax("low_level", 445, 0, 100),
		minmax("high_rate", 446, 0, 100), minmax("high_depth", 447, 0, 100), scl("high_pre_delay", 448, 0, 2, 0, 80),
		minmax("high_level", 449, 0, 100), minmax("direct_mix", 450, 0, 100),
		// Aliases matching the Waza Air's simple "Rate/Depth/Effect Level" UI:
		// they target the chorus's low band.
		minmax("rate", 442, 0, 100), minmax("depth", 443, 0, 100), minmax("effect_level", 445, 0, 100),
	}},
	"AC GUITAR SIM": {15, []knob{
		scl("high", 2064, 50, 1, 0, 100), minmax("body", 2065, 0, 100),
		scl("low", 2066, 50, 1, 0, 100), minmax("level", 2068, 0, 100),
	}},
	"PHASER 90E": {6, []knob{
		lst("script", 2109, 0, 1), minmax("speed", 2110, 0, 100),
	}},
	"FLANGER 117E": {6, []knob{
		minmax("manual", 2111, 0, 100), minmax("width", 2112, 0, 100),
		minmax("speed", 2113, 0, 100), minmax("regen", 2114, 0, 100),
	}},
}

// applyKnobs writes the given knob values for an effect into the record. `fx`
// selects the FX block (true) or the MOD block (false). Unknown effect or knob
// names are ignored, so a typo never corrupts the patch.
func applyKnobs(raw []byte, effect string, fx bool, values map[string]float64) {
	ek, ok := effectKnobTable[effect]
	if !ok {
		return
	}
	delta := 0
	if fx {
		delta = ek.delta
	}
	// Normalise the supplied keys once (e.g. "EFFECT LEVEL" -> "effect_level")
	// so agents can use either the on-device label or the canonical key.
	norm := make(map[string]float64, len(values))
	for k, v := range values {
		norm[device.CanonicalKey(k)] = v
	}
	// Write in table order so the chorus rate/depth/effect_level aliases win
	// over the low_* names.
	for _, k := range ek.knobs {
		v, present := norm[k.name]
		if !present {
			continue
		}
		kk := k
		kk.offset += delta
		kk.set(raw, v)
	}
}

// readKnobs decodes every knob of an effect from the record into a map of
// canonical name -> value. `fx` selects the FX block (true) or MOD (false).
func readKnobs(raw []byte, effect string, fx bool) map[string]float64 {
	ek, ok := effectKnobTable[effect]
	if !ok {
		return nil
	}
	delta := 0
	if fx {
		delta = ek.delta
	}
	out := make(map[string]float64, len(ek.knobs))
	for _, k := range ek.knobs {
		kk := k
		kk.offset += delta
		out[k.name] = kk.get(raw)
	}
	return out
}
