package design

import (
	"strings"
	"testing"

	"github.com/d-led/guitar-modeler-mcp/internal/catalog"
	"github.com/d-led/guitar-modeler-mcp/internal/rig"
)

func TestDesignOrdersChainAndResolvesHardware(t *testing.T) {
	d := NewDesigner(catalog.New())
	res, err := d.Design(Request{
		Name: "Brown Sound",
		Amp:  "Marshall JCM800",
		FX: []FXBlock{
			{Type: "Tape Echo", Enabled: true},
			{Type: "Green JRC-OD", Enabled: true},
		},
	})
	if err != nil {
		t.Fatalf("Design: %v", err)
	}

	types := make([]string, len(res.Spec.Blocks))
	for i, b := range res.Spec.Blocks {
		types[i] = b.Type
	}

	want := []string{"Green JRC-OD", "Amp", "Cab", "Tape Echo"}
	if len(types) != len(want) {
		t.Fatalf("chain = %v, want %v", types, want)
	}
	for i := range want {
		if types[i] != want[i] {
			t.Fatalf("chain = %v, want %v", types, want)
		}
	}

	// The translated amp must be a Marshall JCM800 model.
	ampBlock := res.Spec.Blocks[1]
	if ampBlock.Params["Type"] == "" {
		t.Fatal("amp type not resolved")
	}
}

func TestDesignAppliesAmpAndCabParams(t *testing.T) {
	d := NewDesigner(catalog.New())
	res, err := d.Design(Request{
		Name:      "Dialed",
		Amp:       "65 Black SR",
		AmpParams: map[string]any{"GainA": 62.0, "Master": 55.0, "Treble": 58.0},
		CabParams: map[string]any{"Breakup": 30.0, "OnAxis": false},
	})
	if err != nil {
		t.Fatalf("Design: %v", err)
	}

	amp := blockByType(res.Spec.Blocks, "Amp")
	cab := blockByType(res.Spec.Blocks, "Cab")
	if amp.Params["Type"] != "65 Black SR" {
		t.Fatalf("amp type = %v, want 65 Black SR", amp.Params["Type"])
	}
	if amp.Params["GainA"] != 62.0 || amp.Params["Master"] != 55.0 || amp.Params["Treble"] != 58.0 {
		t.Fatalf("amp params = %v, want GainA 62, Master 55, Treble 58", amp.Params)
	}
	if cab.Params["CabType"] != "1x12 Black Panel Lux" {
		t.Fatalf("cab type = %v, want default 1x12 Black Panel Lux", cab.Params["CabType"])
	}
	if cab.Params["Breakup"] != 30.0 || cab.Params["OnAxis"] != false {
		t.Fatalf("cab params = %v, want Breakup 30, OnAxis false", cab.Params)
	}
}

// blockByType returns the first chain block of the given type, or a zero block.
func blockByType(blocks []rig.Block, typ string) rig.Block {
	for _, b := range blocks {
		if b.Type == typ {
			return b
		}
	}
	return rig.Block{}
}

func TestDesignAutoAssignsExpressionPedal(t *testing.T) {
	d := NewDesigner(catalog.New())
	res, err := d.Design(Request{
		Name: "Wah",
		Amp:  "65 Black SR",
		FX:   []FXBlock{{Type: "Black Wah", Enabled: false}},
	})
	if err != nil {
		t.Fatalf("Design: %v", err)
	}
	if len(res.Spec.Pedals) != 1 || res.Spec.Pedals[0].Module != "Black Wah" || res.Spec.Pedals[0].Param != "Pedal" {
		t.Fatalf("pedals = %+v, want Black Wah -> Pedal", res.Spec.Pedals)
	}
	found := false
	for _, n := range res.Notes {
		if strings.Contains(n, "expression pedal 1") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected an expression-pedal note, got %v", res.Notes)
	}
}

