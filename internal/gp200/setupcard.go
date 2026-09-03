package gp200

import (
	"fmt"
	"html"
	"strings"

	"github.com/d-led/guitar-modeler-mcp/internal/cardchain"
)

// ParamDesc is one editable parameter of a block, with its raw device value
// and its resting/default value.
type ParamDesc struct {
	Name    string
	Value   any
	Default any
}

// ModuleDesc describes one block of a preset for display: which effect it
// holds, whether it is on, and its named parameter values.
type ModuleDesc struct {
	Module     string
	Effect     string
	InspiredBy string
	Enabled    bool
	Params     []ParamDesc
}

// Describe flattens a preset into a display-ready list of block descriptions in
// playback (routing) order, resolving effect codes to names and parameter
// positions to the editor's labels.
func Describe(p Preset) []ModuleDesc {
	desc := make([]ModuleDesc, 0, effectBlockCount)
	for i := 0; i < effectBlockCount; i++ {
		slot := int(p.Routing[i])
		if slot < 0 || slot >= effectBlockCount {
			slot = i
		}
		desc = append(desc, describeBlock(p.Blocks[slot], slot))
	}
	return desc
}

func describeBlock(blk Block, slot int) ModuleDesc {
	name := EffectName(blk.EffectID)
	return ModuleDesc{
		Module:     ModuleForBlock(slot),
		Effect:     name,
		InspiredBy: InspiredBy(name),
		Enabled:    blk.Enabled,
		Params:     describeParams(blk),
	}
}

func describeParams(blk Block) []ParamDesc {
	names := ParamNames(blk.EffectID)
	defaults := DefaultParams(blk.EffectID)
	out := make([]ParamDesc, 0, len(names))
	for i, name := range names {
		if name == "" {
			continue
		}
		out = append(out, ParamDesc{Name: name, Value: blk.Params[i], Default: defaults[i]})
	}
	return out
}

// changed reports whether a parameter deviates from its resting default.
func changed(p ParamDesc) bool {
	return fmt.Sprintf("%v", p.Value) != fmt.Sprintf("%v", p.Default)
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
.params{color:#444;font-size:.85em}
.hl{color:#2563eb;font-weight:600}
` + cardchain.CSS + `
</style></head><body>`)
	fmt.Fprintf(&b, "<h1>%s</h1><h2>%s — setup card</h2>", html.EscapeString(p.PatchName), html.EscapeString(m.Display))

	if stored, truncated := StoredName(p.PatchName); truncated {
		fmt.Fprintf(&b, "<p class=\"inspired\">Note: the device stores preset names up to %d characters; this preset reads as %q on the unit.</p>", NameLimit, html.EscapeString(stored))
	}

	desc := Describe(p)
	b.WriteString(chainHint(desc))

	for i, d := range desc {
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

	if foot := footswitchTable(p); foot != "" {
		b.WriteString("<h2>Footswitches</h2>")
		b.WriteString(foot)
	}

	b.WriteString("</body></html>")
	return b.String()
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
		rows = append(rows, fmt.Sprintf("<tr><td class=\"module\">CTRL %d</td><td>%s</td></tr>", c.Index+1, html.EscapeString(strings.Join(blocks, " + "))))
	}
	if len(rows) == 0 {
		return ""
	}
	return "<table>" + strings.Join(rows, "") + "</table>"
}
