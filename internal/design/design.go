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
	// Device selects the target hardware. "gigboard" (default) is currently
	// the only supported backend.
	Device    string  `json:"device,omitempty"`
	Name      string  `json:"name"`
	Song      string  `json:"song,omitempty"`
	Amp       string  `json:"amp"`           // device model or real-hardware description
	Cab       string  `json:"cab,omitempty"` // device model or description
	Mic       string  `json:"mic,omitempty"` // device model or description
	Tempo     float64 `json:"tempo,omitempty"`
	InputGain float64 `json:"input_gain,omitempty"`
	// OutputLevel is the rig's overall output level in dB (RigVolume). When nil
	// the designer defaults to +6 dB, compensating the amp master's −6 dB so a
	// fresh rig lands at unity.
	OutputLevel *float64  `json:"output_level,omitempty"`
	FX          []FXBlock `json:"fx,omitempty"`

	// Routing selects the signal-chain topology: "" or "S" (serial, default),
	// "SPS-1" (serial → parallel → serial) or "PS-1" (parallel from the input).
	Routing rig.Routing `json:"routing,omitempty"`

	// Amp2, Cab2 and Mic2 add a second, parallel amp path (the dual-amp
	// configuration). When Amp2 is set the designer splits the chain into two
	// amp paths; when it is empty the single amp is shared by both paths.
	Amp2 string `json:"amp2,omitempty"`
	Cab2 string `json:"cab2,omitempty"`
	Mic2 string `json:"mic2,omitempty"`

	// PathAFX and PathBFX place effects on the first and second parallel paths
	// respectively (used for a shared-amp split, e.g. wet/dry/wet).
	PathAFX []FXBlock `json:"path_a_fx,omitempty"`
	PathBFX []FXBlock `json:"path_b_fx,omitempty"`

	// Parallel-path mixer controls. Levels are dB (default -6), pans -100..100
	// (default 0; -100/+100 hard-pans the two paths), delay ms (default 0).
	Para1Level *float64 `json:"para1_level,omitempty"`
	Para2Level *float64 `json:"para2_level,omitempty"`
	Para1Pan   *float64 `json:"para1_pan,omitempty"`
	Para2Pan   *float64 `json:"para2_pan,omitempty"`
	ParaDelay  *float64 `json:"para_delay,omitempty"`

	// Footswitches assigns the four stomp switches (FS5..FS8) to control
	// modules, e.g. [{"module":"Wham"}] toggles the whammy on/off. Module must
	// be a module in the chain.
	Footswitches []rig.Footswitch `json:"footswitches,omitempty"`
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

// defaultOutputLevel compensates the amp master's −6 dB (50% master) so a fresh
// serial rig lands at ~0 dB net, matching the device's own presets (whose
// RigVolume sits around +5 dB).
const defaultOutputLevel = 6.0

