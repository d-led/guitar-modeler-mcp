package rig

import (
	"encoding/base64"
	"testing"

	"github.com/d-led/guitar-modeler-mcp/internal/catalog"
)

func footswitchChildren(t *testing.T, file *RigFile) map[string]any {
	t.Helper()
	content, err := file.Decode()
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	fs := decodeSection(content.FootSwitch)
	return fs["data"].(map[string]any)["FootSwitch"].(map[string]any)["children"].(map[string]any)
}

func footswitchField(t *testing.T, children map[string]any, key string) string {
	t.Helper()
	v, ok := children[key]
	if !ok {
		t.Fatalf("FootSwitch child %s missing", key)
	}
	item, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("FootSwitch child %s is %T, want object", key, v)
	}
	s, _ := item["string"].(string)
	return s
}

func TestFootswitchAssignment(t *testing.T) {
	b, err := NewBuilder(catalog.New())
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	file, err := b.Build(Spec{
		Name: "Whammy Toe",
		Blocks: []Block{
			{Type: "Green JRC-OD", Enabled: true},
			{Type: "Wham", Enabled: true},
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
			{Type: "Tape Echo", Enabled: true},
		},
		Footswitches: []Footswitch{
			{Module: "Wham"},
			{Module: "Green JRC-OD"},
			{Module: "Tape Echo"},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	children := footswitchChildren(t, file)

	got := map[string]string{}
	for _, n := range []string{"5", "6", "7", "8"} {
		got["Module"+n] = footswitchField(t, children, "Module"+n)
		got["Operation"+n] = footswitchField(t, children, "Operation"+n)
	}

	want := map[string]string{
		"Module5": "Wham", "Operation5": "On",
		"Module6": "Green JRC-OD", "Operation6": "On",
		"Module7": "Tape Echo", "Operation7": "On",
		"Module8": "Unassigned", "Operation8": "",
	}
	for k, v := range want {
		if got[k] != v {
			t.Fatalf("%s = %q, want %q", k, got[k], v)
		}
	}
}

func TestFootswitchDefaultsToOn(t *testing.T) {
	b, err := NewBuilder(catalog.New())
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	file, err := b.Build(Spec{
		Name: "Default Op",
		Blocks: []Block{
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
			{Type: "DynIII Comp", Enabled: true},
		},
		Footswitches: []Footswitch{{Module: "DynIII Comp"}},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	children := footswitchChildren(t, file)
	if got := footswitchField(t, children, "Operation5"); got != "On" {
		t.Fatalf("Operation5 = %q, want %q (default)", got, "On")
	}
}

func TestFootswitchCustomOperation(t *testing.T) {
	b, err := NewBuilder(catalog.New())
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	file, err := b.Build(Spec{
		Name: "Custom Op",
		Blocks: []Block{
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
			{Type: "Tremolo", Enabled: true},
		},
		Footswitches: []Footswitch{{Module: "Tremolo", Operation: "Speed"}},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	children := footswitchChildren(t, file)
	if got := footswitchField(t, children, "Operation5"); got != "Speed" {
		t.Fatalf("Operation5 = %q, want %q", got, "Speed")
	}
}

func TestFootswitchUnknownModule(t *testing.T) {
	b, err := NewBuilder(catalog.New())
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	_, err = b.Build(Spec{
		Name: "Bad Switch",
		Blocks: []Block{
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
		},
		Footswitches: []Footswitch{{Module: "Not In Chain"}},
	})
	if err == nil {
		t.Fatal("expected an error for a footswitch module not in the chain")
	}
}

func TestFootswitchTooMany(t *testing.T) {
	b, err := NewBuilder(catalog.New())
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	_, err = b.Build(Spec{
		Name: "Five Switches",
		Blocks: []Block{
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
		},
		Footswitches: []Footswitch{
			{Module: "Amp"}, {Module: "Cab"}, {Module: "Amp"}, {Module: "Cab"}, {Module: "Amp"},
		},
	})
	if err == nil {
		t.Fatal("expected an error for more than 4 footswitches")
	}
}

func TestFootswitchMatchesInstanceName(t *testing.T) {
	b, err := NewBuilder(catalog.New())
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	// A dual-amp rig has both "Amp" and "Amp 2"; the switch must be able to
	// target the second amp by its instance name.
	file, err := b.Build(Spec{
		Name:    "Dual Amp Switch",
		Routing: RoutingSPS,
		PathA: []Block{
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
		},
		PathB: []Block{
			{Type: "Amp", Params: map[string]any{"Type": "85 M-2 Lead"}},
			{Type: "Cab", Params: map[string]any{"CabType": "4x12 Classic 30W"}},
		},
		Footswitches: []Footswitch{{Module: "amp 2"}},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	children := footswitchChildren(t, file)
	if got := footswitchField(t, children, "Module5"); got != "Amp 2" {
		t.Fatalf("Module5 = %q, want %q", got, "Amp 2")
	}
}

func TestFootswitchSceneMode(t *testing.T) {
	b, err := NewBuilder(catalog.New())
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	file, err := b.Build(Spec{
		Name: "Scene Switch",
		Blocks: []Block{
			{Type: "Green JRC-OD", Enabled: true},
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
			{Type: "Tape Echo", Enabled: true},
		},
		Footswitches: []Footswitch{
			{Module: "Green JRC-OD", Mode: "Scene"},
			{Module: "Tape Echo"},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	children := footswitchChildren(t, file)
	if got := footswitchField(t, children, "ModeNew5"); got != "Scene" {
		t.Fatalf("ModeNew5 = %q, want Scene", got)
	}
	if got := footswitchField(t, children, "ModeNew6"); got != "Toggle" {
		t.Fatalf("ModeNew6 = %q, want Toggle (default)", got)
	}
}

func TestFootswitchRejectsUnknownMode(t *testing.T) {
	b, err := NewBuilder(catalog.New())
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	_, err = b.Build(Spec{
		Name: "Bad Mode",
		Blocks: []Block{
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
		},
		Footswitches: []Footswitch{{Module: "Amp", Mode: "Bogus"}},
	})
	if err == nil {
		t.Fatal("expected an error for an unknown footswitch mode")
	}
}

func TestFootswitchSceneSnapshotHeaders(t *testing.T) {
	b, err := NewBuilder(catalog.New())
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	file, err := b.Build(Spec{
		Name: "Scene Snapshot",
		Blocks: []Block{
			{Type: "Green JRC-OD", Enabled: true},
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
			{Type: "Tape Echo", Enabled: true},
		},
		Footswitches: []Footswitch{{
			Module: "Green JRC-OD",
			Mode:   "Scene",
			Label:  "DRIVE",
			Scene: &SceneSnapshot{
				On:  []string{"Green JRC-OD"},
				Off: []string{"Tape Echo"},
			},
		}},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	children := footswitchChildren(t, file)
	if got := footswitchField(t, children, "UserFootSwitchText5"); got != "DRIVE" {
		t.Fatalf("UserFootSwitchText5 = %q, want DRIVE", got)
	}
	if got := footswitchField(t, children, "ModeNew5"); got != "Scene" {
		t.Fatalf("ModeNew5 = %q, want Scene", got)
	}

	raw, err := base64.StdEncoding.DecodeString(children["Scene5"].(map[string]any)["state"].(string))
	if err != nil {
		t.Fatalf("decode scene blob: %v", err)
	}
	// Chain: slot1=Green JRC-OD, slot2=Amp, slot3=Cab, slot4=Tape Echo.
	// Scene turns Green JRC-OD on (1) and Tape Echo off (2); Amp/Cab unchanged (0).
	got := []byte{raw[0], raw[36], raw[36*2], raw[36*3]}
	want := []byte{1, 0, 0, 2}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("scene header slot %d = %d, want %d (headers: %v)", i+1, got[i], want[i], got)
		}
	}
	// State2Scene (the revert state) must stay "no change".
	state2, err := base64.StdEncoding.DecodeString(children["State2Scene5"].(map[string]any)["state"].(string))
	if err != nil {
		t.Fatalf("decode State2Scene5: %v", err)
	}
	for i := 0; i < 4; i++ {
		if state2[i*36] != 0 {
			t.Fatalf("State2Scene5 slot %d header = %d, want 0 (no change)", i+1, state2[i*36])
		}
	}
}

func TestFootswitchSceneRejectsUnknownBlock(t *testing.T) {
	b, err := NewBuilder(catalog.New())
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	_, err = b.Build(Spec{
		Name: "Bad Scene",
		Blocks: []Block{
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
		},
		Footswitches: []Footswitch{{
			Module: "Amp",
			Mode:   "Scene",
			Scene:  &SceneSnapshot{On: []string{"Not In Chain"}},
		}},
	})
	if err == nil {
		t.Fatal("expected an error for a scene block not in the chain")
	}
}
