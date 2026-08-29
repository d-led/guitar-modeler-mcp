package catalog

import "strings"

// classifyGain labels an amp's drive character so an agent can match the amp to
// the song: clean tones need a "clean" amp, lead tones a "crunch" or "high
// gain" one. The label is derived from the amp's channel, description, real
// model and style — the same signals a human would read off the device.
func classifyGain(a Amp) string {
	if a.Bass {
		return "bass"
	}

	channel := strings.ToLower(a.Channel)
	real := strings.ToLower(a.RealModel)
	desc := strings.ToLower(a.Description)
	name := strings.ToLower(a.Model)

	// Explicit, strongest signals first.
	switch {
	case strings.Contains(channel, "clean") || strings.Contains(desc, "clean"):
		return "clean"
	case strings.Contains(channel, "lead"), strings.Contains(channel, "drive"),
		strings.Contains(channel, "overdrive"), strings.Contains(channel, "red"),
		strings.Contains(name, "lead"):
		return "high gain"
	case strings.Contains(channel, "crunch"), strings.Contains(channel, "raw"),
		strings.Contains(channel, "vintage"), strings.Contains(channel, "blue"),
		strings.Contains(channel, "green"), strings.Contains(channel, "orange"):
		return "crunch"
	case strings.Contains(desc, "high-gain"), strings.Contains(desc, "high gain"):
		return "high gain"
	case strings.Contains(real, "jcm800"), strings.Contains(real, "jcm 800"):
		return "high gain"
	case strings.Contains(real, "plexi"), strings.Contains(real, "super lead"),
		strings.Contains(real, "jtm"):
		return "crunch"
	}

	// Genre fallbacks.
	for _, s := range a.Style {
		switch strings.ToLower(s) {
		case "clean", "ambient":
			return "clean"
		case "metal", "high-gain":
			return "high gain"
		case "crunch":
			return "crunch"
		}
	}

	return "edge of breakup"
}

// init derives the gain label for every amp at package load.
func init() {
	for i := range amps {
		amps[i].Gain = classifyGain(amps[i])
	}
}
