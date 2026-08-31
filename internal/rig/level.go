package rig

import (
	"fmt"
	"math"
	"strings"
)

// LevelStage is one gain stage contributing to a rig's output level.
type LevelStage struct {
	Stage string  `json:"stage"`
	DB    float64 `json:"db"`
	Note  string  `json:"note,omitempty"`
}

// LevelEstimate is the estimated output level of a rig and the RigVolume that
// would bring it to the requested target level.
type LevelEstimate struct {
	Routing              string       `json:"routing"`
	Stages               []LevelStage `json:"stages"`
	EstimatedLevelDB     float64      `json:"estimated_level_db"`
	TargetDB             float64      `json:"target_db"`
	OutputRigVolume      float64      `json:"output_rig_volume"`
	RecommendedRigVolume float64      `json:"recommended_rig_volume"`
	// Notes flags what the dB sum deliberately leaves out, so the estimate is
	// read as a relative hint rather than an absolute measurement.
	Notes []string `json:"notes,omitempty"`
}

// EstimateLevel sums the level-relevant stages of a rig (input gain, amp
// master, cab out gain, volume pedals, the parallel-path mixer and the output
// RigVolume) into a net output level in dB. Amp master and volume-pedal
// positions are linear percentage knobs, converted with 20·log10(v/100) — an
// estimate, not a measurement. The recommended RigVolume is the output level
// to set to reach targetDB (clamped to the device's observed -10..+20 dB).
func EstimateLevel(file *RigFile, targetDB float64) (LevelEstimate, error) {
	content, err := file.Decode()
	if err != nil {
		return LevelEstimate{}, err
	}
	est := estimateLevel(content.Data.Patch)
	est.TargetDB = targetDB
	est.RecommendedRigVolume = round1(clamp(est.OutputRigVolume+(targetDB-est.EstimatedLevelDB), -10, 20))
	return est, nil
}

// estimateLevel sums the level-relevant stages of a built patch into a net
// output level. It is shared by EstimateLevel and the build-time plausibility
// check so both agree on the numbers.
func estimateLevel(patch Patch) LevelEstimate {
	est := LevelEstimate{
		Routing: nodeString(patch.Children["Chain"], "Routing"),
	}

	total := 0.0
	add := func(stage string, db float64, note string) {
		est.Stages = append(est.Stages, LevelStage{Stage: stage, DB: round1(db), Note: note})
		total += db
	}

	if in, ok := patch.Children["Input"]; ok {
		add("input gain", nodeNumber(in, "InputGain"), "")
	}

	sawAmp := addTypeStages(patch, "Amp", add, func(name string, node *Node) (string, float64, string) {
		master := nodeNumber(node, "Master")
		return "amp master (" + name + ")", percentToDB(master), fmt.Sprintf("master %s", percent(master))
	})
	addTypeStages(patch, "Cab", add, func(name string, node *Node) (string, float64, string) {
		return "cab out gain (" + name + ")", nodeNumber(node, "OutGain"), ""
	})
	addTypeStages(patch, "IR", add, func(name string, node *Node) (string, float64, string) {
		level, note := irStage(node)
		return "IR (" + name + ")", level, note
	})
	addTypeStages(patch, "Volume", add, func(name string, node *Node) (string, float64, string) {
		v := nodeNumber(node, "Volume")
		return "volume pedal (" + name + ")", percentToDB(v), fmt.Sprintf("position %s", percent(v))
	})

	if est.Routing != "" && est.Routing != "S" {
		chain := patch.Children["Chain"]
		p1 := nodeNumber(chain, "Para1Level")
		p2 := nodeNumber(chain, "Para2Level")
		add("parallel paths (louder)", math.Max(p1, p2), fmt.Sprintf("path A %s, path B %s", dB(p1), dB(p2)))
	}

	if out, ok := patch.Children["Output"]; ok {
		est.OutputRigVolume = nodeNumber(out, "RigVolume")
		add("output rig volume", est.OutputRigVolume, "")
	}

	if sawAmp {
		est.Notes = append(est.Notes,
			"amp preamp gain (GainA/GainB) and any drive-pedal Level are not in this sum: a high-gain amp plays louder than the estimate suggests. Set loudness with the amp Master (power-amp volume), drive with Gain, and use RigVolume only as a final trim.")
	}

	est.EstimatedLevelDB = round1(total)
	return est
}

// addTypeStages appends one stage per instance of base in the patch child order
// and reports whether any instance was found.
func addTypeStages(patch Patch, base string, add func(string, float64, string), measure func(string, *Node) (string, float64, string)) bool {
	found := false
	for _, name := range patch.ChildOrder {
		if !isInstanceOf(name, base) {
			continue
		}
		found = true
		label, db, note := measure(name, patch.Children[name])
		add(label, db, note)
	}
	return found
}

