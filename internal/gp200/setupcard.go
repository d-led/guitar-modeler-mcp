package gp200

import (
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/d-led/guitar-modeler-mcp/internal/cardchain"
)

// ParamDesc is one editable parameter of a block: its display name, the value
// the preset sets, the resting default, the option display names when the
// parameter is a switch/combox (nil for a plain knob), and its display unit
// (Hz, ms, …) when one applies.
type ParamDesc struct {
	Name    string
	Value   float32
	Default float32
	Options []string
	Unit    string
}

// ModuleDesc describes one block of a preset for display: which effect it
// holds, whether it is on, which CTRL footswitch toggles it (when any), and its
// named parameter values.
type ModuleDesc struct {
	Module     string
	Effect     string
	InspiredBy string
	Enabled    bool
	Switch     string
	Params     []ParamDesc
}

// Describe flattens a preset into a display-ready list of block descriptions in
// playback (routing) order, resolving effect codes to names and parameter
// positions to the editor's labels, and annotating footswitch-controlled blocks.
func Describe(p Preset) []ModuleDesc {
	switchFor := ctrlSwitchMap(p.Ctrl)
	desc := make([]ModuleDesc, 0, effectBlockCount)
	for i := 0; i < effectBlockCount; i++ {
		slot := int(p.Routing[i])
		if slot < 0 || slot >= effectBlockCount {
			slot = i
		}
		desc = append(desc, describeBlock(p.Blocks[slot], slot, switchFor[slot]))
	}
	return desc
}

// ctrlSwitchMap returns, per physical block, the CTRL footswitches that toggle
// it (e.g. "CTRL 2", or "CTRL 1 + CTRL 3" when several do).
func ctrlSwitchMap(ctrl [8]CtrlAssignment) [effectBlockCount]string {
	var out [effectBlockCount]string
	for _, c := range ctrl {
		if c.BlockMask == 0 {
			continue
		}
		for bit := 0; bit < effectBlockCount; bit++ {
			if c.BlockMask&(1<<uint(bit)) == 0 {
				continue
			}
			if out[bit] != "" {
				out[bit] += " + "
			}
			out[bit] += fmt.Sprintf("CTRL %d", c.Index+1)
		}
	}
	return out
}

func describeBlock(blk Block, slot int, switchName string) ModuleDesc {
	name := EffectName(blk.EffectID)
	return ModuleDesc{
		Module:     ModuleForBlock(slot),
		Effect:     name,
		InspiredBy: InspiredBy(name),
		Enabled:    blk.Enabled,
		Switch:     switchName,
		Params:     describeParams(blk),
	}
}

func describeParams(blk Block) []ParamDesc {
	defs := Params(blk.EffectID)
	defaults := DefaultParams(blk.EffectID)
	out := make([]ParamDesc, 0, len(defs))
	for _, def := range defs {
		out = append(out, ParamDesc{
			Name:    def.Name,
			Value:   blk.Params[def.Index],
			Default: defaults[def.Index],
			Options: def.Options,
			Unit:    paramUnit(def),
		})
	}
	return out
}

// paramUnit infers a parameter's display unit from its name and range, so the
// card reads like the editor's knobs. It returns "" for a dimensionless knob,
// "?" when the parameter carries a physical quantity whose unit is ambiguous
// (e.g. a 0..100 "Rate" or "Pre Delay" that could be Hz, ms or a percentage),
// and the unit itself otherwise.
func paramUnit(def ParamDef) string {
	name := strings.ToLower(def.Name)
	switch {
	case strings.Contains(name, "rate") || strings.Contains(name, "speed"):
		if def.Max <= 20 {
			return "Hz" // chorus/flanger/phaser rate, 0.1..10 Hz
		}
		return "?"
	case strings.Contains(name, "time"):
		return "ms" // delay time, 20..4000 ms
	case strings.Contains(name, "pre delay"):
		return "?"
	case strings.Contains(name, "cut") && def.Max > 1000:
		return "Hz" // cabinet low/high cut
	}
	return ""
}

// changed reports whether a parameter deviates from its resting default.
func changed(p ParamDesc) bool {
	return p.Value != p.Default
}

// formatParam renders a parameter value: the option name for a switch/combox,
// otherwise the number without float noise. The unit is appended separately so
// it can be styled faintly.
func formatParam(p ParamDesc) string {
	if len(p.Options) > 0 {
		i := int(p.Value)
		if i >= 0 && i < len(p.Options) {
			return p.Options[i]
		}
	}
	return strconv.FormatFloat(float64(p.Value), 'f', -1, 32)
}

