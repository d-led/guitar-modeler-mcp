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
