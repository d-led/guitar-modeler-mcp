package gp200

import (
	"embed"
	"fmt"
	"html"
	"html/template"
	"strconv"
	"strings"

	"github.com/d-led/guitar-modeler-mcp/internal/cardchain"
)

//go:embed css.tmpl html.tmpl
var cardFS embed.FS

var (
	cardCSS  = readCard("css.tmpl")
	cardTmpl = template.Must(template.New("gp200-card").Parse(readCard("html.tmpl")))
)

func readCard(name string) string {
	b, err := cardFS.ReadFile(name)
	if err != nil {
		panic(err)
	}
	return string(b)
}

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

// ChainSlot reports the block label and effect for the chain hint.
func (d ModuleDesc) ChainSlot() (string, string) { return d.Module, d.Effect }

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
	return cardchain.RenderSerial(desc)
}

// SetupCardHTML renders a printable setup card for a preset. It is the
// companion report for the .prst file the design tool writes. Note is optional
// prose for the card (why this tone, how to play it, the rest of the rig and
// hardware); it is printed at the bottom of the card and never written to the
// .prst file.
func SetupCardHTML(m Model, p Preset, note string) string {
	var b strings.Builder
	if err := cardTmpl.Execute(&b, cardPage{
		Title:      p.PatchName + " — " + m.Display,
		H1:         p.PatchName,
		H2:         m.Display + " — setup card",
		CSS:        template.CSS(cardCSS),       // #nosec G203 -- trusted package CSS
		ChainCSS:   template.CSS(cardchain.CSS), // #nosec G203 -- trusted package CSS
		StoredNote: storedNoteHTML(p.PatchName),
		Chain:      template.HTML(chainHint(Describe(p))), // #nosec G203 -- trusted chain HTML
		Modules:    cardModules(p),
		Buttons:    footButtons(p),
		Pedals:     footPedals(p),
		Note:       note,
	}); err != nil {
		panic(err)
	}
	return b.String()
}

// cardPage is the data for the GP-200 setup-card template.
type cardPage struct {
	Title      string
	H1, H2     string
	CSS        template.CSS
	ChainCSS   template.CSS
	StoredNote template.HTML
	Chain      template.HTML
	Modules    []moduleCard
	Buttons    []footBtn
	Pedals     []pedalRow
	Note       string
}

// cardParam is one named parameter value shown on the card.
type cardParam struct {
	Name    string
	Value   template.HTML
	Changed bool
}

// moduleCard is one block rendered as a table.
type moduleCard struct {
	Slot       int
	Module     string
	Effect     string
	Inspired   string
	Enabled    bool
	State      string
	Switch     string
	ShowParams bool
	Params     []cardParam
}

// footBtn is one CTRL footswitch box in the hardware grid.
type footBtn struct {
	Class  string
	Number int
	Mod    string
	Op     string
}

// pedalRow is one expression-pedal assignment chip.
type pedalRow struct {
	Page   string
	Item   int
	Target string
	Param  string
	Range  string
}

// storedNoteHTML renders the truncated-name warning, or empty when the name
// fits on the device.
func storedNoteHTML(name string) template.HTML {
	stored, truncated := StoredName(name)
	return cardchain.StoredNameNote(stored, truncated, NameLimit)
}

func cardModules(p Preset) []moduleCard {
	desc := Describe(p)
	cards := make([]moduleCard, 0, len(desc))
	for i, d := range desc {
		cards = append(cards, moduleCard{Slot: i + 1, Module: d.Module, Effect: d.Effect, Inspired: d.InspiredBy, Enabled: d.Enabled, State: cardchain.StateLabel(d.Enabled), Switch: d.Switch, ShowParams: d.Enabled || d.Switch != "", Params: cardParams(d.Params)})
	}
	return cards
}

func cardParams(ps []ParamDesc) []cardParam {
	out := make([]cardParam, 0, len(ps))
	for _, pd := range ps {
		out = append(out, cardParam{Name: pd.Name, Value: template.HTML(html.EscapeString(formatParam(pd)) + formatParamUnit(pd)), Changed: changed(pd)}) // #nosec G203 -- pre-escaped trusted HTML
	}
	return out
}

func footButtons(p Preset) []footBtn {
	btns := make([]footBtn, 0, len(p.Ctrl))
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
		btns = append(btns, footBtn{Class: cls, Number: c.Index + 1, Mod: mod, Op: op})
	}
	return btns
}

func footPedals(p Preset) []pedalRow {
	var rows []pedalRow
	for _, e := range p.Exp {
		if e.Block < 0 || e.Block > 10 {
			continue
		}
		rows = append(rows, pedalRow{
			Page:   expPageNames[e.Page],
			Item:   e.Item + 1,
			Target: ModuleForBlock(e.Block),
			Param:  expParamName(p, e.Block, e.ParamIndex),
			Range:  strconv.FormatFloat(float64(e.Min), 'f', -1, 32) + "–" + strconv.FormatFloat(float64(e.Max), 'f', -1, 32),
		})
	}
	return rows
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
