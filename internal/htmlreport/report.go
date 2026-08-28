// Package htmlreport renders a human-readable HTML page describing a rig:
// the amp, cabinet, microphone and effect chain with the actual settings used.
package htmlreport

import (
	"fmt"
	"html/template"
	"strings"
	"time"

	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/catalog"
	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/rig"
)

type paramKV struct {
	Key   string
	Value string
}

type moduleInfo struct {
	Name        string
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
	Song      string
	Tempo     string
	Generated string
	Chain     []moduleInfo
}

var page = template.Must(template.New("report").Parse(reportHTML))

// Render builds the HTML report for a rig.
func Render(rf *rig.RigFile, song string, cat *catalog.Catalog) (string, error) {
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

	var sb strings.Builder
	if err := page.Execute(&sb, pageData{
		Name:      rf.Name(),
		Song:      song,
		Tempo:     tempo,
		Generated: time.Now().Format("2006-01-02 15:04"),
		Chain:     chain,
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

	params := make([]paramKV, 0, len(node.ChildOrder))
	for _, key := range node.ChildOrder {
		item, ok := node.Children[key]
		if !ok {
			continue
		}
		if key == "PresetName" || key == "PresetName2" {
			continue
		}
		params = append(params, paramKV{Key: key, Value: itemValue(item)})
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
  .module { border: 1px solid #e3e3e8; border-radius: 10px; padding: 14px 16px; margin-bottom: 12px; }
  @media (prefers-color-scheme: dark) { .module { border-color: #2c2c2e; } }
  .module h2 { margin: 0 0 2px; font-size: 1.1em; }
  .module .cat { font-size: .78em; color: #888; text-transform: uppercase; letter-spacing: .05em; }
  .module .desc { color: #666; margin: 4px 0 10px; }
  .params { display: grid; grid-template-columns: repeat(auto-fill, minmax(150px, 1fr)); gap: 6px 16px; }
  .param { font-size: .9em; }
  .param .k { color: #888; margin-right: 6px; }
  .param .v { font-weight: 600; }
</style>
</head>
<body>
<div class="wrap">
  <h1>{{.Name}}</h1>
  <div class="sub">{{if .Song}}Song: <strong>{{.Song}}</strong> · {{end}}Generated {{.Generated}}</div>
  <div class="meta">
    {{if .Tempo}}<div><div class="label">Tempo</div><div>{{.Tempo}} BPM</div></div>{{end}}
  </div>
  <div class="chain">
    {{range .Chain}}{{if .On}}<span class="chip">{{.Name}}</span>{{else}}<span class="chip off">{{.Name}}</span>{{end}}<span class="arrow">→</span>{{end}}
  </div>
  {{range .Chain}}
  <div class="module">
    <h2>{{.Name}}</h2>
    <div class="cat">{{.Category}}</div>
    {{if .Amp}}<div class="desc">{{.Amp.Brand}} {{.Amp.RealModel}}{{if .Amp.Wattage}} · {{.Amp.Wattage}}{{end}}{{if .Amp.Style}} · {{range $i, $s := .Amp.Style}}{{if $i}}, {{end}}{{$s}}{{end}}{{end}}</div>
    {{else if .Cab}}<div class="desc">{{.Cab.Speakers}} · {{.Cab.SpeakersRef}}{{if .Mic}} · mic {{.Mic.RealModel}}{{end}}</div>
    {{else}}<div class="desc">{{.Description}}</div>{{end}}
    <div class="params">
      {{range .Params}}<div class="param"><span class="k">{{.Key}}</span><span class="v">{{.Value}}</span></div>{{end}}
    </div>
  </div>
  {{end}}
</div>
</body>
</html>`
