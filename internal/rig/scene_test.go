package rig

import (
	"encoding/base64"
	"testing"

	"github.com/d-led/guitar-modeler-mcp/internal/catalog"
)

func sceneSlotNames(t *testing.T, file *RigFile) []string {
	t.Helper()
	content, err := file.Decode()
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	fs := decodeSection(content.FootSwitch)
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

	fs := decodeSection(content.FootSwitch)
	fsw := fs["data"].(map[string]any)["FootSwitch"].(map[string]any)["children"].(map[string]any)
	for _, n := range []string{"5", "6", "7", "8"} {
		if fsw["Module"+n].(map[string]any)["string"] != "Unassigned" {
			t.Fatalf("Module%s not reset", n)
		}
	}

	pedal := decodeSection(content.Pedal1)
	pedalChildren := pedal["data"].(map[string]any)["Pedal1"].(map[string]any)["children"].(map[string]any)
	if pedalChildren["Module1"].(map[string]any)["string"] != "Unassigned" {
		t.Fatal("Pedal1 Module1 not reset")
	}
}

// TestBuildAssignsExpressionPedal verifies an expression-pedal assignment is
// written into the Pedal1 section (module, param and sweep range).
func TestBuildAssignsExpressionPedal(t *testing.T) {
	b := newTestBuilder(t)
	file, err := b.Build(Spec{
		Name: "Wah Rig",
		Blocks: []Block{
			{Type: "Black Wah", Enabled: false},
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
		},
		Pedals: []Pedal{{Module: "Black Wah", Param: "Pedal"}},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	content, err := file.Decode()
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	pedal := decodeSection(content.Pedal1)
	children := pedal["data"].(map[string]any)["Pedal1"].(map[string]any)["children"].(map[string]any)
	if children["Module1"].(map[string]any)["string"] != "Black Wah" {
		t.Fatalf("Pedal1 Module1 = %v, want Black Wah", children["Module1"])
	}
	if children["Param1"].(map[string]any)["string"] != "Pedal" {
		t.Fatalf("Pedal1 Param1 = %v, want Pedal", children["Param1"])
	}
	if children["Min1"].(map[string]any)["value"] != float64(0) {
		t.Fatalf("Pedal1 Min1 = %v, want 0", children["Min1"])
	}
	if children["Max1"].(map[string]any)["value"] != float64(100) {
		t.Fatalf("Pedal1 Max1 = %v, want 100", children["Max1"])
	}
}

// TestBuildRejectsUnknownPedalModule ensures a pedal referencing a module
// outside the chain is refused rather than silently written.
func TestBuildRejectsUnknownPedalModule(t *testing.T) {
	b := newTestBuilder(t)
	_, err := b.Build(Spec{
		Name:   "Wah Rig",
		Blocks: []Block{{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}}},
		Pedals: []Pedal{{Module: "Bogus", Param: "Pedal"}},
	})
	if err == nil {
		t.Fatal("expected an error for a pedal module not in the chain")
	}
}

// TestBuildMarksFirstSceneActiveByDefault ensures a rig with scene switches
// defines its starting point: LastScene points at the first Scene-mode switch.
func TestBuildMarksFirstSceneActiveByDefault(t *testing.T) {
	b := newTestBuilder(t)
	file, err := b.Build(Spec{
		Name: "Scene Rig",
		Blocks: []Block{
			{Type: "Green JRC-OD", Enabled: true},
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
		},
		Footswitches: []Footswitch{
			{Module: "Green JRC-OD", Mode: "Scene", Label: "LEAD", Scene: &SceneSnapshot{On: []string{"Green JRC-OD"}}},
			{Module: "Green JRC-OD", Mode: "Toggle", Label: "BOOST"},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	content, err := file.Decode()
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	fs := decodeSection(content.FootSwitch)
	children := fs["data"].(map[string]any)["FootSwitch"].(map[string]any)["children"].(map[string]any)
	if got := children["LastScene"].(map[string]any)["value"]; got != float64(0) {
		t.Fatalf("LastScene = %v, want 0 (first scene on FS5 active)", got)
	}
	if got := children["LastSceneState"].(map[string]any)["value"]; got != float64(0) {
		t.Fatalf("LastSceneState = %v, want 0", got)
	}
}

// TestBuildNoSceneLeavesLastSceneInactive ensures toggle-only rigs do not
// claim a scene is active.
func TestBuildNoSceneLeavesLastSceneInactive(t *testing.T) {
	b := newTestBuilder(t)
	file, err := b.Build(Spec{
		Name: "Toggle Rig",
		Blocks: []Block{
			{Type: "Green JRC-OD", Enabled: true},
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
		},
		Footswitches: []Footswitch{{Module: "Green JRC-OD"}},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	content, err := file.Decode()
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	fs := decodeSection(content.FootSwitch)
	children := fs["data"].(map[string]any)["FootSwitch"].(map[string]any)["children"].(map[string]any)
	if got := children["LastScene"].(map[string]any)["value"]; got != float64(-1) {
		t.Fatalf("LastScene = %v, want -1 (no scene active)", got)
	}
}
