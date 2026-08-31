package catalog

import "strings"

// classifyFXGain labels a distortion effect's drive strength so an agent can
// match the pedal to the part, mirroring classifyGain for amps. Only
// distortion-category effects carry a label; everything else returns "".
func classifyFXGain(f FX) string {
	if f.Category != "distortion" {
		return ""
	}
	switch strings.ToLower(f.Name) {
	case "white boost":
		return "boost"
	case "green jrc-od", "d250 drive", "glorious drive", "s1 drive", "b2 drive":
		return "overdrive"
	case "k drive", "dc distort", "d1 dist", "mx dist", "anxiety od", "anxiety od v2", "black op":
		return "distortion"
	case "b dist 7000":
		return "bass"
	case "tri fuzz", "round fuzz", "oct fuzz":
		return "fuzz"
	case "8-bit crush":
		return "bitcrusher"
	default:
		return "overdrive"
	}
}

// init derives the drive-strength label for every effect at package load, the
// same way classifyGain does for amps.
func init() {
	for i := range fx {
		fx[i].Gain = classifyFXGain(fx[i])
	}
}
