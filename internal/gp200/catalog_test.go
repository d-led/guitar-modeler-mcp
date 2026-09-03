package gp200

import (
	"strings"
	"testing"
)

func TestCatalogShape(t *testing.T) {
	if got := len(Effects()); got != 305 {
		t.Fatalf("Effects() = %d, want 305", got)
	}
	if got := len(SlotModules()); got != 11 {
		t.Fatalf("SlotModules() = %d, want 11", got)
	}
	if got := len(Amps()); got != 76 {
		t.Fatalf("Amps() = %d, want 76", got)
	}
	if got := len(Cabs()); got != 90 {
		t.Fatalf("Cabs() = %d, want 90", got)
	}
}

func TestEffectCodeRoundTrip(t *testing.T) {
	for _, name := range []string{"COMP", "Green OD", "UK 800", "Dark Twin", "Room", "Pure", "V-Wah", "Hyper EQ", "Volume", "O-Phase"} {
		code, ok := EffectCode(name)
		if !ok {
			t.Fatalf("EffectCode(%q) not found", name)
		}
		e, ok := EffectByCode(code)
		if !ok {
			t.Fatalf("EffectByCode(%#x) not found", code)
		}
		if e.Name != name {
			t.Fatalf("EffectByCode(%#x).Name = %q, want %q", code, e.Name, name)
		}
	}
}

func TestInspiredBy(t *testing.T) {
	for _, tc := range []struct{ name, want string }{
		{"UK 800", "Marshall® JCM800"},
		{"Green OD", "Ibanez® TS-808 Tube Screamer"},
		{"Dark Twin", "Fender® '65 Twin Reverb"},
		{"Plate", "EMT 140 plate reverb"},
		{"V-Wah", "VOX® V846"},
		{"Penesas", "Klon® Centaur (Gold Overdrive)"},
	} {
		if got := InspiredBy(tc.name); got != tc.want {
			t.Fatalf("InspiredBy(%q) = %q, want %q", tc.name, got, tc.want)
		}
	}
}

func TestAmbiguousNames(t *testing.T) {
	// "Tube" is both a DST overdrive and a DLY delay; the slot hint disambiguates.
	if got := len(CodesForName("Tube")); got != 2 {
		t.Fatalf("CodesForName(Tube) = %d codes, want 2", got)
	}
	// User IR spans 20 cab slots.
	if got := len(CodesForName("User IR")); got != 20 {
		t.Fatalf("CodesForName(User IR) = %d codes, want 20", got)
	}
	// SnapTone spans 10 NAM capture slots.
	if got := len(CodesForName("SnapTone")); got != 10 {
		t.Fatalf("CodesForName(SnapTone) = %d codes, want 10", got)
	}
}

func TestDefaultParams(t *testing.T) {
	// COMP: Sustain 20, Volume 50.
	d := DefaultParams(0)
	if d[0] != 20 || d[1] != 50 {
		t.Fatalf("COMP defaults = %v, want Sustain 20 / Volume 50", d[:2])
	}
	// Slapback (184549381): Mix 20, Time 150, Feedback 50.
	d = DefaultParams(184549381)
	if d[0] != 20 || d[1] != 150 || d[2] != 50 {
		t.Fatalf("Slapback defaults = %v", d[:3])
	}
}

func TestParamNames(t *testing.T) {
	names := ParamNames(16777289) // Hammy
	if names[0] != "Range" || names[3] != "Position" {
		t.Fatalf("Hammy param names = %v", names)
	}
}

func TestSetParam(t *testing.T) {
	b := Block{EffectID: 0} // COMP
	if err := SetParam(&b, "sustain", 80); err != nil {
		t.Fatalf("SetParam: %v", err)
	}
	if b.Params[0] != 80 {
		t.Fatalf("sustain = %v, want 80", b.Params[0])
	}
	if err := SetParam(&b, "does-not-exist", 1); err == nil {
		t.Fatal("SetParam accepted an unknown parameter name")
	}
}

func TestParamsKnob(t *testing.T) {
	// Green OD is a three-knob Tube Screamer model.
	ps := Params(50331648)
	if len(ps) != 3 {
		t.Fatalf("Green OD has %d params, want 3", len(ps))
	}
	first := ps[0]
	if first.Name != "Gain" || first.Kind != "knob" || first.Min != 0 || first.Max != 100 {
		t.Fatalf("Green OD param 0 = %+v", first)
	}
}

func TestParamsSwitchOptions(t *testing.T) {
	// A switch/combox carries option names, not a numeric range.
	ps := Params(67108864) // A-Chorus
	for _, p := range ps {
		if p.Name == "Sync" {
			if p.Kind != "switch" || len(p.Options) != 2 || p.Options[1] != "ON" {
				t.Fatalf("A-Chorus Sync = %+v", p)
			}
			return
		}
	}
	t.Fatal("A-Chorus has no Sync parameter")
}

func TestApplyNamedParams(t *testing.T) {
	b := Block{EffectID: 50331648} // Green OD: Gain, Tone, Volume
	rejected := ApplyNamedParams(&b, map[string]float32{"gain": 60, "level": 78})
	if len(rejected) != 1 || !strings.Contains(rejected[0], "level") {
		t.Fatalf("rejected = %v, want a message about level", rejected)
	}
	if b.Params[0] != 60 {
		t.Fatalf("gain = %v, want 60", b.Params[0])
	}
}

func TestSetParamRejectsOutOfRange(t *testing.T) {
	// A-Chorus Rate is 0.1..10 Hz; 15 Hz would sound like noise.
	b := Block{EffectID: 67108864}
	if err := SetParam(&b, "rate", 15); err == nil {
		t.Fatal("SetParam accepted a rate outside 0.1..10 Hz")
	}
	if err := SetParam(&b, "rate", 0.5); err != nil {
		t.Fatalf("SetParam rejected an in-range rate: %v", err)
	}
}
