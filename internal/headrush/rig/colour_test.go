package rig

import (
	"strings"
	"testing"
)

// moduleColour reads the Colour tag a built module node was written with.
func moduleColour(t *testing.T, file *RigFile, name string) string {
	t.Helper()
	c, err := file.Decode()
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	node := c.Data.Patch.Children[name]
	if node == nil {
		t.Fatalf("no module %q in the patch", name)
	}
	item := node.Children["Colour"]
	if item == nil || item.Str == nil {
		t.Fatalf("module %q carries no colour tag", name)
	}
	return *item.Str
}

func TestBuildAppliesBlockColour(t *testing.T) {
	b := newTestBuilder(t)
	file, err := b.Build(Spec{
		Name: "Colour",
		Blocks: []Block{
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux", "MicType": "Dyn 57"}},
			{Type: "Gray Comp", Enabled: true, Colour: "Purple"},
			{Type: "Dyn Delay", Enabled: true}, // no colour → module default Green
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if got := moduleColour(t, file, "Amp"); got != "Green" {
		t.Fatalf("Amp colour = %q, want the default Green", got)
	}
	if got := moduleColour(t, file, "Gray Comp"); got != "Purple" {
		t.Fatalf("Gray Comp colour = %q, want Purple", got)
	}
	if got := moduleColour(t, file, "Dyn Delay"); got != "Green" {
		t.Fatalf("Dyn Delay colour = %q, want the default Green", got)
	}
}

func TestBuildRejectsInvalidBlockColour(t *testing.T) {
	b := newTestBuilder(t)
	_, err := b.Build(Spec{
		Name: "Bad Colour",
		Blocks: []Block{
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux", "MicType": "Dyn 57"}},
			{Type: "Gray Comp", Enabled: true, Colour: "Mauve"},
		},
	})
	if err == nil {
		t.Fatal("expected an invalid-colour error")
	}
	if !strings.Contains(err.Error(), "invalid colour") {
		t.Fatalf("error = %v, want an invalid-colour message", err)
	}
}
