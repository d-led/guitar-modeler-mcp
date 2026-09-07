package catalog

import "strings"

// classifyFXFamily labels an effect's family — the group of model variants that
// are interchangeable for the same job — so an agent can enumerate the
// alternatives to a pick instead of always taking the first model in a
// category. Families are finer than categories where one category holds many
// unrelated effect types (modulation, expression, dynamics).
func classifyFXFamily(f FX) string {
	switch f.Category {
	case "distortion":
		return "drive"
	case "dynamics":
		switch strings.ToLower(f.Name) {
		case "dyniii comp", "gray comp", "side comp":
			return "compressor"
		case "gate", "noise filter":
			return "noise gate"
		case "hold":
			return "hold"
		case "auto swell":
			return "swell"
		}
	case "eq":
		return "eq"
	case "expression":
		switch strings.ToLower(f.Name) {
		case "black wah", "more wah", "shine wah", "white bass wah":
			return "wah"
		case "wham", "chord wham":
			return "whammy"
		case "volume":
			return "volume"
		case "panner":
			return "panner"
		case "feedback":
			return "feedback"
		case "time warp":
			return "time warp"
		case "harm":
			return "harmonizer"
		}
	case "modulation":
		switch strings.ToLower(f.Name) {
		case "chorus", "multi chorus", "dim chorus", "detune":
			return "chorus"
		case "flanger", "air flanger":
			return "flanger"
		case "vibrato", "air vibrato":
			return "vibrato"
		case "tremolo":
			return "tremolo"
		case "rotary":
			return "rotary"
		case "stone phaser", "orange phaser", "vibe phaser", "tron phaser":
			return "phaser"
		case "ring mod":
			return "ring mod"
		case "stereo doubler":
			return "doubler"
		case "smart harm":
			return "harmonizer"
		case "octaves", "octaves up":
			return "octave"
		case "drop tune":
			return "pitch"
		case "env filter", "tron filter", "air filter":
			return "filter"
		}
	case "delay":
		return "delay"
	case "reverb":
		return "reverb"
	case "utility":
		return "utility"
	}
	return ""
}

// init derives the family label for every effect at package load, alongside
// classifyFXGain's drive-strength and classifyFXColor's character labels.
func init() {
	for i := range fx {
		fx[i].Family = classifyFXFamily(fx[i])
	}
}
