// Code generated from the GE150 Edit app's preset.json. DO NOT EDIT the values:
// the knob names, ranges and kinds are the app's, and a list's position inside
// its module is the effect_type it belongs to.
package mooer

// ge150Knob is one knob of a GE150 effect type: its JSON key, its stored
// range and how our internal value maps to it.
type ge150Knob struct {
	Name string
	Min  int
	Max  int
	Kind ge150Kind
}

// ge150Kind says how a knob's value maps between our internal scale and
// the stored JSON integer.
type ge150Kind int

const (
	// ge150KnobScale: our 0-100 value maps linearly onto Min..Max.
	ge150KnobScale ge150Kind = iota
	// ge150KnobMs: our value is already milliseconds (time knobs).
	ge150KnobMs
	// ge150KnobSel: our value is a selector index (SUB-*, TUBE, MIC).
	ge150KnobSel
)

// The app repeats the same knob list for several effect types of a module (seven
// of the nine delay types share one list, for example). Every repeated list is
// declared once here and referenced by name in ge150Knobs below; a list that
// only one type uses stays inline in the table, next to that type.
var (
	// FX/COMP: the compressor and limiter voices.
	ge150CompQKnobs     = []ge150Knob{{"Q", 0, 100, ge150KnobScale}, {"POSITION", 0, 100, ge150KnobScale}, {"PEAK", 0, 100, ge150KnobScale}, {"LEVEL", 0, 100, ge150KnobScale}}
	ge150CompRateKnobs  = []ge150Knob{{"RATE", 0, 100, ge150KnobScale}, {"RANGE", 0, 100, ge150KnobScale}, {"PEAK", 0, 100, ge150KnobScale}, {"LEVEL", 0, 100, ge150KnobScale}}
	ge150CompThresKnobs = []ge150Knob{{"ATTACK", 0, 100, ge150KnobScale}, {"THRES", 0, 100, ge150KnobScale}, {"RATIO", 0, 100, ge150KnobScale}, {"LEVEL", 0, 100, ge150KnobScale}}

	// MOD: the rate/level family, the rate/mix family and the range pair.
	ge150ModLevelDepthKnobs = []ge150Knob{{"RATE", 0, 100, ge150KnobScale}, {"LEVEL", 0, 100, ge150KnobScale}, {"DEPTH", 0, 100, ge150KnobScale}}
	ge150ModFeedbackKnobs   = []ge150Knob{{"RATE", 0, 100, ge150KnobScale}, {"MIX", 0, 100, ge150KnobScale}, {"FEEDBACK", 0, 100, ge150KnobScale}}
	ge150ModToneKnobs       = []ge150Knob{{"RATE", 0, 100, ge150KnobScale}, {"MIX", 0, 100, ge150KnobScale}, {"TONE", 0, 100, ge150KnobScale}}
	ge150ModToneDepthKnobs  = []ge150Knob{{"RATE", 0, 100, ge150KnobScale}, {"MIX", 0, 100, ge150KnobScale}, {"TONE", 0, 100, ge150KnobScale}, {"DEPTH", 0, 100, ge150KnobScale}}
	ge150ModRangeKnobs      = []ge150Knob{{"RATE", 0, 100, ge150KnobScale}, {"MIX", 0, 100, ge150KnobScale}, {"RANGE", 0, 100, ge150KnobScale}}

	// DELAY: every delay type but the gated one (extra THRES) and the dual one.
	ge150DelayKnobs = []ge150Knob{{"LEVEL", 0, 100, ge150KnobScale}, {"FEEDBACK", 0, 100, ge150KnobScale}, {"TIME", 40, 2500, ge150KnobMs}, {"SUB-D", 0, 0, ge150KnobSel}}
)

