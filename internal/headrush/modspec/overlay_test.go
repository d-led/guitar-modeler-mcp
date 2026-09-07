package modspec

import "testing"

func TestOverlayMarksSyncParams(t *testing.T) {
	m, ok := Get("Tape Echo")
	if !ok {
		t.Fatal("Tape Echo missing")
	}
	delay := m["Delay"]
	if delay.Kind != "sync" {
		t.Fatalf("Tape Echo.Delay kind = %q, want sync", delay.Kind)
	}
	// A note value must be accepted.
	if err := delay.Validate("1/4"); err != nil {
		t.Fatalf("Delay \"1/4\" should be valid: %v", err)
	}
	// A number in range must be accepted.
	if err := delay.Validate(300.0); err != nil {
		t.Fatalf("Delay 300 should be valid: %v", err)
	}
	// A bad note value must be rejected.
	if err := delay.Validate("Bogus"); err == nil {
		t.Fatal("Delay \"Bogus\" should be rejected")
	}
}

func TestOverlayAddsAmpTremSpeed2(t *testing.T) {
	m, ok := Get("Amp")
	if !ok {
		t.Fatal("Amp missing")
	}
	if _, ok := m["TremSpeed2"]; !ok {
		t.Fatal("Amp.TremSpeed2 missing")
	}
	if m["TremSpeed2"].Kind != "sync" {
		t.Fatalf("TremSpeed2 kind = %q, want sync", m["TremSpeed2"].Kind)
	}
}

func TestOverlayFixesEditorRangeBugs(t *testing.T) {
	if m, _ := Get("8-Bit Crush"); *m["AntiAliasing"].Max != 100 {
		t.Fatalf("AntiAliasing max = %v, want 100", *m["AntiAliasing"].Max)
	}
	if m, _ := Get("Flanger"); *m["Feedback"].Min != -100 {
		t.Fatalf("Flanger.Feedback min = %v, want -100", *m["Feedback"].Min)
	}
	if m, _ := Get("Tron Filter"); *m["Reso"].Max != 100 {
		t.Fatalf("Tron Filter.Reso max = %v, want 100", *m["Reso"].Max)
	}
}

func TestOverlayAddsMissingSetValue(t *testing.T) {
	m, _ := Get("AIR Reverb")
	found := false
	for _, v := range m["Mode"].Values {
		if v == "NonLinear" {
			found = true
		}
	}
	if !found {
		t.Fatal("AIR Reverb.Mode missing NonLinear")
	}
}

// TestAllParamsHaveDescriptions guards the parameter dictionary: every parameter
// exposed by the spec must have a plain-language description.
func TestAllParamsHaveDescriptions(t *testing.T) {
	missing := map[string][]string{}
	for _, name := range Modules() {
		m, _ := Get(name)
		for key, p := range m {
			if p.Description == "" {
				missing[name] = append(missing[name], key)
			}
		}
	}
	if len(missing) > 0 {
		for name, keys := range missing {
			t.Errorf("module %q: params without description: %v", name, keys)
		}
	}
}
