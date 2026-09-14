package mooer

import (
	"math"
	"strconv"
)

// Conversion between the shared Mooer Preset and a GE100 Pro preset body.
//
// A GE100 Pro slot holds a module kind, the index of a model within that
// module's list, and ten raw knob values. ge100ProModules is the single
// description of the nine modules - the wire kind, the knobs the device stores,
// how to read them out of a preset and how to write them back - so a knob that
// moves on the wire moves in one place.
//
// A knob is stored in the unit the editor labels it with, counted in the knob's
// own step: the editor's parameter tables give every knob a unit, a minimum, a
// maximum and a step, and its serialiser stores value*(max-min). A 0-10 knob
// therefore lands on the shared preset's 0-100 scale, but the EQ's decibel knobs
// do not - they run 0..32 with 16 for flat - and its frequency knobs are Hz. The
// knobs that disagree with the shared scale are converted here (ge100ProBandToWire).
// Where a knob is a physical unit the shared Preset has no field for - the cab's
// LOW CUT / HIGH CUT are Hz - nothing is written and the device keeps its own
// default.
type ge100ProKnob struct {
	// key is the shared preset's name for the knob, e.g. "band2", "threshold".
	key string
	// wire is the position the device stores it at, which is not always the
	// position the shared preset lists it in: the noise gate's threshold moves
	// with the model, and a parametric EQ's gains sit between its frequencies.
	wire uint8
}

type ge100ProModule struct {
	// kind is the module number on the wire.
	kind uint8
	// params names the module's knobs in wire order for the modules whose models
	// all store the same ones. It excludes the hidden model selector, which
	// travels in the slot's own model field.
	params []string
	// knobs names the knobs of one model, for the modules whose models move them.
	// It defaults to params, in wire order.
	knobs func(model uint8) []ge100ProKnob
	// values reads the module's knobs out of a preset, keyed by knob name.
	values func(Preset) map[string]uint16
	// apply writes a slot back into the preset.
	apply func(*Preset, ge100ProSlot, map[string]uint16)
}

// knobsOf returns a module's knobs for one model, in wire order.
func (m ge100ProModule) knobsOf(model uint8) []ge100ProKnob {
	if m.knobs != nil {
		return m.knobs(model)
	}
	knobs := make([]ge100ProKnob, 0, len(m.params))
	for i, name := range m.params {
		knobs = append(knobs, ge100ProKnob{key: name, wire: uint8(i)})
	}
	return knobs
}

// ge100ProModules is every module the device can hold, keyed by module name.
var ge100ProModules = map[string]ge100ProModule{
	"fx": {
		kind:   ge100ProKindFX,
		params: []string{"q", "position", "peak", "level"},
		values: fxValues,
		apply:  applyFXSlot,
	},
	"od": {
		kind:   ge100ProKindDS,
		params: []string{"volume", "tone", "gain"},
		values: driveValues,
		apply:  applyDriveSlot,
	},
	"amp": {
		kind:   ge100ProKindAmp,
		params: []string{"gain", "bass", "mid", "treble", "presence", "master"},
		values: ampValues,
		apply:  applyAmpSlot,
	},
	"cab": {
		// LOW CUT (Hz), HIGH CUT (Hz) and ROOM: the first two are frequencies the
		// shared preset does not carry.
		kind:   ge100ProKindCab,
		params: nil,
		values: func(Preset) map[string]uint16 { return nil },
		apply:  applyCabSlot,
	},
	"ns": {
		kind:   ge100ProKindNS,
		knobs:  ge100ProNSKnobs,
		values: noiseGateValues,
		apply:  applyNoiseGateSlot,
	},
	"eq": {
		kind:   ge100ProKindEQ,
		knobs:  ge100ProEQKnobs,
		values: eqValues,
		apply:  applyEQSlot,
	},
	"mod": {
		kind:   ge100ProKindMod,
		params: []string{"rate", "level", "depth"},
		values: modValues,
		apply:  applyModSlot,
	},
	"delay": {
		kind:   ge100ProKindDelay,
		params: []string{"level", "feedback", "time_ms", "subdivision"},
		values: delayValues,
		apply:  applyDelaySlot,
	},
	"reverb": {
		kind:   ge100ProKindReverb,
		params: []string{"pre_delay", "level", "decay", "tone"},
		values: reverbValues,
		apply:  applyReverbSlot,
	},
}

// ge100ProEQModel is one of the device's EQ models: where it keeps each band's
// gain, and whether its knobs are all band gains.
type ge100ProEQModel struct {
	bands []ge100ProKnob
	// parametric marks a model that mixes gains with frequency knobs, which the
	// shared preset has nowhere to carry.
	parametric bool
}

