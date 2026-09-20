package rig

import (
	"fmt"
	"math"
	"strings"

	"github.com/d-led/guitar-modeler-mcp/internal/headrush/catalog"
	"github.com/d-led/guitar-modeler-mcp/internal/headrush/modspec"
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
// preamp gain and master, cab out gain, drive/compressor/EQ knobs, volume
// pedals, the parallel-path mixer and the output RigVolume) into a net output
// level in dB. Amp gain/master and volume-pedal positions are linear percentage
// knobs, converted with 20·log10(v/100); a compressor's output Level is a
// makeup balance whose unity/bypass point sits at 50 (noon) on the device, not
// 100, so it is converted about 50 — an estimate, not a measurement. The
// recommended RigVolume is the output level to set to reach targetDB (clamped
// to the device's observed -10..+20 dB).
func EstimateLevel(file *RigFile, targetDB float64) (LevelEstimate, error) {
	content, err := file.Decode()
	if err != nil {
		return LevelEstimate{}, err
	}
	sceneOn, sceneNote := sceneAudibleBlocks(content)
	est := estimateLevelWithScenes(catalog.New(), content.Data.Patch, sceneOn)
	if sceneNote != "" {
		est.Notes = append(est.Notes, sceneNote)
	}
	est.TargetDB = targetDB
	est.RecommendedRigVolume = round1(clamp(est.OutputRigVolume+(targetDB-est.EstimatedLevelDB), -10, 20))
	return est, nil
}

// estimateLevel sums the level-relevant stages of a built patch into a net
// output level. It is shared by EstimateLevel and the build-time plausibility
// check so both agree on the numbers.
func estimateLevel(cat *catalog.Catalog, patch Patch) LevelEstimate {
	return estimateLevelWithScenes(cat, patch, nil)
}

// estimateLevelWithScenes is estimateLevel plus the set of blocks a scene
// snapshot turns on (sceneOn), so a drive bypassed in the patch but engaged by
// a scene is still counted in the net level. For a parallel rig the two paths
// are estimated separately (each block brings its own level), and the louder
// path carries the net.
func estimateLevelWithScenes(cat *catalog.Catalog, patch Patch, sceneOn map[string]bool) LevelEstimate {
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

	shared, pathA, pathB := parallelSections(patch)
	if len(pathA) == 0 && len(pathB) == 0 {
		addBlocks(cat, patch, blockNames(patch), sceneOn, add)
	} else {
		chain := patch.Children["Chain"]
		addBlocks(cat, patch, shared, sceneOn, add)
		aStages := blocksLevelStages(cat, patch, pathA, sceneOn)
		bStages := blocksLevelStages(cat, patch, pathB, sceneOn)
		aDB := stageDB(aStages) + nodeNumber(chain, "Para1Level")
		bDB := stageDB(bStages) + nodeNumber(chain, "Para2Level")
		showPath(&est, "path A", pathA, aStages, nodeNumber(chain, "Para1Level"))
		showPath(&est, "path B", pathB, bStages, nodeNumber(chain, "Para2Level"))
		add("parallel paths (louder)", math.Max(aDB, bDB), fmt.Sprintf("path A %s, path B %s", dB(aDB), dB(bDB)))
	}

	if out, ok := patch.Children["Output"]; ok {
		est.OutputRigVolume = nodeNumber(out, "RigVolume")
		add("output rig volume", est.OutputRigVolume, "")
	}

	if hasBlockType(patch, "AMP") {
		est.Notes = append(est.Notes,
			"the estimate sums each block's output-level and EQ-gain knobs (drive Level, compressor makeup, EQ bands) plus the amp's preamp gain and Master; it still omits wet/dry Mix and a drive's saturation, so a pushed amp or heavy drive plays louder than the sum suggests.")
	}

	est.EstimatedLevelDB = round1(total)
	return est
}

// blockNames returns the movable signal blocks of the patch in child order,
// excluding the Chain/Rig/Input/Output/Mix bookkeeping nodes.
func blockNames(patch Patch) []string {
	var names []string
	for _, name := range patch.ChildOrder {
		if !structuralBlock(name) {
			names = append(names, name)
		}
	}
	return names
}

// structuralBlock reports whether a patch child is a bookkeeping node rather
// than a signal block.
func structuralBlock(name string) bool {
	switch baseType(name) {
	case "CHAIN", "RIG", "INPUT", "OUTPUT", "MIX":
		return true
	}
	return false
}

// hasBlockType reports whether any signal block in the patch has the given
// uppercase base type.
func hasBlockType(patch Patch, base string) bool {
	for _, name := range blockNames(patch) {
		if baseType(name) == base {
			return true
		}
	}
	return false
}

// addBlocks appends the named blocks' level stages to the running total.
func addBlocks(cat *catalog.Catalog, patch Patch, names []string, sceneOn map[string]bool, add func(string, float64, string)) {
	for _, name := range names {
		for _, s := range blockStages(cat, patch, name, sceneOn) {
			add(s.Stage, s.DB, s.Note)
		}
	}
}

// blockStages returns the level stages a single block contributes: amp preamp
// gain and master, cab out gain, IR gain, volume-pedal position, or an effect's
// level knobs. A bypassed effect contributes nothing unless a scene turns it on
// (sceneOn); structural blocks (amp/cab/IR/volume) are always counted.
func blockStages(cat *catalog.Catalog, patch Patch, name string, sceneOn map[string]bool) []LevelStage {
	node := patch.Children[name]
	if node == nil {
		return nil
	}
	base := baseType(name)
	on := nodeBool(node, "On")
	engaged := sceneEngaged(name, sceneOn) && !on
	if isFXType(base) && !on && !engaged {
		return nil
	}
	stages := blockStageLevels(cat, name, node, base)
	if engaged {
		markEngaged(stages)
	}
	return stages
}

// blockStageLevels maps one block to its level stages by device type.
func blockStageLevels(cat *catalog.Catalog, name string, node *Node, base string) []LevelStage {
	switch base {
	case "AMP":
		return ampStages(name, node)
	case "CAB":
		return cabStages(name, node)
	case "IR", "IR (1024)":
		level, note := irStage(node)
		return []LevelStage{{Stage: "IR (" + name + ")", DB: level, Note: note}}
	case "VOLUME":
		v := nodeNumber(node, "Volume")
		return []LevelStage{{Stage: "volume pedal (" + name + ")", DB: percentToDB(v), Note: fmt.Sprintf("position %s", percent(v))}}
	default:
		return fxLevelStages(cat, name, node)
	}
}

// ampStages returns an amp's level stages: power-amp Master and the louder of
// the two preamp gains.
func ampStages(name string, node *Node) []LevelStage {
	master := nodeNumber(node, "Master")
	gain := math.Max(nodeNumber(node, "GainA"), nodeNumber(node, "GainB"))
	return []LevelStage{
		{Stage: "amp master (" + name + ")", DB: percentToDB(master), Note: fmt.Sprintf("master %s", percent(master))},
		{Stage: "amp preamp gain (" + name + ")", DB: percentToDB(gain), Note: fmt.Sprintf("gain %s (louder of GainA/GainB)", percent(gain))},
	}
}

// cabStages returns a cab's level stage: its OutGain.
func cabStages(name string, node *Node) []LevelStage {
	return []LevelStage{{Stage: "cab out gain (" + name + ")", DB: nodeNumber(node, "OutGain"), Note: ""}}
}

// isFXType reports whether a base type is a movable effect (distortion,
// dynamics, eq, delay, …) rather than a structural block.
func isFXType(base string) bool {
	switch base {
	case "AMP", "CAB", "IR", "IR (1024)", "VOLUME":
		return false
	}
	return true
}

// sceneEngaged reports whether a scene switch turns the named block on.
func sceneEngaged(name string, sceneOn map[string]bool) bool {
	return sceneOn != nil && sceneOn[name]
}

// markEngaged annotates a block's stages with the scene that engages it.
func markEngaged(stages []LevelStage) {
	for i := range stages {
		stages[i].Note += " (engaged by a scene)"
	}
}

// blocksLevelStages concatenates the level stages of the named blocks.
func blocksLevelStages(cat *catalog.Catalog, patch Patch, names []string, sceneOn map[string]bool) []LevelStage {
	var stages []LevelStage
	for _, name := range names {
		stages = append(stages, blockStages(cat, patch, name, sceneOn)...)
	}
	return stages
}

// stageDB sums the dB of a set of level stages.
func stageDB(stages []LevelStage) float64 {
	total := 0.0
	for _, s := range stages {
		total += s.DB
	}
	return total
}

// showPath appends one parallel path's block stages and mixer level for display
// only: the "parallel paths (louder)" stage already carries the louder path
// into the total, so these must not be counted twice.
func showPath(est *LevelEstimate, label string, names []string, stages []LevelStage, mixer float64) {
	for _, s := range stages {
		est.Stages = append(est.Stages, LevelStage{Stage: label + " " + s.Stage, DB: round1(s.DB), Note: s.Note})
	}
	est.Stages = append(est.Stages, LevelStage{Stage: label + " mixer level", DB: round1(mixer), Note: fmt.Sprintf("blocks %s", strings.Join(names, ", "))})
}

// parallelSections returns the blocks on each side of a parallel split and the
// shared (prefix + suffix) blocks, read from the Chain node's ModuleType1..11
// slots with the same slot budgets the builder lays out. A serial chain returns
// empty paths. The slot ranges are the device's fixed layout, not a heuristic.
func parallelSections(patch Patch) (shared, pathA, pathB []string) {
	chain := patch.Children["Chain"]
	if chain == nil {
		return nil, nil, nil
	}
	slots := make([]string, 11)
	for i := 0; i < 11; i++ {
		slots[i] = nodeString(chain, fmt.Sprintf("ModuleType%d", i+1))
	}
	section := func(from, to int) []string {
		var out []string
		for i := from; i <= to; i++ {
			if slots[i] != "" && slots[i] != "Empty Slot" {
				out = append(out, slots[i])
			}
		}
		return out
	}
	// 0-based slot ranges mirror the builder's budgets (rig.go): SPS-1 = 3
	// prefix + 3 path A + 3 path B + 2 suffix; PS-1 = 3 path A + 5 path B + 3
	// suffix.
	switch nodeString(chain, "Routing") {
	case "SPS-1":
		shared = append(section(0, 2), section(9, 10)...)
		pathA = section(3, 5)
		pathB = section(6, 8)
	case "PS-1":
		shared = section(8, 10)
		pathA = section(0, 2)
		pathB = section(3, 7)
	}
	return shared, pathA, pathB
}

// sceneAudibleBlocks returns the blocks a Scene-mode switch turns on, plus a
// note naming which switch engages which block. A block bypassed in the patch
// but turned on by a scene would otherwise be invisible to the level estimate.
func sceneAudibleBlocks(content *Content) (map[string]bool, string) {
	on := make(map[string]bool)
	var parts []string
	for _, fs := range footswitchAssignments(content) {
		if fs.Scene == nil {
			continue
		}
		for _, name := range fs.Scene.On {
			if on[name] {
				continue
			}
			on[name] = true
			parts = append(parts, fmt.Sprintf("%s engages %s", fs.Switch, name))
		}
	}
	if len(parts) == 0 {
		return nil, ""
	}
	return on, strings.Join(parts, "; ")
}

// fxLevelStages returns one level stage per level-relevant knob of an effect
// module, driven by the module's category and parameter spec: EQ bands and
// trims are dB values added as-is; a drive output Level is a percent volume
// knob converted about 100 (unity), while a compressor's output Level is a
// makeup balance converted about 50 — its unity/bypass point on the device.
// Time-based and modulation effects contribute nothing — their Mix blends wet
// into dry rather than raising the overall level.
func fxLevelStages(cat *catalog.Catalog, name string, node *Node) []LevelStage {
	fx, ok := cat.FXByName(baseType(name))
	if !ok {
		return nil
	}
	spec, ok := modspec.Get(fx.Name)
	if !ok {
		return nil
	}

	var stages []LevelStage
	add := func(key string, dbv float64, note string) {
		stages = append(stages, LevelStage{Stage: fx.Name + " " + key, DB: dbv, Note: note})
	}

	switch fx.Category {
	case "eq":
		eqLevelStages(spec, node, add)
	case "dynamics":
		dynamicsLevelStages(spec, node, add)
	case "distortion":
		distortionLevelStages(spec, node, add)
	}
	return stages
}

// eqLevelStages adds an EQ's overall output-trim knob as a level stage. The
// frequency-band knobs (LoGain, MidGain, HiGain, …) reshape tone at their
// centre frequencies and do not sum to a broadband level change, so they are
// deliberately not counted.
func eqLevelStages(spec modspec.Module, node *Node, add func(string, float64, string)) {
	for key, p := range spec {
		if p.Kind == "range" && strings.TrimSpace(p.Unit) == "dB" && isEQTrimKey(key) {
			v := nodeNumber(node, key)
			add(key, v, dB(v))
		}
	}
}

// isEQTrimKey reports whether an EQ knob name is the module's overall output
// trim (plain Gain/Level/Output/Volume) rather than a frequency band, whose
// names carry a band prefix (Lo, LoMid, Mid, HiMid, Hi).
func isEQTrimKey(key string) bool {
	switch key {
	case "Gain", "Level", "Output", "Volume":
		return true
	}
	return false
}

// dynamicsLevelStages adds a compressor's output Level (a percent makeup
// balance, unity at 50) or makeup Gain (dB); the detector controls
// (threshold/ratio/attack/release) are not level.
func dynamicsLevelStages(spec modspec.Module, node *Node, add func(string, float64, string)) {
	for key, p := range spec {
		if p.Kind != "range" {
			continue
		}
		unit := strings.TrimSpace(p.Unit)
		switch {
		case unit == "dB" && key == "Gain":
			v := nodeNumber(node, key)
			add(key, v, dB(v))
		case unit == "%" && (key == "Level" || key == "Volume" || key == "Output"):
			// Unity/bypass is at 50 (noon), not 100, so the 100% default
			// reads ≈ +6 dB over bypass and sounds louder when switched on.
			v := nodeNumber(node, key)
			add(key, percentAbout(v, 50), percent(v))
		}
	}
}

// distortionLevelStages adds a drive's output-level knobs (Level, Volume,
// Output, Master, DistLev); Gain and Drive are the drive amount, not level.
func distortionLevelStages(spec modspec.Module, node *Node, add func(string, float64, string)) {
	for key, p := range spec {
		if p.Kind == "range" && strings.TrimSpace(p.Unit) == "%" && isDriveLevelKey(key) {
			v := nodeNumber(node, key)
			add(key, percentToDB(v), percent(v))
		}
	}
}

// isDriveLevelKey reports whether a drive's percent knob is an output-level
// knob rather than a drive/sustain control (Drive, Gain, Sustain).
func isDriveLevelKey(key string) bool {
	switch key {
	case "Level", "Volume", "Output", "Master", "DistLev":
		return true
	}
	return false
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
func validatePlausible(cat *catalog.Catalog, patch Patch) error {
	est := estimateLevel(cat, patch)

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

// percentToDB converts a 0..100 percentage knob to a dB estimate about a unity
// reference of 100 (full = 0 dB, 0 = mute).
func percentToDB(p float64) float64 { return percentAbout(p, 100) }

// percentAbout converts a percentage knob to a dB estimate relative to the ref
// value that reads unity (0 dB): 20·log10(p/ref). Plain volume/level knobs use
// ref 100; a compressor's output Level is a makeup balance whose unity/bypass
// point on the device is 50 (noon), so it is converted about 50.
func percentAbout(p, ref float64) float64 {
	if p <= 0 {
		return -60
	}
	return 20 * math.Log10(p/ref)
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