func TestDesignRespectsExplicitPedals(t *testing.T) {
	d := NewDesigner(catalog.New())
	res, err := d.Design(Request{
		Name:   "Wah",
		Amp:    "65 Black SR",
		FX:     []FXBlock{{Type: "Black Wah", Enabled: false}},
		Pedals: []rig.Pedal{{Module: "Black Wah", Param: "Pedal", Min: 10, Max: 90}},
	})
	if err != nil {
		t.Fatalf("Design: %v", err)
	}
	if len(res.Spec.Pedals) != 1 || res.Spec.Pedals[0].Min != 10 || res.Spec.Pedals[0].Max != 90 {
		t.Fatalf("pedals = %+v, want explicit Min 10 Max 90", res.Spec.Pedals)
	}
}

func TestDesignDefaultsCabForFenderAmp(t *testing.T) {
	d := NewDesigner(catalog.New())
	res, err := d.Design(Request{Name: "Clean", Amp: "65 Black SR"})
	if err != nil {
		t.Fatalf("Design: %v", err)
	}
	cabBlock := res.Spec.Blocks[1]
	if cabBlock.Params["CabType"] != "1x12 Black Panel Lux" {
		t.Fatalf("default cab = %v, want 1x12 Black Panel Lux", cabBlock.Params["CabType"])
	}
}

func TestDesignDefaultsOutputLevelToCompensateMaster(t *testing.T) {
	d := NewDesigner(catalog.New())
	res, err := d.Design(Request{Name: "Clean", Amp: "65 Black SR"})
	if err != nil {
		t.Fatalf("Design: %v", err)
	}
	// The amp master defaults to 50% (−6 dB); the designer defaults the output
	// to +6 dB so a fresh rig lands at unity.
	if res.Spec.OutputVolume != 6 {
		t.Fatalf("default output volume = %v, want 6", res.Spec.OutputVolume)
	}
}

func TestDesignRespectsExplicitOutputLevel(t *testing.T) {
	d := NewDesigner(catalog.New())
	level := -3.0
	res, err := d.Design(Request{Name: "Clean", Amp: "65 Black SR", OutputLevel: &level})
	if err != nil {
		t.Fatalf("Design: %v", err)
	}
	if res.Spec.OutputVolume != -3 {
		t.Fatalf("output volume = %v, want -3", res.Spec.OutputVolume)
	}
}

func TestDesignRequiresAmp(t *testing.T) {
	d := NewDesigner(catalog.New())
	if _, err := d.Design(Request{Name: "No Amp"}); err == nil {
		t.Fatal("expected error when amp is missing")
	}
}

func TestDesignRejectsUnknownFX(t *testing.T) {
	d := NewDesigner(catalog.New())
	_, err := d.Design(Request{Name: "x", Amp: "65 Black SR", FX: []FXBlock{{Type: "Bogus"}}})
	if err == nil {
		t.Fatal("expected error for unknown effect")
	}
}

func TestDesignRejectsUnknownDevice(t *testing.T) {
	d := NewDesigner(catalog.New())
	_, err := d.Design(Request{Name: "x", Amp: "65 Black SR", Device: "quad-cortex"})
	if err == nil {
		t.Fatal("expected an error for an unsupported device")
	}
}

func TestDesignDualAmpBuildsParallelPaths(t *testing.T) {
	d := NewDesigner(catalog.New())
	res, err := d.Design(Request{
		Name:    "Two Heads",
		Amp:     "65 Black SR",
		Amp2:    "67 Black Duo",
		Routing: rig.RoutingSPS,
		FX: []FXBlock{
			{Type: "Green JRC-OD", Enabled: true},
			{Type: "Eleven Reverb", Enabled: true},
		},
	})
	if err != nil {
		t.Fatalf("Design: %v", err)
	}

	spec := res.Spec
	if spec.Routing != rig.RoutingSPS {
		t.Fatalf("routing = %q, want SPS-1", spec.Routing)
	}
	assertBlockTypes(t, "prefix", spec.Prefix, []string{"Green JRC-OD"})
	assertBlockTypes(t, "path A", spec.PathA, []string{"Amp", "Cab"})
	assertBlockTypes(t, "path B", spec.PathB, []string{"Amp", "Cab"})
	assertBlockTypes(t, "suffix", spec.Suffix, []string{"Eleven Reverb"})

	ampB := spec.PathB[0]
	if ampB.Params["Type"] != "67 Black Duo" {
		t.Fatalf("path B amp type = %v, want 67 Black Duo", ampB.Params["Type"])
	}
}

