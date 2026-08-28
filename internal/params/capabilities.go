package params

import (
	"sort"
	"strings"

	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/catalog"
)

// capabilityParams maps a parameter name to the capabilities it implies, so an
// agent can discover what a module can do (e.g. a "Delay" param means the
// module delays; a "Mode" full of semitone values means it pitch-shifts).
var capabilityParams = map[string][]string{
	// Delay / echo
	"Delay": {"delay"}, "Delay1": {"delay"}, "Delay2": {"delay"},
	"PreDelay": {"delay"}, "Pre-Delay": {"delay"},
	"Feedback": {"delay"}, "Fdbk": {"delay"}, "FdbkHighCut": {"delay"},
	"FdbkReso": {"delay"}, "FdbkResoFreq": {"delay"},
	// Reverb
	"ReverbTime": {"reverb"}, "Decay": {"reverb"}, "Size": {"reverb"},
	"DiffusionOn": {"reverb"}, "Damp": {"reverb"}, "Color": {"reverb"},
	// Modulation
	"Rate": {"modulation"}, "Speed": {"modulation"}, "LFORate": {"modulation"},
	"LFOSpeed": {"modulation"}, "Depth": {"modulation"}, "LFOShape": {"modulation"},
	"LFODepth": {"modulation"}, "Warp": {"modulation"},
	"Voices": {"chorus"}, "ChorusMode": {"chorus"},
	"Tremolo": {"tremolo"}, "TremSpeed": {"tremolo"}, "TremDepth": {"tremolo"},
	"TremSync": {"tremolo", "tempo sync"},
	"Cabinet":  {"rotary"}, "StopMode": {"rotary"},
	// Pitch / harmony
	"Pitch": {"pitch shift"}, "Pitch1": {"pitch shift"}, "Pitch2": {"pitch shift"},
	"Pitch2Vol": {"pitch shift"}, "Glide": {"pitch shift"},
	"Interval": {"pitch shift"}, "Detune": {"pitch shift"}, "Detune1": {"pitch shift"},
	"Detune2": {"pitch shift"}, "OctaveA": {"octave"}, "OctaveB": {"octave"},
	"Key": {"harmonizer"}, "Learn": {"harmonizer"},
	// Dynamics / compression
	"Threshold": {"compressor"}, "Ratio": {"compressor"}, "Knee": {"compressor"},
	"ThresholdComp": {"compressor"}, "ThresholdExp": {"expander"},
	"Attack": {"compressor"}, "Release": {"compressor"}, "MakeUp": {"compressor"},
	"Sustain": {"compressor"}, "Sensitivity": {"dynamics"},
	"Hi_Thresh": {"noise gate"}, "Lo_Thresh": {"noise gate"}, "FilterThreshold": {"noise gate"},
	// Drive / distortion
	"Drive": {"drive"}, "Distortion": {"distortion"}, "Dist": {"distortion"},
	"DistLev": {"distortion"}, "Fuzz": {"fuzz"},
	// Filter / wah / envelope
	"Cutoff": {"filter"}, "Reso": {"filter"}, "LPFreq": {"filter"},
	"EnvFbk": {"envelope filter"}, "EnvMix": {"envelope filter"}, "EnvRate": {"envelope filter"},
	// Lo-fi / bit-crusher
	"Bit1": {"bitcrusher"}, "BitGainScale": {"bitcrusher"}, "AntiAliasing": {"bitcrusher"},
	"Resampling": {"bitcrusher"},
	// Tape
	"Hiss": {"tape"}, "Wow": {"tape"}, "HeadTilt": {"tape"}, "RecordLevel": {"tape"},
	// Stereo
	"Stereo": {"stereo"}, "Width": {"stereo"}, "Separation": {"stereo"},
	"Pan": {"stereo"}, "Balance": {"stereo"},
	// EQ
	"Bass": {"eq"}, "Treble": {"eq"}, "Mid": {"eq"}, "Presence": {"eq"},
	"Resonance": {"eq"}, "MidFreq": {"eq"}, "Freq": {"eq"}, "Frequency": {"eq"},
	"Q": {"eq"}, "LoCut": {"eq"}, "LowCut": {"eq"}, "HiCut": {"eq"}, "HighCut": {"eq"},
	"HiMidFreq": {"eq"}, "LoMidFreq": {"eq"}, "MidGain": {"eq"}, "HiMidGain": {"eq"},
	"LoMidGain": {"eq"}, "HiMids": {"eq"}, "LoMids": {"eq"}, "Low": {"eq"}, "High": {"eq"},
	// Gain
	"Gain": {"gain"}, "GainA": {"gain"}, "GainB": {"gain"}, "GainSwitch": {"gain"},
	// Tempo sync
	"Sync": {"tempo sync"}, "LFOSync": {"tempo sync"},
}

// valueCapabilities derives a capability from the enumerated options of a
// parameter (e.g. a Mode whose options are semitone intervals).
func valueCapabilities(paramName string, values []string) []string {
	if len(values) == 0 {
		return nil
	}
	joined := strings.Join(values, " ")
	switch {
	case strings.Contains(joined, "Semi"):
		return []string{"pitch shift"}
	case strings.Contains(joined, "Plate") || strings.Contains(joined, "Hall") ||
		strings.Contains(joined, "Room") || strings.Contains(joined, "Church") ||
		strings.Contains(joined, "Cathedral") || strings.Contains(joined, "Arena") ||
		strings.Contains(joined, "Theater") || strings.Contains(joined, "Reflection") ||
		strings.Contains(joined, "Garage") || strings.Contains(joined, "Studio") ||
		strings.Contains(joined, "NonLinear"):
		return []string{"reverb"}
	case strings.Contains(joined, "Chorus"):
		return []string{"chorus"}
	case strings.Contains(joined, "Flanger"):
		return []string{"flanger"}
	}
	return nil
}

// Capabilities computes the capability keywords for a module from its
// parameter spec, sorted and de-duplicated.
func Capabilities(cat *catalog.Catalog, moduleName string) []string {
	spec, err := Describe(cat, moduleName)
	if err != nil {
		return nil
	}
	set := map[string]bool{}
	for key, p := range spec {
		for _, c := range capabilityParams[key] {
			set[c] = true
		}
		for _, c := range valueCapabilities(key, p.Values) {
			set[c] = true
		}
	}
	out := make([]string, 0, len(set))
	for c := range set {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}
