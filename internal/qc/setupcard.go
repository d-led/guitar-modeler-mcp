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

// SetupCardHTML renders a self-contained, printable setup card for a decoded
// preset: the signal chain (in order), each block's name and the hardware it
// is based on, and every knob with its value — the values the preset sets
// explicitly, and the catalog defaults for the rest, so the whole tone can be
// reproduced by hand from the card alone.
func SetupCardHTML(cat *Catalog, preset *BinaryPreset) string {
	var b strings.Builder
	b.WriteString("<!doctype html><html><head><meta charset=\"utf-8\">")
	fmt.Fprintf(&b, "<title>%s — Quad Cortex</title>", html.EscapeString(preset.Name))
	b.WriteString(`<style>
body{font-family:system-ui,-apple-system,sans-serif;max-width:760px;margin:2rem auto;padding:0 1rem;color:#1a1a1a}
h1{margin-bottom:.25rem}h2{font-size:1rem;color:#555;margin-top:0}
h3{font-size:.95rem;margin:1.25rem 0 .5rem;text-transform:uppercase;letter-spacing:.03em;color:#444}
table{width:100%;border-collapse:collapse;margin-bottom:.5rem}
td,th{border-bottom:1px solid #e2e2e2;padding:.45rem .5rem;text-align:left;vertical-align:top}
.block{font-weight:600}.inspired{color:#666;font-size:.85em}
.params{color:#444;font-size:.85em;font-variant-numeric:tabular-nums}
.chain{color:#333;font-size:.9em;margin:0 0 1rem}
.note{color:#8a5a00;background:#fff7e6;border:1px solid #f0d9a8;padding:.6rem .8rem;border-radius:6px;font-size:.85em}
</style></head><body>`)
	fmt.Fprintf(&b, "<h1>%s</h1><h2>Neural DSP Quad Cortex — setup card</h2>", html.EscapeString(preset.Name))

	if preset.AuthorName != "" {
		fmt.Fprintf(&b, "<p class=\"inspired\">by %s</p>", html.EscapeString(preset.AuthorName))
	}
	fmt.Fprintf(&b, "<p class=\"inspired\">volume %.3g · pan %.3g</p>", preset.Volume, preset.Pan)

	chain := chainNames(cat, preset)
	if chain != "" {
		fmt.Fprintf(&b, "<p class=\"chain\">%s</p>", html.EscapeString(chain))
	}

	for _, c := range preset.Chains {
		row := c.GetRow() + 1 // screen rows are 1..4
		fmt.Fprintf(&b, "<h3>Row %d</h3>", row)
		for _, model := range c.Models {
			writeBlock(&b, cat, model)
		}
	}

	b.WriteString("<p class=\"note\">" + html.EscapeString(Caveat) + "</p>")
	b.WriteString("</body></html>")
	return b.String()
}

// chainNames renders the preset's signal path as one line: "Input → A → B →
// Output". Rows after the first are appended as separate chains.
func chainNames(cat *Catalog, preset *BinaryPreset) string {
	var chains []string
	for _, c := range preset.Chains {
		names := make([]string, 0, len(c.Models)+2)
		names = append(names, "Input")
		for _, model := range c.Models {
			names = append(names, modelName(cat, model))
		}
		names = append(names, "Output")
		chains = append(chains, strings.Join(names, " → "))
	}
	return strings.Join(chains, "  ·  ")
}

func modelName(cat *Catalog, model *Model) string {
	if m, ok := cat.Model(int(model.GetHash())); ok {
		return m.Name
	}
	return fmt.Sprintf("model %d", model.GetHash())
}

// writeBlock renders one grid block as a table: its model name, the hardware
// it is based on, and every knob with a value.
func writeBlock(b *strings.Builder, cat *Catalog, model *Model) {
	name := modelName(cat, model)
	based := ""
	var params []string
	if m, ok := cat.Model(int(model.GetHash())); ok {
		based = m.BasedOn
		params = blockParams(m, model)
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

// blockParams returns every knob of a model with its value, in the model's
// parameter order. Knobs the preset sets explicitly show that value; the rest
// show the catalog default, so the card is self-contained.
func blockParams(m *ModelSpec, model *Model) []string {
	wireByIndex := map[uint32]float64{}
	for _, p := range model.Params {
		if len(p.ParamValues) > 0 {
			wireByIndex[p.GetIndex()] = float64(p.ParamValues[0].GetFloatValue())
		}
	}
	var out []string
	for i, spec := range m.Params {
		if !isKnob(spec) {
			continue
		}
		var value string
		if wire, ok := wireByIndex[uint32(i)]; ok {
			value = formatWire(spec, wire)
		} else {
			value = formatDefault(spec)
		}
		out = append(out, spec.Name+": "+value)
	}
	return out
}

// isKnob reports whether a parameter is a dial-able knob worth printing:
// wire placeholders, meters and notification markers are not.
func isKnob(spec ParamSpec) bool {
	if spec.padding || spec.Type == "grMeter" {
		return false
	}
	if strings.HasPrefix(spec.Name, "NOTIFICATION_") {
		return false
	}
	return spec.Name != ""
}

// formatDefault renders a catalog default value in screen units.
func formatDefault(spec ParamSpec) string {
	if spec.Steps > 0 {
		if idx, err := spec.ValueToOption(spec.Default); err == nil && idx < len(spec.StepNames) {
			return spec.StepNames[idx]
		}
	}
	return formatReal(spec, spec.Default)
}

// formatWire renders one parameter's wire value in the parameter's own units,
// or the selected option name for a list parameter.
func formatWire(spec ParamSpec, wire float64) string {
	if spec.Steps > 0 {
		if idx, err := spec.ValueToOption(wire); err == nil && idx < len(spec.StepNames) {
			return spec.StepNames[idx]
		}
		return strconv.FormatFloat(wire, 'g', -1, 64)
	}
	if real, err := spec.Denormalize(wire); err == nil {
		return formatReal(spec, real)
	}
	return strconv.FormatFloat(wire, 'g', -1, 64)
}

// formatReal rounds a screen value for its unit and appends the unit.
func formatReal(spec ParamSpec, real float64) string {
	rounded := roundForDisplay(real, spec.Units)
	text := strconv.FormatFloat(rounded, 'f', -1, 64)
	if spec.Units != "" {
		return text + " " + spec.Units
	}
	return text
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
