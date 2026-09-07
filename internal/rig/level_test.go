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
	// Gain and Master both default to 50% = -6 dB each; everything else is 0.
	if est.EstimatedLevelDB != -12 {
		t.Fatalf("estimated = %v, want -12", est.EstimatedLevelDB)
	}
	if est.RecommendedRigVolume != 12 {
		t.Fatalf("recommended RigVolume = %v, want 12", est.RecommendedRigVolume)
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
	if est.EstimatedLevelDB != -6 {
		t.Fatalf("estimated = %v, want -6 (amp -12 + rigvolume +6)", est.EstimatedLevelDB)
	}
	if est.RecommendedRigVolume != 12 {
		t.Fatalf("recommended = %v, want 12 (reaching 0 from -6)", est.RecommendedRigVolume)
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
	// amp -12 dB (gain + master) + IR gain +6 dB (mix 100 = full wet) = -6 dB.
	if est.EstimatedLevelDB != -6 {
		t.Fatalf("estimated = %v, want -6 (amp -12 + IR gain +6)", est.EstimatedLevelDB)
	}
}

func TestEstimateLevelWithIRMixBlend(t *testing.T) {
	b := newTestBuilder(t)
	file, err := b.Build(Spec{
		Name: "Level",
		Blocks: []Block{
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			// Mix 0 = dry passthrough, so the IR contributes nothing; the
			// estimate stays at the amp's -12 dB (gain + master at 50% each).
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
	if est.EstimatedLevelDB != -12 {
		t.Fatalf("estimated = %v, want -12 (mix 0 passes dry, no IR gain)", est.EstimatedLevelDB)
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
	// amp -12 (gain + master) + mixer (max of -6/-6 = -6) = -18
	if est.EstimatedLevelDB != -18 {
		t.Fatalf("estimated = %v, want -18", est.EstimatedLevelDB)
	}
}

func TestEstimateLevelIncludesPreampGain(t *testing.T) {
	build := func(gain float64) (LevelEstimate, error) {
		b := newTestBuilder(t)
		file, err := b.Build(Spec{
			Name: "Level",
			Blocks: []Block{
				{Type: "Amp", Params: map[string]any{"Type": "65 Black SR", "GainA": gain}},
				{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
			},
		})
		if err != nil {
			t.Fatalf("Build: %v", err)
		}
		return EstimateLevel(file, 0)
	}

	clean, err := build(25)
	if err != nil {
		t.Fatalf("EstimateLevel (clean): %v", err)
	}
	driven, err := build(78)
	if err != nil {
		t.Fatalf("EstimateLevel (driven): %v", err)
	}
	if !(driven.EstimatedLevelDB > clean.EstimatedLevelDB) {
		t.Fatalf("preamp gain should raise the estimate: gain 25 = %v, gain 78 = %v", clean.EstimatedLevelDB, driven.EstimatedLevelDB)
	}

	found := false
	for _, s := range driven.Stages {
		if strings.Contains(s.Stage, "preamp gain") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a preamp-gain stage, got %v", driven.Stages)
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
