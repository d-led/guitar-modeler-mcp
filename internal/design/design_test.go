package design

import (
	"testing"

	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/catalog"
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