// The device's EQ models, in the index order a preset picks one with: 3-Band EQ,
// Mooer G, Mooer HM, Mooer G-6, Mooer B and Custom EQ. bands lists the model's
// band knobs in shared-preset band order, so bands[1] is where "band2" lives.
var ge100ProEQModels = []ge100ProEQModel{
	{bands: ge100ProBands(3)}, // 3-Band EQ: Low, Mid, High
	{bands: ge100ProBands(5)}, // Mooer G
	{bands: ge100ProBands(5)}, // Mooer HM
	{bands: ge100ProBands(6)}, // Mooer G-6
	{bands: ge100ProBands(5)}, // Mooer B
	{
		// Custom EQ is parametric: its six knobs alternate three gains with
		// three frequencies, so only the gains are bands.
		bands:      []ge100ProKnob{{"band1", 0}, {"band2", 2}, {"band3", 4}},
		parametric: true,
	},
}

// ge100ProBands names the band knobs of a graphic model band1..bandN, in wire
// order.
func ge100ProBands(n int) []ge100ProKnob {
	bands := make([]ge100ProKnob, 0, n)
	for i := 0; i < n; i++ {
		bands = append(bands, ge100ProKnob{key: "band" + strconv.Itoa(i+1), wire: uint8(i)})
	}
	return bands
}

func ge100ProEQKnobs(model uint8) []ge100ProKnob {
	return ge100ProTableAt(ge100ProEQModels, model).bands
}

// The gate's three models and the knobs each carries, in wire order: NOISE
// KILLER and INTEL REDUCER have one knob each - the threshold under the editor's
// own name for it - where NOISE GATE has three and stores the threshold last.
var ge100ProNSModelKnobs = [][]ge100ProKnob{
	{{"threshold", 0}},
	{{"threshold", 0}},
	{{"attack", 0}, {"release", 1}, {"threshold", 2}},
}

func ge100ProNSKnobs(model uint8) []ge100ProKnob {
	return ge100ProTableAt(ge100ProNSModelKnobs, model)
}

// ge100ProTableAt picks the entry a preset's model index selects, or the zero
// value when the index is outside the table - a file can carry any index, and a
// model the device does not have holds no knobs.
func ge100ProTableAt[T any](table []T, model uint8) T {
	if int(model) >= len(table) {
		var zero T
		return zero
	}
	return table[model]
}

// An EQ band is stored in decibels: the editor's tables give every band knob min
// -16 / max +16 dB and a 0.1 dB step, and its serialiser stores value*(max-min),
// so the wire byte is 0..32 counting dB with 16 for flat. The shared preset's
// bands are on its own 0-100 scale instead, where 50 is flat and the range spans
// +-10 dB. Writing 50 verbatim asks the device for +34 dB, which pins every band
// at +16 dB: a flat boost across the spectrum rather than the curve designed.
const (
	// ge100ProBandFlat is the wire value of 0 dB.
	ge100ProBandFlat = 16
	// ge100ProBandPerDB is how many of the shared scale's units make one dB.
	ge100ProBandPerDB = 5
)

// ge100ProBandToWire maps a shared EQ band onto the device's dB scale, rounding
// to the whole decibels the band knobs hold.
func ge100ProBandToWire(v uint8) uint8 {
	dB := math.Round(float64(int(v)-noon) / ge100ProBandPerDB)
	return uint8(ge100ProBandFlat + int(dB)) // #nosec G115 -- the shared scale spans +-10 dB, so this is 6..26
}

// ge100ProBandFromWire maps the device's dB band back onto the shared scale. A
// band beyond the shared +-10 dB reads as the nearest end of it, since the
// shared preset has no place for the device's extra range.
func ge100ProBandFromWire(wire uint8) uint8 {
	v := noon + ge100ProBandPerDB*(int(wire)-ge100ProBandFlat)
	return uint8(min(max(v, 0), 100))
}

// ge100ProChainOrder is the order modules take in the ten slots. Slots are a
// signal chain, not the device's module numbering, so the gate and any wah sit
// in front of the drive and the time-based modules sit after the amp and cab -
// the shape a factory preset has.
var ge100ProChainOrder = []string{"ns", "fx", "od", "amp", "cab", "eq", "mod", "delay", "reverb"}

// ge100ProModuleKind maps a module to its wire kind.
func ge100ProModuleKind(module string) (uint8, bool) {
	m, ok := ge100ProModules[module]
	return m.kind, ok
}

// ge100ProModuleAt maps a wire kind back to its module.
func ge100ProModuleAt(kind uint8) (string, bool) {
	for name, module := range ge100ProModules {
		if module.kind == kind {
			return name, true
		}
	}
	return "", false
}

