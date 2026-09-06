package mooer

import (
	"embed"
	"fmt"
	"html"
	"html/template"
	"strings"

	"github.com/d-led/guitar-modeler-mcp/internal/cardchain"
)

//go:embed css.tmpl html.tmpl
var cardFS embed.FS

var (
	cardCSS  = readCard("css.tmpl")
	cardTmpl = template.Must(template.New("mooer-card").Parse(readCard("html.tmpl")))
)

func readCard(name string) string {
	b, err := cardFS.ReadFile(name)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// ParamDesc is one editable parameter of a module, with its raw device value
// and its resting/default value (nil when the default is unknown).
type ParamDesc struct {
	Name    string
	Value   any
	Default any
}

// ModuleDesc describes one module of a preset for display: which effect it
// holds, whether it is on, and its parameter values.
type ModuleDesc struct {
	Module     string
	Effect     string
	InspiredBy string
	Enabled    bool
	Params     []ParamDesc
}

// describeFuncs maps a chain module name to the function that reads its label,
// enabled flag, effect index and parameters out of a preset.
var describeFuncs = map[string]func(Preset) (string, bool, uint8, []ParamDesc){
	"fx": func(p Preset) (string, bool, uint8, []ParamDesc) {
		return "FX", p.FX.Enabled, p.FX.Type, fxParams(p.FX)
	},
	"od": func(p Preset) (string, bool, uint8, []ParamDesc) {
		return "DS/OD", p.Drive.Enabled, p.Drive.Type, driveParams(p.Drive)
	},
	"amp": func(p Preset) (string, bool, uint8, []ParamDesc) {
		return "AMP", p.Amp.Enabled, p.Amp.Type, ampParams(p.Amp)
	},
	"cab": func(p Preset) (string, bool, uint8, []ParamDesc) {
		return "CAB", p.Cab.Enabled, p.Cab.Type, cabParams(p.Cab)
	},
	"ns": func(p Preset) (string, bool, uint8, []ParamDesc) {
		return "NS", p.NoiseGate.Enabled, p.NoiseGate.Type, nsParams(p.NoiseGate)
	},
	"eq": func(p Preset) (string, bool, uint8, []ParamDesc) {
		return "EQ", p.EQ.Enabled, p.EQ.Type, eqParams(p.EQ)
	},
	"mod": func(p Preset) (string, bool, uint8, []ParamDesc) {
		return "MOD", p.Mod.Enabled, p.Mod.Type, modParams(p.Mod)
	},
	"delay": func(p Preset) (string, bool, uint8, []ParamDesc) {
		return "DELAY", p.Delay.Enabled, p.Delay.Type, delayParams(p.Delay)
	},
	"reverb": func(p Preset) (string, bool, uint8, []ParamDesc) {
		return "REVERB", p.Reverb.Enabled, p.Reverb.Type, reverbParams(p.Reverb)
	},
}

// Describe flattens a preset into a display-ready list of module descriptions
// in the model's chain order (ModuleOrder), resolving effect_type indices to
// names via the model's catalog.
func Describe(p Preset, m Model) []ModuleDesc {
	order := m.ModuleOrder
	if len(order) == 0 {
		order = ModuleOrder
	}
	desc := make([]ModuleDesc, 0, len(order))
	for _, module := range order {
		fn, ok := describeFuncs[module]
		if !ok {
			continue
		}
		label, enabled, index, params := fn(p)
		desc = append(desc, describeModule(label, module, enabled, index, m, params))
	}
	return desc
}

func describeModule(label, module string, enabled bool, index uint8, m Model, params []ParamDesc) ModuleDesc {
	effect := m.EffectName(module, index)
	inspired, _ := m.InspiredFX(module, effect)
	switch module {
	case "amp":
		inspired, _ = m.InspiredAmp(effect)
	case "cab":
		inspired, _ = m.InspiredCab(effect)
	}
	return ModuleDesc{Module: label, Effect: effect, InspiredBy: inspired, Enabled: enabled, Params: params}
}

func fxParams(f FX) []ParamDesc {
	return []ParamDesc{
		{"Q", f.Q, noon}, {"Position", f.Position, noon}, {"Peak", f.Peak, noon}, {"Level", f.Level, noon},
	}
}
func driveParams(d Drive) []ParamDesc {
	return []ParamDesc{
		{"Volume", d.Volume, noon}, {"Tone", d.Tone, noon}, {"Gain", d.Gain, noon},
	}
}
func ampParams(a Amp) []ParamDesc {
	return []ParamDesc{
		{"Gain", a.Gain, noon}, {"Bass", a.Bass, noon}, {"Mid", a.Mid, noon},
		{"Treble", a.Treble, noon}, {"Presence", a.Presence, noon}, {"Master", a.Master, noon},
	}
}
func cabParams(c Cab) []ParamDesc {
	return []ParamDesc{
		{"Mic", c.Mic, 0}, {"Center", c.Center, noon}, {"Distance", c.Distance, noon}, {"Tube", c.Tube, noon},
	}
}
func nsParams(n NoiseGate) []ParamDesc {
	return []ParamDesc{
		{"Attack", n.Attack, noon}, {"Release", n.Release, noon}, {"Threshold", n.Threshold, 0},
	}
}
func eqParams(e EQ) []ParamDesc {
	out := make([]ParamDesc, 0, 12)
	for i, v := range e.Bands {
		out = append(out, ParamDesc{fmt.Sprintf("Band %d", i+1), v, noon})
	}
	for i, v := range e.BandsExtra {
		out = append(out, ParamDesc{fmt.Sprintf("Band %d", i+7), v, noon})
	}
	return out
}
func modParams(m Mod) []ParamDesc {
	return []ParamDesc{
		{"Rate", m.Rate, noon}, {"Level", m.Level, noon}, {"Depth", m.Depth, noon},
		{"Param 4", m.Param4, noon}, {"Param 5", m.Param5, noon},
	}
}
func delayParams(d Delay) []ParamDesc {
	return []ParamDesc{
		{"Level", d.Level, noon}, {"Feedback", d.Feedback, noon}, {"Time (ms)", d.TimeMS, neutralDelayTime},
		{"Subdivision", d.Subdivision, 0}, {"Param 5", d.Param5, noon}, {"Param 6", d.Param6, noon},
	}
}
func reverbParams(r Reverb) []ParamDesc {
	return []ParamDesc{
		{"Pre-Delay", r.PreDelay, noon}, {"Level", r.Level, noon}, {"Decay", r.Decay, noon}, {"Tone", r.Tone, noon},
	}
}

// changed reports whether a parameter deviates from its resting default.
func changed(p ParamDesc) bool {
	if p.Default == nil {
		return false
	}
	return fmt.Sprintf("%v", p.Value) != fmt.Sprintf("%v", p.Default)
}

// chainHint renders the fixed nine-slot chain with each slot's selected model,
// so the models are attributable to their slot positions at a glance.
func chainHint(desc []ModuleDesc) string {
	steps := make([]cardchain.Step, 0, len(desc))
	for i, d := range desc {
		steps = append(steps, cardchain.Step{Slot: i + 1, Module: d.Module, Effect: d.Effect})
	}
	return cardchain.Render(steps)
}

// SetupCardHTML renders a printable setup card for a preset on a device. It is
// the human-readable output for devices without preset file transfer, and the
// companion report for devices that can also write a .mo file. Note is optional
// prose for the card (why this tone, how to play it, the rest of the rig and
// hardware); it is printed at the bottom of the card and never written to the
// .mo file.
func SetupCardHTML(m Model, p Preset, note string) string {
	var b strings.Builder
	if err := cardTmpl.Execute(&b, cardPage{
		Title:      p.Name + " — " + m.Display,
		H1:         p.Name,
		H2:         m.Display + " — setup card",
		CSS:        template.CSS(cardCSS),       // #nosec G203 -- trusted package CSS
		ChainCSS:   template.CSS(cardchain.CSS), // #nosec G203 -- trusted package CSS
		StoredNote: storedNoteHTML(p.Name),
		Chain:      template.HTML(chainHint(Describe(p, m))), // #nosec G203 -- trusted chain HTML
		Modules:    cardModules(p, m),
		Note:       note,
	}); err != nil {
		panic(err)
	}
	return b.String()
}

// cardPage is the data for the Mooer setup-card template.
type cardPage struct {
	Title      string
	H1, H2     string
	CSS        template.CSS
	ChainCSS   template.CSS
	StoredNote template.HTML
	Chain      template.HTML
	Modules    []moduleCard
	Note       string
}

// cardParam is one named parameter value shown on the card.
type cardParam struct {
	Name    string
	Value   string
	Changed bool
}

// moduleCard is one chain module rendered as a table.
type moduleCard struct {
	Slot     int
	Module   string
	Effect   string
	Inspired string
	Enabled  bool
	State    string
	Params   []cardParam
}

// storedNoteHTML renders the truncated-name warning, or empty when the name
// fits on the device.
func storedNoteHTML(name string) template.HTML {
	stored, truncated := StoredName(name)
	if !truncated {
		return ""
	}
	return template.HTML(fmt.Sprintf("Note: the device stores preset names up to %d characters; this preset reads as %q on the unit.", NameLimit, html.EscapeString(stored))) // #nosec G203 -- pre-escaped trusted HTML
}

func cardModules(p Preset, m Model) []moduleCard {
	desc := Describe(p, m)
	cards := make([]moduleCard, 0, len(desc))
	for i, d := range desc {
		state := "ON"
		if !d.Enabled {
			state = "OFF"
		}
		cards = append(cards, moduleCard{Slot: i + 1, Module: d.Module, Effect: d.Effect, Inspired: d.InspiredBy, Enabled: d.Enabled, State: state, Params: cardParams(d.Params)})
	}
	return cards
}

func cardParams(ps []ParamDesc) []cardParam {
	out := make([]cardParam, 0, len(ps))
	for _, p := range ps {
		out = append(out, cardParam{Name: p.Name, Value: fmt.Sprintf("%v", p.Value), Changed: changed(p)})
	}
	return out
}
