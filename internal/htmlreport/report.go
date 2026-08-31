// Package htmlreport renders a human-readable HTML page describing a rig:
// the amp, cabinet, microphone and effect chain with the actual settings used.
package htmlreport

import (
	"fmt"
	"html/template"
	"strconv"
	"strings"
	"time"

	"github.com/d-led/guitar-modeler-mcp/internal/cardchain"
	"github.com/d-led/guitar-modeler-mcp/internal/catalog"
	"github.com/d-led/guitar-modeler-mcp/internal/rig"
)

type paramKV struct {
	Key     string
	Value   string
	Changed bool
}

type moduleInfo struct {
	Name        string
	Slot        int // 1-based grid slot, matching the chain picture
	Category    string
	Description string
	On          bool
	Params      []paramKV

	Amp *catalog.Amp
	Cab *catalog.Cab
	Mic *catalog.Mic
}

type pageData struct {
	Name      string
	Note      string
	Tempo     string
	Generated string
	Chain     []moduleInfo
	ChainCSS  template.CSS
	ChainHTML template.HTML
	Buttons   []rig.ButtonAssign
	Pedals    []rig.PedalAssign
}

var page = template.Must(template.New("report").Parse(reportHTML))

// Render builds the HTML report for a rig.
func Render(rf *rig.RigFile, note string, cat *catalog.Catalog) (string, error) {
	content, err := rf.Decode()
	if err != nil {
		return "", err
	}
	patch := content.Data.Patch

	tempo := ""
	if rigNode, ok := patch.Children["Rig"]; ok {
		if item, ok := rigNode.Children["Tempo"]; ok && item.Value != nil {
			tempo = formatNumber(*item.Value)
		}
	}

	chain := make([]moduleInfo, 0, len(patch.ChildOrder))
	for _, name := range patch.ChildOrder {
		if isFixed(name) {
			continue
		}
		node := patch.Children[name]
		if node == nil {
			continue
		}
		chain = append(chain, moduleInfoFor(name, node, cat))
	}

	// Lay the module cards out in grid-slot order and stamp each with its slot
	// number, so the circled numbers in the text match the chain picture.
	byName := make(map[string]moduleInfo, len(chain))
	for _, info := range chain {
		byName[info.Name] = info
	}
	_, slots := chainLayout(patch)
	modules := make([]moduleInfo, 0, len(chain))
	for i, slotName := range slots {
		if slotName == "" || slotName == "Empty Slot" {
			continue
		}
		info, ok := byName[slotName]
		if !ok {
			continue
		}
		info.Slot = i + 1
		modules = append(modules, info)
	}

	hw, err := rig.HardwareAssignments(rf)
	if err != nil {
		return "", err
	}

	var sb strings.Builder
	if err := page.Execute(&sb, pageData{
		Name:      rf.Name(),
		Note:      note,
		Tempo:     tempo,
		Generated: time.Now().Format("2006-01-02 15:04"),
		Chain:     modules,
		ChainCSS:  template.CSS(cardchain.CSS),
		ChainHTML: template.HTML(chainHTML(patch)),
		Buttons:   hw.Buttons,
		Pedals:    hw.Pedals,
	}); err != nil {
		return "", err
	}
	return sb.String(), nil
}

func isFixed(name string) bool {
	switch name {
	case "Chain", "Rig", "Input", "Output", "Mix":
		return true
	}
	return false
}

