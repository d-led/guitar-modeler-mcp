package thr

import (
	"fmt"
	"html"
	"strings"

	"github.com/d-led/guitar-modeler-mcp/internal/cardchain"
)

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
	cardchain.Head(&b, s.Name+" — "+d.Display, `.off{color:#999}.inspired{color:#666;font-size:.85em}
.params{color:#444;font-size:.85em;font-variant-numeric:tabular-nums}`)
	fmt.Fprintf(&b, "<h1>%s</h1><h2>%s — setup card</h2>", html.EscapeString(s.Name), html.EscapeString(d.Display))

	b.WriteString(chainHint(d.Chain, s))

	for i, module := range d.Chain {
		d.writeChainModule(&b, module, s, i+1)
	}

	if knobs := levelKnobs(s.Levels); len(knobs) > 0 {
		writeModuleCard(&b, "LEVELS", "", "", true, knobs, 0)
	}

	if d.Note != "" {
		writeNote(&b, d.Note)
	}
	b.WriteString("<p class=\"inspired\">Knob values are 0&ndash;100 unless noted (ms). Knobs marked (noon) are at 12 o'clock; set them to noon to reproduce the tone.</p>")
	b.WriteString("</body></html>")
	return b.String()
}

// writeChainModule renders one module of the fixed signal chain as a card.
func (d Device) writeChainModule(b *strings.Builder, module string, s Spec, slot int) {
	switch module {
	case "COMPRESSOR":
		writeModuleCard(b, module, onOff(s.Compressor), "", s.Compressor, compressorKnobs(s.CompParams), slot)
	case "NOISE GATE":
		writeModuleCard(b, module, onOff(s.NoiseGate), "", s.NoiseGate, gateKnobs(s.GateParams), slot)
	case "AMP":
		d.writeAmpCard(b, module, s, slot)
	case "CAB":
		writeModuleCard(b, module, s.Cab, "", true, nil, slot)
	case "MOD":
		writeModuleCard(b, module, s.Mod, "", true, modKnobs(s.ModParams), slot)
	case "ECHO":
		writeModuleCard(b, module, s.Echo, "", true, echoKnobs(s.EchoParams), slot)
	case "REVERB":
		writeModuleCard(b, module, s.Reverb, "", true, reverbKnobs(s.ReverbParams), slot)
	}
}

func (d Device) writeAmpCard(b *strings.Builder, module string, s Spec, slot int) {
	cell, ok := d.ampCell(s.Amp)
	if !ok {
		writeModuleCard(b, module, "OFF", "", false, nil, slot)
		return
	}
	writeModuleCard(b, module, cell.Name, cell.InspiredBy, true, ampKnobs(s.AmpParams), slot)
	if cell.Description != "" {
		writeNote(b, cell.Description)
	}
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

// writeModuleCard writes one module as a small table: the module label, its
// selection or on/off state, the real hardware it emulates (when known), and
// the knob values that were specified. Unset knobs are omitted.
func writeModuleCard(b *strings.Builder, module, effect, inspired string, enabled bool, knobs []knob, slot int) {
	if effect == "" {
		effect = "OFF"
	}
	class := ""
	if !enabled {
		class = " class=\"off\""
	}
	badge := ""
	if slot > 0 {
		badge = fmt.Sprintf("<span class=\"slotbadge\">%d</span>", slot)
	}
	fmt.Fprintf(b, "<table><tr><td class=\"module\"%s>%s%s</td><td class=\"effect\">%s</td></tr>", class, badge, html.EscapeString(module), html.EscapeString(effect))
	if inspired != "" {
		fmt.Fprintf(b, "<tr><td></td><td><span class=\"inspired\">based on %s</span></td></tr>", html.EscapeString(inspired))
	}
	if len(knobs) > 0 {
		parts := make([]string, 0, len(knobs))
		for _, k := range knobs {
			parts = append(parts, fmt.Sprintf("%s: %d", k.name, k.value))
		}
		fmt.Fprintf(b, "<tr><td class=\"params\">%s</td><td></td></tr>", html.EscapeString(strings.Join(parts, " · ")))
	}
	b.WriteString("</table>")
}

func writeNote(b *strings.Builder, note string) {
	fmt.Fprintf(b, "<p class=\"inspired\">%s</p>", html.EscapeString(note))
}