// formatParamUnit renders a parameter's unit as a faint suffix, or "" when the
// parameter is dimensionless.
func formatParamUnit(p ParamDesc) string {
	if p.Unit == "" {
		return ""
	}
	return " <span class=\"unit\">" + html.EscapeString(p.Unit) + "</span>"
}

// chainHint renders the eleven fixed blocks in playback order, so the models
// are attributable to their slot positions at a glance.
func chainHint(desc []ModuleDesc) string {
	steps := make([]cardchain.Step, 0, len(desc))
	for i, d := range desc {
		steps = append(steps, cardchain.Step{Slot: i + 1, Module: d.Module, Effect: d.Effect})
	}
	return cardchain.Render(steps)
}

// SetupCardHTML renders a printable setup card for a preset. It is the
// companion report for the .prst file the design tool writes.
func SetupCardHTML(m Model, p Preset) string {
	var b strings.Builder
	b.WriteString("<!doctype html><html><head><meta charset=\"utf-8\">")
	fmt.Fprintf(&b, "<title>%s — %s</title>", html.EscapeString(p.PatchName), html.EscapeString(m.Display))
	b.WriteString(`<style>
body{font-family:system-ui,-apple-system,sans-serif;max-width:760px;margin:2rem auto;padding:0 1rem;color:#1a1a1a}
h1{margin-bottom:.25rem}h2{font-size:1rem;color:#555;margin-top:0}
table{width:100%;border-collapse:collapse;margin-bottom:1.25rem}
td,th{border-bottom:1px solid #e2e2e2;padding:.45rem .5rem;text-align:left;vertical-align:top}
.module{font-weight:600;white-space:nowrap}
.effect{font-weight:600}.off{color:#999}.inspired{color:#666;font-size:.85em}
.params{color:#444;font-size:.85em;font-variant-numeric:tabular-nums}
.hl{color:#2563eb;font-weight:600}
.unit{color:#9b9b9b;font-size:.8em}
.switch{color:#0a7d3c;font-weight:600;font-size:.85em}
.buttons{display:grid;grid-template-columns:repeat(auto-fill,minmax(88px,1fr));gap:10px;margin-bottom:12px}
.btn{border:1px solid #e3e3e8;border-radius:12px;padding:10px 8px;text-align:center}
.btn .num{display:inline-block;min-width:22px;height:22px;line-height:22px;border-radius:999px;background:#e8e8ed;font-size:.74em;font-weight:700;margin-bottom:6px}
.btn .mod{font-weight:700;font-size:.9em;overflow-wrap:anywhere}
.btn .op{font-size:.76em;color:#888}
.btn.on{background:#34c75918;border-color:#34c75955}
.btn.on .num{background:#34c759;color:#fff}
.btn.off .num{background:#8e8e93;color:#fff}
.btn.empty{opacity:.42}
.pedals{margin-top:12px;font-size:.9em;display:flex;flex-direction:column;gap:6px}
.pedal{display:flex;flex-wrap:wrap;gap:6px;align-items:center}
.pedal .name{font-weight:700}
.chip{padding:2px 8px;border-radius:999px;font-size:.85em;background:#e8e8ed}
` + cardchain.CSS + `
</style></head><body>`)
	fmt.Fprintf(&b, "<h1>%s</h1><h2>%s — setup card</h2>", html.EscapeString(p.PatchName), html.EscapeString(m.Display))

	if stored, truncated := StoredName(p.PatchName); truncated {
		fmt.Fprintf(&b, "<p class=\"inspired\">Note: the device stores preset names up to %d characters; this preset reads as %q on the unit.</p>", NameLimit, html.EscapeString(stored))
	}

	desc := Describe(p)
	b.WriteString(chainHint(desc))

	for i, d := range desc {
		writeBlockCard(&b, d, i+1)
	}

	if foot := hardwareBoxes(p); foot != "" {
		b.WriteString("<h2>Hardware</h2>")
		b.WriteString(foot)
	}

	b.WriteString("</body></html>")
	return b.String()
}

