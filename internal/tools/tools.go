// Package tools wires the domain packages into MCP tools.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/assets"
	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/catalog"
	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/design"
	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/docs"
	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/htmlreport"
	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/mcp"
	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/params"
	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/rig"
)

// Registrar binds the catalog, rig builder and designer to the MCP server.
type Registrar struct {
	cat     *catalog.Catalog
	builder *rig.Builder
	design  *design.Designer
}

// NewRegistrar creates a Registrar.
func NewRegistrar(cat *catalog.Catalog, b *rig.Builder, d *design.Designer) *Registrar {
	return &Registrar{cat: cat, builder: b, design: d}
}

// Register adds all tools to the server.
func (r *Registrar) Register(s *mcp.Server) {
	s.Register(mcp.Tool{
		Name:        "get_guide",
		Description: "Return the agent guide: how the device's signal chain works, the parallel routing topologies and constraints, the effect categories, and the recommended workflow. Read this first when the task is unfamiliar.",
		InputSchema: objectSchema(map[string]any{}),
		Handler: func(_ context.Context, _ map[string]any) (string, error) {
			return docs.Guide(), nil
		},
	})
	s.Register(mcp.Tool{
		Name:        "get_fx_placement",
		Description: "Return the effect-placement guidance: which effect categories go before vs after the amp, and how each routing topology's sections map onto the available slots (so you know how many effects fit where).",
		InputSchema: objectSchema(map[string]any{}),
		Handler: func(_ context.Context, _ map[string]any) (string, error) {
			return marshal(design.PlacementGuide())
		},
	})
	s.Register(mcp.Tool{
		Name:        "search_catalog",
		Description: "Fuzzy-search every amp, cab, mic and effect by device name, the real hardware it emulates (modeled_after), category or description. Works in both directions: \"JCM800\" finds \"82 Lead 800 100W\" and vice versa. Use kind to restrict to amp/cab/mic/fx.",
		InputSchema: objectSchema(map[string]any{
			"query": stringSchema("Search text, e.g. \"JCM800\", \"Tube Screamer\", \"Twin Reverb\", \"SM57\" or \"Tape Echo\"."),
			"kind":  stringSchema("Optional: restrict to \"amp\", \"cab\", \"mic\" or \"fx\"."),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			query := argString(args, "query")
			if query == "" {
				return "", fmt.Errorf("a \"query\" is required")
			}
			return marshal(r.cat.Search(query, argString(args, "kind")))
		},
	})
	s.Register(mcp.Tool{
		Name:        "catalog_list_amps",
		Description: "List every amp model available on the HeadRush Gigboard, with the real hardware each emulates. Use the optional query to filter.",
		InputSchema: objectSchema(map[string]any{"query": stringSchema("Optional case-insensitive filter over brand/model/style.")}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			query := argString(args, "query")
			return marshal(params.AmpListings(r.cat, query))
		},
	})
	s.Register(mcp.Tool{
		Name:        "catalog_list_cabs",
		Description: "List every cabinet model available on the HeadRush Gigboard.",
		InputSchema: objectSchema(map[string]any{"query": stringSchema("Optional case-insensitive filter.")}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			query := argString(args, "query")
			return marshal(r.cat.CabsMatching(query))
		},
	})
	s.Register(mcp.Tool{
		Name:        "catalog_list_mics",
		Description: "List every microphone model available for cabinet emulation.",
		InputSchema: objectSchema(map[string]any{"query": stringSchema("Optional case-insensitive filter.")}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			query := argString(args, "query")
			return marshal(r.cat.MicsMatching(query))
		},
	})
	s.Register(mcp.Tool{
		Name:        "catalog_list_fx",
		Description: "List effect modules that can be placed in a rig chain. Pass a query to filter by name, category, description or capability (e.g. query=\"pitch shift\" or query=\"delay\"); without a query the full list is returned.",
		InputSchema: objectSchema(map[string]any{"query": stringSchema("Optional filter over name/category/description/capabilities, e.g. \"pitch shift\" or \"reverb\".")}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return marshal(params.FXListingsMatching(r.cat, argString(args, "query")))
		},
	})
	s.Register(mcp.Tool{
		Name:        "catalog_list_fx_categories",
		Description: "List the effect categories (distortion, dynamics, eq, expression, modulation, delay, reverb, utility) with module counts, so you can pick a category before listing its effects.",
		InputSchema: objectSchema(map[string]any{}),
		Handler: func(_ context.Context, _ map[string]any) (string, error) {
			return marshal(params.FXCategories(r.cat))
		},
	})
	s.Register(mcp.Tool{
		Name:        "catalog_list_fx_by_category",
		Description: "List the effect modules in one category (e.g. category=\"delay\" or \"reverb\"). See catalog_list_fx_categories for the valid category names.",
		InputSchema: objectSchema(map[string]any{"category": stringSchema("Effect category, e.g. \"delay\", \"reverb\", \"distortion\", \"modulation\", \"dynamics\", \"eq\", \"expression\", \"utility\".")}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			category := argString(args, "category")
			if category == "" {
				return "", fmt.Errorf("a \"category\" is required; see catalog_list_fx_categories")
			}
			matches := params.FXListingsByCategory(r.cat, category)
			if len(matches) == 0 {
				return "", fmt.Errorf("unknown effect category %q; see catalog_list_fx_categories", category)
			}
			return marshal(matches)
		},
	})
	s.Register(mcp.Tool{
		Name:        "catalog_list_block_presets",
		Description: "List the named factory presets for an effect module (e.g. type=\"Tape Echo\").",
		InputSchema: objectSchema(map[string]any{"type": stringSchema("The effect module display name.")}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			typ := argString(args, "type")
			if typ == "" {
				return "", fmt.Errorf("a module \"type\" is required")
			}
			if f, ok := r.cat.FXByName(typ); ok {
				typ = f.Name
			}
			presets, err := assets.Presets(strings.ToUpper(typ))
			if err != nil {
				return "", fmt.Errorf("no presets for module %q: %w", typ, err)
			}
			return marshal(presets)
		},
	})

	s.Register(mcp.Tool{
		Name:        "catalog_list_module_params",
		Description: "Describe one or more modules' editable parameters: kind (range/toggle/set), label, unit and the allowed values/range, so only valid inputs are produced.",
		InputSchema: objectSchema(map[string]any{
			"type":  stringSchema("Module display name, e.g. \"Tape Echo\", \"Amp\" or \"Cab\"."),
			"types": arraySchema("Optional list of module names to describe in one call (alternative to type).", stringSchema("A module display name.")),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			names := argStrings(args["types"])
			if len(names) > 0 {
				if len(names) == 1 {
					return r.describeModule(names[0])
				}
				return marshal(params.DescribeMany(r.cat, names))
			}
			typ := argString(args, "type")
			if typ == "" {
				return "", fmt.Errorf("a module \"type\" (or \"types\" list) is required")
			}
			return r.describeModule(typ)
		},
	})

	s.Register(mcp.Tool{
		Name:        "translate_amp",
		Description: "Translate a real-world amplifier description (e.g. \"Marshall JCM800\" or \"blackface deluxe reverb\") into the closest HeadRush amp models.",
		InputSchema: objectSchema(map[string]any{"query": stringSchema("Free-form hardware description.")}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return marshal(r.cat.TranslateAmp(argString(args, "query")))
		},
	})
	s.Register(mcp.Tool{
		Name:        "translate_cab",
		Description: "Translate a cabinet description into the closest HeadRush cabinet models.",
		InputSchema: objectSchema(map[string]any{"query": stringSchema("Free-form cabinet description.")}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return marshal(r.cat.TranslateCab(argString(args, "query")))
		},
	})
	s.Register(mcp.Tool{
		Name:        "translate_mic",
		Description: "Translate a microphone description into the closest HeadRush microphone models.",
		InputSchema: objectSchema(map[string]any{"query": stringSchema("Free-form microphone description.")}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return marshal(r.cat.TranslateMic(argString(args, "query")))
		},
	})

	s.Register(mcp.Tool{
		Name:        "design_rig",
		Description: "Dial in a tone: translate hardware into device models, order the effects into a signal chain, write a .rig file and a human-readable HTML report. The chain can be serial (default) or parallel: pass routing=\"SPS-1\" (serial → two parallel paths → serial) with amp2 for a dual-amp rig, or routing=\"PS-1\" (parallel from the input).",
		InputSchema: objectSchema(map[string]any{
			"name":       stringSchema("Rig/patch name."),
			"song":       stringSchema("Optional song the tone is for."),
			"amp":        stringSchema("Amp: device model or real-hardware description."),
			"cab":        stringSchema("Optional cab: device model or description."),
			"mic":        stringSchema("Optional mic: device model or description."),
			"routing":    stringSchema("Signal-chain topology: \"S\" (serial, default), \"SPS-1\" (serial → parallel → serial) or \"PS-1\" (parallel from the input)."),
			"amp2":       stringSchema("Optional second amp for a dual-amp parallel rig (device model or description). Same model as amp = same amp on both channels."),
			"cab2":       stringSchema("Optional cab for the second amp path."),
			"mic2":       stringSchema("Optional mic for the second amp path."),
			"tempo":      numberSchema("Optional tempo in BPM."),
			"input_gain": numberSchema("Optional input gain in dB."), "output_level": numberSchema("Optional overall rig output level in dB (RigVolume; default +6 dB to compensate the amp master's −6 dB)."), "output_dir": stringSchema("Directory to write the files into (default: current directory)."),
			"fx":           arraySchema("Optional effects, in any order; they will be placed sensibly.", fxItemSchema()),
			"path_a_fx":    arraySchema("Optional effects for parallel path A (shared-amp SPS-1).", fxItemSchema()),
			"path_b_fx":    arraySchema("Optional effects for parallel path B (shared-amp SPS-1).", fxItemSchema()),
			"para1_level":  numberSchema("Optional level of path A in dB (default -6)."),
			"para2_level":  numberSchema("Optional level of path B in dB (default -6)."),
			"para1_pan":    numberSchema("Optional pan of path A, -100..100 (default 0; -100 = hard left)."),
			"para2_pan":    numberSchema("Optional pan of path B, -100..100 (default 0; 100 = hard right)."),
			"para_delay":   numberSchema("Optional delay of path B in ms (default 0)."),
			"footswitches": arraySchema("Optional assignments for the 4 stomp switches (FS5..FS8), in order. Each toggles a module on/off, e.g. [{\"module\":\"Wham\"}] puts the whammy on switch 5.", footswitchItemSchema()),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return r.designRig(args)
		},
	})

	s.Register(mcp.Tool{
		Name:        "render_report",
		Description: "Render the human-readable HTML report for an existing .rig file.",
		InputSchema: objectSchema(map[string]any{
			"rig_file":   stringSchema("Path to the .rig file."),
			"song":       stringSchema("Optional song annotation."),
			"output_dir": stringSchema("Directory to write the HTML file into (default: same as rig file)."),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return r.renderReport(args)
		},
	})

	s.Register(mcp.Tool{
		Name:        "rig_decode",
		Description: "Decode an existing .rig file into its signal chain, parallel-path mixer (levels, pans, delay) and per-module parameter values, so you can analyze or verify a preset.",
		InputSchema: objectSchema(map[string]any{
			"rig_file": stringSchema("Path to the .rig file to decode."),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return r.decodeRig(args)
		},
	})

	s.Register(mcp.Tool{
		Name:        "estimate_rig_level",
		Description: "Estimate a rig's output level: sum the input gain, amp master, cab out gain, volume pedals, parallel-path mixer and output RigVolume into a net dB figure, and recommend the RigVolume to reach a target level. Use this when a rig sounds too quiet or too loud.",
		InputSchema: objectSchema(map[string]any{
			"rig_file":  stringSchema("Path to the .rig file to analyze."),
			"target_db": numberSchema("Optional target output level in dB (default 0 = unity)."),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			path := argString(args, "rig_file")
			if path == "" {
				return "", fmt.Errorf("rig_file is required")
			}
			file, err := readRigFile(path)
			if err != nil {
				return "", err
			}
			est, err := rig.EstimateLevel(file, argFloat(args, "target_db"))
			if err != nil {
				return "", err
			}
			return marshal(est)
		},
	})
}

func (r *Registrar) designRig(args map[string]any) (string, error) {
	req := design.Request{
		Name:         argString(args, "name"),
		Song:         argString(args, "song"),
		Amp:          argString(args, "amp"),
		Cab:          argString(args, "cab"),
		Mic:          argString(args, "mic"),
		Routing:      rig.Routing(argString(args, "routing")),
		Amp2:         argString(args, "amp2"),
		Cab2:         argString(args, "cab2"),
		Mic2:         argString(args, "mic2"),
		Tempo:        argFloat(args, "tempo"),
		InputGain:    argFloat(args, "input_gain"),
		OutputLevel:  argFloatPtr(args, "output_level"),
		FX:           parseFX(args["fx"]),
		PathAFX:      parseFX(args["path_a_fx"]),
		PathBFX:      parseFX(args["path_b_fx"]),
		Para1Level:   argFloatPtr(args, "para1_level"),
		Para2Level:   argFloatPtr(args, "para2_level"),
		Para1Pan:     argFloatPtr(args, "para1_pan"),
		Para2Pan:     argFloatPtr(args, "para2_pan"),
		ParaDelay:    argFloatPtr(args, "para_delay"),
		Footswitches: parseFootswitches(args["footswitches"]),
	}
	res, err := r.design.Design(req)
	if err != nil {
		return "", err
	}

	file, err := r.builder.Build(res.Spec)
	if err != nil {
		return "", err
	}

	outDir := argString(args, "output_dir")
	if outDir == "" {
		outDir = "."
	}
	rigPath, err := file.Write(outDir)
	if err != nil {
		return "", err
	}

	html, err := htmlreport.Render(file, req.Song, r.cat)
	if err != nil {
		return "", err
	}
	htmlPath := filepath.Join(outDir, file.Name()+".html")
	if err := os.WriteFile(htmlPath, []byte(html), 0o644); err != nil {
		return "", err
	}

	return summarize(file, res.Notes, req.Song, rigPath, htmlPath), nil
}

func (r *Registrar) renderReport(args map[string]any) (string, error) {
	path := argString(args, "rig_file")
	if path == "" {
		return "", fmt.Errorf("rig_file is required")
	}
	file, err := readRigFile(path)
	if err != nil {
		return "", err
	}
	html, err := htmlreport.Render(file, argString(args, "song"), r.cat)
	if err != nil {
		return "", err
	}
	outDir := argString(args, "output_dir")
	if outDir == "" {
		outDir = filepath.Dir(path)
	}
	htmlPath := filepath.Join(outDir, file.Name()+".html")
	if err := os.WriteFile(htmlPath, []byte(html), 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("Wrote report to %s", htmlPath), nil
}

func (r *Registrar) decodeRig(args map[string]any) (string, error) {
	path := argString(args, "rig_file")
	if path == "" {
		return "", fmt.Errorf("rig_file is required")
	}
	file, err := readRigFile(path)
	if err != nil {
		return "", err
	}
	summary, err := rig.Describe(file)
	if err != nil {
		return "", err
	}
	return marshal(summary)
}

func (r *Registrar) describeModule(typ string) (string, error) {
	spec, err := params.Describe(r.cat, typ)
	if err != nil {
		return "", err
	}
	return marshal(spec)
}

func readRigFile(path string) (*rig.RigFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var file rig.RigFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse rig file: %w", err)
	}
	return &file, nil
}

func summarize(file *rig.RigFile, notes []string, song, rigPath, htmlPath string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Rig %q written.\n", file.Name())
	if song != "" {
		fmt.Fprintf(&b, "Song: %s\n", song)
	}
	for _, n := range notes {
		fmt.Fprintf(&b, "- %s\n", n)
	}

	// Report the hardware assignments and levels so the caller can confirm the
	// switches and gain staging at a glance.
	if summary, err := rig.Describe(file); err == nil {
		fmt.Fprintf(&b, "Footswitches: %s.\n", rig.FootswitchLine(summary.Footswitches))
		fmt.Fprintf(&b, "Levels: input %+g dB, output %+g dB.\n", summary.InputGain, summary.OutputVolume)
	}

	fmt.Fprintf(&b, "Rig file: %s\n", rigPath)
	fmt.Fprintf(&b, "Report:  %s\n", htmlPath)
	return b.String()
}

func parseFX(raw any) []design.FXBlock {
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]design.FXBlock, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		fx := design.FXBlock{
			Type:    argString(m, "type"),
			Enabled: argBool(m, "enabled", true),
		}
		if p, ok := m["params"].(map[string]any); ok {
			fx.Params = p
		}
		if fx.Type != "" {
			out = append(out, fx)
		}
	}
	return out
}

// parseFootswitches reads the footswitches array argument. Entries may be
// objects ({"module":"Wham"}) or plain strings ("Wham").
func parseFootswitches(raw any) []rig.Footswitch {
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]rig.Footswitch, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			if s = strings.TrimSpace(s); s != "" {
				out = append(out, rig.Footswitch{Module: s})
			}
			continue
		}
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		sw := rig.Footswitch{
			Module:    argString(m, "module"),
			Operation: argString(m, "operation"),
		}
		if sw.Module != "" {
			out = append(out, sw)
		}
	}
	return out
}

// ---- argument helpers ----

func argString(args map[string]any, key string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// argStrings returns the string elements of an array argument, or nil when the
// argument is absent or not an array.
func argStrings(raw any) []string {
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

func argFloat(args map[string]any, key string) float64 {
	if v, ok := args[key]; ok {
		switch n := v.(type) {
		case float64:
			return n
		case int:
			return float64(n)
		case int64:
			return float64(n)
		}
	}
	return 0
}

// argFloatPtr returns the numeric value as a pointer, or nil when the argument
// is absent or not a number. Used for optional parameters where nil means
// "keep the default".
func argFloatPtr(args map[string]any, key string) *float64 {
	v, ok := args[key]
	if !ok {
		return nil
	}
	switch n := v.(type) {
	case float64:
		return &n
	case int:
		f := float64(n)
		return &f
	case int64:
		f := float64(n)
		return &f
	}
	return nil
}

func argBool(args map[string]any, key string, def bool) bool {
	if v, ok := args[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
}

func marshal(v any) (string, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ---- JSON schema helpers ----

func objectSchema(props map[string]any) map[string]any {
	return map[string]any{
		"type":       "object",
		"properties": props,
	}
}

func stringSchema(desc string) map[string]any {
	return map[string]any{"type": "string", "description": desc}
}

func numberSchema(desc string) map[string]any {
	return map[string]any{"type": "number", "description": desc}
}

func arraySchema(desc string, items map[string]any) map[string]any {
	return map[string]any{"type": "array", "description": desc, "items": items}
}

func fxItemSchema() map[string]any {
	return objectSchema(map[string]any{
		"type":    stringSchema("Effect module display name, e.g. \"Tape Echo\"."),
		"enabled": map[string]any{"type": "boolean", "description": "Whether the effect is on."},
		"params":  map[string]any{"type": "object", "description": "Parameter overrides; values are numbers, booleans or strings."},
	})
}

func footswitchItemSchema() map[string]any {
	return objectSchema(map[string]any{
		"module":    stringSchema("Module instance name to control, e.g. \"Wham\" or \"Amp 2\"."),
		"operation": stringSchema("What the switch controls; \"On\" toggles the module on/off (default)."),
	})
}
