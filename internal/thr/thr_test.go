package thr
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
	if len(d.Modulation) != 4 || len(d.EchoRev) != 4 {
		t.Fatalf("modulation/echoRev = %d/%d, want 4/4", len(d.Modulation), len(d.EchoRev))
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
		Mod:        "CHORUS",
		EchoRev:    "spring reverb",
		Compressor: true,
		NoiseGate:  true,
	})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if s.Amp != "SPECIAL CLASSIC" {
		t.Fatalf("amp = %q, want SPECIAL CLASSIC (BROWN → CLASSIC)", s.Amp)
	}
	if s.Mod != "CHORUS" || s.EchoRev != "SPRING REVERB" {
		t.Fatalf("mod/echoRev = %q/%q", s.Mod, s.EchoRev)
	}
	if !s.Compressor || !s.NoiseGate {
		t.Fatal("compressor/noise gate should be on")
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
	s, err := d.Resolve(Spec{Name: "THR Clean", Amp: "Twin Reverb", Mod: "CHORUS", EchoRev: "HALL REVERB"})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	html := d.SetupCardHTML(s)
	for _, want := range []string{"THR Clean", "Yamaha THR-II", "CLEAN CLASSIC", "Fender Twin Reverb", "CHORUS", "HALL REVERB", "OFF"} {
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