// writeBlockCard renders one block as a table: the slot badge, module, on/off
// state, any CTRL footswitch that toggles it, its effect model and the hardware
// it emulates. Parameter values are listed only when there is something to dial
// in — the block is on, or a footswitch can switch it on — so a block that is
// simply off carries no "set me" highlights.
func writeBlockCard(b *strings.Builder, d ModuleDesc, slot int) {
	state := "ON"
	modClass := "module"
	if !d.Enabled {
		state = "OFF"
		modClass += " off"
	}
	fmt.Fprintf(b, "<table><tr><th class=\"%s\"><span class=\"slotbadge\">%d</span>%s</th><th>%s", modClass, slot, html.EscapeString(d.Module), state)
	if d.Switch != "" {
		fmt.Fprintf(b, " <span class=\"switch\">%s</span>", html.EscapeString(d.Switch))
	}
	b.WriteString("</th></tr>")
	fmt.Fprintf(b, "<tr><td class=\"effect\">%s</td><td>", html.EscapeString(d.Effect))
	if d.InspiredBy != "" {
		fmt.Fprintf(b, "<span class=\"inspired\">based on %s</span>", html.EscapeString(d.InspiredBy))
	}
	b.WriteString("</td></tr>")
	if !d.Enabled && d.Switch == "" {
		b.WriteString("</table>")
		return
	}
	b.WriteString("<tr><td class=\"params\">")
	for j, pd := range d.Params {
		if j > 0 {
			b.WriteString(" · ")
		}
		name := html.EscapeString(pd.Name)
		val := html.EscapeString(formatParam(pd)) + formatParamUnit(pd)
		if changed(pd) {
			fmt.Fprintf(b, "<span class=\"hl\">%s: %s</span>", name, val)
		} else {
			fmt.Fprintf(b, "%s: %s", name, val)
		}
	}
	b.WriteString("</td><td></td></tr></table>")
}

// hardwareBoxes renders the CTRL footswitches as a grid of boxes and the EXP
// pedal assignments as chips, mirroring the HeadRush report's hardware section.
func hardwareBoxes(p Preset) string {
	var b strings.Builder
	b.WriteString("<div class=\"buttons\">")
	for _, c := range p.Ctrl {
		blocks := blockNames(c.BlockMask)
		cls := "btn"
		mod := "—"
		op := ""
		if len(blocks) > 0 {
			mod = strings.Join(blocks, " + ")
			if c.State == 1 {
				cls += " on"
				op = "on"
			} else {
				cls += " off"
				op = "off"
			}
		} else {
			cls += " empty"
		}
		fmt.Fprintf(&b, "<div class=\"%s\"><div class=\"num\">%d</div><div class=\"mod\">%s</div>", cls, c.Index+1, html.EscapeString(mod))
		if op != "" {
			fmt.Fprintf(&b, "<div class=\"op\">%s</div>", op)
		}
		b.WriteString("</div>")
	}
	b.WriteString("</div>")

	var pedals []string
	for _, e := range p.Exp {
		if e.Block < 0 || e.Block > 10 {
			continue
		}
		target := ModuleForBlock(e.Block)
		param := expParamName(p, e.Block, e.ParamIndex)
		pedals = append(pedals, fmt.Sprintf("<div class=\"pedal\"><span class=\"name\">%s P%d</span><span class=\"chip\">%s → %s (%s–%s)</span></div>",
			expPageNames[e.Page], e.Item+1, html.EscapeString(target), html.EscapeString(param),
			strconv.FormatFloat(float64(e.Min), 'f', -1, 32), strconv.FormatFloat(float64(e.Max), 'f', -1, 32)))
	}
	if len(pedals) > 0 {
		b.WriteString("<div class=\"pedals\">" + strings.Join(pedals, "") + "</div>")
	}
	return b.String()
}

// blockNames returns the block names set in a CTRL footswitch mask.
func blockNames(mask uint16) []string {
	var names []string
	for bit := 0; bit <= 11; bit++ {
		if mask&(1<<uint(bit)) == 0 {
			continue
		}
		if bit == 11 {
			names = append(names, "FX LOOP")
		} else {
			names = append(names, ModuleForBlock(bit))
		}
	}
	return names
}

// expPageNames labels the three EXP pages (0 = EXP1 Mode A, 1 = EXP1 Mode B,
// 2 = EXP2).
var expPageNames = []string{"EXP1 A", "EXP1 B", "EXP2"}

// expParamName resolves a target block's parameter name at a given index.
func expParamName(p Preset, block, index int) string {
	if index < 0 || index >= blockParamsCount {
		return fmt.Sprintf("param %d", index)
	}
	names := ParamNames(p.Blocks[block].EffectID)
	if name := names[index]; name != "" {
		return name
	}
	return fmt.Sprintf("param %d", index)
}
