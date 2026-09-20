package rig

import (
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
)

// RawContent returns the rig's full inner document as a generic tree so callers
// can inspect every section — FootSwitch, Pedal1/2 and the Patch nodes with
// their childorder/children pairs — without the typed Summary collapsing
// anything.
func (f *RigFile) RawContent() (map[string]any, error) {
	var raw map[string]any
	if err := json.Unmarshal([]byte(f.Content), &raw); err != nil {
		return nil, fmt.Errorf("decode raw rig content: %w", err)
	}
	return raw, nil
}

// DiffEntry is one field-level difference between two rigs: the JSON path of
// the changed field and its value in each rig (null when the field is absent on
// that side).
type DiffEntry struct {
	Path string `json:"path"`
	A    any    `json:"a"`
	B    any    `json:"b"`
}

// DiffRigs returns the field-level differences between two rigs, keyed by each
// changed field's JSON path. Equal fields are omitted, so the result is empty
// for identical rigs. Comparing a generated rig against a device save shows
// exactly which fields the device rewrites.
func DiffRigs(a, b *RigFile) ([]DiffEntry, error) {
	ar, err := a.RawContent()
	if err != nil {
		return nil, err
	}
	br, err := b.RawContent()
	if err != nil {
		return nil, err
	}
	out := []DiffEntry{}
	diffRigRec("", ar, br, &out)
	return out, nil
}

func diffRigRec(path string, a, b any, out *[]DiffEntry) {
	am, aok := a.(map[string]any)
	bm, bok := b.(map[string]any)
	if !aok || !bok {
		if !reflect.DeepEqual(a, b) {
			*out = append(*out, DiffEntry{Path: path, A: a, B: b})
		}
		return
	}
	for _, k := range unionSortedKeys(am, bm) {
		av, aHas := am[k]
		bv, bHas := bm[k]
		child := joinPath(path, k)
		if aHas && bHas {
			diffRigRec(child, av, bv, out)
			continue
		}
		if aHas {
			*out = append(*out, DiffEntry{Path: child, A: av})
		} else {
			*out = append(*out, DiffEntry{Path: child, B: bv})
		}
	}
}

func joinPath(path, k string) string {
	if path == "" {
		return k
	}
	return path + "." + k
}

func unionSortedKeys(a, b map[string]any) []string {
	seen := make(map[string]bool, len(a)+len(b))
	for k := range a {
		seen[k] = true
	}
	for k := range b {
		seen[k] = true
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
