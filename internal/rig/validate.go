package rig

import (
	"fmt"
	"strings"

	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/assets"
	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/modspec"
)

// structuralParams are rig-level fields that the builder sets itself; they are
// never validated against the per-module parameter spec.
var structuralParams = map[string]bool{
	"On":          true,
	"Colour":      true,
	"PresetName":  true,
	"PresetName2": true,
}

// colourPalette is the set of module colours the device accepts.
var colourPalette = map[string]bool{
	"Blue": true, "Yellow": true, "Green": true, "Purple": true, "Red": true,
	"Dark Green": true, "Orange": true, "Light Blue": true, "Pink": true,
}

// cabLevelBounds caps the Cab block's output trims (dB) so a cabinet never
// accidentally boosts a rig by an absurd amount. The Cab module has no editor
// spec, so these bounds are enforced here rather than via modspec.
var cabLevelBounds = map[string]struct{ min, max float64 }{
	"OutGain":     {-12, 12},
	"OutGain2":    {-12, 12},
	"AmpCompGain": {-12, 12},
}

// blockParamKeys returns the parameter names the device itself exposes for a
// module, taken from its embedded factory block.
func blockParamKeys(moduleName string) (map[string]bool, error) {
	raw, err := assets.DefaultBlock(strings.ToUpper(moduleName))
	if err != nil {
		return nil, fmt.Errorf("no block for module %q: %w", moduleName, err)
	}
	node, err := blockNodeFromJSON(raw)
	if err != nil {
		return nil, err
	}
	keys := make(map[string]bool, len(node.ChildOrder))
	for _, k := range node.ChildOrder {
		keys[k] = true
	}
	return keys, nil
}

// validateBlockParams rejects parameter values the device would not accept:
// unknown parameter names, unknown model/cabinet/mic selections, out-of-range
// numbers and invalid enum options. Amp/cab/mic model names are checked against
// the backup-derived catalog, which is authoritative where the editor's own
// list has typos and omissions.
func (b *Builder) validateBlockParams(canon string, params map[string]any) error {
	if len(params) == 0 {
		return nil
	}

	blockKeys, err := blockParamKeys(canon)
	if err != nil {
		blockKeys = map[string]bool{}
	}
	spec, hasSpec := modspec.Get(canon)
	known := blockKeys
	if hasSpec {
		for k := range spec {
			known[k] = true
		}
	}

	for key, value := range params {
		if structuralParams[key] {
			if key == "Colour" {
				if s, ok := value.(string); ok && !colourPalette[s] {
					return fmt.Errorf("module %q: invalid colour %q", canon, s)
				}
			}
			continue
		}

		switch {
		case canon == "Amp" && (key == "Type" || key == "Type2"):
			if s, ok := value.(string); ok {
				if _, found := b.cat.Amp(s); !found {
					return fmt.Errorf("unknown amp model %q", s)
				}
			}
			continue
		case canon == "Cab" && (key == "CabType" || key == "CabType2"):
			if s, ok := value.(string); ok {
				if _, found := b.cat.Cab(s); !found {
					return fmt.Errorf("unknown cabinet model %q", s)
				}
			}
			continue
		case canon == "Cab" && (key == "MicType" || key == "MicType2"):
			if s, ok := value.(string); ok {
				if _, found := b.cat.Mic(s); !found {
					return fmt.Errorf("unknown microphone model %q", s)
				}
			}
			continue
		}

		if !known[key] {
			return fmt.Errorf("module %q has no parameter %q", canon, key)
		}
		if canon == "Cab" {
			if bounds, ok := cabLevelBounds[key]; ok {
				if n, ok := asFloat(value); ok && (n < bounds.min || n > bounds.max) {
					return fmt.Errorf("module %q: %s = %v dB, must be within [%v, %v]", canon, key, n, bounds.min, bounds.max)
				}
			}
		}
		if hasSpec {
			if p, ok := spec[key]; ok {
				if err := p.Validate(value); err != nil {
					return fmt.Errorf("module %q: %w", canon, err)
				}
			}
		}
	}
	return nil
}

// asFloat extracts a numeric value from a parameter override (float64, int or
// int64), reporting whether the value was numeric.
func asFloat(value any) (float64, bool) {
	switch v := value.(type) {
	case float64:
		return v, true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	}
	return 0, false
}
