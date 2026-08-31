package qc

import (
	"math"
	"testing"

	"google.golang.org/protobuf/proto"
)

func TestBuildPresetLaysOutASerialChain(t *testing.T) {
	cat := mustCatalog(t)
	preset, err := BuildPreset(cat, DesignSpec{
		Name:   "JCM800 Rhythm",
		Author: "test",
		Blocks: []BlockSpec{
			{Model: "Ibanez TS808", Params: map[string]float64{"OVERDRIVE": 5}},
			{Model: "JCM800", Params: map[string]float64{"GAIN": 5, "OUTPUT": 0}},
			{Model: "Mesa Rectifier", Params: map[string]float64{}},
		},
	})
	if err != nil {
		t.Fatalf("BuildPreset: %v", err)
	}

	wantEq(t, "chains", len(preset.Chains), 1)
	chain := preset.Chains[0]
	wantEq(t, "in port", chain.GetInPortid(), InputInput1)
	wantEq(t, "out port", chain.GetOutPortid(), OutputMultiple)
	wantEq(t, "models", len(chain.Models), 3)
	// TS808 = drive id 5, JCM800 = amp id 1001; the cab resolves to category 12.
	wantEq(t, "block 0 hash", chain.Models[0].GetHash(), uint32(5))
	wantEq(t, "block 1 hash", chain.Models[1].GetHash(), uint32(1001))
	if got := chain.Models[2].GetHash(); got < 12000 || got >= 13000 {
		t.Fatalf("block 2 hash = %d, want a guitar cab (12xxx)", got)
	}
	wantEq(t, "output control count", len(chain.OutputControl), 1)
	wantEq(t, "output control hash", chain.OutputControl[0].GetHash(), uint32(laneOutputHash))
}

func TestBuildPresetEncodesGainLinearly(t *testing.T) {
	cat := mustCatalog(t)
	preset, err := BuildPreset(cat, DesignSpec{
		Name:   "Gain Check",
		Blocks: []BlockSpec{{Model: "JCM800", Params: map[string]float64{"GAIN": 5, "OUTPUT": 0}}},
	})
	if err != nil {
		t.Fatalf("BuildPreset: %v", err)
	}
	amp := preset.Chains[0].Models[0]
	byName := map[string]float64{}
	for _, p := range amp.Params {
		byName[paramName(cat, 1001, p.GetIndex())] = float64(p.ParamValues[0].GetFloatValue())
	}
	if got := byName["GAIN"]; math.Abs(got-0.5) > 1e-9 {
		t.Fatalf("GAIN 5 on a 0..10 knob = wire %g, want 0.5", got)
	}
	// OUTPUT 0 dB on the -60..+12 dB skew-3.8018 knob lands at wire 0.5 too.
	if got := byName["OUTPUT"]; math.Abs(got-0.5) > 1e-3 {
		t.Fatalf("OUTPUT 0 dB = wire %g, want ~0.5", got)
	}
}

func TestBuildPresetRejectsUnknownParameter(t *testing.T) {
	cat := mustCatalog(t)
	_, err := BuildPreset(cat, DesignSpec{
		Name:   "Bad",
		Blocks: []BlockSpec{{Model: "JCM800", Params: map[string]float64{"NOT A KNOB": 1}}},
	})
	if err == nil {
		t.Fatal("unknown parameter accepted, want error")
	}
}

func TestBuildPresetEncodesAndDecodesRoundTrip(t *testing.T) {
	cat := mustCatalog(t)
	want, err := BuildPreset(cat, DesignSpec{
		Name: "Round Trip",
		Blocks: []BlockSpec{
			{Model: "JCM800", Params: map[string]float64{"GAIN": 7, "MASTER": 6}},
			{Model: "Tape Delay (M)", Params: map[string]float64{"MIX": 35}},
		},
	})
	if err != nil {
		t.Fatalf("BuildPreset: %v", err)
	}
	data, err := EncodePreset("QA00XXXXX", want)
	if err != nil {
		t.Fatalf("EncodePreset: %v", err)
	}
	got, err := DecodePreset("QA00XXXXX", data)
	if err != nil {
		t.Fatalf("DecodePreset: %v", err)
	}
	if !proto.Equal(got, want) {
		t.Fatalf("round trip changed the preset:\n got %v\nwant %v", got, want)
	}
}

// paramName resolves a model's parameter at a wire index, for test assertions.
func paramName(cat *Catalog, modelID int, index uint32) string {
	m, ok := cat.Model(modelID)
	if !ok || int(index) >= len(m.Params) {
		return ""
	}
	return m.Params[index].Name
}
