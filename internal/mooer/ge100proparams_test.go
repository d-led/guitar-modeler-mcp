package mooer

import (
	"slices"
	"strings"
	"testing"
)

// ge100ProDesign designs a tone the way mooer_design does, with an amp so the
// spec is a preset the device would load.
func ge100ProDesign(t *testing.T, spec Spec) (Preset, error) {
	t.Helper()
	m, ok := ModelByName("ge100pro")
	if !ok {
		t.Fatal("ge100pro not registered")
	}
	if spec.Amp == "" {
		spec.Amp = "MARKV DS"
	}
	return m.BuildPreset(spec)
}

// ge100ProSlotOf renders a designed tone and returns the knobs the device
// stores for one module.
func ge100ProSlotOf(t *testing.T, spec Spec, module string) [ge100ProParamCount]uint16 {
	t.Helper()
	p, err := ge100ProDesign(t, spec)
	if err != nil {
		t.Fatalf("designing the tone failed: %v", err)
	}
	m, _ := ModelByName("ge100pro")
	file, err := parseGE100ProFile(MarshalMOFor(m, p))
	if err != nil {
		t.Fatalf("our output is not a frame dump: %v", err)
	}
	body, err := file.body()
	if err != nil {
		t.Fatalf("our output carries no preset body: %v", err)
	}
	kind, ok := ge100ProModuleKind(module)
	if !ok {
		t.Fatalf("the device has no %s module", module)
	}
	for _, slot := range body.Slots {
		if !slot.empty() && slot.Kind == kind {
			return slot.Params
		}
	}
	t.Fatalf("the tone holds no %s slot", module)
	return [ge100ProParamCount]uint16{}
}

func wantGE100ProRefusal(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Fatalf("the design was accepted, want a message naming %q", want)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("err = %q, want it to mention %q", err, want)
	}
}

// An EQ band is stored in decibels, not on the shared preset's scale: the
// editor's tables give every band knob -16..+16 dB and 0 dB is 16 on the wire.
// The shared 50 is flat, so writing it verbatim asked the device for +34 dB -
// which pins every band at +16 dB, a flat boost across the spectrum instead of
// the curve that was designed.
func TestGE100ProEQBandIsWrittenInDB(t *testing.T) {
	knobs := ge100ProSlotOf(t, Spec{
		Name: "EQ SCALE",
		FX: []FXSpec{{
			Module: "eq", Type: "Mooer HM", Enabled: true,
			Params: Params{"band1": 100, "band2": 60, "band3": 50, "band4": 30, "band5": 0},
		}},
	}, "eq")

	want := []uint16{26, 18, 16, 12, 6} // +10, +2, flat, -4, -10 dB
	if got := knobs[:len(want)]; !slices.Equal(got, want) {
		t.Fatalf("EQ band knobs = %v, want %v", got, want)
	}
	if knobs[len(want)] != 0 {
		t.Fatalf("the sixth knob = %d, want 0: Mooer HM has five bands", knobs[len(want)])
	}
}

// A band is stored in whole decibels, so a preset that goes out and comes back
// keeps the curve: 60 on the shared scale is +2 dB and reads as 60 again.
func TestGE100ProEQBandsRoundTrip(t *testing.T) {
	m, _ := ModelByName("ge100pro")
	want := ge100ProBuilt(t, Spec{
		Name: "EQ ROUND TRIP",
		FX: []FXSpec{{
			Module: "eq", Type: "Mooer G-6", Enabled: true,
			Params: Params{"band1": 60, "band2": 45, "band3": 50, "band4": 55, "band5": 40, "band6": 65},
		}},
	})

	got, err := UnmarshalMOFor(m, MarshalMOFor(m, want))
	if err != nil {
		t.Fatalf("reading our own preset failed: %v", err)
	}
	if got.EQ.Bands != want.EQ.Bands {
		t.Fatalf("EQ bands = %v, want %v", got.EQ.Bands, want.EQ.Bands)
	}
}

// ge100ProBuilt is ge100ProDesign for the tests that must not fail.
func ge100ProBuilt(t *testing.T, spec Spec) Preset {
	t.Helper()
	p, err := ge100ProDesign(t, spec)
	if err != nil {
		t.Fatalf("designing the tone failed: %v", err)
	}
	return p
}

// The model decides how many bands the preset may carry, so a band the model has
// no knob for is refused rather than written into whatever knob comes next.
func TestGE100ProRejectsAnEQBandTheModelHasNotGot(t *testing.T) {
	_, err := ge100ProDesign(t, Spec{
		Name: "SIX BANDS",
		FX: []FXSpec{{
			Module: "eq", Type: "Mooer HM", Enabled: true,
			Params: Params{"band6": 60},
		}},
	})

	wantGE100ProRefusal(t, err, `eq "Mooer HM" has no knob "band6"`)
}

// Custom EQ's knobs alternate gains with frequencies, and the shared preset
// carries no frequency: the model is refused with the reason instead of being
// written with three knobs left at nothing.
func TestGE100ProRejectsAParametricEQModel(t *testing.T) {
	_, err := ge100ProDesign(t, Spec{
		Name: "PARAMETRIC",
		FX:   []FXSpec{{Module: "eq", Type: "Custom EQ", Enabled: true, Params: Params{"band1": 60}}},
	})

	wantGE100ProRefusal(t, err, "alternates gain and frequency knobs")
}

