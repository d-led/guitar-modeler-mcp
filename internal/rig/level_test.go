package rig

import (
	"strings"
	"testing"
)

func TestEstimateLevelDefaultSerialRig(t *testing.T) {
	b := newTestBuilder(t)
	file, err := b.Build(Spec{
		Name: "Level",
		Blocks: []Block{
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	est, err := EstimateLevel(file, 0)
	if err != nil {
		t.Fatalf("EstimateLevel: %v", err)
	}
	// Master defaults to 50% = -6 dB; everything else is 0.
	if est.EstimatedLevelDB != -6 {
		t.Fatalf("estimated = %v, want -6", est.EstimatedLevelDB)
	}
	if est.RecommendedRigVolume != 6 {
		t.Fatalf("recommended RigVolume = %v, want 6", est.RecommendedRigVolume)
	}
}

func TestEstimateLevelWithOutputVolume(t *testing.T) {
	b := newTestBuilder(t)
	file, err := b.Build(Spec{
		Name:         "Level",
		OutputVolume: 6,
		Blocks: []Block{
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	est, err := EstimateLevel(file, 0)
	if err != nil {
		t.Fatalf("EstimateLevel: %v", err)
	}
	if est.EstimatedLevelDB != 0 {
		t.Fatalf("estimated = %v, want 0 (master -6 + rigvolume +6)", est.EstimatedLevelDB)
	}
	if est.RecommendedRigVolume != 6 {
		t.Fatalf("recommended = %v, want 6 (already at target)", est.RecommendedRigVolume)
	}
}

func TestEstimateLevelWithIR(t *testing.T) {
	b := newTestBuilder(t)
	file, err := b.Build(Spec{
		Name: "Level",
		Blocks: []Block{
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "IR", Params: map[string]any{"IR": "[directory](York)[name](Mix 01)", "Gain": 6.0}},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	est, err := EstimateLevel(file, 0)
	if err != nil {
		t.Fatalf("EstimateLevel: %v", err)
	}
	// amp master -6 dB + IR gain +6 dB (mix 100 = full wet) = 0 dB.
	if est.EstimatedLevelDB != 0 {
		t.Fatalf("estimated = %v, want 0 (amp master -6 + IR gain +6)", est.EstimatedLevelDB)
	}
}

func TestEstimateLevelWithIRMixBlend(t *testing.T) {
	b := newTestBuilder(t)
	file, err := b.Build(Spec{
		Name: "Level",
		Blocks: []Block{
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			// Mix 0 = dry passthrough, so the IR contributes nothing; mix 100 of
			// a 0 dB IR is also unity. Either way the estimate stays at the amp
			// master's -6 dB.
			{Type: "IR", Params: map[string]any{"IR": "[directory](York)[name](Mix 01)", "Mix": 0.0}},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	est, err := EstimateLevel(file, 0)
	if err != nil {
		t.Fatalf("EstimateLevel: %v", err)
	}
	if est.EstimatedLevelDB != -6 {
		t.Fatalf("estimated = %v, want -6 (mix 0 passes dry, no IR gain)", est.EstimatedLevelDB)
	}
}

func TestEstimateLevelParallelRig(t *testing.T) {
	b := newTestBuilder(t)
	file, err := b.Build(Spec{
		Name:    "Level",
		Routing: RoutingSPS,
		Prefix:  []Block{ampBlock("65 Black SR"), cabBlock("1x12 Black Panel Lux")},
		PathA:   []Block{{Type: "Tape Echo", Enabled: true}},
		PathB:   []Block{{Type: "Eleven Reverb", Enabled: true}},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	est, err := EstimateLevel(file, 0)
	if err != nil {
		t.Fatalf("EstimateLevel: %v", err)
	}
	if est.Routing != "SPS-1" {
		t.Fatalf("routing = %q", est.Routing)
	}
	// amp master -6 + mixer (max of -6/-6 = -6) = -12
	if est.EstimatedLevelDB != -12 {
		t.Fatalf("estimated = %v, want -12", est.EstimatedLevelDB)
	}
}

func TestEstimateLevelNotesPreampGainBlindSpot(t *testing.T) {
	b := newTestBuilder(t)
	file, err := b.Build(Spec{
		Name: "Level",
		Blocks: []Block{
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR", "GainA": 78.0}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	est, err := EstimateLevel(file, 0)
	if err != nil {
		t.Fatalf("EstimateLevel: %v", err)
	}
	// The amp preamp gain is deliberately excluded from the dB sum; the estimate
	// must say so rather than pretending the number is an absolute loudness.
	if len(est.Notes) == 0 {
		t.Fatal("expected a note flagging the preamp-gain blind spot, got none")
	}
	if !strings.Contains(est.Notes[0], "preamp gain") {
		t.Fatalf("note = %q, want it to mention the preamp gain", est.Notes[0])
	}
}

func TestBuildRefusesVeryLoudRig(t *testing.T) {
	b := newTestBuilder(t)
	_, err := b.Build(Spec{
		Name:         "Too Loud",
		InputGain:    12,
		OutputVolume: 20,
		Blocks: []Block{
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR", "Master": 100.0}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux", "OutGain": 12.0}},
		},
	})
	if err == nil {
		t.Fatal("expected the plausibility check to refuse a very loud rig")
	}
	if !strings.Contains(err.Error(), "very loud") {
		t.Fatalf("expected a 'very loud' plausibility error, got: %v", err)
	}
}

func TestBuildRefusesMutedRig(t *testing.T) {
	b := newTestBuilder(t)
	_, err := b.Build(Spec{
		Name: "Muted",
		Blocks: []Block{
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR", "Master": 0.0}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
		},
	})
	if err == nil {
		t.Fatal("expected the plausibility check to refuse a muted rig")
	}
	if !strings.Contains(err.Error(), "muted") {
		t.Fatalf("expected a 'muted' plausibility error, got: %v", err)
	}
}
