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

	// Explicit, strongest signals first, kept in priority order so the table
	// reads top-to-bottom the way a human reads the amp.
	signals := []struct {
		field string
		words []string
		class string
	}{
		{channel, []string{"clean"}, "clean"},
		{desc, []string{"clean"}, "clean"},
		{channel, []string{"lead", "drive", "overdrive", "red"}, "high gain"},
		{name, []string{"lead"}, "high gain"},
		{channel, []string{"crunch", "raw", "vintage", "blue", "green", "orange"}, "crunch"},
		{desc, []string{"high-gain", "high gain"}, "high gain"},
		{real, []string{"jcm800", "jcm 800"}, "high gain"},
		{real, []string{"plexi", "super lead", "jtm"}, "crunch"},
	}
	for _, s := range signals {
		if containsAny(s.field, s.words...) {
			return s.class
		}
	}

	return gainFromStyle(a.Style)
}

func gainFromStyle(styles []string) string {
	for _, s := range styles {
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

// containsAny reports whether haystack contains any of the needles.
func containsAny(haystack string, needles ...string) bool {
	for _, n := range needles {
		if strings.Contains(haystack, n) {
			return true
		}
	}
	return false
}

// init derives the gain label for every amp at package load.
func init() {
	for i := range amps {
		amps[i].Gain = classifyGain(amps[i])
	}
}
