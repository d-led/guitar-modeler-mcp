package rig

import (
	"encoding/base64"
	"testing"

	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/catalog"
)

func sceneSlotNames(t *testing.T, file *RigFile) []string {
	t.Helper()
	content, err := file.Decode()
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	fs := content.FootSwitch.(map[string]any)
	children := fs["data"].(map[string]any)["FootSwitch"].(map[string]any)["children"].(map[string]any)
	blob := children["Scene5"].(map[string]any)["state"].(string)
	raw, err := base64.StdEncoding.DecodeString(blob)
	if err != nil {
		t.Fatalf("decode scene blob: %v", err)
	}
	if len(raw) != 11*36 {
		t.Fatalf("scene blob is %d bytes, want %d", len(raw), 11*36)
	}
	names := make([]string, 11)
	for i := 0; i < 11; i++ {
		field := raw[i*36+4 : i*36+36]
		for j, b := range field {
			if b == 0 {
				names[i] = string(field[:j])
				break
			}
		}
	}
	return names
}

// TestSceneBlobMatchesChain guards the invariant the device relies on: the
// serialized scene state must list the same modules as the Chain slots.
func TestSceneBlobMatchesChain(t *testing.T) {
	b, err := NewBuilder(catalog.New())
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	file, err := b.Build(Spec{
		Name: "Scene Test",
		Blocks: []Block{
			{Type: "Green JRC-OD", Enabled: true},
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
			{Type: "Tape Echo", Enabled: true},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	want := []string{"Green JRC-OD", "Amp", "Cab", "Tape Echo", "Empty Slot", "Empty Slot", "Empty Slot", "Empty Slot", "Empty Slot", "Empty Slot", "Empty Slot"}
	got := sceneSlotNames(t, file)
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("scene slot %d = %q, want %q (full: %v)", i, got[i], want[i], got)
		}
	}
}

// TestFootSwitchAndPedalsReset verifies the generated rig does not reference
// modules from the template chain.
func TestFootSwitchAndPedalsReset(t *testing.T) {
	b, err := NewBuilder(catalog.New())
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	file, err := b.Build(Spec{
		Name: "Reset Test",
		Blocks: []Block{
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

	fs := content.FootSwitch.(map[string]any)
	fsw := fs["data"].(map[string]any)["FootSwitch"].(map[string]any)["children"].(map[string]any)
	for _, n := range []string{"5", "6", "7", "8"} {
		if fsw["Module"+n].(map[string]any)["string"] != "Unassigned" {
			t.Fatalf("Module%s not reset", n)
		}
	}

	pedal := content.Pedal1.(map[string]any)
	pedalChildren := pedal["data"].(map[string]any)["Pedal1"].(map[string]any)["children"].(map[string]any)
	if pedalChildren["Module1"].(map[string]any)["string"] != "Unassigned" {
		t.Fatal("Pedal1 Module1 not reset")
	}
}