func moduleInfoFor(name string, node *rig.Node, cat *catalog.Catalog) moduleInfo {
	info := moduleInfo{Name: name, On: nodeEnabled(node)}

	defaults := rig.Defaults(name)
	params := make([]paramKV, 0, len(node.ChildOrder))
	for _, key := range node.ChildOrder {
		item, ok := node.Children[key]
		if !ok {
			continue
		}
		if key == "PresetName" || key == "PresetName2" {
			continue
		}
		val := itemValue(item)
		changed := false
		if def, ok := defaults[key]; ok {
			changed = val != itemValue(def)
		}
		params = append(params, paramKV{Key: key, Value: val, Changed: changed})
	}
	info.Params = params

	switch {
	case strings.EqualFold(name, "Amp"):
		model := nodeString(node, "Type")
		if a, ok := cat.Amp(model); ok {
			info.Amp = &a
			info.Description = a.Description
		}
	case strings.EqualFold(name, "Cab"):
		cab := nodeString(node, "CabType")
		mic := nodeString(node, "MicType")
		if c, ok := cat.Cab(cab); ok {
			info.Cab = &c
			info.Description = c.Description
		}
		if m, ok := cat.Mic(mic); ok {
			info.Mic = &m
		}
	default:
		if f, ok := cat.FXByName(name); ok {
			info.Category = f.Category
			info.Description = f.Description
		}
	}
	return info
}

func nodeEnabled(node *rig.Node) bool {
	if item, ok := node.Children["On"]; ok && item.State != nil {
		return *item.State
	}
	return false
}

func nodeString(node *rig.Node, key string) string {
	if item, ok := node.Children[key]; ok && item.Str != nil {
		return *item.Str
	}
	return ""
}

func itemValue(item *rig.Item) string {
	switch {
	case item.Value != nil:
		return formatNumber(*item.Value)
	case item.Str != nil:
		return *item.Str
	case item.State != nil:
		if *item.State {
			return "on"
		}
		return "off"
	}
	return ""
}

func formatNumber(v float64) string {
	if v == float64(int64(v)) {
		return fmt.Sprintf("%d", int64(v))
	}
	return strings.TrimRight(fmt.Sprintf("%.2f", v), "0")
}

// chainHTML renders the rig's signal chain as a numbered cardchain fragment,
// dimming any slot whose module starts bypassed.
func chainHTML(patch rig.Patch) string {
	routing, slots := chainLayout(patch)
	if len(slots) == 0 {
		return ""
	}
	return cardchain.Render(chainSteps(routing, slots, bypassed(patch)))
}

// bypassed returns the set of module instance names that start switched off.
func bypassed(patch rig.Patch) map[string]bool {
	off := make(map[string]bool)
	for _, name := range patch.ChildOrder {
		if node := patch.Children[name]; node != nil && !nodeEnabled(node) {
			off[name] = true
		}
	}
	return off
}

// chainLayout reads the routing topology and the 11 grid slots (with "Empty
// Slot" placeholders) from the Chain node.
func chainLayout(patch rig.Patch) (string, []string) {
	chain, ok := patch.Children["Chain"]
	if !ok {
		return "", nil
	}
	routing := ""
	if it, ok := chain.Children["Routing"]; ok && it.Str != nil {
		routing = *it.Str
	}
	slots := make([]string, 0, 11)
	for i := 1; i <= 11; i++ {
		it, ok := chain.Children["ModuleType"+strconv.Itoa(i)]
		if !ok || it.Str == nil {
			continue
		}
		slots = append(slots, *it.Str)
	}
	return routing, slots
}

// chainSteps lays the 11 grid slots out into a numbered chain, honouring the
// Gigboard's three routing topologies: S (serial), SPS-1 (3 shared slots → two
// parallel paths of 3 → 2 shared slots) and PS-1 (two parallel paths at the
// input → 3 shared slots). Parallel paths become a junction with branches A
// and B, each branch keeping its own absolute slot numbers.
func chainSteps(routing string, slots []string, off map[string]bool) []cardchain.Step {
	mk := func(names []string, start int) []cardchain.Step {
		steps := make([]cardchain.Step, 0, len(names))
		for i, n := range names {
			steps = append(steps, slotStep(start+i, n, off[n]))
		}
		return steps
	}
	junction := func(a, b []cardchain.Step) cardchain.Step {
		return cardchain.Step{Branches: []cardchain.Branch{
			{Label: "A", Steps: a},
			{Label: "B", Steps: b},
		}}
	}

	switch routing {
	case "SPS-1":
		if len(slots) >= 9 {
			steps := mk(slots[0:3], 1)
			steps = append(steps, junction(mk(slots[3:6], 4), mk(slots[6:9], 7)))
			return append(steps, mk(slots[9:], 10)...)
		}
	case "PS-1":
		if len(slots) >= 8 {
			steps := []cardchain.Step{junction(mk(slots[0:3], 1), mk(slots[3:8], 4))}
			return append(steps, mk(slots[8:], 9)...)
		}
	}
	return mk(slots, 1)
}

