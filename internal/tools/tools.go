// Package tools wires the domain packages into MCP tools.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/d-led/guitar-modeler-mcp/internal/assets"
	"github.com/d-led/guitar-modeler-mcp/internal/catalog"
	"github.com/d-led/guitar-modeler-mcp/internal/design"
	"github.com/d-led/guitar-modeler-mcp/internal/docs"
	"github.com/d-led/guitar-modeler-mcp/internal/htmlreport"
	"github.com/d-led/guitar-modeler-mcp/internal/mcp"
	"github.com/d-led/guitar-modeler-mcp/internal/mooer"
	"github.com/d-led/guitar-modeler-mcp/internal/params"
	"github.com/d-led/guitar-modeler-mcp/internal/presetmap"
	"github.com/d-led/guitar-modeler-mcp/internal/rig"
	"github.com/d-led/guitar-modeler-mcp/internal/setlist"
	"github.com/d-led/guitar-modeler-mcp/internal/thr"
	"github.com/d-led/guitar-modeler-mcp/internal/waza"
)

// Registrar binds the catalog, rig builder, designer and cross-device mapping
// table to the MCP server.
type Registrar struct {
	cat     *catalog.Catalog
	builder *rig.Builder
	design  *design.Designer
	table   *presetmap.Table
}

// NewRegistrar creates a Registrar.
func NewRegistrar(cat *catalog.Catalog, b *rig.Builder, d *design.Designer, tbl *presetmap.Table) *Registrar {
	return &Registrar{cat: cat, builder: b, design: d, table: tbl}
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
			"device":     stringSchema("Target hardware modeler. Currently only \"gigboard\" (default) is supported."),
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

	s.Register(mcp.Tool{
		Name:        "create_setlist",
		Description: "Write a device .setlist file that steps through the given rig files in order, so one song can use several incompatible chains (e.g. clean, drive, solo). Reads each .rig file's id and name, and writes <output_dir>/<name>.setlist.",
		InputSchema: objectSchema(map[string]any{
			"name":       stringSchema("Setlist name (becomes the .setlist file name)."),
			"rig_files":  arraySchema("Paths to the .rig files, in setlist order.", stringSchema("Path to a .rig file.")),
			"output_dir": stringSchema("Directory to write the .setlist into (default: the first rig file's directory)."),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return r.createSetlist(args)
		},
	})

	s.Register(mcp.Tool{
		Name:        "device_list",
		Description: "List the supported target devices and whether each supports preset file exchange (file_ext) or only a printable setup card.",
		InputSchema: objectSchema(map[string]any{}),
		Handler: func(_ context.Context, _ map[string]any) (string, error) {
			return marshal(deviceList())
		},
	})

	s.Register(mcp.Tool{
		Name:        "mooer_catalog_list_amps",
		Description: "List amp models for a Mooer device (ge150pro, ge200, ge150, ge100pro). Returns the effect_type index, screen name and the real amp it emulates (inspired_by).",
		InputSchema: objectSchema(map[string]any{
			"model": stringSchema("Mooer model: ge150pro, ge200, ge150 or ge100pro (default ge150pro)."),
			"query": stringSchema("Optional case-insensitive filter over name or inspired_by."),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return r.mooerListAmps(args)
		},
	})

	s.Register(mcp.Tool{
		Name:        "mooer_catalog_list_cabs",
		Description: "List cabinet models for a Mooer device, with the real cabinet each emulates.",
		InputSchema: objectSchema(map[string]any{
			"model": stringSchema("Mooer model: ge150pro, ge200, ge150 or ge100pro (default ge150pro)."),
			"query": stringSchema("Optional case-insensitive filter over name or inspired_by."),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return r.mooerListCabs(args)
		},
	})

	s.Register(mcp.Tool{
		Name:        "mooer_catalog_list_fx",
		Description: "List effect modules for a Mooer device, per module (fx, od, mod, delay, reverb, ns, eq).",
		InputSchema: objectSchema(map[string]any{
			"model":  stringSchema("Mooer model: ge150pro, ge200, ge150 or ge100pro (default ge150pro)."),
			"module": stringSchema("Optional module filter: fx, od, mod, delay, reverb, ns or eq."),
			"query":  stringSchema("Optional case-insensitive filter over name or inspired_by."),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return r.mooerListFX(args)
		},
	})

	s.Register(mcp.Tool{
		Name:        "mooer_design",
		Description: "Dial in a tone for a Mooer device: resolve the amp/cab/effects to model indices, then write a .mo file (file-capable models) and a printable HTML setup card.",
		InputSchema: objectSchema(map[string]any{
			"model":      stringSchema("Mooer model: ge150pro, ge200, ge150 or ge100pro (default ge150pro)."),
			"name":       stringSchema("Preset name."),
			"amp":        stringSchema("Amp: device model name or a real-hardware description, e.g. \"Marshall JCM800\"."),
			"cab":        stringSchema("Optional cab: device model name or description."),
			"fx":         arraySchema("Optional effects; each names a module and an effect within it.", mooerFXItemSchema()),
			"output_dir": stringSchema("Directory to write the files into (default: current directory)."),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return r.mooerDesign(args)
		},
	})

	s.Register(mcp.Tool{
		Name:        "render_setup_card",
		Description: "Render the printable HTML setup card for an existing Mooer .mo preset file.",
		InputSchema: objectSchema(map[string]any{
			"model":       stringSchema("Mooer model that produced the .mo file (default ge150pro)."),
			"preset_file": stringSchema("Path to the .mo file."),
			"output_dir":  stringSchema("Directory to write the HTML card into (default: same as the .mo file)."),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return r.renderSetupCard(args)
		},
	})

	s.Register(mcp.Tool{
		Name:        "map_preset",
		Description: "Map a preset from one device to another. A .rig file maps to a Mooer preset (GE150 Pro Li) plus a printable setup card; a .mo file maps back to a Gigboard .rig.",
		InputSchema: objectSchema(map[string]any{
			"input_file": stringSchema("Path to the source preset (.rig or .mo)."),
			"output_dir": stringSchema("Directory to write the target into (default: the source file's directory)."),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return r.mapPreset(args)
		},
	})

	s.Register(mcp.Tool{
		Name:        "waza_catalog_list_amps",
		Description: "List the five amp types of the Boss Waza Air, with the real hardware each emulates.",
		InputSchema: objectSchema(map[string]any{}),
		Handler: func(_ context.Context, _ map[string]any) (string, error) {
			return r.wazaListAmps()
		},
	})

	s.Register(mcp.Tool{
		Name:        "waza_catalog_list_fx",
		Description: "List the Boss Waza Air effects (booster, mod/fx, delay, reverb) with the real hardware each emulates.",
		InputSchema: objectSchema(map[string]any{"query": stringSchema("Optional case-insensitive filter over name or inspired_by.")}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return r.wazaListFX(args)
		},
	})

	s.Register(mcp.Tool{
		Name:        "waza_catalog_list_modes",
		Description: "List the four XSONIC AIRSTEP BW footswitch modes for the Boss Waza Air: which footswitch toggles which effect or channel in each mode.",
		InputSchema: objectSchema(map[string]any{}),
		Handler: func(_ context.Context, _ map[string]any) (string, error) {
			return r.wazaListModes()
		},
	})

	s.Register(mcp.Tool{
		Name:        "waza_setup_card",
		Description: "Write a printable HTML setup card for a Boss Waza Air tone, optionally including the AIRSTEP BW footswitch mapping (airstep_mode 1-4).",
		InputSchema: objectSchema(wazaCardProps()),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return r.wazaSetupCard(args)
		},
	})

	s.Register(mcp.Tool{
		Name:        "waza_write_tsl",
		Description: "Write a BOSS TONE STUDIO backup (.tsl) for the Boss Waza Air. Starts from the built-in template patch and applies the chosen name; parameter-level mapping is set in the app.",
		InputSchema: objectSchema(map[string]any{
			"name":       stringSchema("Patch name (up to 16 characters)."),
			"output_dir": stringSchema("Directory to write the .tsl into (default: current directory)."),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return r.wazaWriteTSL(args)
		},
	})

	s.Register(mcp.Tool{
		Name:        "waza_read_tsl",
		Description: "Read a Boss Waza Air .tsl backup and report its name, device and the list of patch names.",
		InputSchema: objectSchema(map[string]any{
			"input_file": stringSchema("Path to the .tsl backup."),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return r.wazaReadTSL(args)
		},
	})

	s.Register(mcp.Tool{
		Name:        "thr_catalog_list_amps",
		Description: "List a Yamaha THR model's amp-selector positions (type x mode) with Yamaha's official description and the community-inferred real amplifier.",
		InputSchema: objectSchema(map[string]any{
			"model": stringSchema("THR model: thr (THR-II, default), thr10, thr10c or thr10x."),
			"query": stringSchema("Optional case-insensitive filter over name, type, mode, description or inspired_by."),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return r.thrListAmps(args)
		},
	})

	s.Register(mcp.Tool{
		Name:        "thr_catalog_list_fx",
		Description: "List a Yamaha THR model's effects and cabinets: the EFFECT knob (chorus/flanger/phaser/tremolo), the ECHO delay types, the REVERB types, and the THR-II cabinet list.",
		InputSchema: objectSchema(map[string]any{
			"model": stringSchema("THR model (default: thr)."),
			"query": stringSchema("Optional case-insensitive filter."),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return r.thrListFX(args)
		},
	})

	s.Register(mcp.Tool{
		Name:        "thr_setup_card",
		Description: "Write a printable HTML setup card for a Yamaha THR tone. The THR has no preset file format, so the card is the only output.",
		InputSchema: objectSchema(mergeMaps(map[string]any{
			"name":       stringSchema("Patch name."),
			"model":      stringSchema("THR model: thr (default), thr10, thr10c or thr10x."),
			"amp":        stringSchema("Amp: CLEAN/CRUNCH/LEAD/HI GAIN/SPECIAL/BASS/ACOUSTIC/FLAT, optionally with CLASSIC/BOUTIQUE/MODERN (e.g. \"CLEAN BOUTIQUE\" or \"Twin Reverb\")."),
			"cab":        stringSchema("Optional cabinet, e.g. \"Brown 4x12\" or \"American 1x12\" (THR-II only)."),
			"mod":        stringSchema("Optional EFFECT knob: CHORUS, FLANGER, PHASER or TREMOLO."),
			"echo":       stringSchema("Optional ECHO type: Tape or Digital Delay."),
			"reverb":     stringSchema("Optional REVERB type: Plate, Hall, Spring or Room."),
			"compressor": map[string]any{"type": "boolean", "description": "Optional app-only compressor on/off."},
			"noise_gate": map[string]any{"type": "boolean", "description": "Optional app-only noise gate on/off."},
			"output_dir": stringSchema("Directory to write the HTML card into (default: current directory)."),
		}, thrKnobProps())),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return r.thrSetupCard(args)
		},
	})
}

// thrKnobProps returns the optional numeric knob values for the THR setup
// card. All are on a 0-100 scale unless noted; unset knobs are omitted.
func thrKnobProps() map[string]any {
	return map[string]any{
		"gain":            numberSchema("Optional amp gain (0-100)."),
		"master":          numberSchema("Optional amp master volume (0-100)."),
		"bass":            numberSchema("Optional amp bass (0-100)."),
		"mid":             numberSchema("Optional amp middle (0-100)."),
		"treble":          numberSchema("Optional amp treble (0-100)."),
		"guitar_vol":      numberSchema("Optional global guitar input volume (0-100)."),
		"audio_vol":       numberSchema("Optional global audio output volume (0-100)."),
		"comp_sustain":    numberSchema("Optional compressor sustain (0-100)."),
		"comp_level":      numberSchema("Optional compressor level (0-100)."),
		"gate_threshold":  numberSchema("Optional noise gate threshold (0-100)."),
		"gate_decay":      numberSchema("Optional noise gate decay (0-100)."),
		"mod_speed":       numberSchema("Optional EFFECT speed (0-100)."),
		"mod_depth":       numberSchema("Optional EFFECT depth (0-100)."),
		"mod_predelay":    numberSchema("Optional EFFECT pre-delay in milliseconds."),
		"mod_feedback":    numberSchema("Optional EFFECT feedback (0-100)."),
		"mod_mix":         numberSchema("Optional EFFECT mix (0-100)."),
		"echo_time":       numberSchema("Optional ECHO time in milliseconds."),
		"echo_feedback":   numberSchema("Optional ECHO feedback (0-100)."),
		"echo_bass":       numberSchema("Optional ECHO bass (0-100)."),
		"echo_treble":     numberSchema("Optional ECHO treble (0-100)."),
		"echo_mix":        numberSchema("Optional ECHO mix (0-100)."),
		"reverb_level":    numberSchema("Optional REVERB level (0-100)."),
		"reverb_decay":    numberSchema("Optional REVERB decay (0-100)."),
		"reverb_predelay": numberSchema("Optional REVERB pre-delay in milliseconds."),
		"reverb_tone":     numberSchema("Optional REVERB tone (0-100)."),
		"reverb_mix":      numberSchema("Optional REVERB mix (0-100)."),
	}
}

// wazaToneProps is the shared argument schema for the Waza Air tone tools.
func wazaToneProps() map[string]any {
	return map[string]any{
		"name":              stringSchema("Patch name."),
		"amp":               stringSchema("Amp type: CLEAN, CRUNCH, LEAD, BROWN or FLAT (or a description, e.g. \"Twin Reverb\")."),
		"amp_gain":          numberSchema("Optional amp gain (0-100)."),
		"amp_volume":        numberSchema("Optional amp volume (0-100)."),
		"booster":           stringSchema("Optional BOOSTER effect, e.g. \"T-SCREAM\" or \"TS-808\"."),
		"mod":               stringSchema("Optional MOD effect, e.g. \"CHORUS\"."),
		"fx":                stringSchema("Optional FX effect (same list as MOD)."),
		"delay":             stringSchema("Optional DELAY effect, e.g. \"TAPE ECHO\"."),
		"delay_time":        numberSchema("Optional delay time in milliseconds."),
		"reverb":            stringSchema("Optional REVERB effect, e.g. \"HALL REVERB\"."),
		"cabinet_resonance": stringSchema("Optional: VINTAGE, MODERN or DEEP."),
		"ambience":          stringSchema("Optional: STUDIO or STAGE."),
		"position":          stringSchema("Optional: SURROUND, STATIC or STAGE."),
		"mode":              stringSchema("Optional: DELAY, DLY+REV or REVERB."),
		"output_dir":        stringSchema("Directory to write the output into (default: current directory)."),
	}
}

// wazaCardProps is the setup-card argument schema: the shared tone props plus
// the optional AIRSTEP BW mode to print on the card.
func wazaCardProps() map[string]any {
	props := wazaToneProps()
	props["airstep_mode"] = numberSchema("Optional AIRSTEP BW mode (1-4) whose footswitch mapping is printed on the card.")
	return props
}

func (r *Registrar) designRig(args map[string]any) (string, error) {
	req := design.Request{
		Device:       argString(args, "device"),
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
	htmlPath := filepath.Join(outDir, file.Name()+".gigboard.html")
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
	htmlPath := filepath.Join(outDir, file.Name()+".gigboard.html")
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

// createSetlist binds the given rig files into a device .setlist, so a song
// can step through several incompatible chains.
func (r *Registrar) createSetlist(args map[string]any) (string, error) {
	name := argString(args, "name")
	if strings.TrimSpace(name) == "" {
		return "", fmt.Errorf("a setlist name is required")
	}
	paths := argStrings(args["rig_files"])
	if len(paths) == 0 {
		return "", fmt.Errorf("rig_files is required")
	}

	entries := make([]setlist.Entry, 0, len(paths))
	for _, p := range paths {
		file, err := readRigFile(p)
		if err != nil {
			return "", fmt.Errorf("read rig %q: %w", p, err)
		}
		entries = append(entries, setlist.Entry{ID: file.ID, Name: file.Name()})
	}

	sl, err := setlist.New(name, entries)
	if err != nil {
		return "", err
	}
	outDir := argString(args, "output_dir")
	if outDir == "" {
		outDir = filepath.Dir(paths[0])
	}
	path, err := sl.Write(outDir)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Setlist %q written to %s\n", sl.Name(), path)
	for i, e := range entries {
		fmt.Fprintf(&b, "  %d. %s\n", i+1, e.Name)
	}
	return b.String(), nil
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
			Mode:      argString(m, "mode"),
			Label:     argString(m, "label"),
		}
		if scene, ok := m["scene"].(map[string]any); ok {
			snap := &rig.SceneSnapshot{
				On:  argStrings(scene["on"]),
				Off: argStrings(scene["off"]),
			}
			if len(snap.On) > 0 || len(snap.Off) > 0 {
				sw.Scene = snap
			}
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

// argInt returns the integer value of a numeric argument, or def when the
// argument is absent or not a number.
func argInt(args map[string]any, key string, def int) int {
	if v, ok := args[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		case int64:
			return int(n)
		}
	}
	return def
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

// mergeMaps returns a new map with the entries of all the given maps. Later
// maps win on key collisions.
func mergeMaps(maps ...map[string]any) map[string]any {
	out := map[string]any{}
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
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
		"mode":      stringSchema("Switch type: \"Toggle\" (default) or \"Scene\" (recalls a multi-block on/off snapshot)."),
		"label":     stringSchema("Optional on-screen text for the switch, e.g. \"DRIVE\"."),
		"scene": objectSchema(map[string]any{
			"on":  arraySchema("Modules the scene turns ON (instance names).", stringSchema("Module instance name.")),
			"off": arraySchema("Modules the scene turns OFF (instance names).", stringSchema("Module instance name.")),
		}),
	})
}

// ---- Mooer device tools ----

// deviceInfo describes one supported device for the device_list tool.
type deviceInfo struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	FileExchange bool   `json:"file_exchange"`
	FileExt      string `json:"file_ext,omitempty"`
}

func deviceList() []deviceInfo {
	list := []deviceInfo{
		{Name: "gigboard", Description: "HeadRush Gigboard", FileExchange: true, FileExt: ".rig"},
	}
	for _, m := range mooer.Models() {
		ext := ""
		if m.FileExchange {
			ext = m.FileExt
		}
		list = append(list, deviceInfo{Name: m.Name, Description: m.Display, FileExchange: m.FileExchange, FileExt: ext})
	}
	w := waza.Default()
	list = append(list, deviceInfo{Name: w.Name, Description: w.Display, FileExchange: w.FileExchange, FileExt: w.FileExt})
	for _, t := range thr.Models() {
		list = append(list, deviceInfo{Name: t.Name, Description: t.Display, FileExchange: t.FileExchange, FileExt: t.FileExt})
	}
	return list
}

// catalogItem is one catalog row for the listing tools.
type catalogItem struct {
	Index      int    `json:"index"`
	Name       string `json:"name"`
	InspiredBy string `json:"inspired_by,omitempty"`
}

func mooerModelName(args map[string]any) string {
	if name := argString(args, "model"); name != "" {
		return name
	}
	return "ge150pro"
}

func mooerModel(args map[string]any) (mooer.Model, error) {
	m, ok := mooer.ModelByName(mooerModelName(args))
	if !ok {
		return mooer.Model{}, fmt.Errorf("unknown Mooer model %q; see device_list", mooerModelName(args))
	}
	return m, nil
}

func filterItems(items []mooer.Item, query string) []catalogItem {
	q := strings.ToLower(strings.TrimSpace(query))
	out := make([]catalogItem, 0, len(items))
	for i, it := range items {
		if q != "" && !strings.Contains(strings.ToLower(it.Name+" "+it.InspiredBy), q) {
			continue
		}
		out = append(out, catalogItem{Index: i, Name: it.Name, InspiredBy: it.InspiredBy})
	}
	return out
}

func (r *Registrar) mooerListAmps(args map[string]any) (string, error) {
	m, err := mooerModel(args)
	if err != nil {
		return "", err
	}
	return marshal(filterItems(m.Amps, argString(args, "query")))
}

func (r *Registrar) mooerListCabs(args map[string]any) (string, error) {
	m, err := mooerModel(args)
	if err != nil {
		return "", err
	}
	return marshal(filterItems(m.Cabs, argString(args, "query")))
}

func (r *Registrar) mooerListFX(args map[string]any) (string, error) {
	m, err := mooerModel(args)
	if err != nil {
		return "", err
	}
	module := strings.ToLower(strings.TrimSpace(argString(args, "module")))
	if module == "ds" {
		module = "od"
	}
	query := argString(args, "query")

	modules := map[string][]catalogItem{}
	for _, mod := range m.ModuleOrder {
		if mod == "amp" || mod == "cab" {
			continue
		}
		if module != "" && mod != module {
			continue
		}
		modules[mod] = filterItems(m.Effects[mod], query)
	}
	return marshal(modules)
}

func (r *Registrar) mooerDesign(args map[string]any) (string, error) {
	m, err := mooerModel(args)
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(argString(args, "name"))
	if name == "" {
		name = "New Preset"
	}
	spec := mooer.Spec{
		Name: name,
		Amp:  argString(args, "amp"),
		Cab:  argString(args, "cab"),
		FX:   parseMooerFX(args["fx"]),
	}
	p, err := m.BuildPreset(spec)
	if err != nil {
		return "", err
	}
	outDir := argString(args, "output_dir")
	if outDir == "" {
		outDir = "."
	}
	return r.writeMooerOutput(m, p, outDir)
}

func parseMooerFX(raw any) []mooer.FXSpec {
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]mooer.FXSpec, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		spec := mooer.FXSpec{
			Module:  argString(m, "module"),
			Type:    argString(m, "type"),
			Enabled: argBool(m, "enabled", true),
		}
		if spec.Module != "" && spec.Type != "" {
			out = append(out, spec)
		}
	}
	return out
}

func sanitizeFileBase(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "preset"
	}
	return strings.ReplaceAll(name, "/", "-")
}

// writeMooerOutput writes a .mo file (when the model supports file exchange)
// and always writes a printable HTML setup card, then returns a text summary.
func (r *Registrar) writeMooerOutput(m mooer.Model, p mooer.Preset, outDir string) (string, error) {
	base := sanitizeFileBase(p.Name)
	var b strings.Builder

	if m.FileExchange {
		path := filepath.Join(outDir, base+m.FileExt)
		if err := mooer.WriteMOFile(path, p); err != nil {
			return "", err
		}
		fmt.Fprintf(&b, "Wrote %s preset to %s\n", m.Display, path)
	} else {
		fmt.Fprintf(&b, "%s does not support preset file transfer; here is a printable setup card.\n", m.Display)
	}

	cardPath := filepath.Join(outDir, base+"."+m.Name+".html")
	if err := os.WriteFile(cardPath, []byte(mooer.SetupCardHTML(m, p)), 0o644); err != nil {
		return "", err
	}
	fmt.Fprintf(&b, "Setup card: %s\n", cardPath)

	for _, d := range mooer.Describe(p, m) {
		state := "off"
		if d.Enabled {
			state = "on"
		}
		fmt.Fprintf(&b, "- %s: %s (%s)\n", d.Module, d.Effect, state)
	}
	fmt.Fprintf(&b, "Parameter values are neutral defaults (raw 0-255, 128 = noon); source knob positions are not copied across devices.\n")
	return b.String(), nil
}

func (r *Registrar) renderSetupCard(args map[string]any) (string, error) {
	m, err := mooerModel(args)
	if err != nil {
		return "", err
	}
	path := argString(args, "preset_file")
	if path == "" {
		return "", fmt.Errorf("preset_file is required")
	}
	p, err := mooer.ReadMOFile(path)
	if err != nil {
		return "", err
	}
	outDir := argString(args, "output_dir")
	if outDir == "" {
		outDir = filepath.Dir(path)
	}
	cardPath := filepath.Join(outDir, sanitizeFileBase(p.Name)+"."+m.Name+".html")
	if err := os.WriteFile(cardPath, []byte(mooer.SetupCardHTML(m, p)), 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("Wrote setup card to %s", cardPath), nil
}

func (r *Registrar) mapPreset(args map[string]any) (string, error) {
	input := argString(args, "input_file")
	if input == "" {
		return "", fmt.Errorf("input_file is required")
	}
	outDir := argString(args, "output_dir")
	if outDir == "" {
		outDir = filepath.Dir(input)
	}

	if strings.EqualFold(filepath.Ext(input), ".mo") {
		p, err := mooer.ReadMOFile(input)
		if err != nil {
			return "", err
		}
		spec, err := r.table.MooerToGigboard(p)
		if err != nil {
			return "", err
		}
		file, err := r.builder.Build(spec)
		if err != nil {
			return "", err
		}
		path, err := file.Write(outDir)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("Mapped Mooer -> Gigboard rig: %s", path), nil
	}

	file, err := readRigFile(input)
	if err != nil {
		return "", err
	}
	p, err := r.table.GigboardToMooer(file)
	if err != nil {
		return "", err
	}
	m, _ := mooer.ModelByName("ge150pro")
	return r.writeMooerOutput(m, p, outDir)
}

func (r *Registrar) wazaListAmps() (string, error) {
	d := waza.Default()
	return marshal(filterWazaItems(d.Amps, ""))
}

func (r *Registrar) wazaListFX(args map[string]any) (string, error) {
	d := waza.Default()
	query := argString(args, "query")
	return marshal(map[string]any{
		"booster": filterWazaItems(d.Boosters, query),
		"mod_fx":  filterWazaItems(d.ModFX, query),
		"delay":   filterWazaItems(d.Delays, query),
		"reverb":  filterWazaItems(d.Reverbs, query),
	})
}

func (r *Registrar) wazaListModes() (string, error) {
	return marshal(waza.DefaultAirStep())
}

func (r *Registrar) wazaSetupCard(args map[string]any) (string, error) {
	spec, err := r.wazaSpec(args)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(spec.Name) == "" {
		spec.Name = "New Patch"
	}
	outDir := argString(args, "output_dir")
	if outDir == "" {
		outDir = "."
	}
	d := waza.Default()

	var card string
	if n := int(argFloat(args, "airstep_mode")); n > 0 {
		mode, ok := waza.DefaultAirStep().Mode(n)
		if !ok {
			return "", fmt.Errorf("unknown AIRSTEP BW mode %d (valid: 1-4)", n)
		}
		card = d.SetupCardHTMLWithAirStep(spec, mode)
	} else {
		card = d.SetupCardHTML(spec)
	}

	path := filepath.Join(outDir, sanitizeFileBase(spec.Name)+"."+d.Name+".html")
	if err := os.WriteFile(path, []byte(card), 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("Wrote Waza Air setup card to %s", path), nil
}

func (r *Registrar) wazaWriteTSL(args map[string]any) (string, error) {
	name := strings.TrimSpace(argString(args, "name"))
	if name == "" {
		name = "New Patch"
	}
	tmpl, err := waza.TemplatePatch()
	if err != nil {
		return "", err
	}
	backup := waza.NewBackup(name)
	backup.SetPatches([]waza.Patch{tmpl.WithName(name)})

	outDir := argString(args, "output_dir")
	if outDir == "" {
		outDir = "."
	}
	path := filepath.Join(outDir, sanitizeFileBase(name)+".tsl")
	if err := waza.WriteTSLFile(path, backup); err != nil {
		return "", err
	}
	return fmt.Sprintf("Wrote Waza Air backup to %s", path), nil
}

func (r *Registrar) wazaReadTSL(args map[string]any) (string, error) {
	path := argString(args, "input_file")
	if path == "" {
		return "", fmt.Errorf("input_file is required")
	}
	b, err := waza.ReadTSLFile(path)
	if err != nil {
		return "", err
	}
	names := make([]string, 0, len(b.Patches()))
	for _, p := range b.Patches() {
		names = append(names, p.Name)
	}
	return marshal(map[string]any{
		"name":       b.Name,
		"device":     b.Device,
		"format_rev": b.FormatRev,
		"patches":    names,
	})
}

// wazaSpec builds and resolves a Waza Air tone from the tool arguments.
func (r *Registrar) wazaSpec(args map[string]any) (waza.Spec, error) {
	d := waza.Default()
	return d.Resolve(waza.Spec{
		Name:         argString(args, "name"),
		Amp:          argString(args, "amp"),
		Booster:      argString(args, "booster"),
		Mod:          argString(args, "mod"),
		FX:           argString(args, "fx"),
		Delay:        argString(args, "delay"),
		Reverb:       argString(args, "reverb"),
		CabResonance: argString(args, "cabinet_resonance"),
		Ambience:     argString(args, "ambience"),
		Position:     argString(args, "position"),
		Mode:         argString(args, "mode"),
		Gain:         int(argFloat(args, "amp_gain")),
		Volume:       int(argFloat(args, "amp_volume")),
		DelayTime:    int(argFloat(args, "delay_time")),
	})
}

func filterWazaItems(items []waza.Item, query string) []catalogItem {
	q := strings.ToLower(strings.TrimSpace(query))
	out := make([]catalogItem, 0, len(items))
	for i, it := range items {
		if q != "" && !strings.Contains(strings.ToLower(it.Name+" "+it.InspiredBy), q) {
			continue
		}
		out = append(out, catalogItem{Index: i, Name: it.Name, InspiredBy: it.InspiredBy})
	}
	return out
}

// ---- Yamaha THR device tools ----

func thrModelName(args map[string]any) string {
	if name := argString(args, "model"); name != "" {
		return name
	}
	return "thr"
}

func thrModel(args map[string]any) (thr.Device, error) {
	d, ok := thr.ModelByName(thrModelName(args))
	if !ok {
		return thr.Device{}, fmt.Errorf("unknown THR model %q; see device_list", thrModelName(args))
	}
	return d, nil
}

func (r *Registrar) thrListAmps(args map[string]any) (string, error) {
	d, err := thrModel(args)
	if err != nil {
		return "", err
	}
	return marshal(filterThrAmps(d.Amps, argString(args, "query")))
}

func (r *Registrar) thrListFX(args map[string]any) (string, error) {
	d, err := thrModel(args)
	if err != nil {
		return "", err
	}
	query := argString(args, "query")
	return marshal(map[string]any{
		"modulation": filterThrItems(d.Modulation, query),
		"echo":       filterThrItems(d.Echo, query),
		"reverb":     filterThrItems(d.Reverb, query),
		"cabs":       filterThrItems(d.Cabs, query),
	})
}

func (r *Registrar) thrSetupCard(args map[string]any) (string, error) {
	d, err := thrModel(args)
	if err != nil {
		return "", err
	}
	spec := thr.NewSpec()
	spec.Name = argString(args, "name")
	spec.Amp = argString(args, "amp")
	spec.Cab = argString(args, "cab")
	spec.Mod = argString(args, "mod")
	spec.Echo = argString(args, "echo")
	spec.Reverb = argString(args, "reverb")
	spec.Compressor = argBool(args, "compressor", false)
	spec.NoiseGate = argBool(args, "noise_gate", false)
	spec.AmpParams.Gain = argInt(args, "gain", thr.Unset)
	spec.AmpParams.Master = argInt(args, "master", thr.Unset)
	spec.AmpParams.Bass = argInt(args, "bass", thr.Unset)
	spec.AmpParams.Mid = argInt(args, "mid", thr.Unset)
	spec.AmpParams.Treble = argInt(args, "treble", thr.Unset)
	spec.ModParams.Speed = argInt(args, "mod_speed", thr.Unset)
	spec.ModParams.Depth = argInt(args, "mod_depth", thr.Unset)
	spec.ModParams.PreDelay = argInt(args, "mod_predelay", thr.Unset)
	spec.ModParams.Feedback = argInt(args, "mod_feedback", thr.Unset)
	spec.ModParams.Mix = argInt(args, "mod_mix", thr.Unset)
	spec.EchoParams.Time = argInt(args, "echo_time", thr.Unset)
	spec.EchoParams.Feedback = argInt(args, "echo_feedback", thr.Unset)
	spec.EchoParams.Bass = argInt(args, "echo_bass", thr.Unset)
	spec.EchoParams.Treble = argInt(args, "echo_treble", thr.Unset)
	spec.EchoParams.Mix = argInt(args, "echo_mix", thr.Unset)
	spec.ReverbParams.Level = argInt(args, "reverb_level", thr.Unset)
	spec.ReverbParams.Decay = argInt(args, "reverb_decay", thr.Unset)
	spec.ReverbParams.PreDelay = argInt(args, "reverb_predelay", thr.Unset)
	spec.ReverbParams.Tone = argInt(args, "reverb_tone", thr.Unset)
	spec.ReverbParams.Mix = argInt(args, "reverb_mix", thr.Unset)
	spec.CompParams.Sustain = argInt(args, "comp_sustain", thr.Unset)
	spec.CompParams.Level = argInt(args, "comp_level", thr.Unset)
	spec.GateParams.Threshold = argInt(args, "gate_threshold", thr.Unset)
	spec.GateParams.Decay = argInt(args, "gate_decay", thr.Unset)
	spec.Levels.Guitar = argInt(args, "guitar_vol", thr.Unset)
	spec.Levels.Audio = argInt(args, "audio_vol", thr.Unset)

	resolved, err := d.Resolve(spec)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(resolved.Name) == "" {
		resolved.Name = "New Patch"
	}
	outDir := argString(args, "output_dir")
	if outDir == "" {
		outDir = "."
	}
	path := filepath.Join(outDir, sanitizeFileBase(resolved.Name)+"."+d.Name+".html")
	if err := os.WriteFile(path, []byte(d.SetupCardHTML(resolved)), 0o644); err != nil {
		return "", err
	}
	return fmt.Sprintf("Wrote %s setup card to %s", d.Display, path), nil
}

func filterThrAmps(cells []thr.AmpCell, query string) []thr.AmpCell {
	q := strings.ToLower(strings.TrimSpace(query))
	out := make([]thr.AmpCell, 0, len(cells))
	for _, c := range cells {
		if q != "" && !strings.Contains(strings.ToLower(c.Name+" "+c.Type+" "+c.Mode+" "+c.InspiredBy+" "+c.Description), q) {
			continue
		}
		out = append(out, c)
	}
	return out
}

func filterThrItems(items []thr.Item, query string) []thr.Item {
	q := strings.ToLower(strings.TrimSpace(query))
	out := make([]thr.Item, 0, len(items))
	for _, it := range items {
		if q != "" && !strings.Contains(strings.ToLower(it.Name+" "+it.InspiredBy), q) {
			continue
		}
		out = append(out, it)
	}
	return out
}

func mooerFXItemSchema() map[string]any {
	return objectSchema(map[string]any{
		"module":  stringSchema("Target module: fx, od, mod, delay, reverb, ns or eq."),
		"type":    stringSchema("Effect name within the module, e.g. \"808\" in module \"od\"."),
		"enabled": map[string]any{"type": "boolean", "description": "Whether the module is on."},
	})
}
