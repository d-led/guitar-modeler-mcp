package qc

import (
	"math"
	"testing"
)

func mustCatalog(t *testing.T) *Catalog {
	t.Helper()
	c, err := defaultCatalog()
	if err != nil {
		t.Fatalf("defaultCatalog: %v", err)
	}
	return c
}

func TestParseModelRepo(t *testing.T) {
	c := mustCatalog(t)

	if m, ok := c.Model(1001); !ok || m.Name != "Marshall JCM800" {
		t.Fatalf("model 1001 = %+v, want Marshall JCM800", m)
	}
	if m, ok := c.Model(1); !ok || m.Name != "Klon Centaur" {
		t.Fatalf("model 1 = %+v, want Klon Centaur", m)
	}

	if m, ok := c.Find("JCM800"); !ok || m.Name != "Marshall JCM800" {
		t.Fatalf("Find(JCM800) = %+v, want Marshall JCM800", m)
	}
	if m, ok := c.Find("Twin Reverb"); !ok || m.Name != "Fender Twin Reverb" {
		t.Fatalf("Find(Twin Reverb) = %+v, want Fender Twin Reverb", m)
	}
}

func TestResolveParamSynonyms(t *testing.T) {
	c := mustCatalog(t)
	twin, ok := c.Find("Fender Twin Reverb")
	if !ok {
		t.Fatal("Fender Twin Reverb not found")
	}

	// Exact names win, even when a synonym list would also apply.
	if spec, _, ok := twin.ResolveParam("TREBLE"); !ok || spec.Name != "TREBLE" {
		t.Errorf("exact TREBLE = %q, want TREBLE", spec.Name)
	}
	// The Fender Twin's gain knob is named VOLUME; MIDDLE is the catalog MID.
	if spec, _, ok := twin.ResolveParam("GAIN"); !ok || spec.Name != "VOLUME" {
		t.Errorf("GAIN → %q, want VOLUME", spec.Name)
	}
	if spec, _, ok := twin.ResolveParam("MIDDLE"); !ok || spec.Name != "MID" {
		t.Errorf("MIDDLE → %q, want MID", spec.Name)
	}
	// PRESENCE has no synonym fallback: it is genuinely absent on this amp.
	if _, _, ok := twin.ResolveParam("PRESENCE"); ok {
		t.Error("PRESENCE should not resolve on the Fender Twin Reverb")
	}

	// DRIVE is not the TS808's knob name — the synonym maps it to OVERDRIVE,
	// which is exactly the guess the agent makes and previously had to retry.
	ts, ok := c.Find("Ibanez TS808")
	if !ok {
		t.Fatal("Ibanez TS808 not found")
	}
	if spec, _, ok := ts.ResolveParam("DRIVE"); !ok || spec.Name != "OVERDRIVE" {
		t.Errorf("DRIVE → %q, want OVERDRIVE", spec.Name)
	}
}

