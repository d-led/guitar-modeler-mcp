package rig

import (
	"fmt"
	"strconv"
	"strings"
)

// Summary is a structured, agent-friendly description of a rig file: the chain
// order, the parallel-path mixer and every module's effective parameter values.
type Summary struct {
	Name         string              `json:"name"`
	ID           string              `json:"id"`
	Color        int                 `json:"color"`
	CreatedAt    int64               `json:"created_at"`
	Version      string              `json:"version"`
	Tempo        float64             `json:"tempo"`
	InputGain    float64             `json:"input_gain"`    // Input node, dB
	OutputVolume float64             `json:"output_volume"` // Output node RigVolume, dB
	Routing      string              `json:"routing"`
	Mixer        MixerSummary        `json:"mixer"`
	Slots        []string            `json:"slots"`
	Modules      []SummaryModule     `json:"modules"`
	Footswitches []FootswitchSummary `json:"footswitches,omitempty"`
}

// FootswitchSummary is one assigned stomp switch.
type FootswitchSummary struct {
	Switch    string `json:"switch"` // FS5..FS8
	Module    string `json:"module"`
	Operation string `json:"operation"`
	Mode      string `json:"mode,omitempty"`  // "Toggle" or "Scene"
	Label     string `json:"label,omitempty"` // on-screen text, e.g. "DRIVE"
}

// FootswitchLine renders the assigned stomp switches as a one-liner
// ("FS5=Wham (On), FS6=Amp (On)") or a "none assigned" message.
func FootswitchLine(fs []FootswitchSummary) string {
	if len(fs) == 0 {
		return "none assigned"
	}
	parts := make([]string, 0, len(fs))
	for _, f := range fs {
		parts = append(parts, fmt.Sprintf("%s=%s (%s)", f.Switch, f.Module, f.Operation))
	}
	return strings.Join(parts, ", ")
}

// MixerSummary is the parallel-path mixer: the per-path level, pan and delay
// that balance the two paths of a split (SPS-1 / PS-1) chain.
type MixerSummary struct {
	Tails      bool    `json:"tails"`
	Para1Level float64 `json:"para1_level"`
	Para2Level float64 `json:"para2_level"`
	Para1Pan   float64 `json:"para1_pan"`
	Para2Pan   float64 `json:"para2_pan"`
	ParaDelay  float64 `json:"para_delay"`
}

// SummaryModule is one module in the chain with its parameter values.
type SummaryModule struct {
	Name   string         `json:"name"`
	On     bool           `json:"on"`
	Params map[string]any `json:"params"`
}

// Describe decodes a rig file into a human- and agent-readable summary.
func Describe(file *RigFile) (Summary, error) {
	content, err := file.Decode()
	if err != nil {
		return Summary{}, err
	}
	patch := content.Data.Patch

	s := Summary{
		Name:      file.Name(),
		ID:        file.ID,
		Color:     file.Color,
		CreatedAt: file.CreatedAt,
		Version:   content.Info.Version,
		Slots:     chainSlots(patch),
	}

	s.Footswitches = footswitchAssignments(content)

	if chain, ok := patch.Children["Chain"]; ok {
		if item, ok := chain.Children["Routing"]; ok && item.Str != nil {
			s.Routing = *item.Str
		}
		s.Mixer = mixerSummary(chain)
	}

	if rigNode, ok := patch.Children["Rig"]; ok {
		if item, ok := rigNode.Children["Tempo"]; ok && item.Value != nil {
			s.Tempo = *item.Value
		}
	}
	if in, ok := patch.Children["Input"]; ok {
		if item, ok := in.Children["InputGain"]; ok && item.Value != nil {
			s.InputGain = *item.Value
		}
	}
	if out, ok := patch.Children["Output"]; ok {
		if item, ok := out.Children["RigVolume"]; ok && item.Value != nil {
			s.OutputVolume = *item.Value
		}
	}

	for _, name := range patch.ChildOrder {
		if isStructuralNode(name) {
			continue
		}
		node, ok := patch.Children[name]
		if !ok {
			continue
		}
		s.Modules = append(s.Modules, SummaryModule{
			Name:   name,
			On:     moduleOn(node),
			Params: moduleParams(node),
		})
	}
	return s, nil
}

func isStructuralNode(name string) bool {
	switch name {
	case "Chain", "Rig", "Input", "Output", "Mix":
		return true
	}
	return false
}

// mixerSummary reads the parallel-path mixer values from the Chain node. Every
// rig carries these fields, even a serial one (where they sit at their
// defaults and have no effect).
func mixerSummary(chain *Node) MixerSummary {
	var m MixerSummary
	if item, ok := chain.Children["Tails"]; ok && item.State != nil {
		m.Tails = *item.State
	}
	num := func(key string) float64 {
		if item, ok := chain.Children[key]; ok && item.Value != nil {
			return *item.Value
		}
		return 0
	}
	m.Para1Level = num("Para1Level")
	m.Para2Level = num("Para2Level")
	m.Para1Pan = num("Para1Pan")
	m.Para2Pan = num("Para2Pan")
	m.ParaDelay = num("ParaDelay")
	return m
}

// footswitchAssignments lists the stomp switches that are assigned to a
// module, in FS5..FS8 order. Unassigned switches are omitted.
func footswitchAssignments(content *Content) []FootswitchSummary {
	children := namedChildren(content.FootSwitch, "FootSwitch")
	if children == nil {
		return nil
	}
	var out []FootswitchSummary
	for _, n := range []string{"5", "6", "7", "8"} {
		module := childString(children, "Module"+n)
		if module == "" || module == "Unassigned" {
			continue
		}
		out = append(out, FootswitchSummary{
			Switch:    "FS" + n,
			Module:    module,
			Operation: childString(children, "Operation"+n),
			Mode:      childString(children, "ModeNew"+n),
			Label:     childString(children, "UserFootSwitchText"+n),
		})
	}
	return out
}

// chainSlots returns the ModuleType1..ModuleType11 values of the Chain node,
// including "Empty Slot" placeholders, so the full 11-slot layout is visible.
func chainSlots(patch Patch) []string {
	chain, ok := patch.Children["Chain"]
	if !ok {
		return nil
	}
	slots := make([]string, 0, 11)
	for i := 1; i <= 11; i++ {
		key := "ModuleType" + strconv.Itoa(i)
		item, ok := chain.Children[key]
		if !ok || item.Str == nil {
			continue
		}
		slots = append(slots, *item.Str)
	}
	return slots
}

func moduleOn(node *Node) bool {
	if item, ok := node.Children["On"]; ok && item.State != nil {
		return *item.State
	}
	return false
}

func moduleParams(node *Node) map[string]any {
	params := make(map[string]any, len(node.ChildOrder))
	for _, key := range node.ChildOrder {
		if key == "PresetName" || key == "PresetName2" || key == "On" || key == "Colour" {
			continue
		}
		item, ok := node.Children[key]
		if !ok {
			continue
		}
		params[key] = itemAny(item)
	}
	return params
}

func itemAny(item *Item) any {
	switch {
	case item.Value != nil:
		return *item.Value
	case item.Str != nil:
		return *item.Str
	case item.State != nil:
		return *item.State
	}
	return nil
}
