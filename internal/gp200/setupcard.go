package gp200

import (
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/d-led/guitar-modeler-mcp/internal/cardchain"
)

// ParamDesc is one editable parameter of a block: its display name, the value
// the preset sets, the resting default, and the option display names when the
// parameter is a switch/combox (nil for a plain knob).
type ParamDesc struct {
	Name    string
	Value   float32
	Default float32
	Options []string
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
		})
	}
	return out
}

// changed reports whether a parameter deviates from its resting default.
func changed(p ParamDesc) bool {
	return p.Value != p.Default
}

// formatParam renders a parameter value: the option name for a switch/combox,
// otherwise the number without float noise.
func formatParam(p ParamDesc) string {
	if len(p.Options) > 0 {
		i := int(p.Value)
		if i >= 0 && i < len(p.Options) {
			return p.Options[i]
		}
	}
	return strconv.FormatFloat(float64(p.Value), 'f', -1, 32)
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
.switch{color:#0a7d3c;font-weight:600;font-size:.85em}
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

	if foot := footswitchTable(p); foot != "" {
		b.WriteString("<h2>Footswitches</h2>")
		b.WriteString(foot)
	}
	if exp := expTable(p); exp != "" {
		b.WriteString("<h2>Expression pedals</h2>")
		b.WriteString(exp)
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
	class := ""
	if !d.Enabled {
		state = "OFF"
		class = " class=\"off\""
	}
	fmt.Fprintf(b, "<table><tr><th class=\"module\"%s><span class=\"slotbadge\">%d</span>%s</th><th>%s", class, slot, html.EscapeString(d.Module), state)
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
		val := html.EscapeString(formatParam(pd))
		if changed(pd) {
			fmt.Fprintf(b, "<span class=\"hl\">%s: %s</span>", name, val)
		} else {
			fmt.Fprintf(b, "%s: %s", name, val)
		}
	}
	b.WriteString("</td><td></td></tr></table>")
}

// footswitchTable renders the assigned CTRL footswitches, or "" when none are
// assigned.
func footswitchTable(p Preset) string {
	var rows []string
	for _, c := range p.Ctrl {
		if c.BlockMask == 0 {
			continue
		}
		var blocks []string
		for bit := 0; bit <= 11; bit++ {
			if c.BlockMask&(1<<uint(bit)) == 0 {
				continue
			}
			if bit == 11 {
				blocks = append(blocks, "FX LOOP")
			} else {
				blocks = append(blocks, ModuleForBlock(bit))
			}
		}
		rows = append(rows, fmt.Sprintf("<tr><td class=\"module\">CTRL %d</td><td>%s (%s)</td></tr>", c.Index+1, html.EscapeString(strings.Join(blocks, " + ")), onOffLabel(c.State)))
	}
	if len(rows) == 0 {
		return ""
	}
	return "<table>" + strings.Join(rows, "") + "</table>"
}

// onOffLabel renders a footswitch's saved toggle position as a word.
func onOffLabel(state uint8) string {
	if state == 1 {
		return "on"
	}
	return "off"
}

// expPageNames labels the three EXP pages (0 = EXP1 Mode A, 1 = EXP1 Mode B,
// 2 = EXP2).
var expPageNames = []string{"EXP1 A", "EXP1 B", "EXP2"}

// expTable renders the expression-pedal assignments, or "" when none target a
// block.
func expTable(p Preset) string {
	var rows []string
	for _, e := range p.Exp {
		if e.Block < 0 || e.Block > 10 {
			continue
		}
		target := ModuleForBlock(e.Block)
		param := expParamName(p, e.Block, e.ParamIndex)
		rows = append(rows, fmt.Sprintf("<tr><td class=\"module\">%s P%d</td><td>%s → %s (%s–%s)</td></tr>",
			expPageNames[e.Page], e.Item+1, html.EscapeString(target), html.EscapeString(param),
			strconv.FormatFloat(float64(e.Min), 'f', -1, 32), strconv.FormatFloat(float64(e.Max), 'f', -1, 32)))
	}
	if len(rows) == 0 {
		return ""
	}
	return "<table>" + strings.Join(rows, "") + "</table>"
}

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
