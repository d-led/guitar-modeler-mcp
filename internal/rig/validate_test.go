package rig

import (
	"strings"
	"testing"

	"github.com/d-led/guitar-modeler-mcp/internal/catalog"
)

func validate(t *testing.T, blocks []Block) error {
	t.Helper()
	b, err := NewBuilder(catalog.New())
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	_, err = b.Build(Spec{Name: "v", Blocks: blocks})
	return err
}

func validBase() []Block {
	return []Block{
		{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
		{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux", "MicType": "Dyn 57"}},
	}
}

func TestValidateRejectsUnknownAmpModel(t *testing.T) {
	err := validate(t, []Block{
		{Type: "Amp", Params: map[string]any{"Type": "Not A Real Amp"}},
		{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
	})
	if err == nil || !strings.Contains(err.Error(), "unknown amp model") {
		t.Fatalf("expected unknown amp model error, got %v", err)
	}
}

func TestValidateRejectsUnknownCabAndMic(t *testing.T) {
	if err := validate(t, []Block{
		{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
		{Type: "Cab", Params: map[string]any{"CabType": "Nope"}},
	}); err == nil || !strings.Contains(err.Error(), "unknown cabinet model") {
		t.Fatalf("expected unknown cabinet error, got %v", err)
	}
	if err := validate(t, []Block{
		{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
		{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux", "MicType": "Mic X"}},
	}); err == nil || !strings.Contains(err.Error(), "unknown microphone model") {
		t.Fatalf("expected unknown microphone error, got %v", err)
	}
}

func TestValidateRejectsOutOfRangeNumber(t *testing.T) {
	err := validate(t, append(validBase(), Block{Type: "White Boost", Params: map[string]any{"Gain": 150}}))
	if err == nil || !strings.Contains(err.Error(), "maximum") {
		t.Fatalf("expected out-of-range error, got %v", err)
	}
}

func TestValidateRejectsInvalidEnum(t *testing.T) {
	err := validate(t, append(validBase(), Block{Type: "Eleven Reverb", Params: map[string]any{"Mode": "Not A Mode"}}))
	if err == nil || !strings.Contains(err.Error(), "not a valid option") {
		t.Fatalf("expected invalid enum error, got %v", err)
	}
}

func TestValidateRejectsUnknownParamName(t *testing.T) {
	err := validate(t, append(validBase(), Block{Type: "White Boost", Params: map[string]any{"Bogus": 1}}))
	if err == nil || !strings.Contains(err.Error(), "no parameter") {
		t.Fatalf("expected unknown parameter error, got %v", err)
	}
}

func TestValidateAcceptsValidParams(t *testing.T) {
	if err := validate(t, append(validBase(),
		Block{Type: "White Boost", Enabled: true, Params: map[string]any{"Gain": 40, "Treble": 60}},
		Block{Type: "Tape Echo", Enabled: true, Params: map[string]any{"Feedback": 50, "Sync": true}},
	)); err != nil {
		t.Fatalf("expected valid params to build, got %v", err)
	}
}
