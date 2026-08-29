package params

import (
	"testing"

	"github.com/d-led/guitar-modeler-mcp/internal/catalog"
)

func has(caps []string, want string) bool {
	for _, c := range caps {
		if c == want {
			return true
		}
	}
	return false
}

func TestCapabilitiesDiscoverPitchShift(t *testing.T) {
	cat := catalog.New()
	if !has(Capabilities(cat, "Pitch Delay"), "pitch shift") {
		t.Fatalf("Pitch Delay capabilities = %v, want pitch shift", Capabilities(cat, "Pitch Delay"))
	}
	if !has(Capabilities(cat, "Pitch Delay"), "delay") {
		t.Fatalf("Pitch Delay capabilities = %v, want delay", Capabilities(cat, "Pitch Delay"))
	}
}

func TestCapabilitiesReverbDriveTremolo(t *testing.T) {
	cat := catalog.New()
	if !has(Capabilities(cat, "Eleven Reverb"), "reverb") {
		t.Fatalf("Eleven Reverb missing reverb capability")
	}
	if !has(Capabilities(cat, "Green JRC-OD"), "drive") {
		t.Fatalf("Green JRC-OD missing drive capability")
	}
	if !has(Capabilities(cat, "Amp"), "tremolo") {
		t.Fatalf("Amp missing tremolo capability")
	}
}
