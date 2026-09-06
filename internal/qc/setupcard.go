package qc

import (
	"embed"
	"fmt"
	"html/template"
	"math"
	"strconv"
	"strings"

	"github.com/d-led/guitar-modeler-mcp/internal/cardchain"
)

//go:embed css.tmpl html.tmpl
var cardFS embed.FS

var (
	cardCSS  = readCard("css.tmpl")
	cardTmpl = template.Must(template.New("qc-card").Parse(readCard("html.tmpl")))
)

func readCard(name string) string {
	b, err := cardFS.ReadFile(name)
	if err != nil {
		panic(err)
	}
	return string(b)
}

// Caveat is the honest framing of the Quad Cortex outputs, surfaced to agents
// in the tool descriptions and printed on every setup card: the HTML card is
// the instructions to dial the tone in by hand, the .pb is a reference
// archive for this tool (not a file the unit imports), and live transfer goes
// over USB via qcctl.
const Caveat = "The HTML card is the setup instructions; reproduce the tone " +
	"from it. The .pb is this tool's reference archive for saving and " +
	"reloading the tone — it is not a file the Quad Cortex imports, and " +
	"qcctl cannot upload it: qc_usb only recalls/dumps preset slots, " +
	"switches scenes and reads the firmware version. qc_design builds a " +
	"single-lane serial chain; split/parallel routing is not modelled yet."

// SetupCardHTML renders a self-contained, printable setup card for a decoded
// preset: the signal chain (in order), each block's name and the hardware it
// is based on, and every knob with its value — the values the preset sets
// explicitly, and the catalog defaults for the rest, so the whole tone can be
// reproduced by hand from the card alone. Note is optional prose for the card
// (why this tone, how to play it, the rest of the rig and hardware); it is
// printed at the bottom of the card and never written to the .pb archive.
func SetupCardHTML(cat *Catalog, preset *BinaryPreset, note string) string {
	var b strings.Builder
	if err := cardTmpl.Execute(&b, cardPage{
		Title:     preset.Name + " — Quad Cortex",
		H1:        preset.Name,
		H2:        "Neural DSP Quad Cortex — setup card",
		Author:    preset.AuthorName,
		VolumePan: fmt.Sprintf("volume %.3g · pan %.3g", preset.Volume, preset.Pan),
		CSS:       template.CSS(cardCSS),       // #nosec G203 -- trusted package CSS
		ChainCSS:  template.CSS(cardchain.CSS), // #nosec G203 -- trusted package CSS
		Rows:      rowsFor(cat, preset),
		Caveat:    Caveat,
		Note:      note,
	}); err != nil {
		panic(err)
	}
	return b.String()
}

// cardPage is the data for the Quad Cortex setup-card template.
type cardPage struct {
	Title     string
	H1, H2    string
	Author    string
	VolumePan string
	CSS       template.CSS
	ChainCSS  template.CSS
	Rows      []chainRow
	Caveat    string
	Note      string
}

// chainRow is one grid row: its screen number, chain hint and blocks.
type chainRow struct {
	Row    int
	Chain  template.HTML
	Blocks []blockCard
}

// blockCard is one grid block rendered as a table.
type blockCard struct {
	Slot   int
	Name   string
	Based  string
	Params []string
}

// rowsFor builds the grid rows with their block cards.
func rowsFor(cat *Catalog, preset *BinaryPreset) []chainRow {
	rows := make([]chainRow, 0, len(preset.Chains))
	for _, c := range preset.Chains {
		rows = append(rows, chainRow{Row: int(c.GetRow()) + 1, Chain: template.HTML(rowChain(cat, c)), Blocks: blocksFor(cat, c)}) // #nosec G203 -- trusted chain HTML
	}
	return rows
}

func blocksFor(cat *Catalog, c *Chain) []blockCard {
	out := make([]blockCard, 0, len(c.Models))
	for i, model := range c.Models {
		name := modelName(cat, model)
		based := ""
		var params []string
		if m, ok := cat.Model(int(model.GetHash())); ok {
			based = m.BasedOn
			params = blockParams(m, model)
		}
		out = append(out, blockCard{Slot: i + 1, Name: name, Based: based, Params: params})
	}
	return out
}

