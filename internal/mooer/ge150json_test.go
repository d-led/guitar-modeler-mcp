package mooer

import (
	"encoding/json"
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
		{"EQ bands", first.EQ.Bands, [6]uint8{50, 50, 50, 50, 50, 50}}, // 16 = flat; 6th band unused
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

// The classic GE150 numbers its mod and delay types differently from the
// GE200, so the catalog must match the GE150 Edit app, not the shared list.
func TestGE150EffectsCatalogMatchesDevice(t *testing.T) {
	m, _ := ModelByName("ge150")
	if got := len(m.Effects["mod"]); got != 19 {
		t.Fatalf("ge150 mod has %d entries, want 19", got)
	}
	if got := len(m.Effects["delay"]); got != 9 {
		t.Fatalf("ge150 delay has %d entries, want 9", got)
	}
	if got := m.EffectName("delay", 4); got != "MOD" {
		t.Fatalf("ge150 delay[4] = %q, want MOD", got)
	}
	if got := m.EffectName("mod", 18); got != "LOFI" {
		t.Fatalf("ge150 mod[18] = %q, want LOFI", got)
	}
}

// ge150Doc parses a marshalled preset back into its JSON form, so tests can
// assert on exact module keys rather than grepping the serialised text.
func ge150Doc(t *testing.T, data []byte) ge150PresetJSON {
	t.Helper()
	var doc ge150PresetJSON
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse GE150 JSON: %v", err)
	}
	return doc
}

// Every effect type writes its own knob keys: a MOD flanger stores MIX and
// FEEDBACK (not the phaser's LEVEL/DEPTH), the NOISE KILLER stores only THRES,
// and the DUAL DELAY stores the two TIME/SUB pairs.
func TestGE150JSONWritesPerEffectKnobs(t *testing.T) {
	m, _ := ModelByName("ge150")
	p := New()
	p.Mod = Mod{Enabled: true, Type: 3, Rate: 40, Level: 60, Depth: 70} // FLANGER
	p.NoiseGate = NoiseGate{Enabled: true, Type: 0, Attack: 10, Release: 20, Threshold: 30}
	p.Delay = Delay{Enabled: true, Type: 8, Level: 50, Feedback: 30, TimeMS: 600, Subdivision: 1, Param5: 200, Param6: 2} // DUAL DELAY

	doc := ge150Doc(t, MarshalMOFor(m, p))

	modData := doc.EffectModule[ge150ModMod].Data
	if _, ok := modData["DEPTH"]; ok {
		t.Fatalf("flanger MOD wrote the phaser's DEPTH key: %v", modData)
	}
	if modData["MIX"] != 60 || modData["FEEDBACK"] != 70 {
		t.Fatalf("flanger MOD data = %v, want MIX 60 FEEDBACK 70", modData)
	}

	nsData := doc.EffectModule[ge150ModNS].Data
	if _, ok := nsData["ATTACK"]; ok {
		t.Fatalf("noise killer wrote ATTACK: %v", nsData)
	}
	if nsData["THRES"] != 30 {
		t.Fatalf("noise killer THRES = %d, want 30", nsData["THRES"])
	}

	delayData := doc.EffectModule[ge150ModDelay].Data
	for key, want := range map[string]int{"TIME A": 600, "SUB A": 1, "TIME B": 200, "SUB B": 2} {
		if delayData[key] != want {
			t.Fatalf("dual delay %s = %d, want %d", key, delayData[key], want)
		}
	}
}

// A pitch shifter stores PITCH on -120..+120 with 0 at centre, so the shared
// 0-100 Rate of 50 lands on 0 and round-trips back to 50.
func TestGE150JSONPitchShiftRange(t *testing.T) {
	m, _ := ModelByName("ge150")
	p := New()
	p.Mod = Mod{Enabled: true, Type: 8, Rate: 50, Level: 50, Depth: 50} // PITCH SHIFT

	doc := ge150Doc(t, MarshalMOFor(m, p))
	if doc.EffectModule[ge150ModMod].Data["PITCH"] != 0 {
		t.Fatalf("pitch centre = %d, want 0", doc.EffectModule[ge150ModMod].Data["PITCH"])
	}

	back, err := UnmarshalMOFor(m, MarshalMOFor(m, p))
	if err != nil {
		t.Fatal(err)
	}
	if back.Mod.Rate != 50 {
		t.Fatalf("pitch round trip Rate = %d, want 50", back.Mod.Rate)
	}
}

// The six-band G-6 writes all six bands, including 3.2KHz, and reads them back.
func TestGE150JSONSixBandEQ(t *testing.T) {
	m, _ := ModelByName("ge150")
	p := New()
	p.EQ = EQ{Enabled: true, Type: 2, Bands: [6]uint8{50, 50, 50, 50, 50, 75}} // MOOER G-6

	doc := ge150Doc(t, MarshalMOFor(m, p))
	if doc.EffectModule[ge150ModEQ].Data["3.2KHz"] != 24 { // 75/100 x 32
		t.Fatalf("G-6 3.2KHz = %d, want 24", doc.EffectModule[ge150ModEQ].Data["3.2KHz"])
	}

	back, err := UnmarshalMOFor(m, MarshalMOFor(m, p))
	if err != nil {
		t.Fatal(err)
	}
	if back.EQ.Bands[5] != 75 {
		t.Fatalf("G-6 sixth band = %d, want 75", back.EQ.Bands[5])
	}
}

// The custom parametric EQ stores gain/frequency pairs; our model keeps the
// gains, fixes the frequencies at the documented default, and drops the extra
// bands on read.
func TestGE150JSONCustomEQGainsOnly(t *testing.T) {
	m, _ := ModelByName("ge150")
	p := New()
	p.EQ = EQ{Enabled: true, Type: 3, Bands: [6]uint8{50, 75, 25, 10, 20, 30}} // CUSTOM EQ

	doc := ge150Doc(t, MarshalMOFor(m, p))
	eqData := doc.EffectModule[ge150ModEQ].Data
	for _, key := range []string{"GAIN1", "FREQ1", "GAIN2", "FREQ2", "GAIN3", "FREQ3"} {
		if _, ok := eqData[key]; !ok {
			t.Fatalf("custom EQ output missing %s: %v", key, eqData)
		}
	}

	back, err := UnmarshalMOFor(m, MarshalMOFor(m, p))
	if err != nil {
		t.Fatal(err)
	}
	want := [6]uint8{50, 75, 25, 50, 50, 50}
	if back.EQ.Bands != want {
		t.Fatalf("custom EQ round trip = %v, want %v", back.EQ.Bands, want)
	}
}
