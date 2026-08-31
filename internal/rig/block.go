package rig

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/d-led/guitar-modeler-mcp/internal/assets"
)

// blockEnvelope mirrors the outer JSON of a .block file: the actual module
// definition is a JSON string in the Content field.
type blockEnvelope struct {
	Content string `json:"content"`
	Type    string `json:"type"`
}

// blockNodeFromJSON parses a .block file into a module Node.
func blockNodeFromJSON(raw []byte) (*Node, error) {
	var outer blockEnvelope
	if err := json.Unmarshal(raw, &outer); err != nil {
		return nil, fmt.Errorf("parse block envelope: %w", err)
	}
	var inner struct {
		Data map[string]*Node `json:"data"`
	}
	if err := json.Unmarshal([]byte(outer.Content), &inner); err != nil {
		return nil, fmt.Errorf("parse block content: %w", err)
	}
	for key, node := range inner.Data {
		if strings.EqualFold(key, outer.Type) {
			return node, nil
		}
	}
	for _, node := range inner.Data {
		return node, nil
	}
	return nil, fmt.Errorf("block %q has no module data", outer.Type)
}

// buildFXNode builds an effect module node from the device's factory default
// block, layering on the rig-only fields (PresetName, On, Colour).
func buildFXNode(name string, enabled bool, params map[string]any) (*Node, error) {
	raw, err := assets.DefaultBlock(strings.ToUpper(name))
	if err != nil {
		return nil, fmt.Errorf("no block definition for %q: %w", name, err)
	}
	node, err := blockNodeFromJSON(raw)
	if err != nil {
		return nil, err
	}

	node.ChildOrder = append([]string{"PresetName"}, node.ChildOrder...)
	node.ChildOrder = append(node.ChildOrder, "On", "Colour")
	if node.Children == nil {
		node.Children = make(map[string]*Item)
	}
	node.Children["PresetName"] = label("")
	node.Children["On"] = boolean(enabled)
	node.Children["Colour"] = str("Green")
	node.Type = "Node"

	applyParams(node.Children, params)
	return node, nil
}

// baseType maps a module instance name ("Amp 2", "IR (1024)", "Tape Echo")
// back to its device module type name as used by the embedded block catalog:
// the instance number is stripped and the name uppercased.
func baseType(name string) string {
	if i := strings.LastIndex(name, " "); i > 0 && isNumber(name[i+1:]) {
		name = name[:i]
	}
	return strings.ToUpper(name)
}

// isNumber reports whether s is a non-empty sequence of digits.
func isNumber(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// Defaults returns the factory default parameter values a module instance
// starts from before any rig-specific overrides, keyed by parameter name.
// Rig-only structural fields (PresetName, On, Colour) are excluded so a
// bypassed or recoloured module is not flagged. nil means the module type has
// no known default.
func Defaults(instanceName string) map[string]*Item {
	var node *Node
	switch base := baseType(instanceName); base {
	case "AMP":
		node = ampNode("", nil)
	case "CAB":
		node = cabNode("", "", nil)
	case "IR", "IR (1024)":
		node = irNode(true, nil)
	default:
		raw, err := assets.DefaultBlock(base)
		if err != nil {
			return nil
		}
		n, err := blockNodeFromJSON(raw)
		if err != nil {
			return nil
		}
		node = n
	}
	if node == nil {
		return nil
	}
	out := make(map[string]*Item, len(node.Children))
	for k, v := range node.Children {
		switch k {
		case "PresetName", "PresetName2", "On", "Colour":
			continue
		}
		out[k] = v
	}
	return out
}
