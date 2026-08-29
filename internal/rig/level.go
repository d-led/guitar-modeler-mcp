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
	est := LevelEstimate{}
	if chain, ok := patch.Children["Chain"]; ok {
		if item, ok := chain.Children["Routing"]; ok && item.Str != nil {
			est.Routing = *item.Str
		}
	}

	total := 0.0
	add := func(stage string, db float64, note string) {
		est.Stages = append(est.Stages, LevelStage{Stage: stage, DB: round1(db), Note: note})
		total += db
	}

	if in, ok := patch.Children["Input"]; ok {
		add("input gain", nodeNumber(in, "InputGain"), "")
	}

	for _, name := range patch.ChildOrder {
		if !isInstanceOf(name, "Amp") {
			continue
		}
		node := patch.Children[name]
		master := nodeNumber(node, "Master")
		add("amp master ("+name+")", percentToDB(master), fmt.Sprintf("master %s", percent(master)))
	}

	for _, name := range patch.ChildOrder {
		if !isInstanceOf(name, "Cab") {
			continue
		}
		add("cab out gain ("+name+")", nodeNumber(patch.Children[name], "OutGain"), "")
	}

	for _, name := range patch.ChildOrder {
		if !isInstanceOf(name, "Volume") {
			continue
		}
		v := nodeNumber(patch.Children[name], "Volume")
		add("volume pedal ("+name+")", percentToDB(v), fmt.Sprintf("position %s", percent(v)))
	}

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

	est.EstimatedLevelDB = round1(total)
	return est
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
