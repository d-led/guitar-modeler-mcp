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
	// The honest limitation is always printed.
	if !strings.Contains(card, "not yet confirmed on hardware") {
		t.Error("setup card missing the hardware-verification caveat")
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
