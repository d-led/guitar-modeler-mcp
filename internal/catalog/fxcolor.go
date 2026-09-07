package catalog

import "strings"

// classifyFXColor labels a time-based effect's character so an agent can pick
// the right flavour for the part, mirroring classifyFXGain for drives. Only
// delay and reverb effects carry a label; everything else returns "".
func classifyFXColor(f FX) string {
	switch f.Category {
	case "delay":
		switch strings.ToLower(f.Name) {
		case "dyn delay":
			return "clean"
		case "air delay":
			return "atmospheric"
		case "bbd delay":
			return "analog"
		case "tape echo":
			return "tape"
		case "pitch delay":
			return "pitch"
		case "reso delay":
			return "resonant"
		case "reverse delay":
			return "reverse"
		}
	case "reverb":
		switch strings.ToLower(f.Name) {
		case "air reverb":
			return "hall"
		case "ambi verb":
			return "ambient"
		case "eleven reverb":
			return "room"
		case "party verb":
			return "modulated"
		case "spring reverb":
			return "spring"
		case "shimmer":
			return "shimmer"
		}
	}
	return ""
}

// init derives the character label for every time-based effect at package
// load, the same way classifyFXGain derives the drive-strength label.
func init() {
	for i := range fx {
		fx[i].Color = classifyFXColor(fx[i])
	}
}
