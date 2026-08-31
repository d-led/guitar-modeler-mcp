package rig

import "encoding/json"

// ButtonAssign is one stomp button. Buttons 1..4 are the device's four
// footswitches (FS5..FS8 in the file); an unassigned button has an empty
// Module.
type ButtonAssign struct {
	Number    int    `json:"number"` // 1..4
	Module    string `json:"module"` // module instance name, "" = unassigned
	Operation string `json:"operation"`
	Mode      string `json:"mode"`          // "Toggle" or "Scene"
	Label     string `json:"label"`         // on-screen switch text (UserFootSwitchText), e.g. "DRIVE"
	Off       bool   `json:"off,omitempty"` // target inactive at load: a bypassed toggle, or a scene other than the active one
}

// PedalTarget is one expression-pedal assignment: the module, its parameter
// and the heel-to-toe sweep.
type PedalTarget struct {
	Module string  `json:"module"`
	Param  string  `json:"param"`
	Min    float64 `json:"min"`
	Max    float64 `json:"max"`
}

// PedalAssign is one expression pedal. Pedal 1 is the built-in pedal, Pedal 2
// the optional external one.
type PedalAssign struct {
	Name    string        `json:"name"`
	Mode    string        `json:"mode,omitempty"` // PedalMode (Pedal 1 only)
	Targets []PedalTarget `json:"targets"`
}

// Hardware is the rig's physical-control assignments: the four stomp buttons
// and the expression pedals.
type Hardware struct {
	Buttons []ButtonAssign `json:"buttons"`
	Pedals  []PedalAssign  `json:"pedals"`
}

// HardwareAssignments decodes a rig file and extracts its physical-control
// assignments for display.
func HardwareAssignments(rf *RigFile) (Hardware, error) {
	content, err := rf.Decode()
	if err != nil {
		return Hardware{}, err
	}

	h := Hardware{}
	patch := content.Data.Patch
	children := namedChildren(decodeSection(content.FootSwitch), "FootSwitch")
	lastScene := sceneIndex(children)
	for i, n := range []string{"5", "6", "7", "8"} {
		module := assignedName(childString(children, "Module"+n))
		// Mode and label only belong to an assigned switch; the template may
		// still carry a stale value on an unassigned button.
		mode, label, operation, off := "", "", "", false
		if module != "" {
			mode = childString(children, "ModeNew"+n)
			label = childString(children, "UserFootSwitchText"+n)
			operation = childString(children, "Operation"+n)
			// A button starts dimmed when its target is inactive at load: a
			// toggle whose module is bypassed, or a scene other than the one
			// the rig loads with (LastScene).
			if mode == "Scene" {
				off = lastScene != i
			} else if operation == "" || operation == "On" {
				off = nodeStartsOff(patch, module)
			}
		}
		h.Buttons = append(h.Buttons, ButtonAssign{
			Number:    i + 1,
			Module:    module,
			Operation: operation,
			Mode:      mode,
			Label:     label,
			Off:       off,
		})
	}
	h.Pedals = append(h.Pedals,
		pedalAssign(decodeSection(content.Pedal1), "Pedal1", "Pedal 1"),
		pedalAssign(decodeSection(content.Pedal2), "Pedal2", "Pedal 2"),
	)
	return h, nil
}

// decodeSection unmarshals a raw content section (FootSwitch, Pedal1, Pedal2)
// back into its object form for inspection.
func decodeSection(raw json.RawMessage) map[string]any {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil
	}
	return m
}

// nodeStartsOff reports whether the patch module's On state is false. Modules
// missing from the patch report false.
func nodeStartsOff(patch Patch, name string) bool {
	node, ok := patch.Children[name]
	if !ok || node == nil {
		return false
	}
	item, ok := node.Children["On"]
	return ok && item.State != nil && !*item.State
}

// sceneIndex returns the active scene's 0-based button index (0..3 for
// FS5..FS8), or -1 when no scene is active. The FootSwitch LastScene field
// stores the footswitch number (5..8), so it is normalised here.
func sceneIndex(children map[string]any) int {
	v, ok := children["LastScene"].(map[string]any)
	if !ok {
		return -1
	}
	num, _ := v["value"].(float64)
	if num >= 5 && num <= 8 {
		return int(num) - 5
	}
	return -1
}

// namedChildren unwraps the {data:{<name>:{children:{…}}}} wrapper used by the
// FootSwitch and Pedal nodes.
func namedChildren(raw any, name string) map[string]any {
	node, ok := raw.(map[string]any)
	if !ok {
		return nil
	}
	data, ok := node["data"].(map[string]any)
	if !ok {
		return nil
	}
	inner, ok := data[name].(map[string]any)
	if !ok {
		return nil
	}
	children, _ := inner["children"].(map[string]any)
	return children
}

func pedalAssign(raw any, key, display string) PedalAssign {
	p := PedalAssign{Name: display}
	children := namedChildren(raw, key)
	if children == nil {
		return p
	}
	p.Mode = childString(children, "PedalMode")
	for _, n := range []string{"1", "2", "3", "4"} {
		module := childString(children, "Module"+n)
		if module == "" || module == "Unassigned" {
			continue
		}
		p.Targets = append(p.Targets, PedalTarget{
			Module: module,
			Param:  childString(children, "Param"+n),
			Min:    childNumber(children, "Min"+n),
			Max:    childNumber(children, "Max"+n),
		})
	}
	return p
}

// assignedName maps the device's "Unassigned" placeholder to "", so callers can
// test emptiness with a single comparison.
func assignedName(s string) string {
	if s == "Unassigned" {
		return ""
	}
	return s
}

// childString reads a string-typed child (its "string" field), or "" when the
// child is absent or of a different type.
func childString(children map[string]any, key string) string {
	item, ok := children[key].(map[string]any)
	if !ok {
		return ""
	}
	s, _ := item["string"].(string)
	return s
}

// childNumber reads a numeric child (its "value" field), or 0 when absent.
func childNumber(children map[string]any, key string) float64 {
	item, ok := children[key].(map[string]any)
	if !ok {
		return 0
	}
	v, _ := item["value"].(float64)
	return v
}