// A knob the model has not got, and the cab knobs the device stores in Hz, are
// messages rather than values written into knobs the caller never named.
func TestGE100ProRejectsKnobsTheDeviceHasNotGot(t *testing.T) {
	for _, tc := range []struct {
		name string
		spec Spec
		want string
	}{
		{
			"a knob of another module",
			Spec{Name: "WRONG MODULE", FX: []FXSpec{{Module: "od", Type: "808", Enabled: true, Params: Params{"decay": 40}}}},
			`od "808" has no knob "decay"`,
		},
		{
			"a cab knob the device stores in Hz",
			Spec{Name: "CAB KNOB", Cab: "CT-BOG OS 412", CabParams: Params{"mic": 3}},
			"the device's cab block carries LOW CUT",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ge100ProDesign(t, tc.spec)
			wantGE100ProRefusal(t, err, tc.want)
		})
	}
}

// A knob runs off the end of its range on three modules, and a value there is
// refused rather than clamped onto the knob.
func TestGE100ProRejectsValuesOutsideTheDevicesRange(t *testing.T) {
	for _, tc := range []struct {
		name string
		spec Spec
		want string
	}{
		{
			"an amp knob past its maximum",
			Spec{Name: "LOUD", AmpParams: Params{"gain": 120}},
			`knob "gain" value 120 is outside 0..100`,
		},
		{
			"a delay time below the knob's minimum",
			Spec{Name: "SHORT", FX: []FXSpec{{Module: "delay", Type: "TAPE", Enabled: true, Params: Params{"time_ms": 10}}}},
			`knob "time_ms" value 10 is outside 40..2500`,
		},
		{
			"a subdivision the device has not got",
			Spec{Name: "SUB", FX: []FXSpec{{Module: "delay", Type: "TAPE", Enabled: true, Params: Params{"subdivision": 20}}}},
			`knob "subdivision" value 20 is outside 0..9`,
		},
		{
			"a reverb pre-delay past the knob",
			Spec{Name: "LONG", FX: []FXSpec{{Module: "reverb", Type: "HALL", Enabled: true, Params: Params{"pre_delay": 600}}}},
			`knob "pre_delay" value 600 is outside 0..500`,
		},
		{
			"a fraction of a step",
			Spec{Name: "FRACTION", AmpParams: Params{"gain": 60.5}},
			"value 60.5 is not a whole number",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ge100ProDesign(t, tc.spec)
			wantGE100ProRefusal(t, err, tc.want)
		})
	}
}

// A design that names only the knobs it cares about is still accepted: the
// device's own default stands where the shared preset has nothing to say.
func TestGE100ProAcceptsADelayWithNoTime(t *testing.T) {
	if _, err := ge100ProDesign(t, Spec{
		Name: "NO TIME",
		FX:   []FXSpec{{Module: "delay", Type: "TAPE", Enabled: true, Params: Params{"level": 20}}},
	}); err != nil {
		t.Fatalf("a delay without a time was refused: %v", err)
	}
}

// The knob tables are indexed by the model a preset stores, so their order has
// to be the catalog's: a table that drifts converts the wrong model's knobs.
func TestGE100ProKnobTablesLineUpWithTheCatalog(t *testing.T) {
	m, _ := ModelByName("ge100pro")
	for _, tc := range []struct {
		module string
		model  string
		knobs  int
	}{
		{"eq", "3-Band EQ", 3},
		{"eq", "Mooer G", 5},
		{"eq", "Mooer HM", 5},
		{"eq", "Mooer G-6", 6},
		{"eq", "Mooer B", 5},
		// Custom EQ's three controls that are gains; the frequencies between
		// them are not bands.
		{"eq", "Custom EQ", 3},
		{"ns", "NOISE KILLER", 1},
		{"ns", "INTEL REDUCER", 1},
		{"ns", "NOISE GATE", 3},
	} {
		index, ok := m.EffectIndex(tc.module, tc.model)
		if !ok {
			t.Fatalf("%s %q is not in the catalog", tc.module, tc.model)
		}
		if got := len(ge100ProModules[tc.module].knobsOf(index)); got != tc.knobs {
			t.Fatalf("%s %q (model %d) carries %d knobs, want %d", tc.module, tc.model, index, got, tc.knobs)
		}
	}
}

// The gate's models carry different knobs and NOISE GATE stores the threshold
// last, so the knobs have to follow the model rather than one fixed order.
func TestGE100ProNoiseGateFollowsTheModelsKnobOrder(t *testing.T) {
	gate := ge100ProSlotOf(t, Spec{
		Name: "GATE ORDER",
		FX: []FXSpec{{
			Module: "ns", Type: "NOISE GATE", Enabled: true,
			Params: Params{"attack": 90, "release": 20, "threshold": 40},
		}},
	}, "ns")

	want := []uint16{90, 20, 40} // attack, release, threshold
	if got := gate[:len(want)]; !slices.Equal(got, want) {
		t.Fatalf("NOISE GATE knobs = %v, want %v", got, want)
	}
}

// A one-knob gate model takes the threshold and nothing else: the knobs it has
// not got stay empty rather than inheriting the shared preset's neutral values.
func TestGE100ProSingleKnobGateWritesOneKnob(t *testing.T) {
	gate := ge100ProSlotOf(t, Spec{
		Name: "ONE KNOB",
		FX:   []FXSpec{{Module: "ns", Type: "INTEL REDUCER", Enabled: true, Params: Params{"threshold": 27}}},
	}, "ns")

	if want := []uint16{27, 0, 0}; !slices.Equal(gate[:len(want)], want) {
		t.Fatalf("INTEL REDUCER knobs = %v, want %v", gate[:len(want)], want)
	}
}
