package modspec

import "strings"

// paramDescriptions is a curated dictionary of plain-language descriptions for
// the device's parameter names, so the MCP can explain what each knob does
// rather than only listing its abstract key.
var paramDescriptions = map[string]string{
	// Gain / drive / distortion
	"Gain":         "Overall input gain or drive level.",
	"Drive":        "Amount of overdrive.",
	"Dist":         "Amount of distortion.",
	"Distortion":   "Amount of distortion.",
	"DistLev":      "Level of the distorted signal.",
	"Fuzz":         "Amount of fuzz.",
	"Level":        "Output level of the module.",
	"Volume":       "Output volume.",
	"Output":       "Output level.",
	"MakeUp":       "Make-up gain applied after compression.",
	"Tone":         "Timbre shaping (tone control).",
	"Hi-Lo":        "High/low gain switch.",
	"HiGain":       "Gain of the high band.",
	"LoGain":       "Gain of the low band.",
	"MidGain":      "Gain of the mid band.",
	"HiMidGain":    "Gain of the high-mid band.",
	"LoMidGain":    "Gain of the low-mid band.",
	"Intensity":    "Intensity or character of the effect.",
	"JCDistortion": "Distortion amount (jazz-chorus style).",
	"Body":         "Low-frequency emphasis (body).",
	"Fat":          "Adds low-end fullness.",
	"Bite":         "Adds high-frequency bite.",
	"Growl":        "Adds low-mid growl.",
	"Sustain":      "Amount of sustain.",
	"Sensitivity":  "How sensitive the effect is to the input signal.",

	// EQ
	"Bass":       "Low-frequency (bass) level.",
	"Treble":     "High-frequency (treble) level.",
	"Mid":        "Mid-frequency level.",
	"HiMids":     "High-mid band level.",
	"LoMids":     "Low-mid band level.",
	"Low":        "Low band level.",
	"High":       "High band level.",
	"Presence":   "High-frequency presence and air.",
	"Resonance":  "Power-amp low-end resonance.",
	"MidFreq":    "Center frequency of the mid band.",
	"HiMidFreq":  "Center frequency of the high-mid band.",
	"LoMidFreq":  "Center frequency of the low-mid band.",
	"Freq":       "Frequency setting.",
	"Frequency":  "Frequency setting.",
	"LPFreq":     "Low-pass filter frequency.",
	"Q":          "Width (resonance) of an EQ band.",
	"Bright":     "Bright switch (adds high-end).",
	"Bottom":     "Low-end emphasis switch.",
	"MidBoost":   "Mid boost switch.",
	"GainSwitch": "Gain structure switch.",
	"Linear":     "Linear (vs logarithmic) response.",

	// Dynamics / compression / gate
	"Threshold":       "Level above or below which the processor kicks in.",
	"ThresholdComp":   "Compression threshold.",
	"ThresholdExp":    "Expander threshold.",
	"Attack":          "How fast the effect responds to transients.",
	"Release":         "How fast the effect lets go after the signal drops.",
	"Ratio":           "Compression ratio.",
	"Knee":            "Softness of the compression curve at the threshold.",
	"Hi_Thresh":       "High-band noise threshold.",
	"Lo_Thresh":       "Low-band noise threshold.",
	"FilterThreshold": "Signal level below which the noise filter closes.",

	// Delay / reverb
	"Delay":        "Delay time.",
	"Delay1":       "Delay time for voice 1.",
	"Delay2":       "Delay time for voice 2.",
	"Feedback":     "Amount of repeats fed back into the delay.",
	"Fdbk":         "Feedback amount.",
	"FdbkHighCut":  "High-frequency cut on the feedback repeats.",
	"FdbkReso":     "Resonance of the feedback filter.",
	"FdbkResoFreq": "Frequency of the feedback resonance.",
	"ReverbTime":   "Length of the reverb tail.",
	"Decay":        "Decay time of the tail.",
	"Damp":         "High-frequency damping in the reverb.",
	"DiffusionOn":  "Enables reverb diffusion.",
	"Size":         "Size of the reverb space.",
	"Pre-Delay":    "Time before the reverb or effect starts.",
	"PreDelay":     "Time before the effect starts.",
	"Time":         "Time setting.",
	"Mix":          "Balance between dry and wet signal.",
	"Blend":        "Dry/wet blend.",
	"Tails":        "Whether the effect tail continues after bypass.",
	"HiCut":        "High-frequency cut.",
	"HighCut":      "High-frequency cut.",
	"LowCut":       "Low-frequency cut.",
	"LoCut":        "Low-frequency cut.",
	"Color":        "Tone colour of the effect.",
	"Hiss":         "Tape hiss amount.",
	"Wow":          "Tape wow (pitch wobble) amount.",
	"HeadTilt":     "Tape head alignment character.",
	"RecordLevel":  "Tape record level.",
	"Resampling":   "Sample-rate reduction (lo-fi).",

	// Modulation
	"Rate":       "Speed of the modulation.",
	"Speed":      "Speed of the modulation.",
	"Depth":      "Depth or intensity of the modulation.",
	"Voices":     "Number of chorus or detune voices.",
	"Detune":     "Amount of pitch detune.",
	"Detune1":    "Detune amount for voice 1.",
	"Detune2":    "Detune amount for voice 2.",
	"Interval":   "Pitch interval.",
	"Pitch":      "Pitch-shift amount.",
	"Pitch1":     "Pitch shift for voice 1.",
	"Pitch2":     "Pitch shift for voice 2.",
	"Pitch2Vol":  "Level of the second pitch voice.",
	"OctaveA":    "Octave shift for voice A.",
	"OctaveB":    "Octave shift for voice B.",
	"Key":        "Musical key for the harmonizer.",
	"Shape":      "Waveform shape of the modulation.",
	"LFOShape":   "Waveform shape of the LFO.",
	"LFORate":    "Rate of the LFO.",
	"LFODepth":   "Depth of the LFO.",
	"LFOSpeed":   "Speed of the LFO.",
	"LFOSync":    "Syncs the LFO to tempo.",
	"Stereo":     "Stereo width.",
	"Width":      "Stereo width.",
	"Separation": "Stereo separation.",
	"Balance":    "Left/right balance.",
	"Pan":        "Stereo pan position.",
	"Phase":      "Phase offset.",
	"Warp":       "Warp or intensity of the effect.",
	"Move":       "Amount of movement.",
	"Cabinet":    "Rotary cabinet emulation.",
	"StopMode":   "Brake/stop behavior of the rotary.",
	"Glide":      "Pitch glide time.",
	"Mod":        "Modulation amount.",

	// Filter / wah
	"Cutoff":  "Filter cutoff frequency.",
	"Reso":    "Filter resonance.",
	"EnvFbk":  "Envelope feedback.",
	"EnvMix":  "Envelope mix.",
	"EnvRate": "Envelope rate.",

	// Amp / cab
	"Master":       "Master volume.",
	"GainA":        "Gain for channel A.",
	"GainB":        "Gain for channel B.",
	"TremSpeed":    "Tremolo speed.",
	"TremDepth":    "Tremolo depth.",
	"TremSync":     "Syncs the tremolo to tempo.",
	"Tremolo":      "Enables the tremolo.",
	"Doubling":     "Enables the doubled (second) amp or cab.",
	"DoubleStates": "Enables independent states for the doubled module.",
	"StereoAmp":    "Stereo amp mode.",
	"ChorusMode":   "Chorus mode (Fixed, Off or Manual).",
	"Type":         "Model or type selection.",
	"Type1":        "Model or type selection.",
	"Type2":        "Model or type selection.",
	"Type4":        "Model or type selection.",
	"Mode":         "Operating mode.",
	"OnAxis":       "On-axis microphone position.",
	"Breakup":      "Speaker breakup amount.",
	"OutGain":      "Output gain of the cabinet.",
	"AmpCompGain":  "Gain compensation applied to the amp.",

	// Bit-crush / utility
	"Bit1":         "Bit 1 level of the bit-crusher.",
	"Bit2":         "Bit 2 level of the bit-crusher.",
	"Bit3":         "Bit 3 level of the bit-crusher.",
	"Bit4":         "Bit 4 level of the bit-crusher.",
	"Bit5":         "Bit 5 level of the bit-crusher.",
	"Bit6":         "Bit 6 level of the bit-crusher.",
	"Bit7":         "Bit 7 level of the bit-crusher.",
	"Bit8":         "Bit 8 level of the bit-crusher.",
	"BitGainScale": "Gain scaling for the bit-crusher.",
	"AntiAliasing": "Anti-aliasing filter amount.",
	"One":          "Pitch interval for voice 1.",
	"Two":          "Pitch interval for voice 2.",
	"Three":        "Pitch interval for voice 3.",
	"Four":         "Pitch interval for voice 4.",
	"MinVolume":    "Minimum volume (expression heel position).",
	"Return":       "Return level.",
	"Send":         "Send level.",
	"FXOut":        "Effects output level.",
	"Offset":       "Offset.",
	"Range":        "Range setting.",
	"Scale":        "Scale selection.",
	"InputGain":    "Input gain.",
	"Sync":         "Syncs the parameter to tempo.",
}

// paramDescription returns a plain-language description for a parameter key,
// normalising "2" (doubled-state) and trailing-digit (voice/band) variants to
// their base name.
func paramDescription(key string) string {
	if d, ok := paramDescriptions[key]; ok {
		return d
	}
	if strings.HasSuffix(key, "2") {
		if d, ok := paramDescriptions[strings.TrimSuffix(key, "2")]; ok {
			return d
		}
	}
	if n := len(key); n > 1 {
		last := key[n-1]
		if last >= '0' && last <= '9' {
			if d, ok := paramDescriptions[key[:n-1]]; ok {
				return d
			}
		}
	}
	return ""
}
