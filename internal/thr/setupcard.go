package thr

import (
	"embed"
	"html/template"
	"strings"

	"github.com/d-led/guitar-modeler-mcp/internal/cardchain"
)

//go:embed css.tmpl html.tmpl
var cardFS embed.FS

var (
	cardCSS  = readCard("css.tmpl")
	cardTmpl = template.Must(template.New("thr-card").Parse(readCard("html.tmpl")))
)

func readCard(name string) string {
	b, err := cardFS.ReadFile(name)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// Spec is a tone to dial in on a THR: the amp-selector position, the cabinet,
// the EFFECT, ECHO and REVERB choices, the two app-only toggles, and the knob
// values for every module. The selection fields are required where noted; the
// knob fields are optional (Unset knobs are omitted from the card).
type Spec struct {
	Name         string
	Amp          string
	Cab          string
	Mod          string
	Echo         string
	Reverb       string
	Compressor   bool
	NoiseGate    bool
	AmpParams    AmpParams
	ModParams    ModParams
	EchoParams   EchoParams
	ReverbParams ReverbParams
	CompParams   CompressorParams
	GateParams   GateParams
	Levels       Levels
	// Note is optional prose for the setup card: why this tone is voiced this
	// way, how to play it, and the rest of the rig and hardware. It is printed
	// at the bottom of the card and never sent to a device.
	Note string
}

// NewSpec returns a Spec with every knob Unset, so the setup card shows only
// the values that are explicitly assigned. Fill in the selection fields (amp
// is required) and any knobs you want on the card.
func NewSpec() Spec {
	return Spec{
		AmpParams:    AmpParams{Gain: Unset, Master: Unset, Bass: Unset, Mid: Unset, Treble: Unset},
		ModParams:    ModParams{Speed: Unset, Depth: Unset, PreDelay: Unset, Feedback: Unset, Mix: Unset},
		EchoParams:   EchoParams{Time: Unset, Feedback: Unset, Bass: Unset, Treble: Unset, Mix: Unset},
		ReverbParams: ReverbParams{Level: Unset, Decay: Unset, PreDelay: Unset, Tone: Unset, Mix: Unset},
		CompParams:   CompressorParams{Sustain: Unset, Level: Unset},
		GateParams:   GateParams{Threshold: Unset, Decay: Unset},
		Levels:       Levels{Guitar: Unset, Audio: Unset},
	}
}

// SetupCardHTML renders a printable setup card for a resolved Spec.
func (d Device) SetupCardHTML(s Spec) string {
	var b strings.Builder
	cardchain.Head(&b, s.Name+" — "+d.Display, cardCSS)
	if err := cardTmpl.Execute(&b, cardPage{
		H1:         s.Name,
		H2:         d.Display + " — setup card",
		Chain:      template.HTML(chainHint(d.Chain, s)), // #nosec G203 -- trusted chain HTML
		Cards:      d.cards(s),
		DeviceNote: d.Note,
		Note:       s.Note,
	}); err != nil {
		panic(err)
	}
	return b.String()
}

// cardPage is the data for the THR setup-card template.
type cardPage struct {
	H1, H2     string
	Chain      template.HTML
	Cards      []moduleCard
	DeviceNote string
	Note       string
}

// cardKnob is one named control value shown on the card.
type cardKnob struct {
	Name  string
	Value int
}

// moduleCard is one module (or the LEVELS summary) rendered as a small table.
type moduleCard struct {
	Module   string
	Effect   string
	Inspired string
	Enabled  bool
	Slot     int
	Knobs    []cardKnob
	Note     string
}

// cards builds the module cards in chain order, followed by the LEVELS summary
// when any level knob is set.
func (d Device) cards(s Spec) []moduleCard {
	cards := make([]moduleCard, 0, len(d.Chain)+1)
	for i, module := range d.Chain {
		cards = append(cards, d.moduleCard(module, s, i+1))
	}
	if knobs := viewKnobs(levelKnobs(s.Levels)); len(knobs) > 0 {
		cards = append(cards, moduleCard{Module: "LEVELS", Effect: "OFF", Enabled: true, Knobs: knobs})
	}
	return cards
}

func (d Device) moduleCard(module string, s Spec, slot int) moduleCard {
	switch module {
	case "COMPRESSOR":
		return moduleCard{Module: module, Effect: onOff(s.Compressor), Enabled: s.Compressor, Slot: slot, Knobs: viewKnobs(compressorKnobs(s.CompParams))}
	case "NOISE GATE":
		return moduleCard{Module: module, Effect: onOff(s.NoiseGate), Enabled: s.NoiseGate, Slot: slot, Knobs: viewKnobs(gateKnobs(s.GateParams))}
	case "AMP":
		return d.ampCard(module, s, slot)
	case "CAB":
		return moduleCard{Module: module, Effect: effectOrOff(s.Cab), Enabled: true, Slot: slot}
	case "MOD":
		return moduleCard{Module: module, Effect: effectOrOff(s.Mod), Enabled: true, Slot: slot, Knobs: viewKnobs(modKnobs(s.ModParams))}
	case "ECHO":
		return moduleCard{Module: module, Effect: effectOrOff(s.Echo), Enabled: true, Slot: slot, Knobs: viewKnobs(echoKnobs(s.EchoParams))}
	case "REVERB":
		return moduleCard{Module: module, Effect: effectOrOff(s.Reverb), Enabled: true, Slot: slot, Knobs: viewKnobs(reverbKnobs(s.ReverbParams))}
	}
	return moduleCard{}
}

func (d Device) ampCard(module string, s Spec, slot int) moduleCard {
	cell, ok := d.ampCell(s.Amp)
	if !ok {
		return moduleCard{Module: module, Effect: "OFF", Enabled: false, Slot: slot}
	}
	return moduleCard{Module: module, Effect: cell.Name, Inspired: cell.InspiredBy, Enabled: true, Slot: slot, Knobs: viewKnobs(ampKnobs(s.AmpParams)), Note: cell.Description}
}

func effectOrOff(v string) string {
	if v == "" {
		return "OFF"
	}
	return v
}

func viewKnobs(ks []knob) []cardKnob {
	out := make([]cardKnob, 0, len(ks))
	for _, k := range ks {
		out = append(out, cardKnob{Name: k.name, Value: k.value})
	}
	return out
}

func onOff(on bool) string {
	if on {
		return "ON"
	}
	return "OFF"
}

// chainHint renders the fixed signal chain with each slot's selection.
func chainHint(chain []string, s Spec) string {
	steps := make([]cardchain.Step, 0, len(chain))
	for i, module := range chain {
		steps = append(steps, cardchain.Step{Slot: i + 1, Module: module, Effect: thrEffect(module, s)})
	}
	return cardchain.Render(steps)
}

func thrEffect(module string, s Spec) string {
	switch module {
	case "COMPRESSOR":
		return onOff(s.Compressor)
	case "NOISE GATE":
		return onOff(s.NoiseGate)
	case "AMP":
		return s.Amp
	case "CAB":
		return s.Cab
	case "MOD":
		return s.Mod
	case "ECHO":
		return s.Echo
	case "REVERB":
		return s.Reverb
	}
	return ""
}
