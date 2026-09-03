package gp200

import "testing"

func TestResolveAmpInspiredBy(t *testing.T) {
	// A real-hardware description resolves to the modeled amp.
	code, err := resolveAmp("Marshall JCM800")
	if err != nil {
		t.Fatalf("resolveAmp: %v", err)
	}
	if name := EffectName(code); name != "UK 800" {
		t.Fatalf("resolveAmp(JCM800) = %q, want UK 800", name)
	}
}

func TestResolveEffectSlotHint(t *testing.T) {
	// "Tube" alone is ambiguous; the slot hint picks the right catalog.
	dly, err := resolveEffect("dly", "Tube")
	if err != nil {
		t.Fatalf("resolveEffect(dly, Tube): %v", err)
	}
	if name := EffectName(dly); name != "Tube" {
		t.Fatalf("dly Tube = %q", name)
	}
	if _, err := resolveEffect("dst", "Tube"); err != nil {
		t.Fatalf("resolveEffect(dst, Tube): %v", err)
	}
}

func TestBuildPreset(t *testing.T) {
	p, err := BuildPreset(Spec{
		Name: "Brown Sound",
		Amp:  "Marshall JCM800",
		Cab:  "Marshall 1960AV",
		FX: []FXSpec{
			{Slot: "dst", Type: "Green OD", Enabled: true},
			{Slot: "dly", Type: "Tape", Enabled: true, Params: map[string]float32{"Mix": 40}},
		},
	})
	if err != nil {
		t.Fatalf("BuildPreset: %v", err)
	}
	if p.PatchName != "Brown Sound" {
		t.Fatalf("PatchName = %q", p.PatchName)
	}
	if name := EffectName(p.Blocks[3].EffectID); name != "UK 800" {
		t.Fatalf("amp = %q, want UK 800", name)
	}
	if !p.Blocks[3].Enabled {
		t.Fatal("amp block should be enabled")
	}
	if name := EffectName(p.Blocks[5].EffectID); name != "UK LD" {
		t.Fatalf("cab = %q, want UK LD", name)
	}
	if name := EffectName(p.Blocks[2].EffectID); name != "Green OD" {
		t.Fatalf("dst = %q, want Green OD", name)
	}
	if name := EffectName(p.Blocks[8].EffectID); name != "Tape" {
		t.Fatalf("dly = %q, want Tape", name)
	}
	// The Tape delay's Mix parameter (idx 0) must be overridden.
	if p.Blocks[8].Params[0] != 40 {
		t.Fatalf("Tape Mix = %v, want 40", p.Blocks[8].Params[0])
	}
}

func TestBuildPresetWriteReadRoundTrip(t *testing.T) {
	p, err := BuildPreset(Spec{Name: "Round Trip", Amp: "Vox AC30", Cab: "Vox AC30 2x12"})
	if err != nil {
		t.Fatalf("BuildPreset: %v", err)
	}
	data, err := p.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	back, err := Unmarshal(data)
	if err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if back.PatchName != "Round Trip" {
		t.Fatalf("round-tripped name = %q", back.PatchName)
	}
	if name := EffectName(back.Blocks[3].EffectID); name != "Foxy 30N" {
		t.Fatalf("round-tripped amp = %q, want Foxy 30N", name)
	}
	// Re-encoding the decoded preset must be stable (byte-identical).
	again, err := back.Marshal()
	if err != nil {
		t.Fatalf("second Marshal: %v", err)
	}
	if string(again) != string(data) {
		t.Fatal("second round-trip not byte-identical")
	}
}

func TestBuildPresetUnknownAmp(t *testing.T) {
	if _, err := BuildPreset(Spec{Amp: "Not A Real Amp"}); err == nil {
		t.Fatal("BuildPreset accepted an unknown amp")
	}
}

func TestBuildPresetUnknownBlock(t *testing.T) {
	_, err := BuildPreset(Spec{Amp: "UK 800", FX: []FXSpec{{Slot: "not-a-block", Type: "COMP"}}})
	if err == nil {
		t.Fatal("BuildPreset accepted an unknown block name")
	}
}
