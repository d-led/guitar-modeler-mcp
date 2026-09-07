package catalog

import "testing"

func TestClassifyFXFamily(t *testing.T) {
	cases := map[string]string{
		"Chorus":        "chorus",
		"Multi Chorus":  "chorus",
		"Dim Chorus":    "chorus",
		"Detune":        "chorus",
		"Flanger":       "flanger",
		"AIR Flanger":   "flanger",
		"Stone Phaser":  "phaser",
		"Vibe Phaser":   "phaser",
		"Vibrato":       "vibrato",
		"Tremolo":       "tremolo",
		"Rotary":        "rotary",
		"Octaves":       "octave",
		"Octaves Up":    "octave",
		"Drop Tune":     "pitch",
		"Smart Harm":    "harmonizer",
		"Harm":          "harmonizer",
		"Env Filter":    "filter",
		"Tron Filter":   "filter",
		"AIR Filter":    "filter",
		"Black Wah":     "wah",
		"More Wah":      "wah",
		"Wham":          "whammy",
		"Chord Wham":    "whammy",
		"Volume":        "volume",
		"Panner":        "panner",
		"DynIII Comp":   "compressor",
		"Gray Comp":     "compressor",
		"Side Comp":     "compressor",
		"Gate":          "noise gate",
		"Noise Filter":  "noise gate",
		"Graphic EQ":    "eq",
		"Para EQ":       "eq",
		"Green JRC-OD":  "drive",
		"White Boost":   "drive",
		"Tape Echo":     "delay",
		"AIR Delay":     "delay",
		"Eleven Reverb": "reverb",
		"IR":            "utility",
	}

	c := New()
	for name, want := range cases {
		f, ok := c.FXByName(name)
		if !ok {
			t.Fatalf("effect %q not found", name)
		}
		if f.Family != want {
			t.Errorf("effect %q family = %q, want %q", name, f.Family, want)
		}
	}
}

func TestVariantsOfExcludesSelfAndGroupsFamily(t *testing.T) {
	c := New()

	variants := c.VariantsOf("Chorus")
	want := map[string]bool{"Multi Chorus": true, "Dim Chorus": true, "Detune": true}
	if len(variants) != len(want) {
		t.Fatalf("Chorus variants = %v, want %v", namesOf(variants), want)
	}
	for _, v := range variants {
		if !want[v.Name] {
			t.Fatalf("unexpected Chorus variant %q", v.Name)
		}
	}

	// A family with no alternatives yields an empty (non-nil) slice.
	if got := c.VariantsOf("Volume"); got == nil || len(got) != 0 {
		t.Fatalf("Volume variants = %v, want empty slice", got)
	}
}

func TestFXByFamily(t *testing.T) {
	c := New()
	if got := c.FXByFamily("chorus"); len(got) != 4 {
		t.Fatalf("chorus family = %v, want 4 models", namesOf(got))
	}
	if got := c.FXByFamily("bogus"); len(got) != 0 {
		t.Fatalf("unknown family = %v, want empty", got)
	}
}

func namesOf(fx []FX) []string {
	out := make([]string, len(fx))
	for i, f := range fx {
		out[i] = f.Name
	}
	return out
}