// moduleModel returns the model index a preset assigns to a module and whether
// the module is in use. The GE100 Pro separates "the slot holds a module" from
// "the module is on"; the shared Preset expresses both with Enabled.
func moduleModel(p Preset, module string) (uint8, bool) {
	switch module {
	case "fx":
		return p.FX.Type, p.FX.Enabled
	case "od":
		return p.Drive.Type, p.Drive.Enabled
	case "amp":
		return p.Amp.Type, p.Amp.Enabled
	case "cab":
		return p.Cab.Type, p.Cab.Enabled
	case "ns":
		return p.NoiseGate.Type, p.NoiseGate.Enabled
	case "eq":
		return p.EQ.Type, p.EQ.Enabled
	case "mod":
		return p.Mod.Type, p.Mod.Enabled
	case "delay":
		return p.Delay.Type, p.Delay.Enabled
	case "reverb":
		return p.Reverb.Type, p.Reverb.Enabled
	}
	return 0, false
}

// presetToGE100ProBody lays a preset out over the device's ten slots in chain
// order, leaving the slots its modules do not use empty.
func presetToGE100ProBody(p Preset) ge100ProBody {
	body := ge100ProEmptyBody(p.Name)

	slot := 0
	for _, module := range ge100ProChainOrder {
		kind, held := ge100ProModuleKind(module)
		model, used := moduleModel(p, module)
		if !held || !used || slot >= ge100ProSlotCount {
			continue
		}
		body.Slots[slot] = ge100ProSlot{
			Present: true,
			Kind:    kind,
			On:      true,
			Model:   model,
			Params:  ge100ProSlotParams(module, model, ge100ProModules[module].values(p)),
			Memory:  emptyGE100ProMemory(),
		}
		slot++
	}
	return body
}

// ge100ProSlotParams places a module's knobs in the model's wire order.
func ge100ProSlotParams(module string, model uint8, values map[string]uint16) [ge100ProParamCount]uint16 {
	var params [ge100ProParamCount]uint16
	for _, knob := range ge100ProModules[module].knobsOf(model) {
		if int(knob.wire) < len(params) {
			params[knob.wire] = values[knob.key]
		}
	}
	return params
}

// ge100ProSlotValues keys a slot's raw knobs the way the module's knobs are
// named.
func ge100ProSlotValues(module string, model uint8, params [ge100ProParamCount]uint16) map[string]uint16 {
	knobs := ge100ProModules[module].knobsOf(model)
	values := make(map[string]uint16, len(knobs))
	for _, knob := range knobs {
		if int(knob.wire) < len(params) {
			values[knob.key] = params[knob.wire]
		}
	}
	return values
}

// emptyGE100ProMemory is the sample reference the device writes for a slot that
// points at no user sample.
func emptyGE100ProMemory() ge100ProMemory {
	return ge100ProMemory{Name: ge100ProMemoryUnset}
}

// ge100ProBodyToPreset reads the modules a body assigns into the shared Preset.
// A chain that repeats a module kind (two noise gates, say) keeps its first
// slot: the shared Preset holds one module per kind.
func ge100ProBodyToPreset(body ge100ProBody) Preset {
	p := New()
	p.Name = body.Name
	read := make(map[uint8]bool, ge100ProSlotCount)
	for _, slot := range body.Slots {
		if slot.empty() || read[slot.Kind] {
			continue
		}
		module, ok := ge100ProModuleAt(slot.Kind)
		if !ok {
			continue
		}
		read[slot.Kind] = true
		ge100ProModules[module].apply(&p, slot, ge100ProSlotValues(module, slot.Model, slot.Params))
	}
	return p
}

// byteValue narrows a wire knob value onto the shared preset's 0-100 scale.
func byteValue(values map[string]uint16, name string) uint8 {
	v := values[name]
	if v > 100 {
		return 100
	}
	return uint8(v) // #nosec G115 -- clamped to 100 above
}

func fxValues(p Preset) map[string]uint16 {
	return map[string]uint16{
		"q": uint16(p.FX.Q), "position": uint16(p.FX.Position),
		"peak": uint16(p.FX.Peak), "level": uint16(p.FX.Level),
	}
}

func applyFXSlot(p *Preset, slot ge100ProSlot, v map[string]uint16) {
	p.FX = FX{
		Enabled: slot.On, Type: slot.Model,
		Q: byteValue(v, "q"), Position: byteValue(v, "position"),
		Peak: byteValue(v, "peak"), Level: byteValue(v, "level"),
	}
}

func driveValues(p Preset) map[string]uint16 {
	return map[string]uint16{
		"volume": uint16(p.Drive.Volume), "tone": uint16(p.Drive.Tone), "gain": uint16(p.Drive.Gain),
	}
}

