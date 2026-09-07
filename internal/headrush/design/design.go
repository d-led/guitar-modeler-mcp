// Package design turns a high-level "dial in a tone" request into a concrete
// rig.Spec: it translates real-world hardware descriptions into device models
// and orders the effects into a musically sensible signal chain.
package design

import (
	"fmt"
	"strings"

	"github.com/d-led/guitar-modeler-mcp/internal/headrush/catalog"
	"github.com/d-led/guitar-modeler-mcp/internal/headrush/rig"
)

// FXBlock is a single effect the caller wants in the chain.
type FXBlock struct {
	Type    string         `json:"type"`
	Enabled bool           `json:"enabled"`
	Params  map[string]any `json:"params,omitempty"`
	// Colour is the module's slot colour tag (e.g. "Blue"); empty means the
	// factory-conventional colour for the effect's category.
	Colour string `json:"colour,omitempty"`
	// Position overrides the effect category's conventional placement: "pre"
	// puts it before the amp, "post" after it (e.g. an octaver tracked from
	// the dry pre-amp signal). Empty = use the category's default.
	Position string `json:"position,omitempty"`
	// Slot pins the effect to an absolute 1-based chain slot (serial routing
	// only), so any layout the device grid allows can be expressed — e.g. a
	// volume pedal at the very front of the chain. Takes precedence over
	// Position.
	Slot *int `json:"slot,omitempty"`
}

