package mooer

import (
	"strings"
	"testing"
)

// The classic GE150's .mo is the JSON document the GE150 Edit app writes, so
// the codec must round-trip a real template and resolve it to the ge150 model.
func TestGE150JSONRoundTripEmptyTemplate(t *testing.T) {
	m, _ := ModelByName("ge150")
	raw := readFixture(t, "ge150-empty.mo")

	first, err := UnmarshalMOFor(m, raw)
	if err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	// The template is an empty preset: every module off, knobs at their defaults.
	checks := []struct {
		name string
		got  any
		want any
	}{
		{"FX", first.FX, FX{Q: 50, Position: 50, Peak: 50, Level: 50}},
		{"Amp", first.Amp, Amp{Gain: 50, Bass: 50, Mid: 50, Treble: 50, Presence: 50, Master: 50}},
		{"Cab", first.Cab, Cab{Mic: 1, Center: 20, Distance: 35}}, // CEBTER typo tolerated
		{"Reverb", first.Reverb, Reverb{PreDelay: 80, Level: 25, Decay: 40, Tone: 40}},
		{"EQ bands", first.EQ.Bands, [6]uint8{50, 50, 50, 50, 50, 0}}, // 16 = flat
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Fatalf("%s = %+v, want %+v", c.name, c.got, c.want)
		}
	}
	if first.Delay.TimeMS != 600 || first.Delay.Subdivision != 1 {
		t.Fatalf("Delay = %+v, want TIME 600 SUB-D 1", first.Delay)
	}

	again, err := UnmarshalMOFor(m, MarshalMOFor(m, first))
	if err != nil {
		t.Fatalf("re-unmarshal: %v", err)
	}
	if again != first {
		t.Fatalf("round trip mismatch:\n got %+v\nwant %+v", again, first)
	}
}

// A GE150 design writes importable JSON: the schema, the device, the nine
// modules and the enabled modules' knobs.
func TestGE150JSONWriteDesign(t *testing.T) {
	m, _ := ModelByName("ge150")
	p := New()
	p.Name = "GE150 Tone"
	p.Amp = Amp{Enabled: true, Type: 6, Gain: 70, Bass: 50, Mid: 45, Treble: 60, Presence: 55, Master: 65}
	p.Cab = Cab{Enabled: true, Type: 5, Mic: 1, Center: 20, Distance: 35, Tube: 1}

	got := string(MarshalMOFor(m, p))
	for _, want := range []string{
		`"schema": "GE150 Preset"`,
		`"device": "MOOER GE150"`,
		`"TYPE": 6`,
		`"GAIN": 70`,
		`"CENTER": 20`,
		`"TUBE": 1`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("output missing %q:\n%s", want, got)
		}
	}
}

// DetectModel must resolve a GE150 JSON .mo to the ge150 model, not the
// GE150 Pro Li's binary record.
func TestDetectModelGE150JSON(t *testing.T) {
	det, err := DetectMOFile("testdata/ge150-empty.mo")
	if err != nil {
		t.Fatalf("DetectMOFile: %v", err)
	}
	if det.Model.Name != "ge150" {
		t.Fatalf("DetectMOFile = %q, want ge150", det.Model.Name)
	}
}
