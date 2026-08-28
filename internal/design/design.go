// Package design turns a high-level "dial in a tone" request into a concrete
// rig.Spec: it translates real-world hardware descriptions into device models
// and orders the effects into a musically sensible signal chain.
package design

import (
	"fmt"
	"strings"

	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/catalog"
	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/rig"
)

// FXBlock is a single effect the caller wants in the chain.
type FXBlock struct {
	Type    string         `json:"type"`
	Enabled bool           `json:"enabled"`
	Params  map[string]any `json:"params,omitempty"`
}

// Request is the input to the designer.
type Request struct {
	Name      string    `json:"name"`
	Song      string    `json:"song,omitempty"`
	Amp       string    `json:"amp"`           // device model or real-hardware description
	Cab       string    `json:"cab,omitempty"` // device model or description
	Mic       string    `json:"mic,omitempty"` // device model or description
	Tempo     float64   `json:"tempo,omitempty"`
	InputGain float64   `json:"input_gain,omitempty"`
	FX        []FXBlock `json:"fx,omitempty"`
}

// Result carries the resolved spec plus human-readable decisions.
type Result struct {
	Spec  rig.Spec
	Notes []string
}

// Designer resolves requests using the device catalog.
type Designer struct {
	cat *catalog.Catalog
}

// NewDesigner creates a Designer.
func NewDesigner(cat *catalog.Catalog) *Designer { return &Designer{cat: cat} }

// Design resolves a request into a buildable rig spec.
func (d *Designer) Design(req Request) (*Result, error) {
	if strings.TrimSpace(req.Name) == "" {
		req.Name = "New Rig"
	}

	ampModel, note, err := d.resolveAmp(req.Amp)
	if err != nil {
		return nil, err
	}
	cabModel := d.resolveCab(req.Cab, ampModel)
	micModel := d.resolveMic(req.Mic)

	notes := []string{note, fmt.Sprintf("cab %q", cabModel), fmt.Sprintf("mic %q", micModel)}

	var pre, post, last []rig.Block
	for _, fx := range req.FX {
		def, ok := d.cat.FXByName(fx.Type)
		if !ok {
			return nil, fmt.Errorf("unknown effect type %q", fx.Type)
		}
		block := rig.Block{Type: def.Name, Enabled: fx.Enabled, Params: fx.Params}
		if block.Params == nil {
			block.Params = map[string]any{}
		}
		if strings.EqualFold(def.Name, "Volume") {
			last = append(last, block) // Volume pedal always goes at the end of the chain
			continue
		}
		switch def.Category {
		case "drive", "dynamics/eq", "filter/wah", "pitch":
			pre = append(pre, block)
		default: // modulation, delay/reverb, utility
			post = append(post, block)
		}
	}

	blocks := make([]rig.Block, 0, len(pre)+len(post)+len(last)+2)
	blocks = append(blocks, pre...)
	blocks = append(blocks, rig.Block{
		Type:    "Amp",
		Enabled: true,
		Params:  map[string]any{"Type": ampModel, "On": true},
	})
	blocks = append(blocks, rig.Block{
		Type:    "Cab",
		Enabled: true,
		Params:  map[string]any{"CabType": cabModel, "MicType": micModel, "On": true},
	})
	blocks = append(blocks, post...)
	blocks = append(blocks, last...)

	tempo := req.Tempo
	if tempo <= 0 {
		tempo = 100
	}

	return &Result{
		Spec: rig.Spec{
			Name:      req.Name,
			Tempo:     tempo,
			InputGain: req.InputGain,
			Blocks:    blocks,
		},
		Notes: notes,
	}, nil
}

func (d *Designer) resolveAmp(query string) (string, string, error) {
	if strings.TrimSpace(query) == "" {
		return "", "", fmt.Errorf("an amp is required")
	}
	if a, ok := d.cat.Amp(query); ok {
		return a.Model, fmt.Sprintf("amp %q (exact match)", a.Model), nil
	}
	matches := d.cat.TranslateAmp(query)
	if len(matches) == 0 {
		return "", "", fmt.Errorf("no HeadRush amp matches %q; list amps with catalog_list_amps", query)
	}
	best := matches[0]
	return best.Amp.Model, fmt.Sprintf("amp %q translated from %q (%s)", best.Amp.Model, query, best.Reason), nil
}

func (d *Designer) resolveCab(query string, ampModel string) string {
	if strings.TrimSpace(query) != "" {
		if c, ok := d.cat.Cab(query); ok {
			return c.Model
		}
		if cs := d.cat.TranslateCab(query); len(cs) > 0 {
			return cs[0].Model
		}
	}
	// Fall back to a cabinet that suits the amp family.
	if amp, ok := d.cat.Amp(ampModel); ok {
		if amp.Bass {
			return "8x10 Blue Line"
		}
		switch amp.Brand {
		case "Vox":
			return "2x12 AC Blue"
		case "Fender":
			return "1x12 Black Panel Lux"
		case "Marshall", "Mesa Boogie", "Soldano", "Bogner", "Peavey", "Engl":
			return "4x12 Green 25W"
		}
	}
	return "1x12 Black Panel Lux"
}

func (d *Designer) resolveMic(query string) string {
	if strings.TrimSpace(query) != "" {
		if m, ok := d.cat.Mic(query); ok {
			return m.Model
		}
		if ms := d.cat.TranslateMic(query); len(ms) > 0 {
			return ms[0].Model
		}
	}
	return "Dyn 57"
}
