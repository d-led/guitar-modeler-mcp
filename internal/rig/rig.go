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

// Routing is the signal-chain topology. The Gigboard has exactly three,
// derived from the Routing field of every device backup rig:
//
//	"S"     serial: one linear path through the 11 slots.
//	"SPS-1" serial → parallel → serial: 3 shared slots, then two parallel
//	        paths of 3 slots each, then 2 shared slots.
//	"PS-1"  parallel → serial: the signal splits at the input into two paths
//	        (3 + 4..5 slots) and merges before the remaining serial slots.
type Routing string

const (
	RoutingSerial Routing = "S"
	RoutingSPS    Routing = "SPS-1"
	RoutingPS     Routing = "PS-1"
)

// Valid reports whether the routing value is one the device understands.
func (r Routing) Valid() bool {
	switch r {
	case "", RoutingSerial, RoutingSPS, RoutingPS:
		return true
	}
	return false
}

// Slot budgets for each routing topology, measured from the device backups.
// The Gigboard has 11 chain slots in total.
const (
	spsPrefixSlots = 3 // SPS-1: shared serial before the split
	spsPathSlots   = 3 // SPS-1: each parallel path
	spsSuffixSlots = 2 // SPS-1: shared serial after the merge

	psPathASlots = 3 // PS-1: first parallel path (split at the input)
	psPathBSlots = 5 // PS-1: second parallel path (observed 4..5 slots)
)

