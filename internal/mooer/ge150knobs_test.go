package mooer

import (
	"strings"
	"testing"
)

// TestGE150KnobListPerEffectType pins which knobs every effect_type of a
// per-type module carries. An entry's position in ge150Knobs IS the effect_type
// the device expects, so an inserted, dropped or swapped list silently retypes
// the presets the writer produces — and the shared lists make that easy to get
// wrong by hand. These names are the GE150 Edit app's preset.json (see
// ge150knobs.go); regenerate them with the app's data, never from the table.
func TestGE150KnobListPerEffectType(t *testing.T) {
	want := map[string][]string{
		"FX/COMP": {
			"Q POSITION PEAK LEVEL",
			"Q POSITION PEAK LEVEL",
			"RATE RANGE PEAK LEVEL",
			"RATE RANGE PEAK LEVEL",
			"RATE RANGE PEAK LEVEL",
			"ATTACK SENS PEAK LEVEL",
			"ATTACK THRES RATIO LEVEL",
			"ATTACK THRES RATIO LEVEL",
		},
		"NS GATE": {
			"THRES",
			"SENS",
			"ATTACK RELEASE THRES",
		},
		"EQ": {
			"100Hz 250Hz 630Hz 1.6KHz 4KHz",
			"80Hz 240Hz 750Hz 2.2KHz 6.6KHz",
			"100Hz 200Hz 400Hz 800Hz 1.6KHz 3.2KHz",
		},
		"MOD": {
			"RATE LEVEL DEPTH",
			"RATE LEVEL DEPTH",
			"RATE LEVEL DEPTH",
			"RATE MIX FEEDBACK",
			"RATE MIX FEEDBACK",
			"RATE MIX TONE",
			"RATE MIX TONE",
			"RATE DEPTH TONE",
			"PITCH MIX TONE",
			"PITCH MIX TONE",
			"RATE MIX TONE",
			"RATE MIX TONE DEPTH",
			"RATE MIX TONE DEPTH",
			"RATE MIX TONE",
			"RATE MIX Q",
			"RATE MIX RANGE",
			"RATE MIX RANGE",
			"RISE LEVEL",
			"SAMPLE MIX BIT",
		},
		"DELAY": {
			"LEVEL FEEDBACK TIME SUB-D",
			"LEVEL FEEDBACK TIME SUB-D",
			"LEVEL FEEDBACK TIME SUB-D",
			"LEVEL FEEDBACK TIME SUB-D",
			"LEVEL FEEDBACK TIME SUB-D",
			"LEVEL FEEDBACK TIME SUB-D",
			"LEVEL FEEDBACK TIME SUB-D",
			"LEVEL FEEDBACK TIME SUB-D THRES",
			"LEVEL FEEDBACK TIME A SUB A TIME B SUB B",
		},
	}

	for module, types := range want {
		lists := ge150Knobs[module]
		if len(lists) != len(types) {
			t.Errorf("%s has %d effect types, want %d", module, len(lists), len(types))
			continue
		}
		for typ, names := range types {
			if got := strings.Join(ge150KnobNames(lists[typ]), " "); got != names {
				t.Errorf("%s effect_type %d knobs = %q, want %q", module, typ, got, names)
			}
		}
	}
}

// TestGE150KnobTableCoversEveryModule: the remaining modules share one knob list
// across all their types, and no module may appear twice or go missing (the
// writer indexes the table by module name).
func TestGE150KnobTableCoversEveryModule(t *testing.T) {
	shared := []string{"DS/OD", "AMP", "CAB", "REVERB"}
	if got, want := len(ge150Knobs), len(shared)+5; got != want {
		t.Errorf("ge150Knobs has %d modules, want %d (5 per-type + %d shared)", got, want, len(shared))
	}
	for _, module := range shared {
		if got := len(ge150Knobs[module]); got != 1 {
			t.Errorf("%s has %d lists, want the single shared one", module, got)
		}
	}
}

func ge150KnobNames(list []ge150Knob) []string {
	out := make([]string, len(list))
	for i, k := range list {
		out[i] = k.Name
	}
	return out
}
