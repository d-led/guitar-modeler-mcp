package qc

import "testing"

func TestDefaultDeviceShape(t *testing.T) {
	d := Default()
	if d.Name != "quad-cortex" || d.Display != "Neural DSP Quad Cortex" {
		t.Fatalf("identity = %q/%q", d.Name, d.Display)
	}
	if d.FileExchange {
		t.Fatal("quad-cortex should not claim file exchange yet (encrypted protobufs)")
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
	if err != nil || got.Name != "Mesa Rectifier V30" {
		t.Fatalf("ResolveCab(Mesa Rectifier) = %q, %v; want Mesa Rectifier V30", got.Name, err)
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
