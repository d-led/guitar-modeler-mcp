package catalog

import "strings"

// fxColorByName maps each time-based effect to its character label: a flat
// table so the classifier stays trivial and the labels are readable at a
// glance.
var fxColorByName = map[string]string{
	"dyn delay":     "clean",
	"air delay":     "atmospheric",
	"bbd delay":     "analog",
	"tape echo":     "tape",
	"pitch delay":   "pitch",
	"reso delay":    "resonant",
	"reverse delay": "reverse",
	"air reverb":    "hall",
	"ambi verb":     "ambient",
	"eleven reverb": "room",
	"party verb":    "modulated",
	"spring reverb": "spring",
	"shimmer":       "shimmer",
}

// classifyFXColor labels a time-based effect's character so an agent can pick
// the right flavour for the part, mirroring classifyFXGain for drives. Only
// delay and reverb effects carry a label; everything else returns "".
func classifyFXColor(f FX) string {
	return fxColorByName[strings.ToLower(f.Name)]
}

// init derives the character label for every time-based effect at package
// load, the same way classifyFXGain derives the drive-strength label.
func init() {
	for i := range fx {
		fx[i].Character = classifyFXColor(fx[i])
	}
}
