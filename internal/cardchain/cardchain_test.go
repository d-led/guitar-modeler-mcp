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

// TestPaletteIsDarkSchemeOptIn guards the theming contract: setup cards embed
// only CSS and print on white paper, so CSS must keep the light palette even
// when the viewer's OS is dark. A host that renders on a dark canvas (the rig
// report) opts in by appending DarkSchemeCSS, which must flip the palette to
// dark and never leave the light slot background in place.
func TestPaletteIsDarkSchemeOptIn(t *testing.T) {
	if strings.Contains(CSS, "prefers-color-scheme") {
		t.Error("CSS alone must not change on a dark OS: light-only cards would lose contrast on white")
	}
	for _, want := range []string{
		"prefers-color-scheme: dark",
		"--cc-slot-bg:#2c2c2e",
		"--cc-par-bg:#1c1c1e",
		"--cc-badge:#636366",
	} {
		if !strings.Contains(DarkSchemeCSS, want) {
			t.Errorf("DarkSchemeCSS should set %q for dark canvases", want)
		}
	}
	if strings.Contains(DarkSchemeCSS, "#f4f4f4") {
		t.Error("dark palette must not keep the light slot background")
	}
}
