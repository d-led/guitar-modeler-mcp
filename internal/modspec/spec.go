// Package modspec holds the parameter specifications (ranges, toggles and
// enumerated options) for every effect module and the amp. The data is
// extracted from headrush-desktop's renderer/config/modules/*.ts so the MCP can
// describe, and the builder can enforce, exactly the values the device accepts.
package modspec

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
)

//go:embed data/params.json
var rawJSON []byte

// Param describes one knob/switch/dropdown of a module.
type Param struct {
	Kind   string   `json:"kind"` // "range", "toggle" or "set"
	Label  string   `json:"label"`
	Min    *float64 `json:"min,omitempty"`
	Max    *float64 `json:"max,omitempty"`
	Step   *float64 `json:"step,omitempty"`
	Unit   string   `json:"unit,omitempty"`
	Off    string   `json:"off,omitempty"`
	On     string   `json:"on,omitempty"`
	Values []string `json:"values,omitempty"`
}

// Module maps parameter names to their specifications.
type Module map[string]Param

type rawParam struct {
	Type   string   `json:"type"`
	Label  string   `json:"label"`
	Min    *float64 `json:"min"`
	Max    *float64 `json:"max"`
	Step   *float64 `json:"step"`
	Unit   string   `json:"unit"`
	Off    string   `json:"off"`
	On     string   `json:"on"`
	Values []string `json:"values"`
}

var (
	once    sync.Once
	modules map[string]Module
	loadErr error
)

func load() {
	once.Do(func() {
		var raw map[string]map[string]rawParam
		if err := json.Unmarshal(rawJSON, &raw); err != nil {
			loadErr = err
			return
		}
		modules = make(map[string]Module, len(raw))
		for name, params := range raw {
			m := make(Module, len(params))
			for pname, rp := range params {
				m[pname] = Param{
					Kind: rp.Type, Label: rp.Label, Min: rp.Min, Max: rp.Max,
					Step: rp.Step, Unit: rp.Unit, Off: rp.Off, On: rp.On, Values: rp.Values,
				}
			}
			modules[name] = m
		}
	})
}

// Get returns the parameter spec for a module, or false if unknown.
func Get(moduleName string) (Module, bool) {
	load()
	m, ok := modules[moduleName]
	return m, ok
}

// Modules lists all module names that have a parameter spec.
func Modules() []string {
	load()
	names := make([]string, 0, len(modules))
	for n := range modules {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Validate checks a parameter value against this spec. A nil error means the
// value is acceptable to the device.
func (p Param) Validate(value any) error {
	switch p.Kind {
	case "range":
		v, ok := asFloat(value)
		if !ok {
			return fmt.Errorf("%s expects a number, got %T", p.Label, value)
		}
		if p.Min != nil && v < *p.Min {
			return fmt.Errorf("%s = %v is below the minimum %v%s", p.Label, v, *p.Min, p.Unit)
		}
		if p.Max != nil && v > *p.Max {
			return fmt.Errorf("%s = %v is above the maximum %v%s", p.Label, v, *p.Max, p.Unit)
		}
	case "set":
		s, ok := value.(string)
		if !ok {
			return fmt.Errorf("%s expects a string option, got %T", p.Label, value)
		}
		if !contains(p.Values, s) {
			return fmt.Errorf("%s: %q is not a valid option (allowed: %v)", p.Label, s, p.Values)
		}
	case "toggle":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("%s expects a boolean, got %T", p.Label, value)
		}
	}
	return nil
}

func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case json.Number:
		f, err := n.Float64()
		return f, err == nil
	}
	return 0, false
}

func contains(values []string, s string) bool {
	for _, v := range values {
		if v == s {
			return true
		}
	}
	return false
}
