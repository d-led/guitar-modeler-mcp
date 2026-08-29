package rig

import (
	"testing"

	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/catalog"
)

func footswitchChildren(t *testing.T, file *RigFile) map[string]any {
	t.Helper()
	content, err := file.Decode()
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	fs := content.FootSwitch.(map[string]any)
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
