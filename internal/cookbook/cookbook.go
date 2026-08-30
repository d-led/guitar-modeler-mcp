// Package cookbook ports presets across modelers by "ingredients" rather than
// by a one-to-one model table. Every block on every device is reduced to a
// Kind (amp, bassamp, cab or fx) plus a small set of canonical feature tags
// ("delay", "pitch", "tape", …). A source block's tags are then matched
// algorithmically — no agent, no fuzzy AI — against the target device's
// blocks, and a mapping table with coverage percentages is produced. The tag
// vocabulary and the keyword rules are static and live in this source, so a
// fancy delay that also pitch-shifts carries both "delay" and "pitch" and can
// stand in for a harmonizer.
package cookbook

import (
	"sort"
	"strings"
	"unicode"
)

// Kinds are the primary category of a block. KindFX covers every effect.
const (
	KindAmp     = "amp"
	KindBassAmp = "bassamp"
	KindCab     = "cab"
	KindFX      = "fx"
)

// ampGainTags statically map a catalog gain character to a feature tag.
var ampGainTags = map[string]string{
	"bass":            "bass",
	"clean":           "clean",
	"edge of breakup": "edge",
	"crunch":          "crunch",
	"high gain":       "highgain",
}

// keywordTags statically maps a lowercased substring to a feature tag. It is
// applied to a block's name, reference and description, which is how a
// sub-feature ("pitch" inside a delay) is encoded without an agent guessing.
// Entries are ordered so more specific phrases win over generic ones.
var keywordTags = []struct {
	kw  string
	tag string
}{
	{"ping pong", "pingpong"},
	{"ring mod", "ringmod"},
	{"ringmod", "ringmod"},
	{"drop tune", "pitch"},
	{"pitch shifter", "pitch"},
	{"pitch shift", "pitch"},
	{"harmonizer", "pitch"},
	{"whammy", "pitch"},
	{"wham", "pitch"},
	{"octaver", "octave"},
	{"octave", "octave"},
	{"oct fuzz", "octave"},
	{"overdrive", "drive"},
	{"distortion", "drive"},
	{"distort", "drive"},
	{"boost", "boost"},
	{"compressor", "comp"},
	{"comp", "comp"},
	{"noise gate", "gate"},
	{"noise", "gate"},
	{"gate", "gate"},
	{"equalizer", "eq"},
	{"equaliser", "eq"},
	{"parametric", "eq"},
	{"graphic", "eq"},
	{"eq", "eq"},
	{"filter", "filter"},
	{"wah", "wah"},
	{"volume pedal", "volume"},
	{"volume", "volume"},
	{"tremolo", "tremolo"},
	{"vibrato", "vibrato"},
	{"rotary", "rotary"},
	{"rotating", "rotary"},
	{"univibe", "univibe"},
	{"uni-vibe", "univibe"},
	{"vibe", "univibe"},
	{"flanger", "flanger"},
	{"flange", "flanger"},
	{"phaser", "phaser"},
	{"phase 90", "phaser"},
	{"phase shifter", "phaser"},
	{"chorus", "chorus"},
	{"doubler", "stereo"},
	{"stereo", "stereo"},
	{"detune", "detune"},
	{"delay", "delay"},
	{"echo", "delay"},
	{"reverb", "reverb"},
	{"verb", "reverb"},
	{"shimmer", "shimmer"},
	{"spring", "spring"},
	{"plate", "plate"},
	{"hall", "hall"},
	{"room", "room"},
	{"tape", "tape"},
	{"analog", "analog"},
	{"digital", "digital"},
	{"ducking", "ducking"},
	{"ducked", "ducking"},
	{"dynamic", "ducking"},
	{"swell", "swell"},
	{"reverse", "reverse"},
	{"acoustic", "acoustic"},
	{"looper", "looper"},
	{"freeze", "freeze"},
	{"hold", "freeze"},
	{"fuzz", "fuzz"},
}

// Ingredient is one block reduced to its kind and feature tags.
type Ingredient struct {
	Device string
	Kind   string
	Name   string
	Ref    string   // modeled-after / based-on / inspired-by text
	Tags   []string // canonical, sorted, unique feature tags (Kind excluded)
	// Params are the block's editable parameter names (raw, on-device names).
	// Empty when the catalog does not enumerate them.
	Params []string
}