func applyDriveSlot(p *Preset, slot ge100ProSlot, v map[string]uint16) {
	p.Drive = Drive{
		Enabled: slot.On, Type: slot.Model,
		Volume: byteValue(v, "volume"), Tone: byteValue(v, "tone"), Gain: byteValue(v, "gain"),
	}
}

func ampValues(p Preset) map[string]uint16 {
	return map[string]uint16{
		"gain": uint16(p.Amp.Gain), "bass": uint16(p.Amp.Bass), "mid": uint16(p.Amp.Mid),
		"treble": uint16(p.Amp.Treble), "presence": uint16(p.Amp.Presence), "master": uint16(p.Amp.Master),
	}
}

func applyAmpSlot(p *Preset, slot ge100ProSlot, v map[string]uint16) {
	p.Amp = Amp{
		Enabled: slot.On, Type: slot.Model,
		Gain: byteValue(v, "gain"), Bass: byteValue(v, "bass"), Mid: byteValue(v, "mid"),
		Treble: byteValue(v, "treble"), Presence: byteValue(v, "presence"), Master: byteValue(v, "master"),
	}
}

// applyCabSlot keeps the model: the cab's own knobs are Hz filters the shared
// preset has no field for.
func applyCabSlot(p *Preset, slot ge100ProSlot, _ map[string]uint16) {
	p.Cab = Cab{Enabled: slot.On, Type: slot.Model}
}

func noiseGateValues(p Preset) map[string]uint16 {
	return map[string]uint16{
		"threshold": uint16(p.NoiseGate.Threshold), "attack": uint16(p.NoiseGate.Attack),
		"release": uint16(p.NoiseGate.Release),
	}
}

func applyNoiseGateSlot(p *Preset, slot ge100ProSlot, v map[string]uint16) {
	p.NoiseGate = NoiseGate{
		Enabled: slot.On, Type: slot.Model,
		Threshold: byteValue(v, "threshold"), Attack: byteValue(v, "attack"),
		Release: byteValue(v, "release"),
	}
}

// eqValues writes the model's own bands, each on the device's dB scale.
func eqValues(p Preset) map[string]uint16 {
	bands := ge100ProEQKnobs(p.EQ.Type)
	values := make(map[string]uint16, len(bands))
	for i, band := range bands {
		values[band.key] = uint16(ge100ProBandToWire(p.EQ.Bands[i]))
	}
	return values
}

func applyEQSlot(p *Preset, slot ge100ProSlot, v map[string]uint16) {
	p.EQ.Enabled, p.EQ.Type = slot.On, slot.Model
	for i, band := range ge100ProEQKnobs(slot.Model) {
		p.EQ.Bands[i] = ge100ProBandFromWire(byteValue(v, band.key))
	}
}

func modValues(p Preset) map[string]uint16 {
	return map[string]uint16{
		"rate": uint16(p.Mod.Rate), "level": uint16(p.Mod.Level), "depth": uint16(p.Mod.Depth),
	}
}

func applyModSlot(p *Preset, slot ge100ProSlot, v map[string]uint16) {
	p.Mod = Mod{
		Enabled: slot.On, Type: slot.Model,
		Rate: byteValue(v, "rate"), Level: byteValue(v, "level"), Depth: byteValue(v, "depth"),
	}
}

func delayValues(p Preset) map[string]uint16 {
	return map[string]uint16{
		"level": uint16(p.Delay.Level), "feedback": uint16(p.Delay.Feedback),
		"time_ms": p.Delay.TimeMS, "subdivision": uint16(p.Delay.Subdivision),
	}
}

func applyDelaySlot(p *Preset, slot ge100ProSlot, v map[string]uint16) {
	p.Delay = Delay{
		Enabled: slot.On, Type: slot.Model,
		Level: byteValue(v, "level"), Feedback: byteValue(v, "feedback"),
		TimeMS: v["time_ms"], Subdivision: byteValue(v, "subdivision"),
	}
}

func reverbValues(p Preset) map[string]uint16 {
	return map[string]uint16{
		"pre_delay": uint16(p.Reverb.PreDelay), "level": uint16(p.Reverb.Level),
		"decay": uint16(p.Reverb.Decay), "tone": uint16(p.Reverb.Tone),
	}
}

func applyReverbSlot(p *Preset, slot ge100ProSlot, v map[string]uint16) {
	p.Reverb = Reverb{
		Enabled: slot.On, Type: slot.Model,
		PreDelay: byteValue(v, "pre_delay"), Level: byteValue(v, "level"),
		Decay: byteValue(v, "decay"), Tone: byteValue(v, "tone"),
	}
}
