package rig

import "testing"

// TestRawContentExposesFootSwitchChildren ensures the raw view keeps the
// FootSwitch section's childorder and children, which the typed Summary does
// not surface.
func TestRawContentExposesFootSwitchChildren(t *testing.T) {
	b := newTestBuilder(t)
	file, err := b.Build(Spec{
		Name: "Raw",
		Blocks: []Block{
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	raw, err := file.RawContent()
	if err != nil {
		t.Fatalf("RawContent: %v", err)
	}
	fsw := raw["FootSwitch"].(map[string]any)["data"].(map[string]any)["FootSwitch"].(map[string]any)
	if _, ok := fsw["children"]; !ok {
		t.Fatal("raw FootSwitch missing children")
	}
	if _, ok := fsw["childorder"]; !ok {
		t.Fatal("raw FootSwitch missing childorder")
	}
}

// TestDiffRigsReportsChangedFields ensures the diff keys differences by JSON
// path and catches a changed rig name.
func TestDiffRigsReportsChangedFields(t *testing.T) {
	a, err := newTestBuilder(t).Build(Spec{
		Name: "RigA",
		Blocks: []Block{
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
		},
	})
	if err != nil {
		t.Fatalf("Build A: %v", err)
	}
	c, err := newTestBuilder(t).Build(Spec{
		Name: "RigB",
		Blocks: []Block{
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
		},
	})
	if err != nil {
		t.Fatalf("Build B: %v", err)
	}

	diff, err := DiffRigs(a, c)
	if err != nil {
		t.Fatalf("DiffRigs: %v", err)
	}
	const want = "data.Patch.children.Rig.children.PresetName.string"
	for _, e := range diff {
		if e.Path == want {
			if e.A != "RIGA" || e.B != "RIGB" {
				t.Fatalf("PresetName diff = A:%v B:%v", e.A, e.B)
			}
			return
		}
	}
	t.Fatalf("no diff entry for %q (got %d entries)", want, len(diff))
}

// TestDiffRigsIdenticalIsEmpty ensures a rig diffed against itself has no
// entries.
func TestDiffRigsIdenticalIsEmpty(t *testing.T) {
	a, err := newTestBuilder(t).Build(Spec{
		Name: "RigA",
		Blocks: []Block{
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	diff, err := DiffRigs(a, a)
	if err != nil {
		t.Fatalf("DiffRigs: %v", err)
	}
	if len(diff) != 0 {
		t.Fatalf("diff of a rig against itself = %d entries, want 0", len(diff))
	}
}
