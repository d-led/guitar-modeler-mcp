package catalog

import "testing"

func TestClassifyFXColor(t *testing.T) {
	cases := map[string]string{
		"Dyn Delay":     "clean",
		"AIR Delay":     "atmospheric",
		"BBD Delay":     "analog",
		"Tape Echo":     "tape",
		"Pitch Delay":   "pitch",
		"Reso Delay":    "resonant",
		"Reverse Delay": "reverse",
		"AIR Reverb":    "hall",
		"Ambi Verb":     "ambient",
		"Eleven Reverb": "room",
		"Party Verb":    "modulated",
		"Spring Reverb": "spring",
		"Shimmer":       "shimmer",
		"Green JRC-OD":  "", // non-time-based effects carry no label
		"Graphic EQ":    "",
	}

	c := New()
	for name, want := range cases {
		f, ok := c.FXByName(name)
		if !ok {
			t.Fatalf("effect %q not found", name)
		}
		if f.Character != want {
			t.Errorf("effect %q character = %q, want %q", name, f.Character, want)
		}
	}
}

func TestSearchDigitalDelayPrefersCleanDelay(t *testing.T) {
	c := New()
	results := c.Search("digital delay", "fx")
	if len(results) == 0 {
		t.Fatal("expected delay results for 'digital delay'")
	}
	// AIR Delay is atmospheric, not a clean digital delay; it must not win.
	if results[0].Name == "AIR Delay" {
		t.Fatalf("top result = %q, want a clean digital delay (Dyn Delay)", results[0].Name)
	}
	if results[0].Name != "Dyn Delay" {
		t.Fatalf("top result = %q, want Dyn Delay", results[0].Name)
	}
}
