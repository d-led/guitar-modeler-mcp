package thr

import (
	"fmt"
	"html"
	"strings"
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
	b.WriteString("<!doctype html><html><head><meta charset=\"utf-8\">")
	fmt.Fprintf(&b, "<title>%s — %s</title>", html.EscapeString(s.Name), html.EscapeString(d.Display))
	b.WriteString(`<style>
body{font-family:system-ui,-apple-system,sans-serif;max-width:720px;margin:2rem auto;padding:0 1rem;color:#1a1a1a}
h1{margin-bottom:.25rem}h2{font-size:1rem;color:#555;margin-top:0}
table{width:100%;border-collapse:collapse;margin-bottom:1rem}
td,th{border-bottom:1px solid #e2e2e2;padding:.45rem .5rem;text-align:left;vertical-align:top}
.module{font-weight:600;white-space:nowrap}
.effect{font-weight:600}.off{color:#999}.inspired{color:#666;font-size:.85em}
.params{color:#444;font-size:.85em;font-variant-numeric:tabular-nums}
</style></head><body>`)
	fmt.Fprintf(&b, "<h1>%s</h1><h2>%s — setup card</h2>", html.EscapeString(s.Name), html.EscapeString(d.Display))

	for _, module := range d.Chain {
		switch module {
		case "COMPRESSOR":
			writeModuleCard(&b, module, onOff(s.Compressor), "", s.Compressor, compressorKnobs(s.CompParams))
		case "NOISE GATE":
			writeModuleCard(&b, module, onOff(s.NoiseGate), "", s.NoiseGate, gateKnobs(s.GateParams))
		case "AMP":
			if cell, ok := d.ampCell(s.Amp); ok {
				writeModuleCard(&b, module, cell.Name, cell.InspiredBy, true, ampKnobs(s.AmpParams))
				if cell.Description != "" {
					writeNote(&b, cell.Description)
				}
			} else {
				writeModuleCard(&b, module, "OFF", "", false, nil)
			}
		case "CAB":
			writeModuleCard(&b, module, s.Cab, "", true, nil)
		case "MOD":
			writeModuleCard(&b, module, s.Mod, "", true, modKnobs(s.ModParams))
		case "ECHO":
			writeModuleCard(&b, module, s.Echo, "", true, echoKnobs(s.EchoParams))
		case "REVERB":
			writeModuleCard(&b, module, s.Reverb, "", true, reverbKnobs(s.ReverbParams))
		}
	}

	if knobs := levelKnobs(s.Levels); len(knobs) > 0 {
		writeModuleCard(&b, "LEVELS", "", "", true, knobs)
	}

	if d.Note != "" {
		writeNote(&b, d.Note)
	}
	b.WriteString("<p class=\"inspired\">Knob values are 0&ndash;100 unless noted (ms). Knobs marked (noon) are at 12 o'clock; set them to noon to reproduce the tone.</p>")
	b.WriteString("</body></html>")
	return b.String()
}

func onOff(on bool) string {
	if on {
		return "ON"
	}
	return "OFF"
}

// writeModuleCard writes one module as a small table: the module label, its
// selection or on/off state, the real hardware it emulates (when known), and
// the knob values that were specified. Unset knobs are omitted.
func writeModuleCard(b *strings.Builder, module, effect, inspired string, enabled bool, knobs []knob) {
	if effect == "" {
		effect = "OFF"
	}
	class := ""
	if !enabled {
		class = " class=\"off\""
	}
	fmt.Fprintf(b, "<table><tr><td class=\"module\"%s>%s</td><td class=\"effect\">%s</td></tr>", class, html.EscapeString(module), html.EscapeString(effect))
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
