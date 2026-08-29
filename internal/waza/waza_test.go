package waza

import (
	"strings"
	"testing"
)

func TestDefaultDeviceShape(t *testing.T) {
	d := Default()
	if d.Name != "wazaair" || d.Display != "BOSS Waza Air" {
		t.Fatalf("device identity = %q/%q", d.Name, d.Display)
	}
	if !d.FileExchange || d.FileExt != ".tsl" {
		t.Fatalf("wazaair should exchange .tsl files, got %v/%q", d.FileExchange, d.FileExt)
	}
	if len(d.Amps) != 5 {
		t.Fatalf("amps = %d, want 5", len(d.Amps))
	}
	if len(d.Boosters) != 20 {
		t.Fatalf("boosters = %d, want 20", len(d.Boosters))
	}
	if len(d.ModFX) != 16 {
		t.Fatalf("mod/fx = %d, want 16", len(d.ModFX))
	}
	if len(d.Delays) != 3 || len(d.Reverbs) != 3 {
		t.Fatalf("delays/reverbs = %d/%d, want 3/3", len(d.Delays), len(d.Reverbs))
	}
}

func TestResolveAmpByNameAndInspiredBy(t *testing.T) {
	d := Default()

	s, err := d.Resolve(Spec{Amp: "BROWN"})
	if err != nil || s.Amp != "BROWN" {
		t.Fatalf("Resolve(BROWN) = %q, %v", s.Amp, err)
	}

	s, err = d.Resolve(Spec{Amp: "Twin Reverb"})
	if err != nil || s.Amp != "CLEAN" {
		t.Fatalf("Resolve(Twin Reverb) = %q, %v; want CLEAN", s.Amp, err)
	}

	s, err = d.Resolve(Spec{Amp: "SLO-100"})
	if err != nil || s.Amp != "BROWN" {
		t.Fatalf("Resolve(SLO-100) = %q, %v; want BROWN", s.Amp, err)
	}

	s, err = d.Resolve(Spec{Amp: "5150"})
	if err != nil || s.Amp != "LEAD" {
		t.Fatalf("Resolve(5150) = %q, %v; want LEAD", s.Amp, err)
	}
}

func TestResolveBoosterByInspiredBy(t *testing.T) {
	d := Default()
	s, err := d.Resolve(Spec{Booster: "TS-808"})
	if err != nil || s.Booster != "T-SCREAM" {
		t.Fatalf("Resolve(TS-808) = %q, %v; want T-SCREAM", s.Booster, err)
	}
	s, err = d.Resolve(Spec{Booster: "BD-2"})
	if err != nil || s.Booster != "BLUES DRIVE" {
		t.Fatalf("Resolve(BD-2) = %q, %v; want BLUES DRIVE", s.Booster, err)
	}
}

func TestResolveUnknownAmp(t *testing.T) {
	d := Default()
	if _, err := d.Resolve(Spec{Amp: "not an amp"}); err == nil {
		t.Fatal("expected an error for an unknown amp")
	}
}

func TestSetupCardHTML(t *testing.T) {
	d := Default()
	s, err := d.Resolve(Spec{
		Name:         "Brown Practice",
		Amp:          "BROWN",
		Booster:      "T-SCREAM",
		Mod:          "CHORUS",
		Delay:        "TAPE ECHO",
		Reverb:       "HALL REVERB",
		CabResonance: "VINTAGE",
		Ambience:     "STAGE",
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	html := d.SetupCardHTML(s)
	for _, want := range []string{"Brown Practice", "BOSS Waza Air", "BROWN", "T-SCREAM", "CHORUS", "TAPE ECHO", "HALL REVERB", "VINTAGE", "STAGE"} {
		if !strings.Contains(html, want) {
			t.Fatalf("setup card missing %q:\n%s", want, html)
		}
	}
}