// assertBlockTypes fails unless the section's blocks have exactly the wanted
// types in order.
func assertBlockTypes(t *testing.T, section string, blocks []rig.Block, want []string) {
	t.Helper()
	got := make([]string, len(blocks))
	for i, b := range blocks {
		got[i] = b.Type
	}
	if len(got) != len(want) {
		t.Fatalf("%s = %v, want %v", section, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s = %v, want %v", section, got, want)
		}
	}
}

func TestDesignSharedAmpUsesSingleAmpNode(t *testing.T) {
	d := NewDesigner(catalog.New())
	res, err := d.Design(Request{
		Name:    "Shared",
		Amp:     "65 Black SR",
		Routing: rig.RoutingSPS,
		PathAFX: []FXBlock{{Type: "Tape Echo", Enabled: true}},
		PathBFX: []FXBlock{{Type: "Eleven Reverb", Enabled: true}},
	})
	if err != nil {
		t.Fatalf("Design: %v", err)
	}
	spec := res.Spec
	if len(spec.Prefix) != 2 || spec.Prefix[0].Type != "Amp" || spec.Prefix[1].Type != "Cab" {
		t.Fatalf("prefix = %v, want [Amp Cab]", spec.Prefix)
	}
	if len(spec.PathA) != 1 || spec.PathA[0].Type != "Tape Echo" {
		t.Fatalf("path A = %v, want [Tape Echo]", spec.PathA)
	}
	if len(spec.PathB) != 1 || spec.PathB[0].Type != "Eleven Reverb" {
		t.Fatalf("path B = %v, want [Eleven Reverb]", spec.PathB)
	}
}

func TestDesignParallelRequiresSecondAmp(t *testing.T) {
	d := NewDesigner(catalog.New())
	if _, err := d.Design(Request{Name: "x", Amp: "65 Black SR", Routing: rig.RoutingPS}); err == nil {
		t.Fatal("expected error: PS-1 needs a second amp (amp2)")
	}
}

func notesMention(t *testing.T, notes []string, substr string) bool {
	t.Helper()
	for _, n := range notes {
		if strings.Contains(n, substr) {
			return true
		}
	}
	return false
}

func TestDesignHintsExpressionModuleNeedsFootswitch(t *testing.T) {
	d := NewDesigner(catalog.New())
	res, err := d.Design(Request{
		Name: "Whammy",
		Amp:  "65 Black SR",
		FX:   []FXBlock{{Type: "Wham", Enabled: true}},
	})
	if err != nil {
		t.Fatalf("Design: %v", err)
	}
	if !notesMention(t, res.Notes, "Wham") || !notesMention(t, res.Notes, "footswitch") {
		t.Fatalf("expected a footswitch hint for Wham, notes = %v", res.Notes)
	}
}

func TestDesignNoFootswitchHintWhenAssigned(t *testing.T) {
	d := NewDesigner(catalog.New())
	res, err := d.Design(Request{
		Name:         "Whammy Toe",
		Amp:          "65 Black SR",
		FX:           []FXBlock{{Type: "Wham", Enabled: true}},
		Footswitches: []rig.Footswitch{{Module: "Wham"}},
	})
	if err != nil {
		t.Fatalf("Design: %v", err)
	}
	if notesMention(t, res.Notes, "has no footswitch") {
		t.Fatalf("did not expect a footswitch hint when Wham is assigned, notes = %v", res.Notes)
	}
}

func TestDesignNoFootswitchHintForNonExpressionModule(t *testing.T) {
	d := NewDesigner(catalog.New())
	res, err := d.Design(Request{
		Name: "Boost",
		Amp:  "65 Black SR",
		FX:   []FXBlock{{Type: "Green JRC-OD", Enabled: true}},
	})
	if err != nil {
		t.Fatalf("Design: %v", err)
	}
	if notesMention(t, res.Notes, "has no footswitch") {
		t.Fatalf("did not expect a footswitch hint for a distortion, notes = %v", res.Notes)
	}
}
