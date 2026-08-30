package cardchain

import (
	"strings"
	"testing"
)

func TestRenderNumberedChain(t *testing.T) {
	html := Render([]Step{
		{Slot: 1, Module: "BOOSTER", Effect: "T-SCREAM"},
		{Slot: 2, Module: "AMP", Effect: "CLEAN"},
		{Slot: 3, Effect: ""},
	})
	for _, want := range []string{"slotno\">1", "BOOSTER: T-SCREAM", "slotno\">2", "AMP: CLEAN", "slotno\">3", "→", "class=\"chain\""} {
		if !strings.Contains(html, want) {
			t.Errorf("chain missing %q: %s", want, html)
		}
	}
}

func TestRenderEmpty(t *testing.T) {
	if Render(nil) != "" {
		t.Fatal("Render(nil) should be empty")
	}
}

func TestRenderEscapes(t *testing.T) {
	html := Render([]Step{{Slot: 1, Effect: "A<B&C"}})
	if strings.Contains(html, "<B&") {
		t.Fatalf("effect not escaped: %s", html)
	}
}

func TestRenderParallelBranches(t *testing.T) {
	html := Render([]Step{
		{Slot: 1, Module: "BOOSTER", Effect: "T-SCREAM"},
		{Branches: []Branch{
			{Label: "A", Steps: []Step{{Slot: 4, Effect: "Amp"}, {Slot: 5, Effect: "Cab"}}},
			{Label: "B", Steps: []Step{{Slot: 7, Effect: "Amp 2"}, {Slot: 8, Effect: "Cab 2"}}},
		}},
		{Slot: 10, Effect: "Eleven Reverb"},
	})
	for _, want := range []string{
		"BOOSTER: T-SCREAM",
		"class=\"par\"",
		"parlabel\">A",
		"parlabel\">B",
		"slotno\">4",
		"slotno\">5",
		"slotno\">7",
		"slotno\">8",
		"slotno\">10",
		"Eleven Reverb",
		`aria-hidden="true">╫`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("parallel chain missing %q: %s", want, html)
		}
	}
}

func TestSlotNumbersAreCircles(t *testing.T) {
	if !strings.Contains(CSS, "border-radius:50%") {
		t.Error("the slot number badge should be a circle")
	}
}