// newIngredient builds an ingredient, deriving feature tags from the primary
// tag, the amp gain character and the keyword dictionary over name/ref/desc.
func newIngredient(device, kind, name, ref, primary, gain, desc string) Ingredient {
	tags := map[string]bool{}
	add := func(ts ...string) {
		for _, t := range ts {
			if t = strings.TrimSpace(t); t != "" {
				tags[strings.ToLower(t)] = true
			}
		}
	}
	add(primary)
	if g, ok := ampGainTags[strings.ToLower(strings.TrimSpace(gain))]; ok {
		add(g)
	}
	add(tagsFor(name + " " + ref + " " + desc)...)

	out := make([]string, 0, len(tags))
	for t := range tags {
		out = append(out, t)
	}
	sort.Strings(out)
	return Ingredient{Device: device, Kind: kind, Name: name, Ref: ref, Tags: out}
}

// tagsFor returns the feature tags a text implies, via the static keyword
// dictionary. It deliberately only applies the more specific entries so a
// block named "delay" does not also claim "pitch".
func tagsFor(text string) []string {
	lower := strings.ToLower(text)
	var out []string
	used := map[string]bool{}
	for _, k := range keywordTags {
		if !strings.Contains(lower, k.kw) {
			continue
		}
		if used[k.kw] {
			continue
		}
		used[k.kw] = true
		out = append(out, k.tag)
	}
	return out
}

// tokenize splits text into lowercase word tokens.
func tokenize(s string) []string {
	return strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})
}

// paramAliases maps raw on-device parameter names to a canonical knob name, so
// the same knob is recognised across devices: "GAIN", "DRIVE" and "OVERDRIVE"
// are all "gain". The table is deliberately conservative — ambiguous names are
// left out rather than mapped wrongly.
var paramAliases = map[string]string{
	"GAIN":          "gain",
	"GAINA":         "gain",
	"GAINB":         "gain",
	"GAIN A":        "gain",
	"GAIN B":        "gain",
	"DRIVE":         "gain",
	"OVERDRIVE":     "gain",
	"DISTORTION":    "gain",
	"DIST":          "gain",
	"SATURATION":    "gain",
	"SUSTAIN":       "gain",
	"LEVEL":         "level",
	"VOLUME":        "level",
	"OUTPUT":        "level",
	"OUT":           "level",
	"OUTPUT LEVEL":  "level",
	"MASTER":        "master",
	"MASTER VOLUME": "master",
	"BASS":          "bass",
	"MID":           "mid",
	"MIDDLE":        "mid",
	"MIDS":          "mid",
	"TREBLE":        "treble",
	"PRESENCE":      "presence",
	"TONE":          "tone",
	"MIX":           "mix",
	"DIRECT MIX":    "mix",
	"WET":           "mix",
	"BLEND":         "mix",
	"FEEDBACK":      "feedback",
	"REPEATS":       "feedback",
	"REGEN":         "feedback",
	"REGENERATION":  "feedback",
	"TIME":          "time",
	"DELAY TIME":    "time",
	"RATE":          "rate",
	"SPEED":         "rate",
	"FREQUENCY":     "rate",
	"FREQ":          "rate",
	"DEPTH":         "depth",
	"INTENSITY":     "depth",
	"THRESHOLD":     "threshold",
	"RATIO":         "ratio",
	"ATTACK":        "attack",
	"RELEASE":       "release",
	"DECAY":         "decay",
	"TAIL":          "decay",
	"LENGTH":        "decay",
	"LOW CUT":       "lowcut",
	"HIGH PASS":     "lowcut",
	"HPF":           "lowcut",
	"HIGH CUT":      "highcut",
	"LOW PASS":      "highcut",
	"LPF":           "highcut",
	"PRE DELAY":     "predelay",
	"PREDELAY":      "predelay",
	"RESONANCE":     "resonance",
	"RESO":          "resonance",
	"SPREAD":        "spread",
	"WIDTH":         "spread",
}

// canonicalParam maps a raw parameter name to its canonical knob name. Unknown
// names fall back to the normalised raw name, so "GAIN" and "DRIVE" meet at
// "gain" while "FLUTTER" stays "flutter".
func canonicalParam(raw string) string {
	name := strings.Join(strings.Fields(strings.ToUpper(strings.TrimSpace(raw))), " ")
	if c, ok := paramAliases[name]; ok {
		return c
	}
	return strings.ToLower(name)
}

// jaccard returns the Jaccard similarity of two token sets.
func jaccard(a, b []string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0
	}
	set := map[string]bool{}
	for _, t := range a {
		set[t] = true
	}
	inter := 0
	union := len(set)
	seen := map[string]bool{}
	for _, t := range b {
		if set[t] {
			if !seen[t] {
				inter++
				seen[t] = true
			}
		} else if !seen[t] {
			union++
			seen[t] = true
		}
	}
	if union == 0 {
		return 0
	}
	return float64(inter) / float64(union)
}