func slotStep(pos int, name string, off bool) cardchain.Step {
	empty := name == "" || name == "Empty Slot"
	effect := name
	if empty {
		effect = ""
	}
	return cardchain.Step{Slot: pos, Effect: effect, Off: empty || off}
}

const reportHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Name}} · HeadRush Gigboard</title>
<style>
  :root { color-scheme: light dark; }
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
         margin: 0; padding: 24px; line-height: 1.45; }
  .wrap { max-width: 860px; margin: 0 auto; }
  h1 { margin: 0 0 4px; font-size: 1.7em; }
  .sub { color: #888; margin-bottom: 24px; }
  .meta { display: flex; gap: 24px; flex-wrap: wrap; margin-bottom: 20px; }
  .meta div { background: #f2f2f7; border-radius: 8px; padding: 10px 14px; }
  @media (prefers-color-scheme: dark) { .meta div { background: #1c1c1e; } }
  .meta .label { font-size: .72em; text-transform: uppercase; letter-spacing: .06em; color: #888; }
  .chain { display: flex; flex-wrap: wrap; gap: 6px; align-items: center; margin-bottom: 28px; }
  .chip { padding: 4px 10px; border-radius: 999px; font-size: .85em; background: #e8e8ed; }
  @media (prefers-color-scheme: dark) { .chip { background: #2c2c2e; } }
  .chip.off { opacity: .45; text-decoration: line-through; }
  .arrow { color: #999; }
  .hw { margin-bottom: 26px; }
  .hw h2 { margin: 0 0 10px; font-size: .8em; text-transform: uppercase; letter-spacing: .08em; color: #888; }
  .buttons { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 10px; }
  .btn { border: 1px solid #e3e3e8; border-radius: 12px; padding: 10px 8px; text-align: center; }
  @media (prefers-color-scheme: dark) { .btn { border-color: #2c2c2e; } }
  .btn .num { display: inline-block; min-width: 22px; height: 22px; line-height: 22px; border-radius: 999px; background: #e8e8ed; font-size: .74em; font-weight: 700; margin-bottom: 6px; }
  @media (prefers-color-scheme: dark) { .btn .num { background: #2c2c2e; } }
  .btn .mod { font-weight: 700; font-size: .9em; overflow-wrap: anywhere; }
  .btn .op { font-size: .76em; color: #888; }
  .btn.assigned { background: #34c75918; border-color: #34c75955; }
  .btn.assigned .num { background: #34c759; color: #fff; }
  .btn.off { background: #8e8e9318; border-color: #8e8e9355; }
  .btn.off .num { background: #8e8e93; color: #fff; }
  .btn.empty { opacity: .42; }
  .pedals { margin-top: 12px; font-size: .9em; display: flex; flex-direction: column; gap: 6px; }
  .pedal { display: flex; flex-wrap: wrap; gap: 6px; align-items: center; }
  .pedal .name { font-weight: 700; }
  .muted { color: #999; }
  .module { border: 1px solid #e3e3e8; border-radius: 10px; padding: 14px 16px; margin-bottom: 12px; }
  @media (prefers-color-scheme: dark) { .module { border-color: #2c2c2e; } }
  .module.off { opacity: .55; }
  .offbadge { margin-left: 8px; font-size: .6em; vertical-align: middle; text-transform: uppercase; letter-spacing: .06em; color: #fff; background: #8e8e93; border-radius: 4px; padding: 2px 6px; }
  .module h2 { margin: 0 0 2px; font-size: 1.1em; }
  .module .cat { font-size: .78em; color: #888; text-transform: uppercase; letter-spacing: .05em; }
  .module .desc { color: #666; margin: 4px 0 10px; }
  .params { display: grid; grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); gap: 6px 16px; }
  .param { font-size: .9em; }
  .param .k { color: #888; margin-right: 6px; }
  .param .v { font-weight: 600; }
  .param.changed .v { color: #2563eb; }
  @media (prefers-color-scheme: dark) { .param.changed .v { color: #60a5fa; } }
  .disclaimer { max-width: 860px; margin: 24px auto 0; padding-top: 16px; border-top: 1px solid #e3e3e8; font-size: .78em; color: #888; }
  @media (prefers-color-scheme: dark) { .disclaimer { border-color: #2c2c2e; } }
  {{.ChainCSS}}
</style>
</head>
<body>
<div class="wrap">
  <h1>{{.Name}}</h1>
  <div class="sub">{{if .Note}}Note: <strong>{{.Note}}</strong> · {{end}}Generated {{.Generated}}</div>
  <div class="meta">
    {{if .Tempo}}<div><div class="label">Tempo</div><div>{{.Tempo}} BPM</div></div>{{end}}
  </div>
  {{.ChainHTML}}
  <div class="hw">
    <h2>Footswitches</h2>
    <div class="buttons">
      {{range .Buttons}}<div class="btn {{if .Module}}assigned{{else}}empty{{end}}{{if .Off}} off{{end}}">
        <div class="num">{{.Number}}</div>
        <div class="mod">{{if .Label}}{{.Label}}{{else if .Module}}{{.Module}}{{else}}—{{end}}</div>{{if .Operation}}<div class="op">{{.Operation}}</div>{{end}}{{if .Mode}}<div class="op">{{.Mode}}</div>{{end}}
      </div>{{end}}
    </div>
    <div class="pedals">
      {{range .Pedals}}
      <div class="pedal"><span class="name">{{.Name}}{{if .Mode}} ({{.Mode}}){{end}}</span>
        {{if .Targets}}{{range .Targets}}<span class="chip">{{.Module}} → {{.Param}}</span>{{end}}{{else}}<span class="muted">unassigned</span>{{end}}
      </div>
      {{end}}
    </div>
  </div>
  {{range .Chain}}
  <div class="module{{if not .On}} off{{end}}">
    <h2>{{if .Slot}}<span class="slotbadge">{{.Slot}}</span>{{end}}{{.Name}}{{if not .On}}<span class="offbadge">off</span>{{end}}</h2>
    <div class="cat">{{.Category}}</div>
    {{if .Amp}}<div class="desc">{{.Amp.Brand}} {{.Amp.RealModel}}{{if .Amp.Wattage}} · {{.Amp.Wattage}}{{end}}{{if .Amp.Style}} · {{range $i, $s := .Amp.Style}}{{if $i}}, {{end}}{{$s}}{{end}}{{end}}</div>
    {{else if .Cab}}<div class="desc">{{.Cab.Speakers}} · {{.Cab.SpeakersRef}}{{if .Mic}} · mic {{.Mic.RealModel}}{{end}}</div>
    {{else}}<div class="desc">{{.Description}}</div>{{end}}
    <div class="params">
      {{range .Params}}<div class="param{{if .Changed}} changed{{end}}"><span class="k">{{.Key}}</span><span class="v">{{.Value}}</span></div>{{end}}
    </div>
  </div>
  {{end}}
</div>
<footer class="disclaimer">
  All trademarks, logos and brand names are the property of their respective owners.
  All company, product and service names used in this report are for identification
  purposes only; use of these names, trademarks and brands does not imply endorsement.
  This project is not affiliated with, endorsed by or sponsored by HeadRush or any of
  the referenced brands.
</footer>
</body>
</html>`
