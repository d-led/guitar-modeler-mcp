package mooer

import (
	"fmt"
	"html"
	"strings"

	"github.com/d-led/guitar-modeler-mcp/internal/cardchain"
)

// ParamDesc is one editable parameter of a module, with its raw device value.
type ParamDesc struct {
	Name  string
	Value any
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

// Describe flattens a preset into a display-ready list of module descriptions,
// resolving effect_type indices to names via the model's catalog.
func Describe(p Preset, m Model) []ModuleDesc {
	return []ModuleDesc{
		describeModule("FX", "fx", p.FX.Enabled, p.FX.Type, m, fxParams(p.FX)),
		describeModule("DS/OD", "od", p.Drive.Enabled, p.Drive.Type, m, driveParams(p.Drive)),
		describeModule("AMP", "amp", p.Amp.Enabled, p.Amp.Type, m, ampParams(p.Amp)),
		describeModule("CAB", "cab", p.Cab.Enabled, p.Cab.Type, m, cabParams(p.Cab)),
		describeModule("NS", "ns", p.NoiseGate.Enabled, p.NoiseGate.Type, m, nsParams(p.NoiseGate)),
		describeModule("EQ", "eq", p.EQ.Enabled, p.EQ.Type, m, eqParams(p.EQ)),
		describeModule("MOD", "mod", p.Mod.Enabled, p.Mod.Type, m, modParams(p.Mod)),
		describeModule("DELAY", "delay", p.Delay.Enabled, p.Delay.Type, m, delayParams(p.Delay)),
		describeModule("REVERB", "reverb", p.Reverb.Enabled, p.Reverb.Type, m, reverbParams(p.Reverb)),
	}
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
	return []ParamDesc{{"Q", f.Q}, {"Position", f.Position}, {"Peak", f.Peak}, {"Level", f.Level}}
}
func driveParams(d Drive) []ParamDesc {
	return []ParamDesc{{"Volume", d.Volume}, {"Tone", d.Tone}, {"Gain", d.Gain}}
}
func ampParams(a Amp) []ParamDesc {
	return []ParamDesc{{"Gain", a.Gain}, {"Bass", a.Bass}, {"Mid", a.Mid}, {"Treble", a.Treble}, {"Presence", a.Presence}, {"Master", a.Master}}
}
func cabParams(c Cab) []ParamDesc {
	return []ParamDesc{{"Mic", c.Mic}, {"Center", c.Center}, {"Distance", c.Distance}, {"Tube", c.Tube}}
}
func nsParams(n NoiseGate) []ParamDesc {
	return []ParamDesc{{"Attack", n.Attack}, {"Release", n.Release}, {"Threshold", n.Threshold}}
}
func eqParams(e EQ) []ParamDesc {
	out := make([]ParamDesc, 0, 12)
	for i, v := range e.Bands {
		out = append(out, ParamDesc{fmt.Sprintf("Band %d", i+1), v})
	}
	for i, v := range e.BandsExtra {
		out = append(out, ParamDesc{fmt.Sprintf("Band %d", i+7), v})
	}
	return out
}
func modParams(m Mod) []ParamDesc {
	return []ParamDesc{{"Rate", m.Rate}, {"Level", m.Level}, {"Depth", m.Depth}, {"Param 4", m.Param4}, {"Param 5", m.Param5}}
}
func delayParams(d Delay) []ParamDesc {
	return []ParamDesc{{"Level", d.Level}, {"Feedback", d.Feedback}, {"Time (ms)", d.TimeMS}, {"Subdivision", d.Subdivision}, {"Param 5", d.Param5}, {"Param 6", d.Param6}}
}
func reverbParams(r Reverb) []ParamDesc {
	return []ParamDesc{{"Pre-Delay", r.PreDelay}, {"Level", r.Level}, {"Decay", r.Decay}, {"Tone", r.Tone}}
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
// companion report for devices that can also write a .mo file.
func SetupCardHTML(m Model, p Preset) string {
	var b strings.Builder
	b.WriteString("<!doctype html><html><head><meta charset=\"utf-8\">")
	fmt.Fprintf(&b, "<title>%s — %s</title>", html.EscapeString(p.Name), html.EscapeString(m.Display))
	b.WriteString(`<style>
body{font-family:system-ui,-apple-system,sans-serif;max-width:760px;margin:2rem auto;padding:0 1rem;color:#1a1a1a}
h1{margin-bottom:.25rem}h2{font-size:1rem;color:#555;margin-top:0}
table{width:100%;border-collapse:collapse;margin-bottom:1.25rem}
td,th{border-bottom:1px solid #e2e2e2;padding:.45rem .5rem;text-align:left;vertical-align:top}
.module{font-weight:600;white-space:nowrap}
.effect{font-weight:600}.off{color:#999}.inspired{color:#666;font-size:.85em}
.params{color:#444;font-size:.85em}
` + cardchain.CSS + `
</style></head><body>`)
	fmt.Fprintf(&b, "<h1>%s</h1><h2>%s — setup card</h2>", html.EscapeString(p.Name), html.EscapeString(m.Display))

	b.WriteString(chainHint(Describe(p, m)))

	for _, d := range Describe(p, m) {
		state := "ON"
		class := ""
		if !d.Enabled {
			state = "OFF"
			class = " class=\"off\""
		}
		fmt.Fprintf(&b, "<table><tr><th class=\"module\"%s>%s</th><th>%s</th></tr>", class, html.EscapeString(d.Module), state)
		fmt.Fprintf(&b, "<tr><td class=\"effect\">%s</td><td>", html.EscapeString(d.Effect))
		if d.InspiredBy != "" {
			fmt.Fprintf(&b, "<span class=\"inspired\">based on %s</span>", html.EscapeString(d.InspiredBy))
		}
		fmt.Fprintf(&b, "</td></tr>")
		if d.Enabled {
			b.WriteString("<tr><td class=\"params\">")
			parts := make([]string, 0, len(d.Params))
			for _, p := range d.Params {
				parts = append(parts, fmt.Sprintf("%s: %v", html.EscapeString(p.Name), p.Value))
			}
			b.WriteString(html.EscapeString(strings.Join(parts, " · ")))
			b.WriteString("</td><td></td></tr>")
		}
		b.WriteString("</table>")
	}

	b.WriteString("<p class=\"inspired\">Values are the device's raw parameter values (0&ndash;255 unless noted).</p>")
	b.WriteString("</body></html>")
	return b.String()
}