// Spec describes the rig to build.
//
// For a serial rig (Routing ""), Blocks holds the whole chain in signal order.
// For a parallel rig, Prefix/PathA/PathB/Suffix hold the chain sections; the
// builder lays them out into the 11 slots according to the routing. A rig must
// contain at least one Amp and one Cab; a second Amp block in PathB produces
// the dual-amp configuration (same or different amp models).
type Spec struct {
	Name      string
	Author    string
	Color     int
	Tempo     float64
	InputGain float64
	// OutputVolume is the rig's overall output level in dB (the Output block's
	// RigVolume), 0 = unity.
	OutputVolume float64

	// Routing selects the topology; empty means serial.
	Routing Routing

	// Blocks is the serial chain, used when Routing is "".
	Blocks []Block

	// Parallel sections (Routing SPS-1 / PS-1).
	Prefix []Block // SPS-1: shared slots before the split
	PathA  []Block // first parallel path
	PathB  []Block // second parallel path
	Suffix []Block // shared slots after the merge

	// Path mix controls. Levels are dB (default -6), pans -100..100
	// (default 0), delay ms (default 0).
	Para1Level *float64
	Para2Level *float64
	Para1Pan   *float64
	Para2Pan   *float64
	ParaDelay  *float64
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

// instanceName derives the device node name for a module: the first instance
// keeps the display name, later instances get a " N" suffix ("Amp" then
// "Amp 2"), matching how the device names repeated modules in its patches.
func instanceName(canon string, seen map[string]int) string {
	seen[canon]++
	if seen[canon] == 1 {
		return canon
	}
	return fmt.Sprintf("%s %d", canon, seen[canon])
}

// paraBounds are the parameter ranges measured from the device backups.
const (
	paraPanMin   = -100
	paraPanMax   = 100
	paraDelayMin = 0
	paraDelayMax = 100
	paraLevelMin = -60
	paraLevelMax = 6
)

// levelBounds cap the rig-level level controls so a rig is never written with
// an accidentally huge boost. Ranges are the device's observed limits.
const (
	inputGainMin = -40
	inputGainMax = 12
	rigVolumeMin = -10
	rigVolumeMax = 20
)

// validateLevels rejects out-of-range path mix and rig-level level values.
func validateLevels(spec Spec) error {
	if err := validatePara(spec); err != nil {
		return err
	}
	if spec.InputGain < inputGainMin || spec.InputGain > inputGainMax {
		return fmt.Errorf("input gain = %v dB, must be within [%v, %v]", spec.InputGain, inputGainMin, inputGainMax)
	}
	if spec.OutputVolume < rigVolumeMin || spec.OutputVolume > rigVolumeMax {
		return fmt.Errorf("output level = %v dB, must be within [%v, %v]", spec.OutputVolume, rigVolumeMin, rigVolumeMax)
	}
	return nil
}

// validatePara rejects out-of-range path mix values.
func validatePara(spec Spec) error {
	check := func(name string, v *float64, min, max float64) error {
		if v != nil && (*v < min || *v > max) {
			return fmt.Errorf("%s = %v, must be within [%v, %v]", name, *v, min, max)
		}
		return nil
	}
	if err := check("Para1Level", spec.Para1Level, paraLevelMin, paraLevelMax); err != nil {
		return err
	}
	if err := check("Para2Level", spec.Para2Level, paraLevelMin, paraLevelMax); err != nil {
		return err
	}
	if err := check("Para1Pan", spec.Para1Pan, paraPanMin, paraPanMax); err != nil {
		return err
	}
	if err := check("Para2Pan", spec.Para2Pan, paraPanMin, paraPanMax); err != nil {
		return err
	}
	return check("ParaDelay", spec.ParaDelay, paraDelayMin, paraDelayMax)
}

// chain holds the validated, canonicalized blocks of a rig split into the
// sections the routing topology defines. Each block's Type is the exact device
// display name.
type chain struct {
	routing Routing
	prefix  []Block // SPS-1 serial prefix
	pathA   []Block
	pathB   []Block
	suffix  []Block
	serial  []Block // RoutingSerial
}

// blocks returns the chain blocks in signal order (prefix → path A → path B →
// suffix), canonicalized so every Type matches the device display name.
func (c chain) blocks() []Block {
	var blocks []Block
	blocks = append(blocks, c.prefix...)
	blocks = append(blocks, c.pathA...)
	blocks = append(blocks, c.pathB...)
	blocks = append(blocks, c.suffix...)
	blocks = append(blocks, c.serial...)
	return blocks
}

// slots lays the blocks out into the 11 chain slots, padding every section to
// its slot budget so the split and merge points stay fixed regardless of how
// many blocks each path actually holds. Repeated modules get their device
// instance names ("Amp", "Amp 2", ...).
func (c chain) slots() []string {
	blocks := c.blocks()
	seen := make(map[string]int, len(blocks))
	names := make([]string, len(blocks))
	for i, b := range blocks {
		names[i] = instanceName(b.Type, seen)
	}

	slots := make([]string, 0, 11)
	n := 0
	place := func(section []Block, budget int) {
		for i := 0; i < budget; i++ {
			if i < len(section) {
				slots = append(slots, names[n])
				n++
			} else {
				slots = append(slots, "Empty Slot")
			}
		}
	}

	switch c.routing {
	case RoutingSPS:
		place(c.prefix, spsPrefixSlots)
		place(c.pathA, spsPathSlots)
		place(c.pathB, spsPathSlots)
		place(c.suffix, spsSuffixSlots)
	case RoutingPS:
		place(c.pathA, psPathASlots)
		place(c.pathB, psPathBSlots)
		place(c.suffix, 11-psPathASlots-psPathBSlots)
	default:
		place(c.serial, 11)
	}
	return slots
}

// buildChain validates the spec and returns the canonical chain layout. It
// enforces the Gigboard's slot budgets, measured from the device backups.
func buildChain(cat *catalog.Catalog, spec Spec) (chain, error) {
	var c chain
	if err := validateLevels(spec); err != nil {
		return c, err
	}

	if !spec.Routing.Valid() {
		return c, fmt.Errorf("unknown routing %q (want S, SPS-1 or PS-1)", spec.Routing)
	}

	c.routing = spec.Routing
	if c.routing == "" {
		c.routing = RoutingSerial
	}

	canonicalize := func(blocks []Block) ([]Block, error) {
		out := make([]Block, 0, len(blocks))
		for _, b := range blocks {
			canon, ok := normalizeBlockName(cat, b.Type)
			if !ok {
				return nil, fmt.Errorf("unknown module type %q", b.Type)
			}
			b.Type = canon
			out = append(out, b)
		}
		return out, nil
	}

	var err error
	switch c.routing {
	case RoutingSerial:
		c.serial, err = canonicalize(spec.Blocks)
	case RoutingSPS:
		if len(spec.Prefix) > spsPrefixSlots {
			return c, fmt.Errorf("SPS-1 prefix has %d blocks, max %d (slots 1-%d)", len(spec.Prefix), spsPrefixSlots, spsPrefixSlots)
		}
		if len(spec.PathA) > spsPathSlots {
			return c, fmt.Errorf("SPS-1 path A has %d blocks, max %d", len(spec.PathA), spsPathSlots)
		}
		if len(spec.PathB) > spsPathSlots {
			return c, fmt.Errorf("SPS-1 path B has %d blocks, max %d", len(spec.PathB), spsPathSlots)
		}
		if len(spec.Suffix) > spsSuffixSlots {
			return c, fmt.Errorf("SPS-1 suffix has %d blocks, max %d (slots 10-11)", len(spec.Suffix), spsSuffixSlots)
		}
		c.prefix, err = canonicalize(spec.Prefix)
		if err == nil {
			c.pathA, err = canonicalize(spec.PathA)
		}
		if err == nil {
			c.pathB, err = canonicalize(spec.PathB)
		}
		if err == nil {
			c.suffix, err = canonicalize(spec.Suffix)
		}
	case RoutingPS:
		if len(spec.Prefix) > 0 {
			return c, fmt.Errorf("PS-1 has no serial prefix; use PathA/PathB/Suffix (the signal splits at the input)")
		}
		if len(spec.PathA) > psPathASlots {
			return c, fmt.Errorf("PS-1 path A has %d blocks, max %d (slots 1-%d)", len(spec.PathA), psPathASlots, psPathASlots)
		}
		if len(spec.PathB) > psPathBSlots {
			return c, fmt.Errorf("PS-1 path B has %d blocks, max %d", len(spec.PathB), psPathBSlots)
		}
		c.pathA, err = canonicalize(spec.PathA)
		if err == nil {
			c.pathB, err = canonicalize(spec.PathB)
		}
		if err == nil {
			c.suffix, err = canonicalize(spec.Suffix)
		}
	}
	if err != nil {
		return c, err
	}

	if len(c.blocks()) > 11 {
		return c, fmt.Errorf("too many blocks: the Gigboard has 11 chain slots, got %d", len(c.blocks()))
	}

	ampSeen, cabSeen := false, false
	for _, b := range c.blocks() {
		switch b.Type {
		case "Amp":
			ampSeen = true
		case "Cab":
			cabSeen = true
		}
	}
	if !ampSeen {
		return c, fmt.Errorf("rig must contain an \"Amp\" block")
	}
	if !cabSeen {
		return c, fmt.Errorf("rig must contain a \"Cab\" block")
	}
	return c, nil
}
