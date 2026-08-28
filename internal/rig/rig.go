// Package rig builds HeadRush Gigboard rig (.rig) files from a declarative
// specification. The builder reproduces the exact on-disk format used by the
// device: an outer JSON envelope whose "content" field is a second JSON
// document, with the signal chain described by the Patch node.
package rig

import (
	"fmt"
	"strings"

	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/catalog"
)

// Item is a single parameter value inside a module node. Exactly one of
// Value/Str/State is meaningful depending on Type:
//
//	0  numeric value (fader/knob)
//	1  boolean state (toggle)
//	3  double-state boolean
//	4  string selection (enumerated type)
//	8  free text label (PresetName)
type Item struct {
	Type  int      `json:"type"`
	Value *float64 `json:"value,omitempty"`
	Str   *string  `json:"string,omitempty"`
	State *bool    `json:"state,omitempty"`
}

// Constructors keep the item encoding in one place.
func num(v float64) *Item   { return &Item{Type: 0, Value: &v} }
func boolean(v bool) *Item  { return &Item{Type: 1, State: &v} }
func dblState(v bool) *Item { return &Item{Type: 3, State: &v} }
func str(v string) *Item    { return &Item{Type: 4, Str: &v} }
func label(v string) *Item  { return &Item{Type: 8, Str: &v} }

// Node is a module (or fixed section) inside the Patch.
type Node struct {
	ChildOrder []string         `json:"childorder"`
	Children   map[string]*Item `json:"children"`
	Type       string           `json:"type"`
}

func newNode(childOrder ...string) *Node {
	return &Node{ChildOrder: childOrder, Children: make(map[string]*Item), Type: "Node"}
}

func (n *Node) set(name string, item *Item) {
	n.Children[name] = item
}

// Block is one module in the signal chain the caller wants to build. Type is
// the device module display name (e.g. "Amp", "Cab", "Tape Echo"). Params
// overrides the module defaults; values are float64, bool or string.
type Block struct {
	Type    string
	Enabled bool
	Params  map[string]any
}

// Spec describes the rig to build. Blocks must be in signal-chain order and
// include exactly one "Amp" and one "Cab".
type Spec struct {
	Name      string
	Author    string
	Color     int
	Tempo     float64
	InputGain float64
	Blocks    []Block
}

// applyParams merges user overrides onto a node's children, preserving the
// parameter type of any existing default value.
func applyParams(children map[string]*Item, params map[string]any) {
	for key, v := range params {
		existing := children[key]
		switch val := v.(type) {
		case float64:
			if existing != nil && existing.Type == 0 {
				existing.Value = &val
			} else {
				children[key] = num(val)
			}
		case int:
			f := float64(val)
			if existing != nil && existing.Type == 0 {
				existing.Value = &f
			} else {
				children[key] = num(f)
			}
		case int64:
			f := float64(val)
			if existing != nil && existing.Type == 0 {
				existing.Value = &f
			} else {
				children[key] = num(f)
			}
		case bool:
			if existing != nil && (existing.Type == 1 || existing.Type == 3) {
				existing.State = &val
			} else {
				children[key] = boolean(val)
			}
		case string:
			if existing != nil && (existing.Type == 4 || existing.Type == 8) {
				existing.Str = &val
			} else {
				children[key] = str(val)
			}
		}
	}
}

// normalizeBlockName returns the canonical device display name for a block,
// matching catalog names case-insensitively.
func normalizeBlockName(cat *catalog.Catalog, name string) (string, bool) {
	for _, f := range cat.FX() {
		if strings.EqualFold(f.Name, name) {
			return f.Name, true
		}
	}
	switch strings.ToLower(name) {
	case "amp":
		return "Amp", true
	case "cab":
		return "Cab", true
	}
	return "", false
}

// validateBlocks ensures the chain is buildable.
func validateBlocks(cat *catalog.Catalog, blocks []Block) error {
	if len(blocks) > 11 {
		return fmt.Errorf("too many blocks: the Gigboard has 11 chain slots, got %d", len(blocks))
	}
	ampSeen, cabSeen := false, false
	for _, b := range blocks {
		canon, ok := normalizeBlockName(cat, b.Type)
		if !ok {
			return fmt.Errorf("unknown module type %q", b.Type)
		}
		switch canon {
		case "Amp":
			ampSeen = true
		case "Cab":
			cabSeen = true
		}
	}
	if !ampSeen {
		return fmt.Errorf("rig must contain an \"Amp\" block")
	}
	if !cabSeen {
		return fmt.Errorf("rig must contain a \"Cab\" block")
	}
	return nil
}
