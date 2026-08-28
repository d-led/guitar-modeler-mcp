package rig

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// Scene blobs (the "state" field with type 24) are the device's serialized view
// of the 11 chain slots: eleven 36-byte records, each a 4-byte header followed
// by a 32-byte, null-terminated module name. The names must match the
// Chain.ModuleTypeN values, otherwise the rig's footswitch scenes disagree with
// its signal chain.
const (
	sceneSlotSize = 36
	sceneSlots    = 11
)

// sceneBlob encodes the chain slot names into the base64 scene-state format.
func sceneBlob(moduleNames []string) string {
	buf := make([]byte, sceneSlots*sceneSlotSize)
	for i := 0; i < sceneSlots; i++ {
		name := "Empty Slot"
		if i < len(moduleNames) {
			name = moduleNames[i]
		}
		copy(buf[i*sceneSlotSize+4:i*sceneSlotSize+sceneSlotSize], name)
	}
	return base64.StdEncoding.EncodeToString(buf)
}

// footSwitchFor rewires the template FootSwitch to the given chain: the scene
// snapshots encode the new slots and the stomp switches are reset to
// "Unassigned" so they never reference modules that are not in the chain.
func footSwitchFor(template []byte, moduleNames []string) (map[string]any, error) {
	fs, err := decodeObject(template)
	if err != nil {
		return nil, fmt.Errorf("parse FootSwitch template: %w", err)
	}
	children, err := objectField(fs, "data", "FootSwitch", "children")
	if err != nil {
		return nil, err
	}

	blob := sceneBlob(moduleNames)
	for _, key := range []string{"Scene5", "Scene6", "Scene7", "Scene8", "State2Scene5", "State2Scene6", "State2Scene7", "State2Scene8"} {
		children[key] = map[string]any{"state": blob, "type": 24}
	}
	for _, n := range []string{"5", "6", "7", "8"} {
		children["Module"+n] = map[string]any{"string": "Unassigned", "type": 8}
		children["Operation"+n] = map[string]any{"string": "", "type": 8}
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