// rowChain renders one grid row as a slot-numbered chain: each model sits in
// its column position so it is attributable to a slot at a glance.
func rowChain(cat *Catalog, c *Chain) string {
	steps := make([]cardchain.Step, 0, len(c.Models))
	for i, model := range c.Models {
		steps = append(steps, cardchain.Step{Slot: i + 1, Effect: modelName(cat, model)})
	}
	return cardchain.Render(steps)
}

func modelName(cat *Catalog, model *Model) string {
	if m, ok := cat.Model(int(model.GetHash())); ok {
		return m.Name
	}
	return fmt.Sprintf("model %d", model.GetHash())
}

// paramKV is one knob with its formatted value and whether the preset set it
// explicitly (versus falling back to the catalog default).
type paramKV struct {
	name  string
	value string
	set   bool
}

// blockParamKVs returns every knob of a model with its value, in the model's
// parameter order. Knobs the preset sets explicitly carry their value; the
// rest carry the catalog default, so the card and the JSON view are both
// self-contained.
func blockParamKVs(m *ModelSpec, model *Model) []paramKV {
	wireByIndex := map[uint32]float64{}
	for _, p := range model.Params {
		if len(p.ParamValues) > 0 {
			wireByIndex[p.GetIndex()] = float64(p.ParamValues[0].GetFloatValue())
		}
	}
	var out []paramKV
	for i, spec := range m.Params {
		if !isKnob(spec) {
			continue
		}
		value := formatDefault(spec)
		set := false
		if wire, ok := wireByIndex[uint32(i)]; ok {
			value = formatWire(spec, wire)
			set = true
		}
		out = append(out, paramKV{name: spec.Name, value: value, set: set})
	}
	return out
}

// blockParams returns every knob as a "NAME: value" string, marking the
// catalog defaults, in the model's parameter order.
func blockParams(m *ModelSpec, model *Model) []string {
	kvs := blockParamKVs(m, model)
	out := make([]string, 0, len(kvs))
	for _, kv := range kvs {
		value := kv.value
		if !kv.set {
			value += " (default)"
		}
		out = append(out, kv.name+": "+value)
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
	if spec.isList() {
		if idx, err := spec.ValueToOption(spec.Default); err == nil && idx < len(spec.StepNames) {
			return spec.StepNames[idx]
		}
	}
	return formatReal(spec, spec.Default)
}

// formatWire renders one parameter's wire value in the parameter's own units,
// or the selected option name for a named list parameter.
func formatWire(spec ParamSpec, wire float64) string {
	if spec.isList() {
		if idx, err := spec.ValueToOption(wire); err == nil && idx < len(spec.StepNames) {
			return spec.StepNames[idx]
		}
		return formatWireRaw(wire)
	}
	if realVal, err := spec.Denormalize(wire); err == nil {
		return formatReal(spec, realVal)
	}
	return formatWireRaw(wire)
}

// formatWireRaw renders a raw 0..1 wire value cleanly for the rare parameter
// whose bounds are unmeasured and therefore cannot be converted to screen
// units. Three decimals suffice for a wire value, and the fixed-point format
// never leaks float-representation noise to the card.
func formatWireRaw(wire float64) string {
	rounded := math.Round(wire*1000) / 1000
	return strconv.FormatFloat(rounded, 'f', -1, 64)
}

// formatReal renders a screen value: the catalog's endpoint labels ("OFF",
// "MIN", ...) at the bounds, or the rounded value plus its unit otherwise.
func formatReal(spec ParamSpec, value float64) string {
	if spec.MinLabel != "" && value <= spec.Min {
		return spec.MinLabel
	}
	if spec.MaxLabel != "" && value >= spec.Max {
		return spec.MaxLabel
	}
	rounded := roundForDisplay(value, spec.Units)
	text := strconv.FormatFloat(rounded, 'f', -1, 64)
	if spec.Units != "" {
		return text + " " + spec.Units
	}
	return text
}

// roundForDisplay rounds a real value to the precision a screen would show for
// its unit, so a knob set to 0 dB reads "0 dB" rather than "-0.0002 dB".
func roundForDisplay(value float64, units string) float64 {
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
	rounded := math.Round(value*p) / p
	if rounded == 0 {
		// Normalise -0 (from rounding a tiny negative) to plain 0.
		return 0
	}
	return rounded
}
