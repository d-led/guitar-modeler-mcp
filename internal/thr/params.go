package thr

// Unset marks a knob whose value was not specified. The setup card omits
// unset knobs, so an agent never prints a value it did not actually choose.
const Unset = -1

// AmpParams are the THR-II amp controls, each on a 0-100 scale.
type AmpParams struct {
	Gain   int
	Master int
	Bass   int
	Mid    int
	Treble int
}

// ModParams are the EFFECT knob controls, each on a 0-100 scale. Not every
// modulation type uses every knob (chorus has no feedback, for example), so
// unused knobs stay Unset and are omitted from the card.
type ModParams struct {
	Speed    int
	Depth    int
	PreDelay int
	Feedback int
	Mix      int
}

// EchoParams are the ECHO knob controls. Time is in milliseconds; the rest are
// 0-100.
type EchoParams struct {
	Time     int
	Feedback int
	Bass     int
	Treble   int
	Mix      int
}

// ReverbParams are the REVERB knob controls. PreDelay is in milliseconds; the
// rest are 0-100.
type ReverbParams struct {
	Level    int
	Decay    int
	PreDelay int
	Tone     int
	Mix      int
}

// CompressorParams are the app-only compressor controls, each 0-100.
type CompressorParams struct {
	Sustain int
	Level   int
}

// GateParams are the app-only noise gate controls, each 0-100.
type GateParams struct {
	Threshold int
	Decay     int
}

// Levels are the global guitar-input and audio-output levels, each 0-100.
type Levels struct {
	Guitar int
	Audio  int
}

// knob is one named control with its value, as displayed on the setup card.
type knob struct {
	name  string
	value int
}

// set keeps the knobs that have a value, in display order.
func set(pairs ...knob) []knob {
	out := make([]knob, 0, len(pairs))
	for _, p := range pairs {
		if p.value >= 0 {
			out = append(out, p)
		}
	}
	return out
}

// noonSet keeps every knob, filling unset ones with the neutral noon position
// (50 on the 0-100 scale) and marking them, so the card is a self-contained
// instruction card: every knob has a dial-in value.
func noonSet(pairs ...knob) []knob {
	out := make([]knob, 0, len(pairs))
	for _, p := range pairs {
		if p.value < 0 {
			p.value = 50
			p.name += " (noon)"
		}
		out = append(out, p)
	}
	return out
}

func ampKnobs(a AmpParams) []knob {
	return noonSet(
		knob{"Gain", a.Gain},
		knob{"Master", a.Master},
		knob{"Bass", a.Bass},
		knob{"Mid", a.Mid},
		knob{"Treble", a.Treble},
	)
}

func modKnobs(m ModParams) []knob {
	// Modulation types do not all use every knob (chorus has no feedback), so
	// unused knobs stay omitted.
	return set(
		knob{"Speed", m.Speed},
		knob{"Depth", m.Depth},
		knob{"Pre-Delay (ms)", m.PreDelay},
		knob{"Feedback", m.Feedback},
		knob{"Mix", m.Mix},
	)
}

func echoKnobs(e EchoParams) []knob {
	return noonSet(
		knob{"Time (ms)", e.Time},
		knob{"Feedback", e.Feedback},
		knob{"Bass", e.Bass},
		knob{"Treble", e.Treble},
		knob{"Mix", e.Mix},
	)
}

func reverbKnobs(r ReverbParams) []knob {
	return noonSet(
		knob{"Level", r.Level},
		knob{"Decay", r.Decay},
		knob{"Pre-Delay (ms)", r.PreDelay},
		knob{"Tone", r.Tone},
		knob{"Mix", r.Mix},
	)
}

func compressorKnobs(c CompressorParams) []knob {
	return noonSet(
		knob{"Sustain", c.Sustain},
		knob{"Level", c.Level},
	)
}

func gateKnobs(g GateParams) []knob {
	return noonSet(
		knob{"Threshold", g.Threshold},
		knob{"Decay", g.Decay},
	)
}

func levelKnobs(l Levels) []knob {
	return noonSet(
		knob{"Guitar", l.Guitar},
		knob{"Audio", l.Audio},
	)
}
