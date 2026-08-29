package rig

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// Scene blobs (the "state" field with type 24) are the device's serialized view
// of the 11 chain slots: eleven 36-byte records, each a 4-byte little-endian
// header followed by a 32-byte, null-terminated module name. The header encodes
// the block's state in the scene: 0 = no change, 1 = turn on, 2 = turn off. The
// names must match the Chain.ModuleTypeN values, otherwise the rig's footswitch
// scenes disagree with its signal chain.
const (
	sceneSlotSize = 36
	sceneSlots    = 11

	sceneNoChange byte = 0
	sceneOn       byte = 1
	sceneOff      byte = 2
)

// sceneBlob encodes the chain slot names into the base64 scene-state format.
// headers[i] is the per-slot state header; a nil headers slice means "no
// change" for every slot.
func sceneBlob(moduleNames []string, headers []byte) string {
	buf := make([]byte, sceneSlots*sceneSlotSize)
	for i := 0; i < sceneSlots; i++ {
		name := "Empty Slot"
		if i < len(moduleNames) {
			name = moduleNames[i]
		}
		if i < len(headers) {
			buf[i*sceneSlotSize] = headers[i]
		}
		copy(buf[i*sceneSlotSize+4:i*sceneSlotSize+sceneSlotSize], name)
	}
	return base64.StdEncoding.EncodeToString(buf)
}

// sceneHeaders maps a scene snapshot onto the chain's 11 slots: 1 for blocks to
// turn on, 2 for blocks to turn off, 0 (no change) for the rest.
func sceneHeaders(moduleNames []string, snap *SceneSnapshot) []byte {
	headers := make([]byte, sceneSlots)
	if snap == nil {
		return headers
	}
	for i, name := range moduleNames {
		for _, on := range snap.On {
			if on == name {
				headers[i] = sceneOn
			}
		}
		for _, off := range snap.Off {
			if off == name {
				headers[i] = sceneOff
			}
		}
	}
	return headers
}

// footSwitchFor rewires the template FootSwitch to the given chain: the scene
// snapshots encode the new slots, the stomp switches (FS5..FS8) are reset to
// "Unassigned", and the requested switches are assigned to the first slots.
func footSwitchFor(template []byte, moduleNames []string, switches []Footswitch) (map[string]any, error) {
	fs, err := decodeObject(template)
	if err != nil {
		return nil, fmt.Errorf("parse FootSwitch template: %w", err)
	}
	children, err := objectField(fs, "data", "FootSwitch", "children")
	if err != nil {
		return nil, err
	}

	blob := sceneBlob(moduleNames, nil)
	for _, key := range []string{"Scene5", "Scene6", "Scene7", "Scene8", "State2Scene5", "State2Scene6", "State2Scene7", "State2Scene8"} {
		children[key] = map[string]any{"state": blob, "type": 24}
	}
	for _, n := range []string{"5", "6", "7", "8"} {
		children["Module"+n] = map[string]any{"string": "Unassigned", "type": 8}
		children["Operation"+n] = map[string]any{"string": "", "type": 8}
	}
	for i, sw := range switches {
		n := fmt.Sprintf("%d", 5+i)
		children["Module"+n] = map[string]any{"string": sw.Module, "type": 8}
		children["Operation"+n] = map[string]any{"string": sw.Operation, "type": 8}
		// The switch mode ("Toggle" or "Scene") is already resolved by the
		// builder; Scene switches recall a block on/off snapshot instead of
		// toggling a single module.
		children["ModeNew"+n] = map[string]any{"string": sw.Mode, "type": 4}
		if sw.Label != "" {
			children["UserFootSwitchText"+n] = map[string]any{"string": sw.Label, "type": 8}
		}
		if sw.Mode == "Scene" && sw.Scene != nil {
			children["Scene"+n] = map[string]any{"state": sceneBlob(moduleNames, sceneHeaders(moduleNames, sw.Scene)), "type": 24}
		}
	}
	return fs, nil
}

// pedalFor resets the template pedal section so it no longer references
// modules from the template's chain.
func pedalFor(template []byte) (map[string]any, error) {
	pedal, err := decodeObject(template)
	if err != nil {
		return nil, fmt.Errorf("parse pedal template: %w", err)
	}
	data, err := objectField(pedal, "data")
	if err != nil {
		return nil, err
	}
	for _, section := range data {
		children, err := objectField(section.(map[string]any), "children")
		if err != nil {
			return nil, err
		}
		for i := 1; i <= 4; i++ {
			children[fmt.Sprintf("Module%d", i)] = map[string]any{"string": "Unassigned", "type": 8}
			children[fmt.Sprintf("Param%d", i)] = map[string]any{"string": "", "type": 8}
		}
	}
	return pedal, nil
}

func decodeObject(raw []byte) (map[string]any, error) {
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, err
	}
	return m, nil
}

// objectField walks a chain of keys through nested objects.
func objectField(root map[string]any, path ...string) (map[string]any, error) {
	cur := root
	for i, key := range path {
		v, ok := cur[key]
		if !ok {
			return nil, fmt.Errorf("missing field %q", key)
		}
		next, ok := v.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("field %q is not an object", key)
		}
		if i == len(path)-1 {
			return next, nil
		}
		cur = next
	}
	return cur, nil
}