// Request is the input to the designer.
type Request struct {
	// Device selects the target hardware. "gigboard" (default) is currently
	// the only supported backend.
	Device string `json:"device,omitempty"`
	Name   string `json:"name"`
	Note   string `json:"note,omitempty"`
	Amp    string `json:"amp"`           // device model or real-hardware description
	Cab    string `json:"cab,omitempty"` // device model or description
	Mic    string `json:"mic,omitempty"` // device model or description
	// AmpParams and CabParams override the amp/cab block's knobs (e.g.
	// "GainA", "Master", "Breakup", "OutGain"). Values are numbers, booleans
	// or strings, keyed by the module's exact parameter names.
	AmpParams map[string]any `json:"amp_params,omitempty"`
	CabParams map[string]any `json:"cab_params,omitempty"`
	Tempo     float64        `json:"tempo,omitempty"`
	InputGain float64        `json:"input_gain,omitempty"`
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

	// Pedals assigns the two expression pedals (Pedal1, Pedal2) to control
	// module parameters, e.g. [{"module":"Black Wah","param":"Pedal"}]. When
	// empty, a wah/whammy/volume in the chain is auto-assigned to Pedal1.
	Pedals []rig.Pedal `json:"pedals,omitempty"`
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

// defaultOutputLevel matches the device's own presets (whose RigVolume sits
// around +5 dB) and offsets a default amp's gain + master (both 50% = −6 dB
// each) enough that a fresh serial rig lands at a healthy, not-hot level. The
// level estimate is a relative hint, not an absolute measurement.
const defaultOutputLevel = 6.0

// Design resolves a request into a buildable rig spec.
func (d *Designer) Design(req Request) (*Result, error) {
	if req.Device != "" && !strings.EqualFold(req.Device, "gigboard") {
		return nil, fmt.Errorf("device %q is not supported yet (supported: gigboard)", req.Device)
	}
	if strings.TrimSpace(req.Name) == "" {
		req.Name = "New Rig"
	}

	ampModel, cabModel, micModel, note, err := d.resolveHardware(req)
	if err != nil {
		return nil, err
	}
	skipCab := hasIR(req.FX)
	notes := hardwareNotes(note, cabModel, micModel, skipCab)

	pre, post, last, pinned, err := d.classifyFX(req.FX)
	if err != nil {
		return nil, err
	}
	if err := validatePinnedRouting(req.Routing, pinned); err != nil {
		return nil, err
	}

	pedals := d.assignExpressionPedals(req, &notes)

	spec := rig.Spec{
		Name:         req.Name,
		Tempo:        resolveTempo(req.Tempo),
		InputGain:    req.InputGain,
		OutputVolume: resolveOutputVolume(req.OutputLevel),
		Routing:      req.Routing,
		Para1Level:   req.Para1Level,
		Para2Level:   req.Para2Level,
		Para1Pan:     req.Para1Pan,
		Para2Pan:     req.Para2Pan,
		ParaDelay:    req.ParaDelay,
		Footswitches: req.Footswitches,
		Pedals:       pedals,
	}

	if err := d.applyRouting(&spec, req, ampModel, cabModel, micModel, skipCab, pre, post, last, pinned, &notes); err != nil {
		return nil, err
	}

	notes = append(notes, d.footswitchHints(req)...)

	return &Result{Spec: spec, Notes: notes}, nil
}

// resolveHardware resolves the amp model and, from it, the cabinet and mic,
// returning the amp-resolution note to surface in the result.
func (d *Designer) resolveHardware(req Request) (ampModel, cabModel, micModel, note string, err error) {
	ampModel, note, err = d.resolveAmp(req.Amp)
	if err != nil {
		return "", "", "", "", err
	}
	return ampModel, d.resolveCab(req.Cab, ampModel), d.resolveMic(req.Mic), note, nil
}

// hardwareNotes records which cab and mic the rig uses, or that an IR loader
// replaces the cabinet.
func hardwareNotes(note, cabModel, micModel string, skipCab bool) []string {
	notes := []string{note}
	if skipCab {
		notes = append(notes, "IR loader replaces the cabinet")
	} else {
		notes = append(notes, fmt.Sprintf("cab %q", cabModel), fmt.Sprintf("mic %q", micModel))
	}
	return notes
}

// validatePinnedRouting rejects absolute slot placement on a parallel chain,
// where the section layout owns the slots.
func validatePinnedRouting(routing rig.Routing, pinned map[int]rig.Block) error {
	if len(pinned) > 0 && routing != "" && routing != rig.RoutingSerial {
		return fmt.Errorf("slot placement is only supported for serial routing (S); use path_a_fx/path_b_fx to place effects on parallel paths")
	}
	return nil
}

// resolveTempo falls back to the device default when no tempo is given.
func resolveTempo(tempo float64) float64 {
	if tempo <= 0 {
		return 100
	}
	return tempo
}

// resolveOutputVolume applies the caller's output level, else the designer's
// +6 dB default that offsets a fresh amp's −12 dB gain staging.
func resolveOutputVolume(level *float64) float64 {
	if level != nil {
		return *level
	}
	return defaultOutputLevel
}

// applyRouting fills the chain sections according to the requested topology.
func (d *Designer) applyRouting(spec *rig.Spec, req Request, ampModel, cabModel, micModel string, skipCab bool, pre, post, last []rig.Block, pinned map[int]rig.Block, notes *[]string) error {
	switch {
	case req.Routing == rig.RoutingSPS && req.Amp2 == "":
		return d.applySharedAmpRouting(spec, req, ampModel, cabModel, micModel, pre, post, last, notes)
	case req.Routing == rig.RoutingSPS:
		return d.applyDualAmpRouting(spec, req, ampModel, cabModel, micModel, pre, post, last, notes, false)
	case req.Routing == rig.RoutingPS:
		return d.applyDualAmpRouting(spec, req, ampModel, cabModel, micModel, pre, post, last, notes, true)
	default:
		d.applySerialRouting(spec, req, ampModel, cabModel, micModel, skipCab, pre, post, last, pinned)
		return nil
	}
}

// applySharedAmpRouting builds the SPS-1 layout with one shared amp feeding two
// parallel effect paths.
func (d *Designer) applySharedAmpRouting(spec *rig.Spec, req Request, ampModel, cabModel, micModel string, pre, post, last []rig.Block, notes *[]string) error {
	spec.Prefix = append(pre, d.ampBlock(ampModel, req.AmpParams), d.cabBlock(cabModel, micModel, req.CabParams))
	pathA, err := d.fxBlocks(req.PathAFX)
	if err != nil {
		return err
	}
	pathB, err := d.fxBlocks(req.PathBFX)
	if err != nil {
		return err
	}
	spec.PathA = pathA
	spec.PathB = pathB
	spec.Suffix = append(post, last...)
	*notes = append(*notes, "shared amp with two parallel effect paths (SPS-1)")
	return nil
}

// applyDualAmpRouting builds a dual-amp parallel layout. When splitAtInput is
// true the chain splits before the amp (PS-1: pre effects join path A);
// otherwise it splits after a shared prefix (SPS-1).
func (d *Designer) applyDualAmpRouting(spec *rig.Spec, req Request, ampModel, cabModel, micModel string, pre, post, last []rig.Block, notes *[]string, splitAtInput bool) error {
	pathA, pathB, note2, cab2Model, err := d.dualAmpPaths(req, ampModel, cabModel, micModel)
	if err != nil {
		return err
	}
	if splitAtInput {
		spec.PathA = append(pre, pathA...)
	} else {
		spec.Prefix = pre
		spec.PathA = pathA
	}
	spec.PathB = pathB
	spec.Suffix = append(post, last...)
	*notes = append(*notes, note2, fmt.Sprintf("cab2 %q", cab2Model))
	return nil
}

// applySerialRouting fills the serial chain: pre → amp → [cab] → post → last,
// with any explicitly pinned effects occupying their requested slots.
func (d *Designer) applySerialRouting(spec *rig.Spec, req Request, ampModel, cabModel, micModel string, skipCab bool, pre, post, last []rig.Block, pinned map[int]rig.Block) {
	blocks := make([]rig.Block, 0, len(pre)+len(post)+len(last)+3)
	blocks = append(blocks, pre...)
	blocks = append(blocks, d.ampBlock(ampModel, req.AmpParams))
	if !skipCab {
		blocks = append(blocks, d.cabBlock(cabModel, micModel, req.CabParams))
	}
	blocks = append(blocks, post...)
	blocks = append(blocks, last...)
	spec.Blocks = blocks
	spec.Pinned = pinned
}

// dualAmpPaths resolves the second amp path and returns both paths' blocks.
func (d *Designer) dualAmpPaths(req Request, ampModel, cabModel, micModel string) (pathA, pathB []rig.Block, note, cab2Model string, err error) {
	amp2Model, note, err := d.resolveAmp(req.Amp2)
	if err != nil {
		return nil, nil, "", "", err
	}
	cab2Model = d.resolveCab(req.Cab2, amp2Model)
	mic2Model := d.resolveMic(req.Mic2)
	pathA = []rig.Block{d.ampBlock(ampModel, req.AmpParams), d.cabBlock(cabModel, micModel, req.CabParams)}
	pathB = []rig.Block{d.ampBlock(amp2Model, nil), d.cabBlock(cab2Model, mic2Model, nil)}
	return pathA, pathB, note, cab2Model, nil
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

// assignExpressionPedals wires an expression-pedal-driven module (wah, whammy,
// volume) to expression pedal 1 when the caller did not specify pedals. A wah
// or whammy with no expression pedal is unplayable, so this must not be left to
// chance. The caller's explicit Pedals, when present, win.
func (d *Designer) assignExpressionPedals(req Request, notes *[]string) []rig.Pedal {
	if len(req.Pedals) > 0 {
		return req.Pedals
	}
	all := append(append(append([]FXBlock{}, req.FX...), req.PathAFX...), req.PathBFX...)
	for _, f := range all {
		def, ok := d.cat.FXByName(f.Type)
		if !ok || def.Category != "expression" {
			continue
		}
		param := expressionParam(def.Name)
		if param == "" {
			continue
		}
		*notes = append(*notes, fmt.Sprintf("assigned expression pedal 1 to %q (%s 0–100)", def.Name, param))
		return []rig.Pedal{{Module: def.Name, Param: param}}
	}
	return nil
}

// expressionParam returns the controller parameter an expression pedal drives
// for a module that needs one: a wah's sweep ("Pedal"), a whammy's pitch
// ("Pitch"), a volume pedal's "Volume". Anything else returns "" (no default).
func expressionParam(name string) string {
	n := strings.ToLower(name)
	switch {
	case strings.Contains(n, "wah"):
		return "Pedal"
	case n == "wham":
		return "Pitch"
	case n == "volume":
		return "Volume"
	default:
		return ""
	}
}

// classifyFX orders effects into pre-amp, post-amp and final (Volume) groups.
// hasIR reports whether any requested effect is an impulse-response loader.
// An IR replaces the cabinet, so the designer drops the cab block for it.
func hasIR(fx []FXBlock) bool {
	for _, f := range fx {
		switch strings.ToLower(strings.TrimSpace(f.Type)) {
		case "ir", "ir (1024)":
			return true
		}
	}
	return false
}

func (d *Designer) classifyFX(fx []FXBlock) (pre, post, last []rig.Block, pinned map[int]rig.Block, err error) {
	pinned = make(map[int]rig.Block)
	for _, f := range fx {
		def, ok := d.cat.FXByName(f.Type)
		if !ok {
			return nil, nil, nil, nil, fmt.Errorf("unknown effect type %q", f.Type)
		}
		block, err := d.buildFXBlock(def, f)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		if f.Slot != nil {
			if err := d.pinBlock(pinned, f, block); err != nil {
				return nil, nil, nil, nil, err
			}
			continue
		}
		place, err := d.placeFXBlock(def, f)
		if err != nil {
			return nil, nil, nil, nil, err
		}
		switch place {
		case "pre":
			pre = append(pre, block)
		case "post":
			post = append(post, block)
		case "last":
			last = append(last, block)
		}
	}
	return pre, post, last, pinned, nil
}

// buildFXBlock resolves one effect into a chain block: its slot colour and
// (normalised, non-nil) parameter map.
func (d *Designer) buildFXBlock(def catalog.FX, f FXBlock) (rig.Block, error) {
	colour, err := d.colourFor(def.Name, f.Colour)
	if err != nil {
		return rig.Block{}, fmt.Errorf("effect %q: %w", def.Name, err)
	}
	params := f.Params
	if params == nil {
		params = map[string]any{}
	}
	return rig.Block{Type: def.Name, Enabled: f.Enabled, Params: params, Colour: colour}, nil
}

// pinBlock pins an effect to an absolute serial slot, rejecting out-of-range
// or duplicate slots.
func (d *Designer) pinBlock(pinned map[int]rig.Block, f FXBlock, block rig.Block) error {
	if *f.Slot < 1 || *f.Slot > 11 {
		return fmt.Errorf("effect %q slot %d is out of range 1..11", block.Type, *f.Slot)
	}
	if _, dup := pinned[*f.Slot]; dup {
		return fmt.Errorf("two effects pinned to slot %d", *f.Slot)
	}
	pinned[*f.Slot] = block
	return nil
}

// placeFXBlock returns which chain section an effect belongs to: its explicit
// position, its category's convention, or "last" for a volume pedal defaulting
// to the end of the chain.
func (d *Designer) placeFXBlock(def catalog.FX, f FXBlock) (string, error) {
	place := strings.ToLower(strings.TrimSpace(f.Position))
	if place == "" {
		if strings.EqualFold(def.Name, "Volume") {
			return "last", nil
		}
		if placeForCategory(def.Category) == "pre-amp" {
			return "pre", nil
		}
		return "post", nil
	}
	switch place {
	case "pre", "pre-amp":
		return "pre", nil
	case "post", "post-amp":
		return "post", nil
	default:
		return "", fmt.Errorf("effect %q position %q is invalid (want \"pre\" or \"post\")", def.Name, f.Position)
	}
}

func (d *Designer) fxBlocks(fx []FXBlock) ([]rig.Block, error) {
	blocks := make([]rig.Block, 0, len(fx))
	for _, f := range fx {
		def, ok := d.cat.FXByName(f.Type)
		if !ok {
			return nil, fmt.Errorf("unknown effect type %q", f.Type)
		}
		block, err := d.buildFXBlock(def, f)
		if err != nil {
			return nil, err
		}
		blocks = append(blocks, block)
	}
	return blocks, nil
}

func (d *Designer) ampBlock(model string, params map[string]any) rig.Block {
	p := make(map[string]any, len(params)+2)
	for k, v := range params {
		p[k] = v
	}
	p["Type"] = model
	p["On"] = true
	return rig.Block{Type: "Amp", Enabled: true, Params: p, Colour: d.defaultColour("Amp")}
}

func (d *Designer) cabBlock(cab, mic string, params map[string]any) rig.Block {
	p := make(map[string]any, len(params)+3)
	for k, v := range params {
		p[k] = v
	}
	p["CabType"] = cab
	p["MicType"] = mic
	p["On"] = true
	return rig.Block{Type: "Cab", Enabled: true, Params: p, Colour: d.defaultColour("Cab")}
}

// colourFor resolves one effect's slot colour: the caller's override wins,
// otherwise the factory-conventional colour for the effect's category. An
// override must be one of the device palette colours.
func (d *Designer) colourFor(moduleType, override string) (string, error) {
	if override != "" {
		if !catalog.ColourValid(override) {
			return "", fmt.Errorf("invalid colour %q (want one of: %s)", override, catalog.ColourList())
		}
		return override, nil
	}
	return d.defaultColour(moduleType), nil
}

// defaultSlotColour overrides the category-derived colour for the fixed
// modules (amp/cab/IR) and the two delays whose factory colour differs from
// the rest of the delay family.
var defaultSlotColour = map[string]string{
	"amp": "Yellow", "cab": "Green", "ir": "Green", "ir (1024)": "Green",
	"pitch delay": "Purple", "reso delay": "Red",
}

// familySlotColour maps an effect family to a more specific colour than its
// category would otherwise get (wahs orange while volume is red, phasers
// orange while chorus is purple, …).
var familySlotColour = map[string]string{
	"hold": "Purple", "wah": "Orange", "whammy": "Purple", "harmonizer": "Blue",
	"flanger": "Orange", "phaser": "Orange", "rotary": "Orange",
	"doubler": "Red", "octave": "Blue", "filter": "Yellow",
}

// categorySlotColour maps an effect category to its factory-conventional slot
// colour, the dominant colour observed across the device's own rigs.
var categorySlotColour = map[string]string{
	"distortion": "Yellow", "eq": "Yellow", "dynamics": "Red",
	"expression": "Red", "modulation": "Purple",
	"delay": "Green", "reverb": "Blue", "utility": "Yellow",
}

// defaultColour is the slot colour the factory rigs conventionally give a
// module type, so a fresh rig reads like a stock one instead of every slot
// defaulting to green.
func (d *Designer) defaultColour(moduleType string) string {
	n := strings.ToLower(strings.TrimSpace(moduleType))
	if c, ok := defaultSlotColour[n]; ok {
		return c
	}
	def, ok := d.cat.FXByName(moduleType)
	if !ok {
		return "Green"
	}
	if c, ok := familySlotColour[def.Family]; ok {
		return c
	}
	if c, ok := categorySlotColour[def.Category]; ok {
		return c
	}
	return "Green"
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
