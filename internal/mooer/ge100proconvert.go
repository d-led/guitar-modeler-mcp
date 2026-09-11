package mooer

import "strconv"

// Conversion between the shared Mooer Preset and a GE100 Pro preset body.
//
// A GE100 Pro slot holds a module kind, the index of a model within that
// module's list, and ten raw knob values. ge100ProModules is the single
// description of the nine modules - the wire kind, the order of the knobs the
// device stores, how to read them out of a preset and how to write them back -
// so a knob that moves on the wire moves in one place.
//
// Values are written in the device's raw scale, the same 0-100 "50 = noon"
// scale the rest of the package uses. Where a knob is a physical unit the shared
// Preset has no field for - the cab's LOW CUT / HIGH CUT are Hz - nothing is
// written and the device keeps its own default.
type ge100ProModule struct {
	// kind is the module number on the wire.
	kind uint8
	// params names the module's knobs in wire order, excluding the hidden model
	// selector, which travels in the slot's own model field.
	params []string
	// values reads the module's knobs out of a preset, keyed by params.
	values func(Preset) map[string]uint16
	// apply writes a slot back into the preset.
	apply func(*Preset, ge100ProSlot, map[string]uint16)
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
		params: []string{"threshold", "attack", "release"},
		values: noiseGateValues,
		apply:  applyNoiseGateSlot,
	},
	"eq": {
		kind:   ge100ProKindEQ,
		params: []string{"band1", "band2", "band3", "band4", "band5", "band6"},
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
		params: []string{"level", "feedback", "time", "subdivision"},
		values: delayValues,
		apply:  applyDelaySlot,
	},
	"reverb": {
		kind:   ge100ProKindReverb,
		params: []string{"predelay", "level", "decay", "tone"},
		values: reverbValues,
		apply:  applyReverbSlot,
	},
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
			Params:  ge100ProSlotParams(module, ge100ProModules[module].values(p)),
			Memory:  emptyGE100ProMemory(),
		}
		slot++
	}
	return body
}

// ge100ProSlotParams places a module's knobs in the model's wire order.
func ge100ProSlotParams(module string, values map[string]uint16) [ge100ProParamCount]uint16 {
	var params [ge100ProParamCount]uint16
	names := ge100ProModules[module].params
	for i := 0; i < len(params) && i < len(names); i++ {
		params[i] = values[names[i]]
	}
	return params
}

// ge100ProSlotValues keys a slot's raw knobs the way the module's params are
// named.
func ge100ProSlotValues(module string, params [ge100ProParamCount]uint16) map[string]uint16 {
	names := ge100ProModules[module].params
	values := make(map[string]uint16, len(names))
	for i, name := range names {
		if i < len(params) {
			values[name] = params[i]
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
		ge100ProModules[module].apply(&p, slot, ge100ProSlotValues(module, slot.Params))
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

// bandValues keys six knobs as band1..band6, the names the shared preset's EQ
// parameters use.
func bandValues(bands [6]uint8) map[string]uint16 {
	values := make(map[string]uint16, len(bands))
	for i, band := range bands {
		values["band"+strconv.Itoa(i+1)] = uint16(band)
	}
	return values
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

func eqValues(p Preset) map[string]uint16 { return bandValues(p.EQ.Bands) }

func applyEQSlot(p *Preset, slot ge100ProSlot, v map[string]uint16) {
	p.EQ.Enabled, p.EQ.Type = slot.On, slot.Model
	for i := range p.EQ.Bands {
		p.EQ.Bands[i] = byteValue(v, "band"+strconv.Itoa(i+1))
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
		"time": p.Delay.TimeMS, "subdivision": uint16(p.Delay.Subdivision),
	}
}

func applyDelaySlot(p *Preset, slot ge100ProSlot, v map[string]uint16) {
	p.Delay = Delay{
		Enabled: slot.On, Type: slot.Model,
		Level: byteValue(v, "level"), Feedback: byteValue(v, "feedback"),
		TimeMS: v["time"], Subdivision: byteValue(v, "subdivision"),
	}
}

func reverbValues(p Preset) map[string]uint16 {
	return map[string]uint16{
		"predelay": uint16(p.Reverb.PreDelay), "level": uint16(p.Reverb.Level),
		"decay": uint16(p.Reverb.Decay), "tone": uint16(p.Reverb.Tone),
	}
}

func applyReverbSlot(p *Preset, slot ge100ProSlot, v map[string]uint16) {
	p.Reverb = Reverb{
		Enabled: slot.On, Type: slot.Model,
		PreDelay: byteValue(v, "predelay"), Level: byteValue(v, "level"),
		Decay: byteValue(v, "decay"), Tone: byteValue(v, "tone"),
	}
}
