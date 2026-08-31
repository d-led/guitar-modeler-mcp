// Package params unifies the catalog (models) and modspec (parameter specs)
// into a single description of a module's editable parameters, so callers (MCP
// tools and CLI) can tell an agent exactly what values the device accepts.
package params

import (
	"fmt"
	"sort"
	"strings"

	"github.com/d-led/guitar-modeler-mcp/internal/catalog"
	"github.com/d-led/guitar-modeler-mcp/internal/modspec"
)

// ModuleNames lists every module that has a parameter description.
func ModuleNames() []string {
	names := modspec.Modules()
	names = append(names, "Cab", "IR", "IR (1024)")
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
	case "IR", "IR (1024)":
		return irParams(), nil
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

// DescribeMany returns the parameter specs for several modules at once:
// canonical module name -> parameter name -> spec. Modules that fail to resolve
// or have no spec are skipped.
func DescribeMany(cat *catalog.Catalog, moduleNames []string) map[string]map[string]modspec.Param {
	out := make(map[string]map[string]modspec.Param, len(moduleNames))
	for _, name := range moduleNames {
		canon, err := Resolve(cat, name)
		if err != nil {
			continue
		}
		spec, err := Describe(cat, canon)
		if err != nil {
			continue
		}
		out[canon] = spec
	}
	return out
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
	out["Type"] = modspec.Param{Kind: "set", Label: "Model", Description: "Emulated amplifier model.", Values: models}
	out["Type2"] = modspec.Param{Kind: "set", Label: "Model", Description: "Emulated amplifier model (doubled state).", Values: models}
	return out
}

// fp is a shorthand for a float64 pointer, used when building Param specs.
func fp(v float64) *float64 { return &v }

// irParams documents the impulse-response loader: a file selector plus the
// gain/filter/mix trims. The IR file reference is the device's
// "[directory](<folder>)[name](<file>)" string; "[IR ROOT]" is the root.
func irParams() map[string]modspec.Param {
	return map[string]modspec.Param{
		"IR":     {Kind: "string", Label: "IR", Description: `Impulse-response selector in the form "[directory](<folder>)[name](<file>)"; use "[IR ROOT]" for the root folder.`},
		"IR2":    {Kind: "string", Label: "IR (Doubling)", Description: `Second impulse response for the Doubling (stereo) state.`},
		"Gain":   {Kind: "range", Label: "Gain", Description: "IR output gain.", Min: fp(-24), Max: fp(24), Unit: " dB"},
		"Gain2":  {Kind: "range", Label: "Gain (Doubling)", Description: "Second IR output gain.", Min: fp(-24), Max: fp(24), Unit: " dB"},
		"HiCut":  {Kind: "range", Label: "High Cut", Description: "High-cut filter.", Min: fp(500), Max: fp(20000), Unit: " Hz"},
		"HiCut2": {Kind: "range", Label: "High Cut (Doubling)", Description: "Second IR high-cut filter.", Min: fp(500), Max: fp(20000), Unit: " Hz"},
		"LoCut":  {Kind: "range", Label: "Low Cut", Description: "Low-cut filter.", Min: fp(20), Max: fp(1000), Unit: " Hz"},
		"LoCut2": {Kind: "range", Label: "Low Cut (Doubling)", Description: "Second IR low-cut filter.", Min: fp(20), Max: fp(1000), Unit: " Hz"},
		"Mix":    {Kind: "range", Label: "Mix", Description: "Wet/dry mix.", Min: fp(0), Max: fp(100), Unit: " %"},
		"Mix2":   {Kind: "range", Label: "Mix (Doubling)", Description: "Second IR wet/dry mix.", Min: fp(0), Max: fp(100), Unit: " %"},
	}
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
		"CabType":     {Kind: "set", Label: "Cabinet", Description: "Cabinet model.", Values: cabs},
		"CabType2":    {Kind: "set", Label: "Cabinet", Description: "Cabinet model (doubled state).", Values: cabs},
		"MicType":     {Kind: "set", Label: "Microphone", Description: "Microphone model.", Values: mics},
		"MicType2":    {Kind: "set", Label: "Microphone", Description: "Microphone model (doubled state).", Values: mics},
		"OnAxis":      {Kind: "toggle", Label: "On Axis", Description: "On-axis microphone position.", Off: "off", On: "on"},
		"OnAxis2":     {Kind: "toggle", Label: "On Axis", Description: "On-axis microphone position (doubled state).", Off: "off", On: "on"},
		"Breakup":     {Kind: "range", Label: "Breakup", Description: "Speaker breakup amount.", Min: fp(0), Max: fp(100), Unit: " %"},
		"Breakup2":    {Kind: "range", Label: "Breakup", Description: "Speaker breakup amount (doubled state).", Min: fp(0), Max: fp(100), Unit: " %"},
		"OutGain":     {Kind: "range", Label: "Out Gain", Description: "Output gain of the cabinet.", Min: fp(-12), Max: fp(12), Unit: " dB"},
		"OutGain2":    {Kind: "range", Label: "Out Gain", Description: "Output gain of the cabinet (doubled state).", Min: fp(-12), Max: fp(12), Unit: " dB"},
		"AmpCompGain": {Kind: "range", Label: "Amp Comp Gain", Description: "Gain compensation applied to the amp.", Min: fp(-12), Max: fp(12), Unit: " dB"},
	}
}
