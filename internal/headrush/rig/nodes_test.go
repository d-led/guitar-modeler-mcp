package rig

import (
	"testing"

	"github.com/d-led/guitar-modeler-mcp/internal/headrush/catalog"
)

func TestOutputNodeWritesToAmpGain(t *testing.T) {
	b, err := NewBuilder(catalog.New())
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	file, err := b.Build(Spec{
		Name: "Output Node",
		Blocks: []Block{
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
		},
		OutputVolume: 4,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	content, err := file.Decode()
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	out := content.Data.Patch.Children["Output"]
	if out == nil {
		t.Fatal("Output node missing")
	}

	item, ok := out.Children["ToAmpGain"]
	if !ok || item.Value == nil {
		t.Fatalf("Output node missing ToAmpGain: %+v", out.Children)
	}
	wantEq(t, "ToAmpGain", *item.Value, 0.0)

	rigVol, ok := out.Children["RigVolume"]
	if !ok || rigVol.Value == nil {
		t.Fatalf("Output node missing RigVolume: %+v", out.Children)
	}
	wantEq(t, "RigVolume", *rigVol.Value, 4.0)
}

// TestGateUsesDeviceSpelling pins the gate module's filter-threshold key to the
// device's "FilterThreshhold" spelling (with the extra h). A renamed
// "FilterThreshold" key is an orphan the device ignores — the value would
// silently never reach the device.
func TestGateUsesDeviceSpelling(t *testing.T) {
	b, err := NewBuilder(catalog.New())
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	file, err := b.Build(Spec{
		Name: "Gate Node",
		Blocks: []Block{
			{Type: "Gate", Enabled: true},
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	content, err := file.Decode()
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	gate := content.Data.Patch.Children["Gate"]
	if gate == nil {
		t.Fatal("Gate node missing")
	}
	if _, ok := gate.Children["FilterThreshhold"]; !ok {
		t.Fatal("Gate node missing the device's FilterThreshhold key")
	}
	if _, ok := gate.Children["FilterThreshold"]; ok {
		t.Fatal("Gate node wrote FilterThreshold, an orphan the device ignores")
	}
	inOrder := false
	for _, k := range gate.ChildOrder {
		if k == "FilterThreshhold" {
			inOrder = true
		}
	}
	if !inOrder {
		t.Fatalf("Gate childorder missing FilterThreshhold: %v", gate.ChildOrder)
	}
}
