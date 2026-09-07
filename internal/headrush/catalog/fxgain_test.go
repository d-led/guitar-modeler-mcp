package catalog

import "testing"

func TestClassifyFXGain(t *testing.T) {
	cases := map[string]string{
		"White Boost":   "boost",
		"Green JRC-OD":  "overdrive",
		"D250 Drive":    "overdrive",
		"S1 Drive":      "overdrive",
		"K Drive":       "distortion",
		"DC Distort":    "distortion",
		"D1 Dist":       "distortion",
		"MX Dist":       "distortion",
		"Black OP":      "distortion",
		"B Dist 7000":   "bass",
		"Tri Fuzz":      "fuzz",
		"Round Fuzz":    "fuzz",
		"8-Bit Crush":   "bitcrusher",
		"Tape Echo":     "", // non-drive effects carry no label
		"Orange Phaser": "",
	}

	c := New()
	for name, want := range cases {
		f, ok := c.FXByName(name)
		if !ok {
			t.Fatalf("effect %q not found", name)
		}
		if f.Gain != want {
			t.Errorf("effect %q gain = %q, want %q", name, f.Gain, want)
		}
	}
}
