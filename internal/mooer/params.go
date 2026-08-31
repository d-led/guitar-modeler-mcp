package mooer

import "strings"

// Params holds raw device parameter values keyed by canonical name. Values are
// in the device's raw 0-100 scale (50 = noon) unless noted (delay time is
// milliseconds).
type Params map[string]float64

// canonicalParamKey normalises an agent-supplied parameter name to a canonical
// key: lower-case with every non-alphanumeric run collapsed to an underscore,
// so "GAIN", "Time (ms)" and "time_ms" all match "time_ms".
func canonicalParamKey(s string) string {
	var b strings.Builder
	lastUnderscore := false
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastUnderscore = false
		} else if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.Trim(b.String(), "_")
}

// normalizeParams canonicalises every key once, so the per-module appliers can
// look keys up directly. Absent/unknown keys are simply not applied.
func normalizeParams(params Params) Params {
	if len(params) == 0 {
		return nil
	}
	out := make(Params, len(params))
	for k, v := range params {
		out[canonicalParamKey(k)] = v
	}
	return out
}

func clampByte(v float64) uint8 {
	switch {
	case v < 0:
		return 0
	case v > 100:
		return 100
	default:
		return uint8(v)
	}
}

func applyByte(params Params, key string, dst *uint8) {
	if v, ok := params[key]; ok {
		*dst = clampByte(v)
	}
}

func applyFXParams(f *FX, p Params) {
	applyByte(p, "q", &f.Q)
	applyByte(p, "position", &f.Position)
	applyByte(p, "peak", &f.Peak)
	applyByte(p, "level", &f.Level)
}

func applyDriveParams(d *Drive, p Params) {
	applyByte(p, "volume", &d.Volume)
	applyByte(p, "tone", &d.Tone)
	applyByte(p, "gain", &d.Gain)
}

func applyAmpParams(a *Amp, p Params) {
	applyByte(p, "gain", &a.Gain)
	applyByte(p, "bass", &a.Bass)
	applyByte(p, "mid", &a.Mid)
	applyByte(p, "treble", &a.Treble)
	applyByte(p, "presence", &a.Presence)
	applyByte(p, "master", &a.Master)
}

func applyCabParams(c *Cab, p Params) {
	applyByte(p, "mic", &c.Mic)
	applyByte(p, "center", &c.Center)
	applyByte(p, "distance", &c.Distance)
	applyByte(p, "tube", &c.Tube)
}

func applyNoiseGateParams(n *NoiseGate, p Params) {
	applyByte(p, "attack", &n.Attack)
	applyByte(p, "release", &n.Release)
	applyByte(p, "threshold", &n.Threshold)
}

func applyEQParams(e *EQ, p Params) {
	bands := [12]string{
		"band1", "band2", "band3", "band4", "band5", "band6",
		"band7", "band8", "band9", "band10", "band11", "band12",
	}
	for i, key := range bands[:6] {
		applyByte(p, key, &e.Bands[i])
	}
	for i, key := range bands[6:] {
		applyByte(p, key, &e.BandsExtra[i])
	}
}

func applyModParams(m *Mod, p Params) {
	applyByte(p, "rate", &m.Rate)
	applyByte(p, "level", &m.Level)
	applyByte(p, "depth", &m.Depth)
	applyByte(p, "param4", &m.Param4)
	applyByte(p, "param5", &m.Param5)
}

func applyDelayParams(d *Delay, p Params) {
	applyByte(p, "level", &d.Level)
	applyByte(p, "feedback", &d.Feedback)
	applyByte(p, "subdivision", &d.Subdivision)
	applyByte(p, "param5", &d.Param5)
	applyByte(p, "param6", &d.Param6)
	// Time accepts "time" or "time_ms"; it is a 16-bit millisecond value.
	if v, ok := p["time_ms"]; ok {
		d.TimeMS = clampUint16(v)
	} else if v, ok := p["time"]; ok {
		d.TimeMS = clampUint16(v)
	}
}

func applyReverbParams(r *Reverb, p Params) {
	applyByte(p, "pre_delay", &r.PreDelay)
	applyByte(p, "level", &r.Level)
	applyByte(p, "decay", &r.Decay)
	applyByte(p, "tone", &r.Tone)
}

func clampUint16(v float64) uint16 {
	switch {
	case v < 0:
		return 0
	case v > 65535:
		return 65535
	default:
		return uint16(v)
	}
}