// ge150Knobs lists each module's effect-type knob lists, in effect_type
// order, so an entry's position is the type it describes and no entry may be
// reordered or dropped. A module whose list is shared by every type has a
// single entry; per-type modules have one entry per type. The custom
// parametric EQ (EQ type 3) is handled separately in ge150json.go.
var ge150Knobs = map[string][][]ge150Knob{
	"FX/COMP": {
		ge150CompQKnobs,
		ge150CompQKnobs,
		ge150CompRateKnobs,
		ge150CompRateKnobs,
		ge150CompRateKnobs,
		{{"ATTACK", 0, 100, ge150KnobScale}, {"SENS", 0, 100, ge150KnobScale}, {"PEAK", 0, 100, ge150KnobScale}, {"LEVEL", 0, 100, ge150KnobScale}},
		ge150CompThresKnobs,
		ge150CompThresKnobs,
	},
	"NS GATE": {
		{{"THRES", 0, 100, ge150KnobScale}},
		{{"SENS", 0, 100, ge150KnobScale}},
		{{"ATTACK", 0, 100, ge150KnobScale}, {"RELEASE", 0, 100, ge150KnobScale}, {"THRES", 0, 100, ge150KnobScale}},
	},
	"EQ": {
		{{"100Hz", 0, 32, ge150KnobScale}, {"250Hz", 0, 32, ge150KnobScale}, {"630Hz", 0, 32, ge150KnobScale}, {"1.6KHz", 0, 32, ge150KnobScale}, {"4KHz", 0, 32, ge150KnobScale}},
		{{"80Hz", 0, 32, ge150KnobScale}, {"240Hz", 0, 32, ge150KnobScale}, {"750Hz", 0, 32, ge150KnobScale}, {"2.2KHz", 0, 32, ge150KnobScale}, {"6.6KHz", 0, 32, ge150KnobScale}},
		{{"100Hz", 0, 32, ge150KnobScale}, {"200Hz", 0, 32, ge150KnobScale}, {"400Hz", 0, 32, ge150KnobScale}, {"800Hz", 0, 32, ge150KnobScale}, {"1.6KHz", 0, 32, ge150KnobScale}, {"3.2KHz", 0, 32, ge150KnobScale}},
	},
	"MOD": {
		ge150ModLevelDepthKnobs,
		ge150ModLevelDepthKnobs,
		ge150ModLevelDepthKnobs,
		ge150ModFeedbackKnobs,
		ge150ModFeedbackKnobs,
		ge150ModToneKnobs,
		ge150ModToneKnobs,
		{{"RATE", 0, 100, ge150KnobScale}, {"DEPTH", 0, 100, ge150KnobScale}, {"TONE", 0, 100, ge150KnobScale}},
		{{"PITCH", -120, 120, ge150KnobScale}, {"MIX", 0, 100, ge150KnobScale}, {"TONE", 0, 100, ge150KnobScale}},
		{{"PITCH", -200, 200, ge150KnobScale}, {"MIX", 0, 100, ge150KnobScale}, {"TONE", 0, 100, ge150KnobScale}},
		ge150ModToneKnobs,
		ge150ModToneDepthKnobs,
		ge150ModToneDepthKnobs,
		ge150ModToneKnobs,
		{{"RATE", 0, 100, ge150KnobScale}, {"MIX", 0, 100, ge150KnobScale}, {"Q", 0, 100, ge150KnobScale}},
		ge150ModRangeKnobs,
		ge150ModRangeKnobs,
		{{"RISE", 0, 100, ge150KnobScale}, {"LEVEL", 0, 100, ge150KnobScale}},
		{{"SAMPLE", 0, 100, ge150KnobScale}, {"MIX", 0, 100, ge150KnobScale}, {"BIT", 0, 100, ge150KnobScale}},
	},
	"DELAY": {
		ge150DelayKnobs,
		ge150DelayKnobs,
		ge150DelayKnobs,
		ge150DelayKnobs,
		ge150DelayKnobs,
		ge150DelayKnobs,
		ge150DelayKnobs,
		{{"LEVEL", 0, 100, ge150KnobScale}, {"FEEDBACK", 0, 100, ge150KnobScale}, {"TIME", 40, 2500, ge150KnobMs}, {"SUB-D", 0, 0, ge150KnobSel}, {"THRES", 0, 100, ge150KnobScale}},
		{{"LEVEL", 0, 100, ge150KnobScale}, {"FEEDBACK", 0, 100, ge150KnobScale}, {"TIME A", 40, 2500, ge150KnobMs}, {"SUB A", 0, 0, ge150KnobSel}, {"TIME B", 40, 2500, ge150KnobMs}, {"SUB B", 0, 0, ge150KnobSel}},
	},
	"DS/OD": {
		{{"VOLUME", 0, 100, ge150KnobScale}, {"TONE", 0, 100, ge150KnobScale}, {"GAIN", 0, 100, ge150KnobScale}},
	},
	"AMP": {
		{{"GAIN", 0, 100, ge150KnobScale}, {"BASS", 0, 100, ge150KnobScale}, {"MID", 0, 100, ge150KnobScale}, {"TREBLE", 0, 100, ge150KnobScale}, {"PRES", 0, 100, ge150KnobScale}, {"MST", 0, 100, ge150KnobScale}},
	},
	"CAB": {
		{{"TUBE", 0, 0, ge150KnobSel}, {"MIC", 0, 0, ge150KnobSel}, {"CENTER", 0, 100, ge150KnobScale}, {"DISTANCE", 0, 100, ge150KnobScale}},
	},
	"REVERB": {
		{{"PRE DELAY", 0, 500, ge150KnobMs}, {"LEVEL", 0, 100, ge150KnobScale}, {"DECAY", 0, 100, ge150KnobScale}, {"TONE", 0, 100, ge150KnobScale}},
	},
}
