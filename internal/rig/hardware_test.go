package rig

import (
	"testing"

	"github.com/d-led/guitar-modeler-mcp/internal/catalog"
)

func TestHardwareAssignmentsButtonsAndPedals(t *testing.T) {
	b, err := NewBuilder(catalog.New())
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	file, err := b.Build(Spec{
		Name: "HW Assign",
		Blocks: []Block{
			{Type: "Wham", Enabled: true},
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
		},
		Footswitches: []Footswitch{{Module: "Wham"}, {Module: "Amp"}},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	hw, err := HardwareAssignments(file)
	if err != nil {
		t.Fatalf("HardwareAssignments: %v", err)
	}

	if len(hw.Buttons) != 4 {
		t.Fatalf("buttons = %d, want 4", len(hw.Buttons))
	}
	got := hw.Buttons[0]
	if got.Number != 1 || got.Module != "Wham" || got.Operation != "On" || got.Mode != "Toggle" {
		t.Fatalf("button 1 = %+v, want {1 Wham On Toggle}", got)
	}
	if hw.Buttons[1].Module != "Amp" {
		t.Fatalf("button 2 module = %q, want Amp", hw.Buttons[1].Module)
	}
	if hw.Buttons[2].Module != "" || hw.Buttons[3].Module != "" {
		t.Fatalf("buttons 3/4 should be unassigned: %+v", hw.Buttons[2:])
	}

	if len(hw.Pedals) != 2 {
		t.Fatalf("pedals = %d, want 2", len(hw.Pedals))
	}
	if hw.Pedals[0].Name != "Pedal 1" || hw.Pedals[0].Mode != "Classic" {
		t.Fatalf("pedal 1 = %+v, want {Pedal 1 Classic}", hw.Pedals[0])
	}
	// The builder intentionally resets pedal targets so a rig never references
	// a module that is not in its chain.
	if len(hw.Pedals[0].Targets) != 0 || len(hw.Pedals[1].Targets) != 0 {
		t.Fatalf("expected no pedal targets: %+v %+v", hw.Pedals[0].Targets, hw.Pedals[1].Targets)
	}
}

func TestHardwareAssignmentsSceneLabel(t *testing.T) {
	b, err := NewBuilder(catalog.New())
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	file, err := b.Build(Spec{
		Name: "Scene Rig",
		Blocks: []Block{
			{Type: "Green JRC-OD", Enabled: true},
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
			{Type: "BBD Delay", Enabled: true},
		},
		Footswitches: []Footswitch{{
			Module: "Green JRC-OD",
			Mode:   "Scene",
			Label:  "DRIVE",
			Scene:  &SceneSnapshot{On: []string{"Green JRC-OD"}, Off: []string{"BBD Delay"}},
		}},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	hw, err := HardwareAssignments(file)
	if err != nil {
		t.Fatalf("HardwareAssignments: %v", err)
	}
	got := hw.Buttons[0]
	if got.Label != "DRIVE" || got.Mode != "Scene" || got.Module != "Green JRC-OD" {
		t.Fatalf("button 1 = %+v, want label DRIVE, mode Scene, module Green JRC-OD", got)
	}
}

func TestHardwareAssignmentsOff(t *testing.T) {
	b, err := NewBuilder(catalog.New())
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	file, err := b.Build(Spec{
		Name: "Off Switches",
		Blocks: []Block{
			{Type: "Chorus", Enabled: false}, // toggle target, starts off
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
			{Type: "Green JRC-OD", Enabled: true}, // scene target
		},
		Footswitches: []Footswitch{
			{Module: "Chorus"}, // Toggle, off at load
			{Module: "Green JRC-OD", Mode: "Scene", Scene: &SceneSnapshot{On: []string{"Green JRC-OD"}}}, // Scene, active at load
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	hw, err := HardwareAssignments(file)
	if err != nil {
		t.Fatalf("HardwareAssignments: %v", err)
	}

	if got := hw.Buttons[0]; !got.Off {
		t.Fatalf("toggle button for a bypassed module should be dimmed: %+v", got)
	}
	if got := hw.Buttons[1]; got.Off {
		t.Fatalf("the active scene must not be dimmed: %+v", got)
	}
}

func TestHardwareAssignmentsDimInactiveScenes(t *testing.T) {
	b, err := NewBuilder(catalog.New())
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	file, err := b.Build(Spec{
		Name: "Two Scenes",
		Blocks: []Block{
			{Type: "Green JRC-OD", Enabled: true},
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
			{Type: "BBD Delay", Enabled: true},
		},
		Footswitches: []Footswitch{
			{Module: "Green JRC-OD", Mode: "Scene", Label: "LEAD", Scene: &SceneSnapshot{On: []string{"Green JRC-OD"}, Off: []string{"BBD Delay"}}},
			{Module: "BBD Delay", Mode: "Scene", Label: "CLEAN", Scene: &SceneSnapshot{On: []string{"BBD Delay"}, Off: []string{"Green JRC-OD"}}},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	hw, err := HardwareAssignments(file)
	if err != nil {
		t.Fatalf("HardwareAssignments: %v", err)
	}

	// Only the first scene is active at load (LastScene=0); the second dims.
	if hw.Buttons[0].Off {
		t.Fatalf("first scene should be lit at load: %+v", hw.Buttons[0])
	}
	if !hw.Buttons[1].Off {
		t.Fatalf("second scene should be dimmed at load: %+v", hw.Buttons[1])
	}
}