// Plausibility thresholds: a rig whose estimated net level exceeds these is
// refused at build time rather than written, to prevent accidentally very loud
// (or silently muted) presets. +20 dB matches the loudest factory presets.
const (
	maxPlausibleLevel = 20  // dB net — above this is very loud
	minPlausibleLevel = -60 // dB net — at/below this the amp is effectively muted
)

// validatePlausible refuses to build a rig that is implausibly loud or silent,
// explaining the problem and how to remediate it.
func validatePlausible(patch Patch) error {
	est := estimateLevel(patch)

	var problems []string
	if est.EstimatedLevelDB > maxPlausibleLevel {
		problems = append(problems, fmt.Sprintf(
			"estimated output level %+.1f dB is very loud (above %+d dB): lower the output level (RigVolume, now %+.1f dB), the amp master and/or the cab out gain",
			est.EstimatedLevelDB, maxPlausibleLevel, est.OutputRigVolume))
	}
	if est.EstimatedLevelDB <= minPlausibleLevel {
		problems = append(problems, fmt.Sprintf(
			"estimated output level %+.1f dB is effectively muted (the amp master is at 0%%): raise the amp master or the output level",
			est.EstimatedLevelDB))
	}

	if len(problems) == 0 {
		return nil
	}
	return fmt.Errorf("rig refused by the plausibility check:\n  - %s", strings.Join(problems, "\n  - "))
}

// isInstanceOf reports whether a node name is the base name or one of its
// numbered instances ("Amp", "Amp 2", ...).
func isInstanceOf(name, base string) bool {
	if name == base {
		return true
	}
	return strings.HasPrefix(name, base+" ")
}

func nodeNumber(node *Node, key string) float64 {
	if node == nil {
		return 0
	}
	if item, ok := node.Children[key]; ok && item.Value != nil {
		return *item.Value
	}
	return 0
}

// nodeNumberOr reads a numeric parameter, falling back to def when the item is
// absent (so an unset knob defaults to the device value rather than 0).
func nodeNumberOr(node *Node, key string, def float64) float64 {
	if node == nil {
		return def
	}
	if item, ok := node.Children[key]; ok && item.Value != nil {
		return *item.Value
	}
	return def
}

// nodeString reads a string (enumerated type or label) parameter, or "".
func nodeString(node *Node, key string) string {
	if node == nil {
		return ""
	}
	if item, ok := node.Children[key]; ok && item.Str != nil {
		return *item.Str
	}
	return ""
}

// nodeBool reads a boolean (state) parameter, defaulting to false.
func nodeBool(node *Node, key string) bool {
	if node == nil {
		return false
	}
	if item, ok := node.Children[key]; ok && item.State != nil {
		return *item.State
	}
	return false
}

// irStage estimates an IR loader's level contribution. Mix is a wet/dry blend
// (0 = dry passthrough, 100 = full wet); the blended level is
// 20·log10((1−m) + m·10^(g/20)). When Doubling is on and a second IR is
// loaded, the louder of the two is used.
func irStage(node *Node) (float64, string) {
	gain := nodeNumberOr(node, "Gain", 0)
	mix := nodeNumberOr(node, "Mix", 100)
	level := blendDB(gain, mix)
	note := fmt.Sprintf("gain %s, mix %.0f%%", dB(gain), mix)
	if nodeBool(node, "Doubling") && nodeString(node, "IR2") != "" {
		g2 := nodeNumberOr(node, "Gain2", 0)
		m2 := nodeNumberOr(node, "Mix2", 100)
		if l2 := blendDB(g2, m2); l2 > level {
			level = l2
		}
		note += fmt.Sprintf(" (doubling: gain %s, mix %.0f%%)", dB(g2), m2)
	}
	return level, note
}

// blendDB estimates the level of a wet/dry blend: mix m in 0..100 and wet gain
// g in dB. At m=100 the result is g; at m=0 it is 0 (dry passthrough).
func blendDB(g, mix float64) float64 {
	m := clamp(mix, 0, 100) / 100
	return 20 * math.Log10((1-m)+m*math.Pow(10, g/20))
}

// percentToDB converts a 0..100 percentage knob to a dB estimate (0 = mute).
func percentToDB(p float64) float64 {
	if p <= 0 {
		return -60
	}
	return 20 * math.Log10(p/100)
}

func percent(p float64) string {
	return fmt.Sprintf("%.0f%%", p)
}

func dB(v float64) string {
	if v == 0 {
		return "0 dB"
	}
	return fmt.Sprintf("%+.1f dB", v)
}

func round1(v float64) float64 {
	r := math.Round(v*10) / 10
	if r == 0 {
		return 0 // normalise -0 to 0
	}
	return r
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
