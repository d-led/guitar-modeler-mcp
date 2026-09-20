package rig

import (
	"encoding/base64"
	"testing"

	"github.com/d-led/guitar-modeler-mcp/internal/headrush/catalog"
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
// defines its starting point: LastScene points at the first Scene switch (0 =
// FS5, the device's 0-based index).
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
	// The first Scene switch is engaged at load: LastScene = 0 (0-based FS5),
	// and the rig carries no ModeN (the device does not write it).
	if got := children["LastScene"].(map[string]any)["value"]; got != float64(0) {
		t.Fatalf("LastScene = %v, want 0 (first scene on FS5 engaged at load)", got)
	}
	if _, ok := children["Mode5"]; ok {
		t.Fatalf("Mode5 should not be written (the device does not use it), got %v", children["Mode5"])
	}
}

// TestBuildNoSceneWritesNoEngagedFlag ensures toggle-only rigs do not claim a
// scene is engaged at load: LastScene is -1 (none).
func TestBuildNoSceneWritesNoEngagedFlag(t *testing.T) {
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
		t.Fatalf("LastScene = %v, want -1 (no scene engaged)", got)
	}
}

// TestBuildEngagedSceneUsesLastSceneIndex pins the device's engaged-scene
// encoding: the first Scene switch is recorded as LastScene = 0 (0-based FS5),
// and no ModeN flag is written.
func TestBuildEngagedSceneUsesLastSceneIndex(t *testing.T) {
	b := newTestBuilder(t)
	file, err := b.Build(Spec{
		Name: "Mixed Switches",
		Blocks: []Block{
			{Type: "Green JRC-OD", Enabled: true},
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
			{Type: "Tape Echo", Enabled: true},
		},
		Footswitches: []Footswitch{
			{Module: "Tape Echo"}, // toggle on FS5
			{Module: "Green JRC-OD", Mode: "Scene", Scene: &SceneSnapshot{On: []string{"Green JRC-OD"}}}, // first scene on FS6
			{Module: "Tape Echo", Mode: "Scene", Scene: &SceneSnapshot{On: []string{"Tape Echo"}}},       // second scene
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

	// The first Scene switch sits on FS6 (0-based index 1).
	if got := children["LastScene"].(map[string]any)["value"]; got != float64(1) {
		t.Fatalf("LastScene = %v, want 1 (first scene is on FS6)", got)
	}
	for _, n := range []string{"5", "6", "7", "8"} {
		if _, ok := children["Mode"+n]; ok {
			t.Fatalf("Mode%s should not be written (the device does not use it)", n)
		}
	}
}

// TestBuildModernSceneFields ensures the writer emits the scene bookkeeping the
// device's current firmware writes back, not the legacy template fields.
func TestBuildModernSceneFields(t *testing.T) {
	b := newTestBuilder(t)
	file, err := b.Build(Spec{
		Name: "Scene Rig",
		Blocks: []Block{
			{Type: "Green JRC-OD", Enabled: true},
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
		},
		Footswitches: []Footswitch{
			{Module: "Green JRC-OD", Mode: "Scene", Scene: &SceneSnapshot{On: []string{"Green JRC-OD"}}},
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

	for _, n := range []string{"5", "6", "7", "8"} {
		assertModernSwitchFields(t, children, n)
	}
	if got := childString(children, "TapTempoColour"); got != "Green" {
		t.Fatalf("TapTempoColour = %q, want \"Green\"", got)
	}
	if _, ok := children["Scene1Slot5Preset"]; ok {
		t.Fatal("Scene1Slot5Preset should not be written (legacy numbering)")
	}
	if got := childString(children, "Scene5Slot1Preset"); got != "No Preset" {
		t.Fatalf("Scene5Slot1Preset = %q, want \"No Preset\"", got)
	}
}

// TestBuildSyncsChildOrder ensures the FootSwitch childorder lists exactly the
// written children. The device ignores a field missing from childorder, which
// is how the engaged-scene LastScene field was lost, so childorder and children
// must agree on every key.
func TestBuildSyncsChildOrder(t *testing.T) {
	b := newTestBuilder(t)
	file, err := b.Build(Spec{
		Name: "Scene Rig",
		Blocks: []Block{
			{Type: "Green JRC-OD", Enabled: true},
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
		},
		Footswitches: []Footswitch{
			{Module: "Green JRC-OD", Mode: "Scene", Scene: &SceneSnapshot{On: []string{"Green JRC-OD"}}},
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
	fsobj := fs["data"].(map[string]any)["FootSwitch"].(map[string]any)
	children := fsobj["children"].(map[string]any)
	inOrder := footSwitchOrder(t, fs)

	if len(inOrder) != len(children) {
		t.Fatalf("childorder has %d entries but children has %d", len(inOrder), len(children))
	}
	for name := range children {
		if !inOrder[name] {
			t.Fatalf("childorder is missing %q (the device ignores fields not listed)", name)
		}
	}
	if !inOrder["LastScene"] {
		t.Fatal("LastScene must be listed in childorder or the scene is not engaged at load")
	}
	assertNoLegacySceneFields(t, inOrder)
}

// footSwitchOrder returns the FootSwitch childorder names as a set.
func footSwitchOrder(t *testing.T, fs map[string]any) map[string]bool {
	t.Helper()
	fsobj := fs["data"].(map[string]any)["FootSwitch"].(map[string]any)
	raw, ok := fsobj["childorder"].([]any)
	if !ok {
		t.Fatal("childorder is missing or not an array")
	}
	order := make(map[string]bool, len(raw))
	for _, k := range raw {
		order[k.(string)] = true
	}
	return order
}

// assertNoLegacySceneFields fails if the device-retired scene fields resurface
// in the childorder.
func assertNoLegacySceneFields(t *testing.T, inOrder map[string]bool) {
	t.Helper()
	for _, n := range []string{"5", "6", "7", "8"} {
		for _, p := range []string{"Mode", "SceneState", "State2ExtAmp"} {
			if inOrder[p+n] {
				t.Fatalf("legacy field %s%s leaked into childorder", p, n)
			}
		}
	}
}

// assertModernSwitchFields checks one footswitch's scene bookkeeping matches the
// device's current save format.
func assertModernSwitchFields(t *testing.T, children map[string]any, n string) {
	t.Helper()
	if _, ok := children["SceneState"+n]; ok {
		t.Fatalf("SceneState%s should not be written (legacy field), got %v", n, children["SceneState"+n])
	}
	if _, ok := children["Mode"+n]; ok {
		t.Fatalf("Mode%s should not be written (legacy field), got %v", n, children["Mode"+n])
	}
	if _, ok := children["State2ExtAmp"+n]; ok {
		t.Fatalf("State2ExtAmp%s should not be written (renamed State2SceneExtAmp), got %v", n, children["State2ExtAmp"+n])
	}
	if got := childString(children, "State2SceneExtAmp"+n); got != "No Change" {
		t.Fatalf("State2SceneExtAmp%s = %q, want \"No Change\"", n, got)
	}
	if got := childString(children, "State2MacroColour"+n); got != "Green" {
		t.Fatalf("State2MacroColour%s = %q, want \"Green\"", n, got)
	}
}
