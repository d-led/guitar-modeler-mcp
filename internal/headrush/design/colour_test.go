package design

import (
	"strings"
	"testing"

	"github.com/d-led/guitar-modeler-mcp/internal/headrush/catalog"
	"github.com/d-led/guitar-modeler-mcp/internal/headrush/rig"
)

// TestDefaultColourMirrorsFactoryRigs locks the factory-conventional slot
// colour per module type — the palette the device's own rigs actually use —
// so a fresh rig reads like a stock one instead of every slot being green.
func TestDefaultColourMirrorsFactoryRigs(t *testing.T) {
	d := NewDesigner(catalog.New())
	cases := []struct{ module, colour string }{
		{"Amp", "Yellow"}, {"Cab", "Green"}, {"IR", "Green"}, {"IR (1024)", "Green"},
		{"Green JRC-OD", "Yellow"}, {"White Boost", "Yellow"}, {"DC Distort", "Yellow"},
		{"Gray Comp", "Red"}, {"DynIII Comp", "Red"}, {"Side Comp", "Red"},
		{"Bass EQ", "Yellow"}, {"Graphic EQ", "Yellow"},
		{"Dyn Delay", "Green"}, {"Tape Echo", "Green"}, {"BBD Delay", "Green"},
		{"Reverse Delay", "Green"}, {"AIR Delay", "Green"},
		{"Pitch Delay", "Purple"}, {"Reso Delay", "Red"},
		{"Eleven Reverb", "Blue"}, {"Spring Reverb", "Blue"}, {"AIR Reverb", "Blue"}, {"Shimmer", "Blue"},
		{"Black Wah", "Orange"}, {"Shine Wah", "Orange"}, {"More Wah", "Orange"},
		{"Volume", "Red"}, {"Wham", "Purple"}, {"Harm", "Blue"},
		{"Chorus", "Purple"}, {"Multi Chorus", "Purple"}, {"Dim Chorus", "Purple"}, {"Detune", "Purple"},
		{"Tremolo", "Purple"}, {"Vibrato", "Purple"}, {"Ring Mod", "Purple"}, {"Drop Tune", "Purple"},
		{"Flanger", "Orange"}, {"AIR Flanger", "Orange"}, {"Rotary", "Orange"},
		{"Stone Phaser", "Orange"}, {"Orange Phaser", "Orange"}, {"Vibe Phaser", "Orange"},
		{"Stereo Doubler", "Red"}, {"Octaves", "Blue"}, {"Octaves Up", "Blue"}, {"Smart Harm", "Blue"},
		{"Tron Filter", "Yellow"}, {"AIR Filter", "Yellow"}, {"Env Filter", "Yellow"},
	}
	for _, tc := range cases {
		if got := d.defaultColour(tc.module); got != tc.colour {
			t.Errorf("defaultColour(%q) = %q, want %q", tc.module, got, tc.colour)
		}
	}
}

// TestDesignColoursSlotsByConvention builds a whole rig through the designer
// and checks that each slot lands on its factory-conventional colour.
func TestDesignColoursSlotsByConvention(t *testing.T) {
	b, err := rig.NewBuilder(catalog.New())
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	d := NewDesigner(catalog.New())
	res, err := d.Design(Request{
		Name: "Stock Colours",
		Amp:  "Marshall JCM800",
		FX: []FXBlock{
			{Type: "Green JRC-OD", Enabled: true},
			{Type: "Gray Comp", Enabled: true},
			{Type: "Dyn Delay", Enabled: true},
			{Type: "Chorus", Enabled: true},
			{Type: "Eleven Reverb", Enabled: true},
		},
	})
	if err != nil {
		t.Fatalf("Design: %v", err)
	}
	file, err := b.Build(res.Spec)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	want := map[string]string{
		"Green JRC-OD": "Yellow", "Gray Comp": "Red",
		"Amp": "Yellow", "Cab": "Green",
		"Dyn Delay": "Green", "Chorus": "Purple", "Eleven Reverb": "Blue",
	}
	for module, colour := range want {
		if got := designModuleColour(t, file, module); got != colour {
			t.Errorf("%s colour = %q, want %q", module, got, colour)
		}
	}
}

// TestDesignAllowsSlotColourOverrides checks that per-effect colour overrides
// (fx[].colour) reach the built rig, while the amp and cab keep their factory
// colour automatically.
func TestDesignAllowsSlotColourOverrides(t *testing.T) {
	b, err := rig.NewBuilder(catalog.New())
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	d := NewDesigner(catalog.New())
	res, err := d.Design(Request{
		Name: "Overrides",
		Amp:  "65 Black SR",
		Cab:  "1x12 Black Panel Lux",
		FX: []FXBlock{
			{Type: "Tape Echo", Enabled: true, Colour: "Red"},
			{Type: "Dyn Delay", Enabled: true}, // no override → conventional Green
		},
	})
	if err != nil {
		t.Fatalf("Design: %v", err)
	}
	file, err := b.Build(res.Spec)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	want := map[string]string{
		"Amp": "Yellow", "Cab": "Green", "Tape Echo": "Red", "Dyn Delay": "Green",
	}
	for module, colour := range want {
		if got := designModuleColour(t, file, module); got != colour {
			t.Errorf("%s colour = %q, want %q", module, got, colour)
		}
	}
}

func TestDesignRejectsInvalidSlotColour(t *testing.T) {
	d := NewDesigner(catalog.New())
	_, err := d.Design(Request{
		Name: "Bad",
		Amp:  "65 Black SR",
		FX:   []FXBlock{{Type: "Chorus", Colour: "Mauve"}},
	})
	if err == nil {
		t.Fatal("expected an invalid fx colour error")
	}
	if !strings.Contains(err.Error(), "invalid colour") {
		t.Fatalf("error = %v, want an invalid-colour message", err)
	}
}

// designModuleColour reads the Colour tag a built module node was written with.
func designModuleColour(t *testing.T, file *rig.RigFile, name string) string {
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
