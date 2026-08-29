package thr

import (
	"fmt"
	"html"
	"strings"
)

// Spec is a tone to dial in on a THR: the amp-selector position, the cabinet,
// the EFFECT, ECHO and REVERB choices, and the two app-only toggles.
type Spec struct {
	Name       string
	Amp        string
	Cab        string
	Mod        string
	Echo       string
	Reverb     string
	Compressor bool
	NoiseGate  bool
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
</style></head><body>`)
	fmt.Fprintf(&b, "<h1>%s</h1><h2>%s — setup card</h2>", html.EscapeString(s.Name), html.EscapeString(d.Display))

	for _, module := range d.Chain {
		switch module {
		case "COMPRESSOR":
			writeModule(&b, module, onOff(s.Compressor), "")
		case "NOISE GATE":
			writeModule(&b, module, onOff(s.NoiseGate), "")
		case "AMP":
			if cell, ok := d.ampCell(s.Amp); ok {
				writeModule(&b, module, cell.Name, cell.InspiredBy)
				if cell.Description != "" {
					writeNote(&b, cell.Description)
				}
			} else {
				writeModule(&b, module, "OFF", "")
			}
		case "CAB":
			writeModule(&b, module, s.Cab, "")
		case "MOD":
			writeModule(&b, module, s.Mod, "")
		case "ECHO":
			writeModule(&b, module, s.Echo, "")
		case "REVERB":
			writeModule(&b, module, s.Reverb, "")
		}
	}

	if d.Note != "" {
		writeNote(&b, d.Note)
	}
	b.WriteString("<p class=\"inspired\">Set the knob positions in the THR Remote app.</p>")
	b.WriteString("</body></html>")
	return b.String()
}

func onOff(on bool) string {
	if on {
		return "ON"
	}
	return "OFF"
}

func writeModule(b *strings.Builder, module, effect, inspired string) {
	if effect == "" {
		effect = "OFF"
	}
	fmt.Fprintf(b, "<table><tr><td class=\"module\">%s</td><td class=\"effect\">%s</td></tr>", html.EscapeString(module), html.EscapeString(effect))
	if inspired != "" {
		fmt.Fprintf(b, "<tr><td></td><td><span class=\"inspired\">based on %s</span></td></tr>", html.EscapeString(inspired))
	}
	b.WriteString("</table>")
}

func writeNote(b *strings.Builder, note string) {
	fmt.Fprintf(b, "<p class=\"inspired\">%s</p>", html.EscapeString(note))
}
