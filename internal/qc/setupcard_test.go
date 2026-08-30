package qc

import (
	"strings"
	"testing"
)

func TestSetupCardHTMLRendersChain(t *testing.T) {
	cat := mustCatalog(t)
	preset, err := BuildPreset(cat, DesignSpec{
		Name:   "Card Tone",
		Author: "tester",
		Blocks: []BlockSpec{
			{Model: "JCM800", Params: map[string]float64{"GAIN": 5, "BASS": 3}},
			{Model: "Tape Delay (M)", Params: map[string]float64{"MIX": 35}},
		},
	})
	if err != nil {
		t.Fatalf("BuildPreset: %v", err)
	}

	card := SetupCardHTML(cat, preset)
	for _, want := range []string{
		"Card Tone",
		"tester",
		"Marshall JCM800",
		"Tape Delay (M)",
		"GAIN",
		"BASS",
		"MIX",
		"setup card",
	} {
		if !strings.Contains(card, want) {
			t.Errorf("setup card missing %q", want)
		}
	}
	// The honest framing is always printed.
	if !strings.Contains(card, "not a file the Quad Cortex imports") {
		t.Error("setup card missing the .pb-is-a-reference-archive note")
	}
}

func TestSetupCardHTMLShowsRealValues(t *testing.T) {
	cat := mustCatalog(t)
	preset, err := BuildPreset(cat, DesignSpec{
		Name:   "Values",
		Blocks: []BlockSpec{{Model: "JCM800", Params: map[string]float64{"GAIN": 5, "OUTPUT": 0}}},
	})
	if err != nil {
		t.Fatalf("BuildPreset: %v", err)
	}
	card := SetupCardHTML(cat, preset)
	// GAIN 5 on a 0..10 knob reads "5" in its own units; OUTPUT 0 reads "0 dB".
	if !strings.Contains(card, "GAIN: 5") {
		t.Error("setup card should show GAIN: 5 in screen units")
	}
	if !strings.Contains(card, "OUTPUT: 0 dB") {
		t.Error("setup card should show OUTPUT: 0 dB")
	}
}

func TestSetupCardIsSelfContained(t *testing.T) {
	cat := mustCatalog(t)
	// No parameter overrides at all: the card must still be a complete
	// instruction card, showing the chain and every knob's default value.
	preset, err := BuildPreset(cat, DesignSpec{
		Name:   "Defaults Only",
		Blocks: []BlockSpec{{Model: "JCM800"}, {Model: "Tape Delay (M)"}},
	})
	if err != nil {
		t.Fatalf("BuildPreset: %v", err)
	}

	card := SetupCardHTML(cat, preset)
	// The signal chain is shown as a slot-numbered hint.
	for _, want := range []string{"slotno\">1", "Marshall JCM800", "slotno\">2", "Tape Delay (M)", "class=\"chain\""} {
		if !strings.Contains(card, want) {
			t.Errorf("setup card missing chain element %q", want)
		}
	}
	// Each block's parameter table repeats the circled slot number, so the
	// values correlate with the chain hint above.
	for _, want := range []string{"slotbadge\">1</span>Marshall JCM800", "slotbadge\">2</span>Tape Delay (M)"} {
		if !strings.Contains(card, want) {
			t.Errorf("setup card missing block slot badge %q", want)
		}
	}
	// Unset knobs fall back to their catalog defaults, so the amp's GAIN
	// (default 5) is printed even though the preset never set it.
	if !strings.Contains(card, "GAIN: 5") {
		t.Error("setup card should show the default GAIN: 5 for an unset amp")
	}
	// The delay's knobs are present too (its default mix, e.g. MIX: 35 is not
	// required — but the knob name must be there).
	if !strings.Contains(card, "MIX:") || !strings.Contains(card, "FEEDBACK:") {
		t.Error("setup card should list the delay's knobs with defaults")
	}
}

func TestFormatRealShowsEndpointLabels(t *testing.T) {
	spec := ParamSpec{Name: "OUTPUT", Units: "dB", Min: -60, Max: 12, MinLabel: "OFF"}
	if got := formatReal(spec, -60); got != "OFF" {
		t.Errorf("formatReal at min = %q, want OFF", got)
	}
	if got := formatReal(spec, 0); got != "0 dB" {
		t.Errorf("formatReal(0) = %q, want 0 dB", got)
	}
}
