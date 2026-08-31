package rig

import (
	"testing"

	"github.com/d-led/guitar-modeler-mcp/internal/catalog"
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
	if *item.Value != 0 {
		t.Fatalf("ToAmpGain = %v, want 0", *item.Value)
	}

	rigVol, ok := out.Children["RigVolume"]
	if !ok || rigVol.Value == nil || *rigVol.Value != 4 {
		t.Fatalf("RigVolume = %+v, want 4", rigVol)
	}
}
