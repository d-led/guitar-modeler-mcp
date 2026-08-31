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
	assertResolvesTo(t, twin, "TREBLE", "TREBLE")
	// The Fender Twin's gain knob is named VOLUME; MIDDLE is the catalog MID.
	assertResolvesTo(t, twin, "GAIN", "VOLUME")
	assertResolvesTo(t, twin, "MIDDLE", "MID")
	// PRESENCE has no synonym fallback: it is genuinely absent on this amp.
	assertNoResolve(t, twin, "PRESENCE")

	// DRIVE is not the TS808's knob name — the synonym maps it to OVERDRIVE,
	// which is exactly the guess the agent makes and previously had to retry.
	ts, ok := c.Find("Ibanez TS808")
	if !ok {
		t.Fatal("Ibanez TS808 not found")
	}
	assertResolvesTo(t, ts, "DRIVE", "OVERDRIVE")
}

// assertResolvesTo fails unless the model resolves param to the named knob.
func assertResolvesTo(t *testing.T, m *ModelSpec, param, want string) {
	t.Helper()
	spec, _, ok := m.ResolveParam(param)
	if !ok {
		t.Fatalf("%s: %q did not resolve, want %q", m.Name, param, want)
	}
	if spec.Name != want {
		t.Fatalf("%s: %q → %q, want %q", m.Name, param, spec.Name, want)
	}
}

// assertNoResolve fails when the model resolves param to any knob.
func assertNoResolve(t *testing.T, m *ModelSpec, param string) {
	t.Helper()
	if _, _, ok := m.ResolveParam(param); ok {
		t.Fatalf("%s: %q should not resolve", m.Name, param)
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
	hp := mustParam(t, lhc, "HPF FREQ")
	real, err := hp.Denormalize(0.25)
	if err != nil {
		t.Fatalf("Denormalize: %v", err)
	}
	assertNear(t, "HPF FREQ at wire 0.25", real, 217, 1)

	// Env. Filter FREQ: 100..10000 Hz, LOG_SKEW (0.3). Wire 0.25 read 197 Hz.
	env, _ := c.Model(24003)
	freq := mustParam(t, env, "FREQ")
	real, err = freq.Denormalize(0.25)
	if err != nil {
		t.Fatalf("Denormalize: %v", err)
	}
	assertNear(t, "Env. Filter FREQ at wire 0.25", real, 197, 1)

	// A cab LEVEL with MIN_CABSIM_DB: -40..6 dB, skew 4.93. 0 dB sits at wire
	// 0.5, and converting 0 dB back must round-trip to 0.
	cab, _ := c.Model(12000) // Default Cabsim
	if cab == nil {
		t.Skip("no model 12000 in this catalog")
	}
	level := mustParam(t, cab, "LEVEL")
	wire, err := level.Normalize(0)
	if err != nil {
		t.Fatalf("Normalize(0): %v", err)
	}
	assertNear(t, "cab LEVEL Normalize(0 dB)", wire, 0.5, 0.01)
	back, err := level.Denormalize(0.5)
	if err != nil {
		t.Fatalf("Denormalize(0.5): %v", err)
	}
	assertNear(t, "cab LEVEL Denormalize(0.5)", back, 0, 0.05)
}

// mustParam returns the named parameter of a model, failing when absent.
func mustParam(t *testing.T, m *ModelSpec, name string) ParamSpec {
	t.Helper()
	if m == nil {
		t.Fatalf("model not found")
	}
	p, _, ok := m.Param(name)
	if !ok {
		t.Fatalf("%s has no %s", m.Name, name)
	}
	return p
}

// assertNear fails unless got is within tol of want.
func assertNear(t *testing.T, name string, got, want, tol float64) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Fatalf("%s = %g, want %g (±%g)", name, got, want, tol)
	}
}

// wantEq fails when got differs from want, naming the checked field.
func wantEq[T comparable](t *testing.T, name string, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %v, want %v", name, got, want)
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
