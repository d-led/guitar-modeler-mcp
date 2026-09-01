// Package cardchain renders a shared, wrap-safe signal-chain visualisation for
// the setup cards and the rig report: each slot is a numbered badge with the
// number in a circle, serial slots are joined by arrows, and a parallel
// junction draws its branches stacked vertically between a split and a merge
// marker. The layout wraps horizontally and never forces the page wider than
// its container.
package cardchain

import (
	"fmt"
	"html"
	"strings"
)

// Step is one slot in a signal chain. Slot is the 1-based position on the
// device grid; Module is a slot label such as "AMP" (empty for free-grid
// slots); Effect is the model in that slot (empty renders "empty"); Off dims
// the slot. When Branches is non-empty the step is a parallel junction: the
// signal splits into the listed branches and merges again afterwards, and the
// step's own Slot/Module/Effect are ignored.
type Step struct {
	Slot   int
	Module string
	Effect string
	Off    bool
	// Branches turns the step into a parallel junction.
	Branches []Branch
}

// Branch is one parallel path of a junction: an optional label ("A", "B") and
// the path's own serial steps.
type Branch struct {
	Label string
	Steps []Step
}

// CSS is the stylesheet for the chain visualisation. Include it once in the
// card's <style> block.
const CSS = `.chain{display:flex;flex-wrap:wrap;align-items:center;gap:.3rem .5rem;margin:0 0 1.25rem}
.chain .slot{display:inline-flex;align-items:center;gap:.4rem;background:#f4f4f4;border:1px solid #ddd;border-radius:6px;padding:.2rem .55rem;font-size:.85em;min-width:0}
.chain .slotno{font-weight:700;color:#fff;background:#6b6b73;border-radius:50%;min-width:1.45em;height:1.45em;line-height:1.45em;text-align:center;flex:none;display:inline-flex;align-items:center;justify-content:center}
.chain .name{overflow-wrap:anywhere}
.chain .arrow{color:#aaa;flex:none}
.chain .off{opacity:.55}
.chain .par{display:flex;flex-direction:column;gap:.3rem;border:1px solid #ddd;border-radius:8px;padding:.4rem .5rem;background:#fafafa}
.chain .branch{display:flex;flex-wrap:wrap;align-items:center;gap:.3rem .5rem}
.chain .parlabel{font-size:.75em;font-weight:700;color:#888;min-width:1.1em;flex:none}
.chain .mark{color:#bbb;flex:none;font-size:1.1em;line-height:1}
.slotbadge{display:inline-flex;align-items:center;justify-content:center;min-width:1.45em;height:1.45em;border-radius:50%;background:#6b6b73;color:#fff;font-weight:700;font-size:.8em;margin-right:.45em;vertical-align:middle}`

// headCSS is the shared setup-card stylesheet preamble, before the chain CSS.
// Every device backend's setup card uses it so the cards render consistently.
const headCSS = `body{font-family:system-ui,-apple-system,sans-serif;max-width:720px;margin:2rem auto;padding:0 1rem;color:#1a1a1a}
h1{margin-bottom:.25rem}h2{font-size:1rem;color:#555;margin-top:0}
table{width:100%;border-collapse:collapse;margin-bottom:1rem}
td,th{border-bottom:1px solid #e2e2e2;padding:.45rem .5rem;text-align:left;vertical-align:top}
.module{font-weight:600;white-space:nowrap}
.effect{font-weight:600}`

// Head writes the shared <head>, <style> preamble and opening <body> of a
// setup card: the escaped title, the common card styles, extraCSS (appended
// inside the style block), the chain CSS, and the opening <body>. Callers
// write the card body and closing tags themselves.
func Head(b *strings.Builder, title, extraCSS string) {
	b.WriteString("<!doctype html><html><head><meta charset=\"utf-8\">")
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(title))
	b.WriteString("</title><style>\n")
	b.WriteString(headCSS)
	if extraCSS != "" {
		b.WriteByte('\n')
		b.WriteString(extraCSS)
	}
	b.WriteString("\n" + CSS + "\n</style></head><body>")
}

// Render returns the numbered chain visualisation as an HTML fragment: serial
// steps joined by arrows, and parallel junctions drawn as stacked branches
// between split and merge markers.
func Render(steps []Step) string {
	if len(steps) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(`<div class="chain">`)
	for i, s := range steps {
		if i > 0 {
			b.WriteString(`<span class="arrow">→</span>`)
		}
		renderNode(&b, s)
	}
	b.WriteString(`</div>`)
	return b.String()
}

func renderNode(b *strings.Builder, s Step) {
	if len(s.Branches) > 0 {
		b.WriteString(`<span class="mark" aria-hidden="true">╫</span>`)
		b.WriteString(`<span class="par">`)
		for _, br := range s.Branches {
			b.WriteString(`<span class="branch">`)
			if br.Label != "" {
				fmt.Fprintf(b, `<span class="parlabel">%s</span>`, html.EscapeString(br.Label))
			}
			for j, st := range br.Steps {
				if j > 0 {
					b.WriteString(`<span class="arrow">→</span>`)
				}
				renderPill(b, st)
			}
			b.WriteString(`</span>`)
		}
		b.WriteString(`</span>`)
		b.WriteString(`<span class="mark" aria-hidden="true">╫</span>`)
		return
	}
	renderPill(b, s)
}

func renderPill(b *strings.Builder, s Step) {
	label := s.Module
	if s.Effect != "" {
		if label != "" {
			label += ": "
		}
		label += s.Effect
	}
	if label == "" {
		label = "empty"
	}
	class := "slot"
	if s.Off {
		class += " off"
	}
	fmt.Fprintf(b, `<span class="%s"><span class="slotno">%d</span><span class="name">%s</span></span>`,
		class, s.Slot, html.EscapeString(label))
}
