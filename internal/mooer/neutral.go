package mooer

// noon is the centre value of the device's 0–100 parameter scale, used as the
// neutral default for "amount" knobs (gain, tone, level, depth, …). Selector
// indices (effect type, mic, subdivision) keep 0, the first option, which is a
// valid selection rather than a minimum amount.
const noon = 50

// neutralDelayTime is a musical, playable delay time in milliseconds for a
// preset whose exact time was not translated from another device.
const neutralDelayTime = 400

func neutralFX() FX       { return FX{Q: noon, Position: noon, Peak: noon, Level: noon} }
func neutralDrive() Drive { return Drive{Volume: noon, Tone: noon, Gain: noon} }
func neutralAmp() Amp {
	return Amp{Gain: noon, Bass: noon, Mid: noon, Treble: noon, Presence: noon, Master: noon}
}
func neutralCab() Cab { return Cab{Mic: 0, Center: noon, Distance: noon, Tube: noon} }

// neutralNoiseGate keeps the threshold at 0 (gate fully open) so an enabled
// gate never clamps the signal; only its attack/release sit at noon.
func neutralNoiseGate() NoiseGate {
	return NoiseGate{Attack: noon, Release: noon, Threshold: 0}
}
func neutralEQ() EQ {
	return EQ{
		Bands:      [6]uint8{noon, noon, noon, noon, noon, noon},
		BandsExtra: [6]uint8{noon, noon, noon, noon, noon, noon},
	}
}
func neutralMod() Mod { return Mod{Rate: noon, Level: noon, Depth: noon, Param4: noon, Param5: noon} }
func neutralDelay() Delay {
	return Delay{Level: noon, Feedback: noon, TimeMS: neutralDelayTime, Subdivision: 0, Param5: noon, Param6: noon}
}
func neutralReverb() Reverb { return Reverb{PreDelay: noon, Level: noon, Decay: noon, Tone: noon} }

// SetModule places an effect_type into one module of the fixed chain and gives
// the module its neutral parameter values. A preset chosen structurally (type
// + on/off only, e.g. from a cross-device mapping) is therefore playable
// rather than "every knob at zero".
func SetModule(p *Preset, module string, index uint8, enabled bool) {
	switch module {
	case "fx":
		f := neutralFX()
		f.Enabled, f.Type = enabled, index
		p.FX = f
	case "od":
		d := neutralDrive()
		d.Enabled, d.Type = enabled, index
		p.Drive = d
	case "amp":
		a := neutralAmp()
		a.Enabled, a.Type = enabled, index
		p.Amp = a
	case "cab":
		c := neutralCab()
		c.Enabled, c.Type = enabled, index
		p.Cab = c
	case "ns":
		n := neutralNoiseGate()
		n.Enabled, n.Type = enabled, index
		p.NoiseGate = n
	case "eq":
		e := neutralEQ()
		e.Enabled, e.Type = enabled, index
		p.EQ = e
	case "mod":
		m := neutralMod()
		m.Enabled, m.Type = enabled, index
		p.Mod = m
	case "delay":
		d := neutralDelay()
		d.Enabled, d.Type = enabled, index
		p.Delay = d
	case "reverb":
		r := neutralReverb()
		r.Enabled, r.Type = enabled, index
		p.Reverb = r
	}
}