func TestParseSkew(t *testing.T) {
	cases := []struct {
		raw  string
		want float64
		err  bool
	}{
		{"", 1.0, false},
		{"LIN_SKEW", 1.0, false},
		{"1", 1.0, false},
		{"1.0", 1.0, false},
		{"LOG_SKEW", 0.3, false},
		{"0.3", 0.3, false},
		{"4.93", 4.93, false},
		{" 0.4", 0.4, false},
		{"nonsense", 0, true},
		{"EXP_SKEW", 0, true},
		{"0", 0, true},
		{"-2", 0, true},
	}
	for _, c := range cases {
		got, err := parseSkew(c.raw)
		if c.err {
			if err == nil {
				t.Errorf("parseSkew(%q) = %g, want error", c.raw, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseSkew(%q): %v", c.raw, err)
			continue
		}
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("parseSkew(%q) = %g, want %g", c.raw, got, c.want)
		}
	}
}

func TestNormalizeLawOnRealModels(t *testing.T) {
	c := mustCatalog(t)

	// Low-High Cut HPF FREQ: 20..20000 Hz, skew 0.3. Wire 0.25 read 217 Hz on
	// the unit's screen (pyquadcortex hardware reading).
	lhc, _ := c.Model(4003)
	hp, _, ok := lhc.Param("HPF FREQ")
	if !ok {
		t.Fatal("Low-High Cut has no HPF FREQ")
	}
	real, err := hp.Denormalize(0.25)
	if err != nil {
		t.Fatalf("Denormalize: %v", err)
	}
	if math.Abs(real-217) > 1 {
		t.Fatalf("HPF FREQ at wire 0.25 = %g Hz, want 217", real)
	}

	// Env. Filter FREQ: 100..10000 Hz, LOG_SKEW (0.3). Wire 0.25 read 197 Hz.
	env, _ := c.Model(24003)
	freq, _, ok := env.Param("FREQ")
	if !ok {
		t.Fatal("Env. Filter has no FREQ")
	}
	real, err = freq.Denormalize(0.25)
	if err != nil {
		t.Fatalf("Denormalize: %v", err)
	}
	if math.Abs(real-197) > 1 {
		t.Fatalf("Env. Filter FREQ at wire 0.25 = %g Hz, want 197", real)
	}

	// A cab LEVEL with MIN_CABSIM_DB: -40..6 dB, skew 4.93. 0 dB sits at wire
	// 0.5, and converting 0 dB back must round-trip to 0.
	cab, _ := c.Model(12000) // Default Cabsim
	if cab == nil {
		t.Skip("no model 12000 in this catalog")
	}
	level, _, ok := cab.Param("LEVEL")
	if !ok {
		t.Fatal("Default Cabsim has no LEVEL")
	}
	wire, err := level.Normalize(0)
	if err != nil {
		t.Fatalf("Normalize(0): %v", err)
	}
	if math.Abs(wire-0.5) > 0.01 {
		t.Fatalf("cab LEVEL Normalize(0 dB) = %g, want ~0.5", wire)
	}
	back, err := level.Denormalize(0.5)
	if err != nil {
		t.Fatalf("Denormalize(0.5): %v", err)
	}
	if math.Abs(back) > 0.05 {
		t.Fatalf("cab LEVEL Denormalize(0.5) = %g dB, want 0", back)
	}
}

func TestLaneLevelLinearLaw(t *testing.T) {
	// Lane/mixer/splitter LEVEL family: -40..+12 dB, linear. 0 dB is unity at
	// wire 10/13 = 0.76923077 (pyquadcortex hardware reading).
	p := ParamSpec{Name: "VOLUME", Min: -40, Max: 12, Skew: 1}
	wire, err := p.Normalize(0)
	if err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if math.Abs(wire-0.76923077) > 1e-4 {
		t.Fatalf("VOLUME Normalize(0 dB) = %g, want 0.76923077", wire)
	}
	if real, _ := p.Denormalize(0.76923077); math.Abs(real) > 1e-4 {
		t.Fatalf("VOLUME Denormalize = %g dB, want 0", real)
	}
}

func TestUnmeasuredBoundRefusesConversion(t *testing.T) {
	xml := `<Models><Category id="20" name="Neural Capture Internal">` +
		`<Model id="20000" name="NC_Recorder">` +
		`<Parameter name="OUT LEVEL" type="float" min="MIN_INPUT_TRIM" max="MAX_INPUT_TRIM" defaultValue="0"/>` +
		`</Model></Category></Models>`
	c, err := parseModelRepo([]byte(xml))
	if err != nil {
		t.Fatalf("parseModelRepo: %v", err)
	}
	m, ok := c.Model(20000)
	if !ok {
		t.Fatal("NC_Recorder not parsed")
	}
	out := m.Params[0]
	if !out.unmeasured {
		t.Fatal("OUT LEVEL should be marked unmeasured")
	}
	if _, err := out.Normalize(0.5); err == nil {
		t.Fatal("Normalize on an unmeasured bound succeeded, want error")
	}
	if _, err := out.Denormalize(0.5); err == nil {
		t.Fatal("Denormalize on an unmeasured bound succeeded, want error")
	}
}

func TestOptionHelpers(t *testing.T) {
	p := ParamSpec{Name: "MODE", Steps: 3, StepNames: []string{"Normal", "Vibrato", "Vibrato Bright Off"}}
	if v, _ := p.OptionToValue(2); math.Abs(v-1.0) > 1e-9 {
		t.Fatalf("OptionToValue(2) = %g, want 1.0", v)
	}
	if v, _ := p.OptionToValue(1); math.Abs(v-0.5) > 1e-9 {
		t.Fatalf("OptionToValue(1) = %g, want 0.5", v)
	}
	if i, err := p.OptionName("vibrato"); err != nil || i != 1 {
		t.Fatalf("OptionName(vibrato) = %d, %v; want 1", i, err)
	}
}
