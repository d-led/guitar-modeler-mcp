package mooer

import (
	"fmt"
	"math"
	"sort"
	"strings"
)

// ValidateSpec checks a design against the knobs a GE100 Pro really stores,
// before BuildPreset applies a single value. A knob the model has not got, a
// value outside the range its knob covers, or an EQ model the shared preset
// cannot express is reported instead of written - clamping whatever arrives is
// how a preset reaches the device asking for +34 dB on every EQ band.
func (ge100ProCodec) ValidateSpec(m Model, s Spec) error {
	return validateGE100ProSpec(m, s)
}

func validateGE100ProSpec(m Model, s Spec) error {
	if err := validateGE100ProModule(m, "amp", s.Amp, s.AmpParams); err != nil {
		return err
	}
	if err := validateGE100ProCabParams(s); err != nil {
		return err
	}
	for _, f := range s.FX {
		if err := validateGE100ProModule(m, normalizeModule(f.Module), f.Type, f.Params); err != nil {
			return err
		}
	}
	return nil
}

// validateGE100ProCabParams refuses cab knobs rather than dropping them: the
// device's cab block carries LOW CUT / HIGH CUT in Hz and ROOM, and the shared
// preset has a field for none of them.
func validateGE100ProCabParams(s Spec) error {
	if len(s.CabParams) == 0 {
		return nil
	}
	return fmt.Errorf("ge100pro cab: the device's cab block carries LOW CUT / HIGH CUT (Hz) and ROOM, which the shared preset has no field for, so %s cannot be set here; dial the cab on the device",
		strings.Join(sortedParamKeys(s.CabParams), ", "))
}

// validateGE100ProModule checks one module's knobs against the model the spec
// names, so a knob the model does not have is a message rather than a value
// written into the next model's knob.
func validateGE100ProModule(m Model, module, effect string, params Params) error {
	params = normalizeParams(params)
	if len(params) == 0 {
		return nil
	}
	def, ok := ge100ProModules[module]
	if !ok {
		return fmt.Errorf("ge100pro has no %q module", module)
	}
	model, ok := m.EffectIndex(module, effect)
	if !ok {
		// An unknown model is BuildPreset's to report: the catalog knows the
		// names the device publishes.
		return nil
	}
	name := m.EffectName(module, model)
	if module == "eq" && ge100ProTableAt(ge100ProEQModels, model).parametric {
		return fmt.Errorf("ge100pro eq %q alternates gain and frequency knobs (Gain 1, Freq 1, ...) and the shared preset carries no frequency; use a graphic EQ model or dial it on the device", name)
	}

	knobs := def.knobsOf(model)
	for _, key := range sortedParamKeys(params) {
		if !knobKnown(knobs, key) {
			return fmt.Errorf("ge100pro %s %q has no knob %q (knobs: %s)", module, name, key, knobKeys(knobs))
		}
		if err := checkGE100ProValue(module, name, key, params[key]); err != nil {
			return err
		}
	}
	return nil
}

// ge100ProKnobRange is the range one knob covers on the shared preset's own
// 0-100 scale.
type ge100ProKnobRange struct {
	min, max float64
	// zeroMeansUnset widens the range to include zero, the shared preset's way
	// of leaving a knob alone, so the device keeps its own default.
	zeroMeansUnset bool
}

// ge100ProKnobRanges narrows the few knobs whose device range is smaller than
// the shared 0-100 scale. Every other knob accepts the whole scale.
var ge100ProKnobRanges = map[string]ge100ProKnobRange{
	// The delay's SUB-D knob selects one of ten note values.
	"subdivision": {min: 0, max: 9},
	// The reverb's pre-delay is milliseconds, and its knob stops at 500.
	"pre_delay": {min: 0, max: 500},
	// The delay's TIME is milliseconds: the device's knob starts at 40 ms, and
	// zero leaves the device's own default in place.
	"time_ms": {min: 40, max: 2500, zeroMeansUnset: true},
}

// ge100ProSharedRange is the range of every knob without a narrower one.
var ge100ProSharedRange = ge100ProKnobRange{min: 0, max: 100}

func (r ge100ProKnobRange) allows(v float64) bool {
	if v == 0 && r.zeroMeansUnset {
		return true
	}
	return v >= r.min && v <= r.max
}

func (r ge100ProKnobRange) String() string {
	if r.zeroMeansUnset {
		return fmt.Sprintf("%g..%g (or 0 to leave the device's default)", r.min, r.max)
	}
	return fmt.Sprintf("%g..%g", r.min, r.max)
}

// checkGE100ProValue refuses a value the device's knob cannot hold. Every knob
// counts whole steps, so a fraction would otherwise be truncated in silence.
func checkGE100ProValue(module, effect, key string, v float64) error {
	limit := ge100ProSharedRange
	if narrower, ok := ge100ProKnobRanges[key]; ok {
		limit = narrower
	}
	if !limit.allows(v) {
		return fmt.Errorf("ge100pro %s %q knob %q value %g is outside %s", module, effect, key, v, limit)
	}
	if v != math.Trunc(v) {
		return fmt.Errorf("ge100pro %s %q knob %q value %g is not a whole number; the device stores whole steps", module, effect, key, v)
	}
	return nil
}

func knobKnown(knobs []ge100ProKnob, key string) bool {
	for _, knob := range knobs {
		if knob.key == key {
			return true
		}
	}
	return false
}

// knobKeys names a model's knobs for an error message.
func knobKeys(knobs []ge100ProKnob) string {
	keys := make([]string, 0, len(knobs))
	for _, knob := range knobs {
		keys = append(keys, knob.key)
	}
	if len(keys) == 0 {
		return "none"
	}
	return strings.Join(keys, ", ")
}

// sortedParamKeys lists a spec's knobs in a stable order, so a design carrying
// two bad knobs always reports the same one.
func sortedParamKeys(params Params) []string {
	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
