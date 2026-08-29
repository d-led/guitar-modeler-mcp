package thr

import (
	"strings"
	"testing"
)

func TestDefaultDeviceShape(t *testing.T) {
	d := Default()
	if d.Name != "thr" || d.Display != "Yamaha THR-II" {
		t.Fatalf("identity = %q/%q", d.Name, d.Display)
	}
	if d.FileExchange {
		t.Fatal("thr should be card-only")
	}
	if len(d.AmpTypes) != 8 {
		t.Fatalf("amp types = %d, want 8", len(d.AmpTypes))
	}
	if len(d.AmpModes) != 3 {
		t.Fatalf("amp modes = %d, want 3", len(d.AmpModes))
	}
	// 8 types × 3 modes = 24 cells, including three FLAT positions.
	if len(d.Amps) != 24 {
		t.Fatalf("amps = %d, want 24", len(d.Amps))
	}
	if len(d.Modulation) != 4 || len(d.Echo) != 2 || len(d.Reverb) != 4 {
		t.Fatalf("modulation/echo/reverb = %d/%d/%d, want 4/2/4", len(d.Modulation), len(d.Echo), len(d.Reverb))
	}
	if len(d.Cabs) != 16 {
		t.Fatalf("cabs = %d, want 16", len(d.Cabs))
	}
}

func TestResolveAmpByNameTypeAndInspiredBy(t *testing.T) {
	d := Default()

	s, err := d.Resolve(Spec{Amp: "CLEAN CLASSIC"})
	if err != nil || s.Amp != "CLEAN CLASSIC" {
		t.Fatalf("Resolve(CLEAN CLASSIC) = %q, %v", s.Amp, err)
	}

	s, err = d.Resolve(Spec{Amp: "clean"})
	if err != nil || s.Amp != "CLEAN CLASSIC" {
		t.Fatalf("Resolve(clean) = %q, %v; want CLEAN CLASSIC", s.Amp, err)
	}

	s, err = d.Resolve(Spec{Amp: "Twin Reverb"})
	if err != nil || s.Amp != "CLEAN CLASSIC" {
		t.Fatalf("Resolve(Twin Reverb) = %q, %v; want CLEAN CLASSIC", s.Amp, err)
	}

	// FLAT is a full selector group with three variants now.
	s, err = d.Resolve(Spec{Amp: "FLAT MODERN"})
	if err != nil || s.Amp != "FLAT MODERN" {
		t.Fatalf("Resolve(FLAT MODERN) = %q, %v", s.Amp, err)
	}
}

func TestResolveEffects(t *testing.T) {
	d := Default()
	s, err := d.Resolve(Spec{
		Amp:        "BROWN",
		Cab:        "brown 4x12",
		Mod:        "CHORUS",
		Echo:       "tape",
		Reverb:     "spring",
		Compressor: true,
		NoiseGate:  true,
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if s.Amp != "SPECIAL CLASSIC" {
		t.Fatalf("amp = %q, want SPECIAL CLASSIC (BROWN → CLASSIC)", s.Amp)
	}
	if s.Cab != "Brown 4x12" {
		t.Fatalf("cab = %q, want Brown 4x12", s.Cab)
	}
	if s.Mod != "CHORUS" || s.Echo != "Tape" || s.Reverb != "Spring" {
		t.Fatalf("mod/echo/reverb = %q/%q/%q", s.Mod, s.Echo, s.Reverb)
	}
	if !s.Compressor || !s.NoiseGate {
		t.Fatal("compressor/noise gate should be on")
	}
}

func TestResolveCabRequiresThrII(t *testing.T) {
	if _, err := Default().Resolve(Spec{Amp: "CLEAN", Cab: "Not a cab"}); err == nil {
		t.Fatal("expected an error for an unknown cabinet")
	}
	legacy, _ := ModelByName("thr10")
	if _, err := legacy.Resolve(Spec{Amp: "Lead", Cab: "Brown 4x12"}); err == nil {
		t.Fatal("legacy THR10 should reject a cabinet")
	}
}

func TestResolveRequiresAmp(t *testing.T) {
	if _, err := Default().Resolve(Spec{Name: "No Amp"}); err == nil {
		t.Fatal("expected an error when no amp is given")
	}
	if _, err := Default().Resolve(Spec{Amp: "not an amp"}); err == nil {
		t.Fatal("expected an error for an unknown amp")
	}
}

func TestSetupCardHTML(t *testing.T) {
	d := Default()
	s, err := d.Resolve(Spec{Name: "THR Clean", Amp: "Twin Reverb", Cab: "California 1x12", Mod: "CHORUS", Echo: "Tape", Reverb: "Hall"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	html := d.SetupCardHTML(s)
	for _, want := range []string{"THR Clean", "Yamaha THR-II", "CLEAN CLASSIC", "Fender Twin Reverb", "California 1x12", "CHORUS", "Tape", "Hall", "OFF"} {
		if !strings.Contains(html, want) {
			t.Fatalf("setup card missing %q", want)
		}
	}
}

func TestLegacyModelsPartial(t *testing.T) {
	for _, name := range []string{"thr10", "thr10c", "thr10x"} {
		m, ok := ModelByName(name)
		if !ok {
			t.Fatalf("model %q not registered", name)
		}
		if m.FileExchange {
			t.Fatalf("%s should be card-only", name)
		}
		if m.Note == "" {
			t.Fatalf("%s should flag its partial catalog", name)
		}
		if len(m.Amps) == 0 {
			t.Fatalf("%s has no amps", name)
		}
	}
}

func TestModelsRegistry(t *testing.T) {
	if len(Models()) != 4 {
		t.Fatalf("Models() = %d, want 4", len(Models()))
	}
	if _, ok := ModelByName("Yamaha THR-II"); !ok {
		t.Fatal("ModelByName(display name) should match")
	}
}
