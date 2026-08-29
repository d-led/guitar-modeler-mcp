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
