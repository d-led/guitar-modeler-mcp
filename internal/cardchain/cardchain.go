// Package cardchain renders a shared, wrap-safe signal-chain visualisation for
// the setup cards: each slot is a numbered pill, joined by arrows, so the
// chosen models map to their slot positions at a glance. The layout wraps
// horizontally and never forces the page wider than its container.
package cardchain

import (
	"fmt"
	"html"
	"strings"
)

// Step is one slot in a signal chain.
type Step struct {
	Slot   int    // 1-based slot position
	Module string // module label, e.g. "AMP" (empty for free-grid slots)
	Effect string // the model in that slot (empty = empty/off)
}

// CSS is the stylesheet for the chain visualisation. Include it once in the
// card's <style> block.
const CSS = `.chain{display:flex;flex-wrap:wrap;align-items:center;gap:.3rem .5rem;margin:0 0 1.25rem}
.chain .slot{display:inline-flex;align-items:center;gap:.4rem;background:#f4f4f4;border:1px solid #ddd;border-radius:6px;padding:.2rem .55rem;font-size:.85em;min-width:0}
.chain .slotno{font-weight:700;color:#777;background:#e7e7e7;border-radius:4px;padding:0 .35rem;min-width:1.3em;text-align:center;flex:none}
.chain .name{overflow-wrap:anywhere}
.chain .arrow{color:#aaa;flex:none}
.chain .off{color:#999}`

// Render returns the slot-numbered chain visualisation as an HTML fragment.
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
		fmt.Fprintf(&b, `<span class="slot"><span class="slotno">%d</span><span class="name">%s</span></span>`,
			s.Slot, html.EscapeString(label))
	}
	b.WriteString(`</div>`)
	return b.String()
}
