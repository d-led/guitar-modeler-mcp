package qc

import (
	"strings"
	"testing"
)

func TestDefaultDeviceShape(t *testing.T) {
	d := Default()
	if d.Name != "quad-cortex" || d.Display != "Neural DSP Quad Cortex" {
		t.Fatalf("identity = %q/%q", d.Name, d.Display)
	}
	if d.FileExchange || d.FileExt != "" {
		t.Fatalf("file exchange = %v/%q, want false (the .pb is a reference archive, not a device import)", d.FileExchange, d.FileExt)
	}
	if len(d.Amps) < 50 {
		t.Fatalf("guitar amps = %d, want a comprehensive list", len(d.Amps))
	}
	if len(d.Cabs) < 30 {
		t.Fatalf("guitar cabs = %d, want a comprehensive list", len(d.Cabs))
	}
	for _, cat := range []string{"drive", "compressor", "equalizer", "delay", "modulation", "reverb", "wah", "pitch", "filter", "gate"} {
		if len(d.Effects[cat]) == 0 {
			t.Fatalf("effect category %q is empty", cat)
		}
	}
}

func TestCatalogSourcedItemsCarryIDs(t *testing.T) {
	d := Default()
	// The first guitar amp in wire-hash order is the Marshall JCM800.
	if len(d.Amps) == 0 || d.Amps[0].Name != "Marshall JCM800" || d.Amps[0].ID != 1001 {
		t.Fatalf("first amp = %+v, want Marshall JCM800 (1001)", d.Amps[0])
	}
	// The Klon Centaur is drive id 1.
	if len(d.Effects["drive"]) == 0 || d.Effects["drive"][0].Name != "Klon Centaur" || d.Effects["drive"][0].ID != 1 {
		t.Fatalf("first drive = %+v, want Klon Centaur (1)", d.Effects["drive"][0])
	}
}

func TestResolveAmpByExactNameAndInspiredBy(t *testing.T) {
	d := Default()

	got, err := d.ResolveAmp("Marshall JCM800")
	if err != nil || got.Name != "Marshall JCM800" {
		t.Fatalf("ResolveAmp(JCM800) = %q, %v", got.Name, err)
	}

	got, err = d.ResolveAmp("JCM800")
	if err != nil || got.Name != "Marshall JCM800" {
		t.Fatalf("ResolveAmp(JCM800 substring) = %q, %v; want Marshall JCM800", got.Name, err)
	}

	got, err = d.ResolveAmp("Twin Reverb")
	if err != nil || got.Name != "Fender Twin Reverb" {
		t.Fatalf("ResolveAmp(Twin Reverb) = %q, %v; want Fender Twin Reverb", got.Name, err)
	}
}

func TestResolveCab(t *testing.T) {
	got, err := Default().ResolveCab("Mesa Rectifier")
	if err != nil || !strings.Contains(got.Name, "Mesa Rectifier") || got.ID == 0 {
		t.Fatalf("ResolveCab(Mesa Rectifier) = %+v, %v; want a Mesa Rectifier cab", got, err)
	}
}

func TestResolveFXCategory(t *testing.T) {
	got, err := Default().ResolveFX("drive", "TS808")
	if err != nil || got.Name != "Ibanez TS808" {
		t.Fatalf("ResolveFX(drive, TS808) = %q, %v; want Ibanez TS808", got.Name, err)
	}

	if _, err := Default().ResolveFX("nope", "x"); err == nil {
		t.Fatal("expected an error for an unknown category")
	}
}

func TestResolveUnknownAmp(t *testing.T) {
	if _, err := Default().ResolveAmp("not an amp"); err == nil {
		t.Fatal("expected an error for an unknown amp")
	}
}
