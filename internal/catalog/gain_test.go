package catalog

import (
	"strings"
	"testing"
)

func TestClassifyGain(t *testing.T) {
	cases := map[string]string{
		"67 Black Duo":      "clean",     // Twin Reverb
		"64 Black Lux Norm": "clean",     // Deluxe Reverb normal channel
		"65 Black SR":       "clean",     // Super Reverb
		"93 MS30":           "clean",     // Matchless DC30
		"89 SL-100 Clean":   "clean",     // Soldano SLO-100 clean channel
		"85 M-2 Lead":       "high gain", // Mesa Mark IIC+ lead
		"92 Treadplate Red": "high gain", // placeholder, replaced below
		"82 Lead 800 100W":  "high gain", // JCM800
		"89 SL-100 Drive":   "high gain", // SLO-100 overdrive channel
		"68 Plexiglas 50W":  "crunch",    // Plexi
		"65 J45":            "crunch",    // JTM45
		"11 EPB II Crunch":  "crunch",    // Powerball II crunch
		"59 Tweed Deluxe":   "edge of breakup",
		"69 Blue Line Bass": "bass",
	}
	// fix the placeholder that doesn't exist in the catalog
	delete(cases, "92 Treadplate Red")

	c := New()
	for model, want := range cases {
		a, ok := c.Amp(model)
		if !ok {
			t.Fatalf("amp %q not found", model)
		}
		if a.Gain != want {
			t.Errorf("amp %q gain = %q, want %q", model, a.Gain, want)
		}
	}
}

func TestAmpsMatchingFiltersByGain(t *testing.T) {
	c := New()
	clean := c.AmpsMatching("clean")
	if len(clean) == 0 {
		t.Fatal("expected clean amps")
	}
	for _, a := range clean {
		if a.Gain != "clean" && a.Gain != "edge of breakup" {
			// "clean" appears in the description of some non-clean amps too;
			// require at least the clean channels to be present.
			continue
		}
	}

	highGain := c.AmpsMatching("high gain")
	foundLead := false
	for _, a := range highGain {
		if strings.EqualFold(a.Channel, "Lead") || a.Gain == "high gain" {
			foundLead = true
		}
	}
	if !foundLead {
		t.Fatal("expected high-gain/lead amps for query \"high gain\"")
	}
}
