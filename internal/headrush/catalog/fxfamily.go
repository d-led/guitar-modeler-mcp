package catalog

import "strings"

// fxFamilyByName maps the modules whose family is finer than their category —
// dynamics, expression and modulation hold several unrelated effect kinds.
var fxFamilyByName = map[string]string{
	"dyniii comp":    "compressor",
	"gray comp":      "compressor",
	"side comp":      "compressor",
	"gate":           "noise gate",
	"noise filter":   "noise gate",
	"hold":           "hold",
	"auto swell":     "swell",
	"black wah":      "wah",
	"more wah":       "wah",
	"shine wah":      "wah",
	"white bass wah": "wah",
	"wham":           "whammy",
	"chord wham":     "whammy",
	"volume":         "volume",
	"panner":         "panner",
	"feedback":       "feedback",
	"time warp":      "time warp",
	"harm":           "harmonizer",
	"chorus":         "chorus",
	"multi chorus":   "chorus",
	"dim chorus":     "chorus",
	"detune":         "chorus",
	"flanger":        "flanger",
	"air flanger":    "flanger",
	"vibrato":        "vibrato",
	"air vibrato":    "vibrato",
	"tremolo":        "tremolo",
	"rotary":         "rotary",
	"stone phaser":   "phaser",
	"orange phaser":  "phaser",
	"vibe phaser":    "phaser",
	"tron phaser":    "phaser",
	"ring mod":       "ring mod",
	"stereo doubler": "doubler",
	"smart harm":     "harmonizer",
	"octaves":        "octave",
	"octaves up":     "octave",
	"drop tune":      "pitch",
	"env filter":     "filter",
	"tron filter":    "filter",
	"air filter":     "filter",
}

// fxFamilyByCategory maps the categories that are a single family: every drive
// is a "drive", every EQ an "eq", and so on.
var fxFamilyByCategory = map[string]string{
	"distortion": "drive",
	"eq":         "eq",
	"delay":      "delay",
	"reverb":     "reverb",
	"utility":    "utility",
}

// classifyFXFamily labels an effect's family — the group of model variants that
// are interchangeable for the same job — so an agent can enumerate the
// alternatives to a pick instead of always taking the first model in a
// category. Families are finer than categories where one category holds many
// unrelated effect types (modulation, expression, dynamics).
func classifyFXFamily(f FX) string {
	if fam, ok := fxFamilyByName[strings.ToLower(f.Name)]; ok {
		return fam
	}
	return fxFamilyByCategory[f.Category]
}

// init derives the family label for every effect at package load, alongside
// classifyFXGain's drive-strength and classifyFXColor's character labels.
func init() {
	for i := range fx {
		fx[i].Family = classifyFXFamily(fx[i])
	}
}
