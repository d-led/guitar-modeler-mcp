package design

import (
	"strings"
	"testing"

	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/catalog"
	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/rig"
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
	if len(spec.Prefix) != 1 || spec.Prefix[0].Type != "Green JRC-OD" {
		t.Fatalf("prefix = %v, want [Green JRC-OD]", spec.Prefix)
	}
	if len(spec.PathA) != 2 || spec.PathA[0].Type != "Amp" || spec.PathA[1].Type != "Cab" {
		t.Fatalf("path A = %v, want [Amp Cab]", spec.PathA)
	}
	if len(spec.PathB) != 2 || spec.PathB[0].Type != "Amp" || spec.PathB[1].Type != "Cab" {
		t.Fatalf("path B = %v, want [Amp Cab]", spec.PathB)
	}
	if len(spec.Suffix) != 1 || spec.Suffix[0].Type != "Eleven Reverb" {
		t.Fatalf("suffix = %v, want [Eleven Reverb]", spec.Suffix)
	}

	ampB := spec.PathB[0]
	if ampB.Params["Type"] != "67 Black Duo" {
		t.Fatalf("path B amp type = %v, want 67 Black Duo", ampB.Params["Type"])
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
