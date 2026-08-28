// Package params unifies the catalog (models) and modspec (parameter specs)
// into a single description of a module's editable parameters, so callers (MCP
// tools and CLI) can tell an agent exactly what values the device accepts.
package params

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/catalog"
	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/modspec"
)

// ModuleNames lists every module that has a parameter description.
func ModuleNames() []string {
	names := modspec.Modules()
	names = append(names, "Cab")
	sort.Strings(names)
	return names
}

// Resolve maps a possibly case-insensitive module name to its canonical name.
func Resolve(cat *catalog.Catalog, name string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "amp":
		return "Amp", nil
	case "cab":
		return "Cab", nil
	}
	if f, ok := cat.FXByName(name); ok {
		return f.Name, nil
	}
	return "", fmt.Errorf("unknown module %q (available: %v)", name, ModuleNames())
}

// Describe returns the parameter specs for a module: parameter name -> spec.
// Amp model and cab/mic selections are taken from the backup-derived catalog,
// which is authoritative where the editor's own lists have typos or gaps.
func Describe(cat *catalog.Catalog, moduleName string) (map[string]modspec.Param, error) {
	canon, err := Resolve(cat, moduleName)
	if err != nil {
		return nil, err
	}
	switch canon {
	case "Cab":
		return cabParams(cat), nil
	case "Amp":
		return ampParams(cat), nil
	}

	spec, ok := modspec.Get(canon)
	if !ok {
		return nil, fmt.Errorf("no parameter spec for module %q (available: %v)", canon, ModuleNames())
	}
	out := make(map[string]modspec.Param, len(spec))
	for k, v := range spec {
		out[k] = v
	}
	return out, nil
}

func ampParams(cat *catalog.Catalog) map[string]modspec.Param {
	spec, ok := modspec.Get("Amp")
	if !ok {
		return nil
	}
	out := make(map[string]modspec.Param, len(spec))
	for k, v := range spec {
		out[k] = v
	}
	models := make([]string, 0, len(cat.Amps()))
	for _, a := range cat.Amps() {
		models = append(models, a.Model)
	}
	out["Type"] = modspec.Param{Kind: "set", Label: "Model", Values: models}
	out["Type2"] = modspec.Param{Kind: "set", Label: "Model", Values: models}
	return out
}

func cabParams(cat *catalog.Catalog) map[string]modspec.Param {
	cabs := make([]string, 0, len(cat.Cabs()))
	for _, c := range cat.Cabs() {
		cabs = append(cabs, c.Model)
	}
	mics := make([]string, 0, len(cat.Mics()))
	for _, m := range cat.Mics() {
		mics = append(mics, m.Model)
	}
	return map[string]modspec.Param{
		"CabType":     {Kind: "set", Label: "Cabinet", Values: cabs},
		"CabType2":    {Kind: "set", Label: "Cabinet", Values: cabs},
		"MicType":     {Kind: "set", Label: "Microphone", Values: mics},
		"MicType2":    {Kind: "set", Label: "Microphone", Values: mics},
		"OnAxis":      {Kind: "toggle", Label: "On Axis", Off: "off", On: "on"},
		"OnAxis2":     {Kind: "toggle", Label: "On Axis", Off: "off", On: "on"},
		"Breakup":     {Kind: "range", Label: "Breakup"},
		"Breakup2":    {Kind: "range", Label: "Breakup"},
		"OutGain":     {Kind: "range", Label: "Out Gain"},
		"OutGain2":    {Kind: "range", Label: "Out Gain"},
		"AmpCompGain": {Kind: "range", Label: "Amp Comp Gain"},
	}
}
