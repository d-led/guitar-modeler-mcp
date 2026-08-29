package waza

import (
	"strings"
	"testing"
)

func TestDefaultAirStepShape(t *testing.T) {
	a := DefaultAirStep()
	if a.Name != "airstep-bw" || a.Display != "XSONIC AIRSTEP BW Edition" {
		t.Fatalf("identity = %q/%q", a.Name, a.Display)
	}
	if len(a.Switches) != 5 {
		t.Fatalf("switches = %d, want 5", len(a.Switches))
	}
	if len(a.Modes) != 4 {
		t.Fatalf("modes = %d, want 4", len(a.Modes))
	}
	for i, m := range a.Modes {
		if m.Number != i+1 {
			t.Fatalf("mode %d has Number %d", i, m.Number)
		}
		if len(m.Bindings) != 5 {
			t.Fatalf("mode %d has %d bindings, want 5", m.Number, len(m.Bindings))
		}
	}
}

func TestAirStepModeBindings(t *testing.T) {
	a := DefaultAirStep()

	m1, ok := a.Mode(1)
	if !ok {
		t.Fatal("mode 1 missing")
	}
	press := presses(m1)
	if press["A"] != "Toggle BOOSTER" || press["B"] != "Toggle FX" || press["E"] != "CH 3/6" {
		t.Fatalf("mode 1 presses = %v", press)
	}
	if long(m1, "C") != "Select CH 1-3" || long(m1, "D") != "Select CH 4-6" {
		t.Fatalf("mode 1 long presses = %v", m1)
	}

	m3, ok := a.Mode(3)
	if !ok {
		t.Fatal("mode 3 missing")
	}
	p3 := presses(m3)
	if p3["D"] != "Toggle DELAY" || p3["E"] != "Toggle REVERB & DELAY2" {
		t.Fatalf("mode 3 presses = %v", p3)
	}
	if long(m3, "A") != "CH 1" || long(m3, "E") != "CH 5" {
		t.Fatalf("mode 3 long presses = %v", m3)
	}

	m4, ok := a.Mode(4)
	if !ok {
		t.Fatal("mode 4 missing")
	}
	if presses(m4)["D"] != "CH 4" || long(m4, "E") != "CH 6" {
		t.Fatalf("mode 4 bindings = %v", m4)
	}
	if long(m4, "A") != "" {
		t.Fatalf("mode 4 FS A should have no long press, got %q", long(m4, "A"))
	}
}

func TestAirStepModeOutOfRange(t *testing.T) {
	a := DefaultAirStep()
	if _, ok := a.Mode(0); ok {
		t.Fatal("Mode(0) should not exist")
	}
	if _, ok := a.Mode(5); ok {
		t.Fatal("Mode(5) should not exist")
	}
}

func TestSetupCardHTMLWithAirStep(t *testing.T) {
	d := Default()
	s, err := d.Resolve(Spec{Name: "Scene Card", Amp: "BROWN", Booster: "T-SCREAM"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	a := DefaultAirStep()
	mode, _ := a.Mode(3)

	html := d.SetupCardHTMLWithAirStep(s, mode)
	for _, want := range []string{"AIRSTEP BW — Mode 3", "Toggle REVERB &amp; DELAY2", "CH 4"} {
		if !strings.Contains(html, want) {
			t.Fatalf("setup card missing %q", want)
		}
	}

	// Without a mode, the card must not mention the foot controller.
	plain := d.SetupCardHTML(s)
	if strings.Contains(plain, "AIRSTEP BW") {
		t.Fatal("plain setup card should not include the AIRSTEP section")
	}
}

func presses(m AirStepMode) map[string]string {
	out := make(map[string]string, len(m.Bindings))
	for _, bi := range m.Bindings {
		out[bi.Switch] = bi.Press
	}
	return out
}

func long(m AirStepMode, sw string) string {
	for _, bi := range m.Bindings {
		if bi.Switch == sw {
			return bi.LongPress
		}
	}
	return ""
}