// Design resolves a request into a buildable rig spec.
func (d *Designer) Design(req Request) (*Result, error) {
	if req.Device != "" && !strings.EqualFold(req.Device, "gigboard") {
		return nil, fmt.Errorf("device %q is not supported yet (supported: gigboard)", req.Device)
	}
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

	pre, post, last, err := d.classifyFX(req.FX)
	if err != nil {
		return nil, err
	}
	tempo := req.Tempo
	if tempo <= 0 {
		tempo = 100
	}

	outputVolume := defaultOutputLevel
	if req.OutputLevel != nil {
		outputVolume = *req.OutputLevel
	}

	spec := rig.Spec{
		Name:         req.Name,
		Tempo:        tempo,
		InputGain:    req.InputGain,
		OutputVolume: outputVolume,
		Routing:      req.Routing,
		Para1Level:   req.Para1Level,
		Para2Level:   req.Para2Level,
		Para1Pan:     req.Para1Pan,
		Para2Pan:     req.Para2Pan,
		ParaDelay:    req.ParaDelay,
		Footswitches: req.Footswitches,
	}

	switch {
	case req.Routing == rig.RoutingSPS && req.Amp2 == "":
		// Shared amp: the amp+cab feed two parallel effect paths.
		spec.Prefix = append(pre, d.ampBlock(ampModel), d.cabBlock(cabModel, micModel))
		spec.PathA = d.fxBlocks(req.PathAFX)
		spec.PathB = d.fxBlocks(req.PathBFX)
		spec.Suffix = append(post, last...)
		notes = append(notes, "shared amp with two parallel effect paths (SPS-1)")
	case req.Routing == rig.RoutingSPS:
		// Dual amp: two full amp paths in parallel.
		amp2Model, note2, err := d.resolveAmp(req.Amp2)
		if err != nil {
			return nil, err
		}
		cab2Model := d.resolveCab(req.Cab2, amp2Model)
		mic2Model := d.resolveMic(req.Mic2)
		spec.Prefix = pre
		spec.PathA = []rig.Block{d.ampBlock(ampModel), d.cabBlock(cabModel, micModel)}
		spec.PathB = []rig.Block{d.ampBlock(amp2Model), d.cabBlock(cab2Model, mic2Model)}
		spec.Suffix = append(post, last...)
		notes = append(notes, note2, fmt.Sprintf("cab2 %q", cab2Model))
	case req.Routing == rig.RoutingPS:
		// Split at the input into two amp paths.
		amp2Model, note2, err := d.resolveAmp(req.Amp2)
		if err != nil {
			return nil, err
		}
		cab2Model := d.resolveCab(req.Cab2, amp2Model)
		mic2Model := d.resolveMic(req.Mic2)
		spec.PathA = append(pre, d.ampBlock(ampModel), d.cabBlock(cabModel, micModel))
		spec.PathB = []rig.Block{d.ampBlock(amp2Model), d.cabBlock(cab2Model, mic2Model)}
		spec.Suffix = append(post, last...)
		notes = append(notes, note2, fmt.Sprintf("cab2 %q", cab2Model))
	default:
		// Serial: pre → amp → cab → post → volume.
		blocks := make([]rig.Block, 0, len(pre)+len(post)+len(last)+2)
		blocks = append(blocks, pre...)
		blocks = append(blocks, d.ampBlock(ampModel))
		blocks = append(blocks, d.cabBlock(cabModel, micModel))
		blocks = append(blocks, post...)
		blocks = append(blocks, last...)
		spec.Blocks = blocks
	}

	notes = append(notes, d.footswitchHints(req)...)

	return &Result{Spec: spec, Notes: notes}, nil
}

// footswitchHints nudges the caller towards assigning a stomp switch to the
// modules that need one. Expression-category modules (wah, whammy, …) are
// built to be toggled by a footswitch; when the request includes one but no
// footswitch targets it, the rig would be unplayable as a stompbox.
func (d *Designer) footswitchHints(req Request) []string {
	types := make([]string, 0, len(req.FX)+len(req.PathAFX)+len(req.PathBFX))
	for _, f := range req.FX {
		types = append(types, f.Type)
	}
	for _, f := range req.PathAFX {
		types = append(types, f.Type)
	}
	for _, f := range req.PathBFX {
		types = append(types, f.Type)
	}

	assigned := make(map[string]bool, len(req.Footswitches))
	for _, sw := range req.Footswitches {
		assigned[strings.ToLower(sw.Module)] = true
	}

	var hints []string
	for _, t := range types {
		def, ok := d.cat.FXByName(t)
		if !ok || def.Category != "expression" {
			continue
		}
		if assigned[strings.ToLower(def.Name)] {
			continue
		}
		hints = append(hints, fmt.Sprintf("%s has no footswitch — pass footswitches: [{\"module\": \"%s\"}] to toggle it on/off", def.Name, def.Name))
	}
	return hints
}

// classifyFX orders effects into pre-amp, post-amp and final (Volume) groups.
func (d *Designer) classifyFX(fx []FXBlock) (pre, post, last []rig.Block, err error) {
	for _, f := range fx {
		def, ok := d.cat.FXByName(f.Type)
		if !ok {
			return nil, nil, nil, fmt.Errorf("unknown effect type %q", f.Type)
		}
		block := rig.Block{Type: def.Name, Enabled: f.Enabled, Params: f.Params}
		if block.Params == nil {
			block.Params = map[string]any{}
		}
		if strings.EqualFold(def.Name, "Volume") {
			last = append(last, block)
			continue
		}
		if placeForCategory(def.Category) == "pre-amp" {
			pre = append(pre, block)
		} else {
			post = append(post, block)
		}
	}
	return pre, post, last, nil
}

func (d *Designer) fxBlocks(fx []FXBlock) []rig.Block {
	blocks := make([]rig.Block, 0, len(fx))
	for _, f := range fx {
		blocks = append(blocks, rig.Block{Type: f.Type, Enabled: f.Enabled, Params: f.Params})
	}
	return blocks
}

func (d *Designer) ampBlock(model string) rig.Block {
	return rig.Block{Type: "Amp", Enabled: true, Params: map[string]any{"Type": model, "On": true}}
}

func (d *Designer) cabBlock(cab, mic string) rig.Block {
	return rig.Block{Type: "Cab", Enabled: true, Params: map[string]any{"CabType": cab, "MicType": mic, "On": true}}
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
