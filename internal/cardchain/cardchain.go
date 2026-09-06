// Package cardchain renders a shared, wrap-safe signal-chain visualisation for
// the setup cards and the rig report: each slot is a numbered badge with the
// number in a circle, serial slots are joined by arrows, and a parallel
// junction draws its branches stacked vertically between a split and a merge
// marker. The layout wraps horizontally and never forces the page wider than
// its container.
package cardchain

import (
	"embed"
	"html/template"
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

//go:embed css.tmpl css-dark.tmpl head.tmpl chain.tmpl
var templateFiles embed.FS

// CSS is the light-theme chain visualisation stylesheet. Every colour sits
// behind a custom property declared on :root, so the palette has a single,
// overridable source. The printable setup cards include just CSS; hosts that
// render on a dark canvas (the rig report) append DarkSchemeCSS after it.
var CSS = embedded("css.tmpl")

// DarkSchemeCSS flips the chain palette for hosts that render on a dark canvas.
// It only overrides the custom properties declared in CSS, so it must appear
// after CSS in the same <style> block.
var DarkSchemeCSS = embedded("css-dark.tmpl")

// embedded reads one of the embedded template/CSS files, trimming a single
// trailing newline so the strings embed byte-for-byte like the constants they
// replace.
func embedded(name string) string {
	b, err := templateFiles.ReadFile(name)
	if err != nil {
		panic(err)
	}
	return strings.TrimSuffix(string(b), "\n")
}

var (
	headTmpl  = template.Must(template.New("head").Parse(embedded("head.tmpl")))
	chainTmpl = template.Must(template.New("chain").Parse(embedded("chain.tmpl")))
)

// Label returns the step's display label: module and effect joined with ": ",
// or "empty" for a free slot.
func (s Step) Label() string {
	label := s.Module
	if s.Effect != "" {
		if label != "" {
			label += ": "
		}
		label += s.Effect
	}
	if label == "" {
		return "empty"
	}
	return label
}

// Head writes the shared <head>, <style> preamble and opening <body> of a
// setup card: the escaped title, the common card styles, extraCSS (appended
// inside the style block), and the chain CSS.
func Head(b *strings.Builder, title, extraCSS string) {
	if err := headTmpl.Execute(b, map[string]any{
		"Title":    title,
		"ExtraCSS": template.CSS(extraCSS), // #nosec G203 -- trusted package CSS
		"CSS":      template.CSS(CSS),      // #nosec G203 -- trusted package CSS
	}); err != nil {
		panic(err)
	}
}

// Render returns the numbered chain visualisation as an HTML fragment: serial
// steps joined by arrows, and parallel junctions drawn as stacked branches
// between split and merge markers.
func Render(steps []Step) string {
	if len(steps) == 0 {
		return ""
	}
	var b strings.Builder
	if err := chainTmpl.Execute(&b, map[string]any{"Steps": steps}); err != nil {
		panic(err)
	}
	return b.String()
}
