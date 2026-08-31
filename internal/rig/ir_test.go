package rig

import (
	"testing"
)

func TestBuildIRBlockWritesSelectorAndDoublingFields(t *testing.T) {
	b := newTestBuilder(t)
	file, err := b.Build(Spec{
		Name: "IR RIG",
		Blocks: []Block{
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "IR", Enabled: true, Params: map[string]any{
				"IR":    "[directory](YorkMixes)[name](YA MES 212 V30 Mix 01)",
				"Gain":  -13.5,
				"HiCut": 20000.0,
				"LoCut": 20.0,
				"Mix":   100.0,
			}},
		},
	})
	if err != nil {
		t.Fatalf("Build (IR instead of Cab): %v", err)
	}

	content, err := file.Decode()
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	node := content.Data.Patch.Children["IR"]
	if node == nil {
		t.Fatal("IR node missing")
	}
	if got := *node.Children["IR"].Str; got != "[directory](YorkMixes)[name](YA MES 212 V30 Mix 01)" {
		t.Fatalf("IR selector = %q", got)
	}
	if got := *node.Children["Gain"].Value; got != -13.5 {
		t.Fatalf("IR Gain = %v, want -13.5", got)
	}
	if got := *node.Children["HiCut"].Value; got != 20000 {
		t.Fatalf("HiCut = %v, want 20000", got)
	}
	if got := *node.Children["LoCut"].Value; got != 20 {
		t.Fatalf("LoCut = %v, want 20", got)
	}
	// Doubling (stereo) fields are present with the device defaults.
	for _, key := range []string{"PresetName2", "Doubling", "DoubleStates", "IR2", "Gain2", "HiCut2", "LoCut2", "Mix2"} {
		if _, ok := node.Children[key]; !ok {
			t.Fatalf("IR node missing %q", key)
		}
	}
}

func TestBuildRequiresCabOrIR(t *testing.T) {
	b := newTestBuilder(t)
	// Amp + IR (no Cab) is accepted: the IR loader replaces the cabinet.
	if _, err := b.Build(Spec{Name: "IR", Blocks: []Block{
		{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
		{Type: "IR", Enabled: true},
	}}); err != nil {
		t.Fatalf("IR should satisfy the cab requirement: %v", err)
	}
	// Amp alone is still rejected.
	if _, err := b.Build(Spec{Name: "Amp Only", Blocks: []Block{
		{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
	}}); err == nil {
		t.Fatal("expected an error for a rig with neither Cab nor IR")
	}
}
