package mooer

import (
	"fmt"
	"html"
	"strings"

	"github.com/d-led/guitar-modeler-mcp/internal/cardchain"
)

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
		switch module {
		case "fx":
			desc = append(desc, describeModule("FX", module, p.FX.Enabled, p.FX.Type, m, fxParams(p.FX)))
		case "od":
			desc = append(desc, describeModule("DS/OD", module, p.Drive.Enabled, p.Drive.Type, m, driveParams(p.Drive)))
		case "amp":
			desc = append(desc, describeModule("AMP", module, p.Amp.Enabled, p.Amp.Type, m, ampParams(p.Amp)))
		case "cab":
			desc = append(desc, describeModule("CAB", module, p.Cab.Enabled, p.Cab.Type, m, cabParams(p.Cab)))
		case "ns":
			desc = append(desc, describeModule("NS", module, p.NoiseGate.Enabled, p.NoiseGate.Type, m, nsParams(p.NoiseGate)))
		case "eq":
			desc = append(desc, describeModule("EQ", module, p.EQ.Enabled, p.EQ.Type, m, eqParams(p.EQ)))
		case "mod":
			desc = append(desc, describeModule("MOD", module, p.Mod.Enabled, p.Mod.Type, m, modParams(p.Mod)))
		case "delay":
			desc = append(desc, describeModule("DELAY", module, p.Delay.Enabled, p.Delay.Type, m, delayParams(p.Delay)))
		case "reverb":
			desc = append(desc, describeModule("REVERB", module, p.Reverb.Enabled, p.Reverb.Type, m, reverbParams(p.Reverb)))
		}
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
.hl{color:#2563eb;font-weight:600}
` + cardchain.CSS + `
</style></head><body>`)
	fmt.Fprintf(&b, "<h1>%s</h1><h2>%s — setup card</h2>", html.EscapeString(p.Name), html.EscapeString(m.Display))

	if stored, truncated := StoredName(p.Name); truncated {
		fmt.Fprintf(&b, "<p class=\"inspired\">Note: the device stores preset names up to %d characters; this preset reads as %q on the unit.</p>", NameLimit, html.EscapeString(stored))
	}

	b.WriteString(chainHint(Describe(p, m)))

	for i, d := range Describe(p, m) {
		state := "ON"
		class := ""
		if !d.Enabled {
			state = "OFF"
			class = " class=\"off\""
		}
		fmt.Fprintf(&b, "<table><tr><th class=\"module\"%s><span class=\"slotbadge\">%d</span>%s</th><th>%s</th></tr>", class, i+1, html.EscapeString(d.Module), state)
		fmt.Fprintf(&b, "<tr><td class=\"effect\">%s</td><td>", html.EscapeString(d.Effect))
		if d.InspiredBy != "" {
			fmt.Fprintf(&b, "<span class=\"inspired\">based on %s</span>", html.EscapeString(d.InspiredBy))
		}
		fmt.Fprintf(&b, "</td></tr>")
		if d.Enabled {
			b.WriteString("<tr><td class=\"params\">")
			for i, p := range d.Params {
				if i > 0 {
					b.WriteString(" · ")
				}
				name := html.EscapeString(p.Name)
				val := html.EscapeString(fmt.Sprintf("%v", p.Value))
				if changed(p) {
					fmt.Fprintf(&b, "<span class=\"hl\">%s: %s</span>", name, val)
				} else {
					fmt.Fprintf(&b, "%s: %s", name, val)
				}
			}
			b.WriteString("</td><td></td></tr>")
		}
		b.WriteString("</table>")
	}

	b.WriteString("<p class=\"inspired\">Values are the device's raw parameter values (0&ndash;255 unless noted).</p>")
	b.WriteString("</body></html>")
	return b.String()
}
