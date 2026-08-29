package qc

import (
	"fmt"
	"html"
	"math"
	"strconv"
	"strings"
)

// Caveat is the honest limitation of the file route, surfaced to agents in
// the tool description and printed on every setup card.
const Caveat = "The .pb is a valid encrypted BinaryPreset and round-trips " +
	"through qc_decode_preset, but loading it onto a unit by copying the file " +
	"is not yet confirmed on hardware. qc_design builds a single-lane serial " +
	"chain; split/parallel routing is not modelled yet."

// SetupCardHTML renders a printable setup card for a decoded preset: the
// signal chain with each block's name, the hardware it is based on, and each
// parameter's name with its real (screen) value.
func SetupCardHTML(cat *Catalog, preset *BinaryPreset) string {
	var b strings.Builder
	b.WriteString("<!doctype html><html><head><meta charset=\"utf-8\">")
	fmt.Fprintf(&b, "<title>%s — Quad Cortex</title>", html.EscapeString(preset.Name))
	b.WriteString(`<style>
body{font-family:system-ui,-apple-system,sans-serif;max-width:720px;margin:2rem auto;padding:0 1rem;color:#1a1a1a}
h1{margin-bottom:.25rem}h2{font-size:1rem;color:#555;margin-top:0}
h3{font-size:.95rem;margin:1.25rem 0 .5rem;text-transform:uppercase;letter-spacing:.03em;color:#444}
table{width:100%;border-collapse:collapse;margin-bottom:.5rem}
td,th{border-bottom:1px solid #e2e2e2;padding:.45rem .5rem;text-align:left;vertical-align:top}
.block{font-weight:600}.inspired{color:#666;font-size:.85em}
.params{color:#444;font-size:.85em;font-variant-numeric:tabular-nums}
.note{color:#8a5a00;background:#fff7e6;border:1px solid #f0d9a8;padding:.6rem .8rem;border-radius:6px;font-size:.85em}
</style></head><body>`)
	fmt.Fprintf(&b, "<h1>%s</h1><h2>Neural DSP Quad Cortex — setup card</h2>", html.EscapeString(preset.Name))

	if preset.AuthorName != "" {
		fmt.Fprintf(&b, "<p class=\"inspired\">by %s</p>", html.EscapeString(preset.AuthorName))
	}
	fmt.Fprintf(&b, "<p class=\"inspired\">volume %.3g · pan %.3g</p>", preset.Volume, preset.Pan)

	for _, chain := range preset.Chains {
		row := chain.GetRow() + 1 // screen rows are 1..4
		fmt.Fprintf(&b, "<h3>Row %d</h3>", row)
		for _, model := range chain.Models {
			writeBlock(&b, cat, model)
		}
	}

	b.WriteString("<p class=\"note\">" + html.EscapeString(Caveat) + "</p>")
	b.WriteString("</body></html>")
	return b.String()
}

// writeBlock renders one grid block as a table: its model name, the hardware
// it is based on, and each parameter with its real value.
func writeBlock(b *strings.Builder, cat *Catalog, model *Model) {
	name := fmt.Sprintf("model %d", model.GetHash())
	based := ""
	var params []string

	if m, ok := cat.Model(int(model.GetHash())); ok {
		name = m.Name
		based = m.BasedOn
		for _, p := range model.Params {
			label := fmt.Sprintf("param %d", p.GetIndex())
			value := ""
			if int(p.GetIndex()) < len(m.Params) {
				spec := m.Params[p.GetIndex()]
				label = spec.Name
				if len(p.ParamValues) > 0 {
					value = formatValue(spec, float64(p.ParamValues[0].GetFloatValue()))
				}
			}
			params = append(params, label+": "+value)
		}
	}

	fmt.Fprintf(b, "<table><tr><td class=\"block\">%s</td><td></td></tr>", html.EscapeString(name))
	if based != "" {
		fmt.Fprintf(b, "<tr><td></td><td class=\"inspired\">based on %s</td></tr>", html.EscapeString(based))
	}
	if len(params) > 0 {
		fmt.Fprintf(b, "<tr><td class=\"params\" colspan=\"2\">%s</td></tr>", html.EscapeString(strings.Join(params, " · ")))
	}
	b.WriteString("</table>")
}

// formatValue renders one parameter's wire value in the parameter's own units,
// or the selected option name for a list parameter.
func formatValue(spec ParamSpec, wire float64) string {
	if spec.Steps > 0 {
		if idx, err := spec.ValueToOption(wire); err == nil && idx < len(spec.StepNames) {
			return spec.StepNames[idx]
		}
		return strconv.FormatFloat(wire, 'g', -1, 64)
	}
	if real, err := spec.Denormalize(wire); err == nil {
		rounded := roundForDisplay(real, spec.Units)
		text := strconv.FormatFloat(rounded, 'f', -1, 64)
		if spec.Units != "" {
			return text + " " + spec.Units
		}
		return text
	}
	return strconv.FormatFloat(wire, 'g', -1, 64)
}

// roundForDisplay rounds a real value to the precision a screen would show for
// its unit, so a knob set to 0 dB reads "0 dB" rather than "-0.0002 dB".
func roundForDisplay(real float64, units string) float64 {
	places := 2
	switch units {
	case "dB", "dB/oct":
		places = 1
	case "Hz":
		places = 0
	case "%", "ms":
		places = 1
	case "s", "Semitones", "Cents":
		places = 2
	}
	p := math.Pow(10, float64(places))
	rounded := math.Round(real*p) / p
	if rounded == 0 {
		// Normalise -0 (from rounding a tiny negative) to plain 0.
		return 0
	}
	return rounded
}
