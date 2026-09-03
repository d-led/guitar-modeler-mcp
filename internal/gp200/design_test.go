package gp200

import (
	"strings"
	"testing"
)

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
	p, rejected, err := BuildPreset(Spec{
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
	if len(rejected) != 0 {
		t.Fatalf("BuildPreset rejected %v, want none", rejected)
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
	p, _, err := BuildPreset(Spec{Name: "Round Trip", Amp: "Vox AC30", Cab: "Vox AC30 2x12"})
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
	if _, _, err := BuildPreset(Spec{Amp: "Not A Real Amp"}); err == nil {
		t.Fatal("BuildPreset accepted an unknown amp")
	}
}

func TestBuildPresetUnknownBlock(t *testing.T) {
	if _, _, err := BuildPreset(Spec{Amp: "UK 800", FX: []FXSpec{{Slot: "not-a-block", Type: "COMP"}}}); err == nil {
		t.Fatal("BuildPreset accepted an unknown block name")
	}
}

func TestBuildPresetRejectsUnknownParams(t *testing.T) {
	p, rejected, err := BuildPreset(Spec{
		Amp: "UK 900",
		FX: []FXSpec{
			{Slot: "eq", Type: "Guitar EQ 2", Enabled: true, Params: map[string]float32{"band2": 2.5, "band4": -1.5}},
		},
	})
	if err != nil {
		t.Fatalf("BuildPreset: %v", err)
	}
	// The GP-200 EQ bands are named by frequency, not "bandN"; both must be
	// reported as rejected and the EQ left at its flat default.
	if len(rejected) != 2 {
		t.Fatalf("rejected = %v, want band2 and band4", rejected)
	}
	for _, name := range []string{"band2", "band4"} {
		if !strings.Contains(rejected[0]+rejected[1], name) {
			t.Fatalf("rejected %v should mention %q", rejected, name)
		}
	}
	if p.Blocks[6].Params[1] != 0 || p.Blocks[6].Params[3] != 0 {
		t.Fatalf("EQ bands should stay flat, got %v", p.Blocks[6].Params[:5])
	}
}

func TestBuildPresetRejectsOutOfRangeParams(t *testing.T) {
	p, rejected, err := BuildPreset(Spec{
		Amp: "UK 900",
		FX: []FXSpec{
			{Slot: "mod", Type: "A-Chorus", Enabled: true, Params: map[string]float32{"rate": 15, "depth": 30}},
		},
	})
	if err != nil {
		t.Fatalf("BuildPreset: %v", err)
	}
	// "rate" 15 is out of the 0.1..10 Hz range; "depth" 30 is fine.
	if len(rejected) != 1 || !strings.Contains(strings.ToLower(rejected[0]), "rate") || !strings.Contains(rejected[0], "0.1") {
		t.Fatalf("rejected = %v, want an out-of-range message for rate", rejected)
	}
	// The chorus keeps the default rate (0.5 Hz), and depth 30 is applied.
	if p.Blocks[7].Params[1] != 0.5 {
		t.Fatalf("chorus rate = %v, want default 0.5", p.Blocks[7].Params[1])
	}
	if p.Blocks[7].Params[0] != 30 {
		t.Fatalf("chorus depth = %v, want 30", p.Blocks[7].Params[0])
	}
}
