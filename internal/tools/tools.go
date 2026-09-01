// Package tools wires the domain packages into MCP tools.
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/d-led/guitar-modeler-mcp/internal/assets"
	"github.com/d-led/guitar-modeler-mcp/internal/catalog"
	"github.com/d-led/guitar-modeler-mcp/internal/cookbook"
	"github.com/d-led/guitar-modeler-mcp/internal/design"
	"github.com/d-led/guitar-modeler-mcp/internal/docs"
	"github.com/d-led/guitar-modeler-mcp/internal/htmlreport"
	"github.com/d-led/guitar-modeler-mcp/internal/mcp"
	"github.com/d-led/guitar-modeler-mcp/internal/mooer"
	"github.com/d-led/guitar-modeler-mcp/internal/params"
	"github.com/d-led/guitar-modeler-mcp/internal/presetmap"
	"github.com/d-led/guitar-modeler-mcp/internal/qc"
	"github.com/d-led/guitar-modeler-mcp/internal/qcctl"
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
		Handler: r.searchCatalog,
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
		Handler:     r.catalogListFXByCategory,
	})
	s.Register(mcp.Tool{
		Name:        "catalog_list_block_presets",
		Description: "List the named factory presets for an effect module (e.g. type=\"Tape Echo\").",
		InputSchema: objectSchema(map[string]any{"type": stringSchema("The effect module display name.")}),
		Handler:     r.catalogListBlockPresets,
	})

	s.Register(mcp.Tool{
		Name:        "catalog_list_module_params",
		Description: "Describe one or more modules' editable parameters: kind (range/toggle/set), label, unit and the allowed values/range, so only valid inputs are produced.",
		InputSchema: objectSchema(map[string]any{
			"type":  stringSchema("Module display name, e.g. \"Tape Echo\", \"Amp\" or \"Cab\"."),
			"types": arraySchema("Optional list of module names to describe in one call (alternative to type).", stringSchema("A module display name.")),
		}),
		Handler: r.catalogListModuleParams,
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
			"note":       stringSchema("Optional free-form note shown on the report (e.g. the tone's character, artist or song)."),
			"amp":        stringSchema("Amp: device model or real-hardware description."),
			"cab":        stringSchema("Optional cab: device model or description."),
			"mic":        stringSchema("Optional mic: device model or description."),
			"amp_params": paramMapSchema("Amp knob overrides, keyed by parameter name (e.g. \"GainA\", \"Master\"); values are numbers, booleans or strings."),
			"cab_params": paramMapSchema("Cab knob overrides, keyed by parameter name (e.g. \"Breakup\", \"OutGain\", \"OnAxis\"); values are numbers, booleans or strings."),
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
			"pedals":       arraySchema("Optional expression-pedal assignments (Pedal1, Pedal2), in order. Each wires a module parameter to a pedal, e.g. [{\"module\":\"Black Wah\",\"param\":\"Pedal\"}] sweeps a wah. A wah/whammy/volume is auto-assigned to Pedal1 when this is omitted.", pedalItemSchema()),
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
			"note":       stringSchema("Optional note annotation."),
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
		Handler: r.estimateRigLevel,
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
			"amp_params": mooerAmpParamsSchema(),
			"cab":        stringSchema("Optional cab: device model name or description."),
			"cab_params": mooerCabParamsSchema(),
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
		Name:        "map_ingredients",
		Description: "Port a preset's blocks from one modeler to another by matching their \"ingredients\" (kind + feature tags such as drive, delay, pitch, tape) algorithmically — no agent guessing. Given the source device, target device and the source block names, it returns a mapping table with a score and reason per block, a per-block knob mapping (source/target/canonical parameter names), plus overall and per-kind coverage. Mismatches are listed, never silently dropped. Device names come from device_list.",
		InputSchema: objectSchema(map[string]any{
			"source_device": stringSchema("Source device name (gigboard, ge200, ge150pro, ge150, ge100pro, wazaair, thr, thr10, thr10c, thr10x, quad-cortex)."),
			"target_device": stringSchema("Target device name, same list as source_device."),
			"blocks":        arraySchema("The source preset's block names, in signal order.", stringSchema("One block name.")),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return r.mapIngredients(args)
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
		Description: "Write a BOSS TONE STUDIO backup (.tsl) for the Boss Waza Air. Starts from a neutral CLEAN template and applies the chosen amp/effect types and knob values; unspecified effects stay off. Pass a `patches` array to pack several named patches into one backup.",
		InputSchema: objectSchema(mergeMaps(map[string]any{
			"name":       stringSchema("Backup name (becomes the .tsl file name)."),
			"output_dir": stringSchema("Directory to write the .tsl into (default: current directory)."),
			"patches": arraySchema("Optional list of patches (each with `name` plus the tone fields below); when given, all are written into this one backup instead of a single top-level patch.",
				objectSchema(mergeMaps(map[string]any{"name": stringSchema("Patch name (up to 16 characters).")}, wazaPatchProps()))),
		}, wazaPatchProps())),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return r.wazaWriteTSL(args)
		},
	})

	s.Register(mcp.Tool{
		Name:        "waza_read_tsl",
		Description: "Read a Boss Waza Air .tsl backup and report its name, device and each patch's decoded amp/effect parameters.",
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

	s.Register(mcp.Tool{
		Name:        "qc_catalog_list_amps",
		Description: "List the Neural DSP Quad Cortex guitar amp models (and bass amps) with the real hardware each is based on.",
		InputSchema: objectSchema(map[string]any{
			"query": stringSchema("Optional case-insensitive filter over name or based-on."),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return r.qcListAmps(args)
		},
	})

	s.Register(mcp.Tool{
		Name:        "qc_catalog_list_cabs",
		Description: "List the Quad Cortex guitar cabinet (cabsim) models with the real cabinet each is based on.",
		InputSchema: objectSchema(map[string]any{
			"query": stringSchema("Optional case-insensitive filter over name or based-on."),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return r.qcListCabs(args)
		},
	})

	s.Register(mcp.Tool{
		Name:        "qc_catalog_list_fx",
		Description: "List the Quad Cortex effect models in one category: drive, compressor, equalizer, delay, modulation, reverb, wah, pitch, filter or gate.",
		InputSchema: objectSchema(map[string]any{
			"category": stringSchema("Effect category: drive, compressor, equalizer, delay, modulation, reverb, wah, pitch, filter or gate."),
			"query":    stringSchema("Optional case-insensitive filter over name or based-on."),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return r.qcListFX(args)
		},
	})

	s.Register(mcp.Tool{
		Name:        "qc_translate_amp",
		Description: "Translate a real-world amplifier description into the closest Quad Cortex amp model (e.g. \"Marshall JCM800\" or \"Fender Twin Reverb\").",
		InputSchema: objectSchema(map[string]any{
			"query": stringSchema("Free-form hardware description."),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return r.qcTranslateAmp(args)
		},
	})

	s.Register(mcp.Tool{
		Name:        "qc_translate_cab",
		Description: "Translate a cabinet description into the closest Quad Cortex cabinet model.",
		InputSchema: objectSchema(map[string]any{
			"query": stringSchema("Free-form cabinet description."),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return r.qcTranslateCab(args)
		},
	})

	s.Register(mcp.Tool{
		Name:        "qc_list_model_params",
		Description: "List one Quad Cortex model's editable parameters with their scale (min, max, default, steps and option names), so a value can be set on the screen's own line. Resolves the model by name or \"based on\" description.",
		InputSchema: objectSchema(map[string]any{
			"model": stringSchema("Model name or \"based on\" description, e.g. \"JCM800\" or \"Mesa Rectifier\"."),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return r.qcListModelParams(args)
		},
	})

	s.Register(mcp.Tool{
		Name:        "qc_decode_preset",
		Description: "Decrypt and decode a Quad Cortex .pb reference archive into a readable summary: the grid rows, each block's model name and each parameter's name with its real (screen) value. The serial is the unit's 9-character serial (empty for cloud files). The .pb is this tool's own storage format, not a file the unit imports.",
		InputSchema: objectSchema(map[string]any{
			"path":   stringSchema("Path to the encrypted .pb preset file."),
			"serial": stringSchema("The unit's 9-character serial number, or empty for cloud files."),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return r.qcDecodePreset(args)
		},
	})

	s.Register(mcp.Tool{
		Name:        "qc_design",
		Description: "Build a serial Quad Cortex preset — amp, then cab, then the effects in the order given — and write a self-contained HTML setup card, a .pb reference archive, and a human-readable .json view. The HTML card is the dial-in instructions; the .pb is for saving and reloading the tone in this tool, NOT a file the unit imports; the .json is this tool's own readable view of the same preset (also not a device or upload format). To put the tone on the unit, dial it in from the card or place a preset in a slot with Cortex Control — qc_usb (qcctl) can recall that slot but cannot upload the .pb. Parameter values are on the screen's own line (GAIN 5 on a 0..10 knob, a dB or % value); list parameters take the option index. Knob names are case-insensitive, and common synonyms resolve automatically (GAIN→VOLUME, MIDDLE→MID, DRIVE→OVERDRIVE, LEVEL→OUTPUT, TIME→DELAY TIME, RATE→CHR RATE, DEPTH→VIB DEPTH); if a model rejects a name, run qc_list_model_params to see its exact knob list. The serial is the unit's 9-character serial (empty for cloud).",
		InputSchema: objectSchema(map[string]any{
			"name":               stringSchema("Preset name (becomes the file name)."),
			"serial":             stringSchema("The unit's 9-character serial number, or empty for cloud files."),
			"amp":                stringSchema("Amp model: device name or \"based on\" description, e.g. \"Marshall JCM800\"."),
			"cab":                stringSchema("Optional cab model name or description."),
			"amp_params":         floatMapSchema("Amp knob values in screen units (e.g. {\"GAIN\": 5, \"OUTPUT\": 0})."),
			"amp_encoded_params": floatMapSchema("Amp knob values already on the device's 0..1 line."),
			"cab_params":         floatMapSchema("Cab knob values in screen units."),
			"cab_encoded_params": floatMapSchema("Cab knob values already on the device's 0..1 line."),
			"fx":                 arraySchema("Effects after the cab, in signal order.", qcFXItemSchema()),
			"author":             stringSchema("Optional author name."),
			"volume":             numberSchema("Optional preset output level (default 1.0 = unity)."),
			"output_dir":         stringSchema("Directory to write the .pb and card into (default: current directory)."),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return r.qcDesign(args)
		},
	})

	s.Register(mcp.Tool{
		Name:        "qc_render_setup_card",
		Description: "Decode a Quad Cortex .pb reference archive and write a printable HTML setup card plus a human-readable .json view next to it. The serial is the unit's 9-character serial (empty for cloud files).",
		InputSchema: objectSchema(map[string]any{
			"path":       stringSchema("Path to the encrypted .pb preset file."),
			"serial":     stringSchema("The unit's 9-character serial number, or empty for cloud files."),
			"output_dir": stringSchema("Directory to write the card into (default: next to the preset)."),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return r.qcRenderSetupCard(args)
		},
	})

	s.Register(mcp.Tool{
		Name:        "qc_usb",
		Description: "Control a connected Quad Cortex over USB by shelling out to the user's qcctl (pyquadcortex). qcctl has only four subcommands — version, recall --setlist --slot, scene --index, dump-preset --setlist --slot — so it reads the firmware version, recalls a preset already in a slot on the unit, switches scenes, and dumps the preset in a slot. It does NOT upload a .pb file: the .pb is only a reference archive; to put a tone on the unit, dial it in from the HTML card or place a preset in a slot with Cortex Control, then recall it. Prerequisites: install qcctl once (`pip install pyquadcortex`; macOS also `brew install hidapi`), and quit the official Cortex Control desktop app first — it holds an exclusive lock on the USB interface and blocks qcctl while running (device Wi-Fi may stay on). Ask the user first: it needs a USB-connected Quad Cortex and it can change the device — only set confirm:true after they agree.",
		InputSchema: objectSchema(map[string]any{
			"command": stringSchema("qcctl command: version, recall, scene or dump-preset."),
			"slot":    stringSchema("Slot name for recall/dump-preset, e.g. 28C."),
			"scene":   numberSchema("Scene number for scene (e.g. 3)."),
			"setlist": stringSchema("Optional setlist path for recall/dump-preset."),
			"confirm": boolSchema("Set true only after the user has confirmed live USB control."),
		}),
		Handler: func(_ context.Context, args map[string]any) (string, error) {
			return r.qcUSB(args)
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
// wazaPatchProps is the tone-definition schema shared by the single-patch and
// multi-patch Waza Air write tools (everything except the name and the output
// directory).
func wazaPatchProps() map[string]any {
	return map[string]any{
		"amp":                stringSchema("Amp type: CLEAN, CRUNCH, LEAD, BROWN or FLAT (or a description, e.g. \"Twin Reverb\")."),
		"amp_gain":           numberSchema("Optional amp gain (0-100)."),
		"amp_volume":         numberSchema("Optional amp volume (0-100)."),
		"amp_bass":           numberSchema("Optional amp bass (0-100)."),
		"amp_middle":         numberSchema("Optional amp middle (0-100)."),
		"amp_treble":         numberSchema("Optional amp treble (0-100)."),
		"amp_presence":       numberSchema("Optional amp presence (0-100)."),
		"booster":            stringSchema("Optional BOOSTER effect, e.g. \"T-SCREAM\" or \"TS-808\"."),
		"booster_drive":      numberSchema("Optional booster drive (0-120)."),
		"booster_bottom":     numberSchema("Optional booster bottom (-50..+50)."),
		"booster_tone":       numberSchema("Optional booster tone (0-100, 50 = neutral)."),
		"booster_solo":       boolSchema("Optional booster solo switch (true/false)."),
		"booster_solo_level": numberSchema("Optional booster solo level (0-100)."),
		"booster_level":      numberSchema("Optional booster level (0-100)."),
		"booster_direct_mix": numberSchema("Optional booster direct mix (0-100)."),
		"mod":                stringSchema("Optional MOD effect, e.g. \"CHORUS\"."),
		"mod_params": objectSchema(map[string]any{
			"rate":         numberSchema("e.g. 35"),
			"depth":        numberSchema("e.g. 60"),
			"effect_level": numberSchema("e.g. 50"),
			"direct_mix":   numberSchema("e.g. 100"),
		}),
		"fx":                stringSchema("Optional FX effect (same list as MOD)."),
		"fx_params":         objectSchema(map[string]any{}),
		"delay":             stringSchema("Optional DELAY effect, e.g. \"TAPE ECHO\"."),
		"delay_time":        numberSchema("Optional delay time in milliseconds."),
		"delay_feedback":    numberSchema("Optional delay feedback (0-100)."),
		"delay_high_cut":    numberSchema("Optional delay high cut (0-14)."),
		"delay_level":       numberSchema("Optional delay wet level (0-120)."),
		"delay_direct_mix":  numberSchema("Optional delay DRY-signal level (0-100, 100 = unity/full dry). Leave unset unless you want to attenuate the dry guitar."),
		"reverb":            stringSchema("Optional REVERB effect, e.g. \"HALL REVERB\"."),
		"reverb_time":       numberSchema("Optional reverb time in seconds (0.1-10.0)."),
		"reverb_pre_delay":  numberSchema("Optional reverb pre-delay in milliseconds (0-500)."),
		"reverb_level":      numberSchema("Optional reverb wet level (0-100)."),
		"reverb_direct_mix": numberSchema("Optional reverb DRY-signal level (0-100, 100 = unity/full dry). Leave unset unless you want to attenuate the dry guitar."),
		"cabinet_resonance": stringSchema("Optional: VINTAGE, MODERN or DEEP."),
		"ambience":          stringSchema("Optional: STUDIO or STAGE."),
		"ambience_level":    numberSchema("Optional ambience level (0-100)."),
		"position":          stringSchema("Optional: SURROUND, STATIC or STAGE."),
		"guitar_position":   numberSchema("Optional guitar position in degrees (-180..+180, SURROUND only)."),
		"mode":              stringSchema("Optional: DELAY, DLY+REV or REVERB."),
		"ns_on":             boolSchema("Optional noise suppressor switch (true = on, false = off)."),
		"ns_threshold":      numberSchema("Optional noise suppressor threshold (0-100)."),
		"ns_release":        numberSchema("Optional noise suppressor release (0-100)."),
	}
}

// wazaToneProps is the argument schema for the single-patch Waza Air tools:
// a name, the tone fields and the output directory.
func wazaToneProps() map[string]any {
	return mergeMaps(
		map[string]any{"name": stringSchema("Patch name.")},
		wazaPatchProps(),
		map[string]any{"output_dir": stringSchema("Directory to write the output into (default: current directory).")},
	)
}

func (r *Registrar) searchCatalog(_ context.Context, args map[string]any) (string, error) {
	query := argString(args, "query")
	if query == "" {
		return "", fmt.Errorf("a \"query\" is required")
	}
	return marshal(r.cat.Search(query, argString(args, "kind")))
}

func (r *Registrar) catalogListFXByCategory(_ context.Context, args map[string]any) (string, error) {
	category := argString(args, "category")
	if category == "" {
		return "", fmt.Errorf("a \"category\" is required; see catalog_list_fx_categories")
	}
	matches := params.FXListingsByCategory(r.cat, category)
	if len(matches) == 0 {
		return "", fmt.Errorf("unknown effect category %q; see catalog_list_fx_categories", category)
	}
	return marshal(matches)
}

func (r *Registrar) catalogListBlockPresets(_ context.Context, args map[string]any) (string, error) {
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
}

func (r *Registrar) catalogListModuleParams(_ context.Context, args map[string]any) (string, error) {
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
}

func (r *Registrar) estimateRigLevel(_ context.Context, args map[string]any) (string, error) {
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
		Note:         argString(args, "note"),
		Amp:          argString(args, "amp"),
		Cab:          argString(args, "cab"),
		Mic:          argString(args, "mic"),
		AmpParams:    argMap(args, "amp_params"),
		CabParams:    argMap(args, "cab_params"),
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
		Pedals:       parsePedals(args["pedals"]),
	}
	res, err := r.design.Design(req)
	if err != nil {
		return "", err
	}

	file, err := r.builder.Build(res.Spec)
	if err != nil {
		return "", err
	}
	if stored, truncated := rig.StoredName(res.Spec.Name); truncated {
		res.Notes = append(res.Notes, fmt.Sprintf(
			"preset name %q was truncated to %q — Gigboard names fit %d characters; put the full title in the note field instead",
			res.Spec.Name, stored, rig.NameLimit))
	}

	outDir := argString(args, "output_dir")
	if outDir == "" {
		outDir = "."
	}
	rigPath, err := file.Write(outDir)
	if err != nil {
		return "", err
	}

	html, err := htmlreport.Render(file, req.Note, r.cat)
	if err != nil {
		return "", err
	}
	htmlPath := filepath.Join(outDir, file.Name()+".gigboard.html")
	if err := os.WriteFile(htmlPath, []byte(html), 0o600); err != nil {
		return "", err
	}

	return summarize(file, res.Notes, req.Note, rigPath, htmlPath), nil
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
	html, err := htmlreport.Render(file, argString(args, "note"), r.cat)
	if err != nil {
		return "", err
	}
	outDir := argString(args, "output_dir")
	if outDir == "" {
		outDir = filepath.Dir(path)
	}
	htmlPath := filepath.Join(outDir, file.Name()+".gigboard.html")
	if err := os.WriteFile(htmlPath, []byte(html), 0o600); err != nil {
		return "", err
	}
	return fmt.Sprintf("Wrote report: %s", fileLink(htmlPath)), nil
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

func summarize(file *rig.RigFile, notes []string, note, rigPath, htmlPath string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Rig %q written.\n", file.Name())
	if note != "" {
		fmt.Fprintf(&b, "Note: %s\n", note)
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

	fmt.Fprintf(&b, "Rig file: %s\n", fileLink(rigPath))
	fmt.Fprintf(&b, "Report:  %s\n", fileLink(htmlPath))
	return b.String()
}

// fileURL returns an absolute file:// URI for a path, so chat clients render it
// as a clickable link that opens in the browser.
func fileURL(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return "file://" + filepath.ToSlash(abs)
}

// fileLink wraps a path as a markdown link whose label is the file name, so the
// user sees a friendly clickable label instead of the raw file:// URI.
func fileLink(path string) string {
	return fmt.Sprintf("[%s](%s)", filepath.Base(path), fileURL(path))
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

// parsePedals reads the pedals array argument: one expression-pedal assignment
// per entry, e.g. {"module":"Black Wah","param":"Pedal"}.
func parsePedals(raw any) []rig.Pedal {
	arr, ok := raw.([]any)
	if !ok {
		return nil
	}
	out := make([]rig.Pedal, 0, len(arr))
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		p := rig.Pedal{
			Module: argString(m, "module"),
			Param:  argString(m, "param"),
			Min:    argFloat(m, "min"),
			Max:    argFloat(m, "max"),
		}
		if p.Module != "" && p.Param != "" {
			out = append(out, p)
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

// argObjects returns the object elements of an array argument (e.g. a patches
// list), or nil when the argument is absent or not an array of objects.
func argObjects(args map[string]any, key string) []map[string]any {
	arr, ok := args[key].([]any)
	if !ok {
		return nil
	}
	out := make([]map[string]any, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

// argMap returns an object argument as-is (a map of parameter overrides whose
// values may be numbers, booleans or strings), or nil when absent or not an
// object.
func argMap(args map[string]any, key string) map[string]any {
	if m, ok := args[key].(map[string]any); ok {
		return m
	}
	return nil
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

// argFloatMap returns the numeric entries of an object argument (e.g.
// mod_params/fx_params), or nil when the argument is absent or not an object.
func argFloatMap(args map[string]any, key string) map[string]float64 {
	obj, ok := args[key].(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]float64, len(obj))
	for k, v := range obj {
		switch n := v.(type) {
		case float64:
			out[k] = n
		case int:
			out[k] = float64(n)
		case int64:
			out[k] = float64(n)
		}
	}
	return out
}

// argBoolPtr returns the boolean argument as a pointer, or nil when absent.
// Used for optional switches where nil means "keep the default".
func argBoolPtr(args map[string]any, key string) *bool {
	v, ok := args[key]
	if !ok {
		return nil
	}
	b, ok := v.(bool)
	if !ok {
		return nil
	}
	return &b
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

func boolSchema(desc string) map[string]any {
	return map[string]any{"type": "boolean", "description": desc}
}

func arraySchema(desc string, items map[string]any) map[string]any {
	return map[string]any{"type": "array", "description": desc, "items": items}
}

// floatMapSchema describes an object whose values are all numbers (named
// parameter settings).
func floatMapSchema(desc string) map[string]any {
	return map[string]any{
		"type":                 "object",
		"description":          desc,
		"additionalProperties": map[string]any{"type": "number"},
	}
}

// paramMapSchema describes an object of parameter overrides whose values may
// be numbers, booleans or strings (not just numbers).
func paramMapSchema(desc string) map[string]any {
	return map[string]any{"type": "object", "description": desc}
}

// qcFXItemSchema describes one Quad Cortex effect block in qc_design.
func qcFXItemSchema() map[string]any {
	return objectSchema(map[string]any{
		"type":           stringSchema("Effect model name or \"based on\" description, e.g. \"TS808\" or \"Tape Delay\"."),
		"params":         floatMapSchema("Knob values in screen units (option index for list parameters)."),
		"encoded_params": floatMapSchema("Knob values already on the device's 0..1 line."),
	})
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

func pedalItemSchema() map[string]any {
	return objectSchema(map[string]any{
		"module": stringSchema("Module instance name to control, e.g. \"Black Wah\", \"Wham\" or \"Volume\"."),
		"param":  stringSchema("Controller target, e.g. \"Pedal\" (wah sweep), \"Pitch\" (whammy) or \"Volume\"."),
		"min":    numberSchema("Optional controller minimum (default 0)."),
		"max":    numberSchema("Optional controller maximum (default 100)."),
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
	ID         int    `json:"id,omitempty"`
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
		Name:      name,
		Amp:       argString(args, "amp"),
		AmpParams: argFloatMap(args, "amp_params"),
		Cab:       argString(args, "cab"),
		CabParams: argFloatMap(args, "cab_params"),
		FX:        parseMooerFX(args["fx"]),
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
			Params:  argFloatMap(m, "params"),
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
	if err := os.WriteFile(cardPath, []byte(mooer.SetupCardHTML(m, p)), 0o600); err != nil {
		return "", err
	}
	fmt.Fprintf(&b, "Setup card: %s\n", cardPath)

	if stored, truncated := mooer.StoredName(p.Name); truncated {
		fmt.Fprintf(&b, "Note: the %s stores preset names up to %d characters; this preset reads as %q on the unit.\n", m.Display, mooer.NameLimit, stored)
	}

	for _, d := range mooer.Describe(p, m) {
		state := "off"
		if d.Enabled {
			state = "on"
		}
		fmt.Fprintf(&b, "- %s: %s (%s)\n", d.Module, d.Effect, state)
	}
	fmt.Fprintf(&b, "Parameter values are neutral defaults (raw 0-100, 50 = noon); source knob positions are not copied across devices.\n")
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
	if err := os.WriteFile(cardPath, []byte(mooer.SetupCardHTML(m, p)), 0o600); err != nil {
		return "", err
	}
	return fmt.Sprintf("Wrote setup card to %s", cardPath), nil
}

func (r *Registrar) mapIngredients(args map[string]any) (string, error) {
	srcName := argString(args, "source_device")
	tgtName := argString(args, "target_device")
	if srcName == "" || tgtName == "" {
		return "", fmt.Errorf("source_device and target_device are required")
	}
	src, err := cookbook.Ingredients(srcName)
	if err != nil {
		return "", err
	}
	tgt, err := cookbook.Ingredients(tgtName)
	if err != nil {
		return "", err
	}
	blocks := argStrings(args["blocks"])
	if len(blocks) == 0 {
		return "", fmt.Errorf("blocks is required: the source preset's block names")
	}
	plan, err := cookbook.Map(src, tgt, tgtName, blocks)
	if err != nil {
		return "", err
	}
	return marshal(plan)
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
	if err := os.WriteFile(path, []byte(card), 0o600); err != nil {
		return "", err
	}
	return fmt.Sprintf("Wrote Waza Air setup card to %s", path), nil
}

func (r *Registrar) wazaWriteTSL(args map[string]any) (string, error) {
	tmpl, err := waza.TemplatePatch()
	if err != nil {
		return "", err
	}
	patches, err := r.wazaPatches(tmpl, args)
	if err != nil {
		return "", err
	}

	name := strings.TrimSpace(argString(args, "name"))
	if name == "" {
		name = patches[0].Name
	}
	if name == "" {
		name = "New Patch"
	}

	backup := waza.NewBackup(name)
	backup.SetPatches(patches)

	outDir := argString(args, "output_dir")
	if outDir == "" {
		outDir = "."
	}
	path := filepath.Join(outDir, sanitizeFileBase(name)+".tsl")
	if err := waza.WriteTSLFile(path, backup); err != nil {
		return "", err
	}
	return fmt.Sprintf("Wrote Waza Air backup (%d patch(es)) to %s", len(patches), path), nil
}

// wazaPatches builds the requested patches from the neutral template: either a
// single top-level patch or the "patches" array.
func (r *Registrar) wazaPatches(tmpl waza.Patch, args map[string]any) ([]waza.Patch, error) {
	if _, multi := args["patches"]; !multi {
		spec, err := r.wazaSpec(args)
		if err != nil {
			return nil, err
		}
		return []waza.Patch{wazaPatch(tmpl, spec)}, nil
	}

	var patches []waza.Patch
	for i, pm := range argObjects(args, "patches") {
		spec, err := r.wazaSpec(pm)
		if err != nil {
			return nil, fmt.Errorf("patches[%d]: %w", i, err)
		}
		patches = append(patches, wazaPatch(tmpl, spec))
	}
	if len(patches) == 0 {
		return nil, fmt.Errorf("patches must contain at least one patch")
	}
	return patches, nil
}

// wazaPatch builds one preset patch from the neutral template plus a resolved
// spec: the specified blocks and knobs are applied, everything else stays off.
func wazaPatch(tmpl waza.Patch, spec waza.Spec) waza.Patch {
	name := strings.TrimSpace(spec.Name)
	if name == "" {
		name = "New Patch"
	}
	return tmpl.WriteParams(waza.Params{
		AmpType:          spec.Amp,
		AmpGain:          spec.Gain,
		AmpVolume:        spec.Volume,
		AmpBass:          spec.Bass,
		AmpMiddle:        spec.Middle,
		AmpTreble:        spec.Treble,
		AmpPresence:      spec.Presence,
		BoosterType:      spec.Booster,
		BoosterDrive:     spec.BoosterDrive,
		BoosterBottom:    spec.BoosterBottom,
		BoosterTone:      spec.BoosterTone,
		BoosterSolo:      spec.BoosterSolo,
		BoosterSoloLevel: spec.BoosterSoloLevel,
		BoosterLevel:     spec.BoosterLevel,
		BoosterDirectMix: spec.BoosterDirectMix,
		ModType:          spec.Mod,
		ModParams:        spec.ModParams,
		FXType:           spec.FX,
		FXParams:         spec.FXParams,
		DelayType:        spec.Delay,
		DelayTime:        spec.DelayTime,
		DelayFeedback:    spec.DelayFeedback,
		DelayHighCut:     spec.DelayHighCut,
		DelayLevel:       spec.DelayLevel,
		DelayDirectMix:   spec.DelayDirectMix,
		ReverbType:       spec.Reverb,
		ReverbTime:       spec.ReverbTime,
		ReverbPreDelay:   spec.ReverbPreDelay,
		ReverbLevel:      spec.ReverbLevel,
		ReverbDirectMix:  spec.ReverbDirectMix,
		Position:         spec.Position,
		GuitarPosition:   spec.GuitarPosition,
		Ambience:         spec.Ambience,
		AmbienceLevel:    spec.AmbienceLevel,
		Mode:             spec.Mode,
		NSOn:             spec.NSOn,
		NSThreshold:      spec.NSThreshold,
		NSRelease:        spec.NSRelease,
	}).WithName(name)
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
	patches := b.Patches()
	decoded := make([]map[string]any, 0, len(patches))
	for _, p := range patches {
		params := p.ReadParams()
		decoded = append(decoded, map[string]any{
			"name":                p.Name,
			"amp":                 params.AmpType,
			"gain":                params.AmpGain,
			"volume":              params.AmpVolume,
			"bass":                params.AmpBass,
			"middle":              params.AmpMiddle,
			"treble":              params.AmpTreble,
			"presence":            params.AmpPresence,
			"booster":             params.BoosterType,
			"booster_drive":       params.BoosterDrive,
			"booster_bottom":      params.BoosterBottom,
			"booster_tone":        params.BoosterTone,
			"booster_solo":        params.BoosterSolo,
			"booster_solo_level":  params.BoosterSoloLevel,
			"booster_level":       params.BoosterLevel,
			"booster_direct_mix":  params.BoosterDirectMix,
			"mod":                 params.ModType,
			"mod_params":          params.ModParams,
			"fx":                  params.FXType,
			"fx_params":           params.FXParams,
			"delay":               params.DelayType,
			"delay_time_ms":       params.DelayTime,
			"delay_feedback":      params.DelayFeedback,
			"delay_high_cut":      params.DelayHighCut,
			"delay_level":         params.DelayLevel,
			"delay_direct_mix":    params.DelayDirectMix,
			"reverb":              params.ReverbType,
			"reverb_time_s":       params.ReverbTime,
			"reverb_pre_delay_ms": params.ReverbPreDelay,
			"reverb_level":        params.ReverbLevel,
			"reverb_direct_mix":   params.ReverbDirectMix,
			"position":            params.Position,
			"guitar_position":     params.GuitarPosition,
			"ambience":            params.Ambience,
			"ambience_level":      params.AmbienceLevel,
			"mode":                params.Mode,
			"ns_on":               params.NSOn != nil && *params.NSOn,
			"ns_threshold":        params.NSThreshold,
			"ns_release":          params.NSRelease,
		})
	}
	return marshal(map[string]any{
		"name":       b.Name,
		"device":     b.Device,
		"format_rev": b.FormatRev,
		"patches":    decoded,
	})
}

// wazaSpec builds and resolves a Waza Air tone from the tool arguments.
func (r *Registrar) wazaSpec(args map[string]any) (waza.Spec, error) {
	d := waza.Default()
	return d.Resolve(waza.Spec{
		Name:             argString(args, "name"),
		Amp:              argString(args, "amp"),
		Booster:          argString(args, "booster"),
		Mod:              argString(args, "mod"),
		FX:               argString(args, "fx"),
		Delay:            argString(args, "delay"),
		Reverb:           argString(args, "reverb"),
		CabResonance:     argString(args, "cabinet_resonance"),
		Ambience:         argString(args, "ambience"),
		Position:         argString(args, "position"),
		Mode:             argString(args, "mode"),
		Gain:             int(argFloat(args, "amp_gain")),
		Volume:           int(argFloat(args, "amp_volume")),
		Bass:             int(argFloat(args, "amp_bass")),
		Middle:           int(argFloat(args, "amp_middle")),
		Treble:           int(argFloat(args, "amp_treble")),
		Presence:         int(argFloat(args, "amp_presence")),
		BoosterDrive:     int(argFloat(args, "booster_drive")),
		BoosterBottom:    int(argFloat(args, "booster_bottom")),
		BoosterTone:      int(argFloat(args, "booster_tone")),
		BoosterSolo:      argBool(args, "booster_solo", false),
		BoosterSoloLevel: int(argFloat(args, "booster_solo_level")),
		BoosterLevel:     int(argFloat(args, "booster_level")),
		BoosterDirectMix: int(argFloat(args, "booster_direct_mix")),
		ModParams:        argFloatMap(args, "mod_params"),
		FXParams:         argFloatMap(args, "fx_params"),
		DelayTime:        int(argFloat(args, "delay_time")),
		DelayFeedback:    int(argFloat(args, "delay_feedback")),
		DelayHighCut:     int(argFloat(args, "delay_high_cut")),
		DelayLevel:       int(argFloat(args, "delay_level")),
		DelayDirectMix:   int(argFloat(args, "delay_direct_mix")),
		ReverbTime:       argFloat(args, "reverb_time"),
		ReverbPreDelay:   int(argFloat(args, "reverb_pre_delay")),
		ReverbLevel:      int(argFloat(args, "reverb_level")),
		ReverbDirectMix:  int(argFloat(args, "reverb_direct_mix")),
		GuitarPosition:   int(argFloat(args, "guitar_position")),
		AmbienceLevel:    int(argFloat(args, "ambience_level")),
		NSOn:             argBoolPtr(args, "ns_on"),
		NSThreshold:      int(argFloat(args, "ns_threshold")),
		NSRelease:        int(argFloat(args, "ns_release")),
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

// ---- Quad Cortex device tools ----

func filterQCItems(items []qc.Item, query string) []catalogItem {
	q := strings.ToLower(strings.TrimSpace(query))
	out := make([]catalogItem, 0, len(items))
	for i, it := range items {
		if q != "" && !strings.Contains(strings.ToLower(it.Name+" "+it.InspiredBy), q) {
			continue
		}
		out = append(out, catalogItem{Index: i, ID: it.ID, Name: it.Name, InspiredBy: it.InspiredBy})
	}
	return out
}

func (r *Registrar) qcListAmps(args map[string]any) (string, error) {
	d := qc.Default()
	amps := append(append([]qc.Item{}, d.Amps...), d.BassAmps...)
	return marshal(map[string]any{
		"device": d.Display,
		"amps":   filterQCItems(amps, argString(args, "query")),
	})
}

func (r *Registrar) qcListCabs(args map[string]any) (string, error) {
	d := qc.Default()
	return marshal(map[string]any{
		"device": d.Display,
		"cabs":   filterQCItems(d.Cabs, argString(args, "query")),
	})
}

func (r *Registrar) qcListFX(args map[string]any) (string, error) {
	d := qc.Default()
	category := strings.ToLower(strings.TrimSpace(argString(args, "category")))
	query := argString(args, "query")
	effects := map[string][]catalogItem{}
	for name, items := range d.Effects {
		if category != "" && name != category {
			continue
		}
		effects[name] = filterQCItems(items, query)
	}
	return marshal(map[string]any{"device": d.Display, "effects": effects})
}

func (r *Registrar) qcTranslateAmp(args map[string]any) (string, error) {
	item, err := qc.Default().ResolveAmp(argString(args, "query"))
	if err != nil {
		return "", err
	}
	return marshal(map[string]any{"id": item.ID, "name": item.Name, "based_on": item.InspiredBy})
}

func (r *Registrar) qcTranslateCab(args map[string]any) (string, error) {
	item, err := qc.Default().ResolveCab(argString(args, "query"))
	if err != nil {
		return "", err
	}
	return marshal(map[string]any{"id": item.ID, "name": item.Name, "based_on": item.InspiredBy})
}

// qcParamJSON is one parameter row for qc_list_model_params.
type qcParamJSON struct {
	Index     int      `json:"index"`
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Units     string   `json:"units,omitempty"`
	Min       float64  `json:"min,omitempty"`
	Max       float64  `json:"max,omitempty"`
	Default   float64  `json:"default,omitempty"`
	Steps     int      `json:"steps,omitempty"`
	StepNames []string `json:"step_names,omitempty"`
}

func (r *Registrar) qcListModelParams(args map[string]any) (string, error) {
	d := qc.Default()
	m, ok := d.Catalog.Find(argString(args, "model"))
	if !ok {
		return "", fmt.Errorf("no Quad Cortex model matches %q", argString(args, "model"))
	}
	params := make([]qcParamJSON, 0, len(m.Params))
	for i, p := range m.Params {
		params = append(params, qcParamJSON{
			Index: i, Name: p.Name, Type: p.Type, Units: p.Units,
			Min: p.Min, Max: p.Max, Default: p.Default,
			Steps: p.Steps, StepNames: p.StepNames,
		})
	}
	return marshal(map[string]any{
		"id":       m.ID,
		"name":     m.Name,
		"category": m.Category,
		"based_on": m.BasedOn,
		"params":   params,
	})
}

// qcDecodePreset decrypts and decodes a .pb preset file, then renders it as a
// readable summary with model names and parameter names resolved.
func (r *Registrar) qcDecodePreset(args map[string]any) (string, error) {
	path := argString(args, "path")
	if path == "" {
		return "", fmt.Errorf("a preset file path is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read preset: %w", err)
	}
	preset, err := qc.DecodePreset(argString(args, "serial"), data)
	if err != nil {
		return "", err
	}
	d := qc.Default()
	chains := make([]map[string]any, 0, len(preset.Chains))
	for _, c := range preset.Chains {
		chains = append(chains, renderChain(d.Catalog, c))
	}
	return marshal(map[string]any{
		"name":         preset.Name,
		"author":       preset.AuthorName,
		"volume":       preset.Volume,
		"pan":          preset.Pan,
		"date":         preset.Date,
		"tempo":        preset.Tempo,
		"scene_labels": preset.SceneLabels,
		"chains":       chains,
	})
}

// renderChain renders one grid lane with its models and their parameters.
func renderChain(cat *qc.Catalog, c *qc.Chain) map[string]any {
	models := make([]map[string]any, 0, len(c.Models))
	for _, model := range c.Models {
		models = append(models, renderModel(cat, model))
	}
	return map[string]any{
		"row":      c.GetRow(),
		"in_port":  c.GetInPortid(),
		"out_port": c.GetOutPortid(),
		"models":   models,
	}
}

// renderModel resolves a model's wire hash to its name and renders each
// parameter with its resolved name and (for measured knobs) its real value.
func renderModel(cat *qc.Catalog, model *qc.Model) map[string]any {
	name, id := "?", model.GetHash()
	if m, ok := cat.Model(int(model.GetHash())); ok {
		name, id = m.Name, uint32(m.ID) // #nosec G115 -- catalog model IDs fit in uint32
	}
	params := make([]map[string]any, 0, len(model.Params))
	for _, p := range model.Params {
		params = append(params, renderParam(cat, model, p))
	}
	return map[string]any{"id": id, "name": name, "column": model.GetColumn(), "params": params}
}

func renderParam(cat *qc.Catalog, model *qc.Model, p *qc.Param) map[string]any {
	pname := fmt.Sprintf("param %d", p.GetIndex())
	var val any
	if len(p.ParamValues) > 0 {
		val = p.ParamValues[0].GetFloatValue()
	}
	if m, ok := cat.Model(int(model.GetHash())); ok && int(p.GetIndex()) < len(m.Params) {
		spec := m.Params[p.GetIndex()]
		pname = spec.Name
		if f, ok := val.(float32); ok {
			if real, err := spec.Denormalize(float64(f)); err == nil {
				val = map[string]any{"wire": f, "real": real, "units": spec.Units}
			}
		}
	}
	return map[string]any{"index": p.GetIndex(), "name": pname, "value": val}
}

// qcRenderSetupCard decodes an existing .pb preset and writes a printable HTML
// setup card next to it.
func (r *Registrar) qcRenderSetupCard(args map[string]any) (string, error) {
	path := argString(args, "path")
	if path == "" {
		return "", fmt.Errorf("a preset file path is required")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read preset: %w", err)
	}
	preset, err := qc.DecodePreset(argString(args, "serial"), data)
	if err != nil {
		return "", err
	}
	d := qc.Default()
	outDir := argString(args, "output_dir")
	if outDir == "" {
		outDir = filepath.Dir(path)
	}
	stem := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	cardPath := filepath.Join(outDir, stem+".html")
	if err := os.WriteFile(cardPath, []byte(qc.SetupCardHTML(d.Catalog, preset)), 0o600); err != nil {
		return "", fmt.Errorf("write setup card: %w", err)
	}
	view, err := qc.PresetJSON(d.Catalog, preset)
	if err != nil {
		return "", err
	}
	jsonPath := filepath.Join(outDir, stem+".json")
	if err := os.WriteFile(jsonPath, []byte(view), 0o600); err != nil {
		return "", fmt.Errorf("write preset JSON view: %w", err)
	}
	return marshal(map[string]any{"card": cardPath, "json": jsonPath, "name": preset.Name, "caveat": qc.Caveat})
}

// qcUSB shells out to qcctl for live Quad Cortex control. It refuses to run
// until the user has confirmed, because it talks to a physical unit.
func (r *Registrar) qcUSB(args map[string]any) (string, error) {
	if !argBool(args, "confirm", false) {
		return "", fmt.Errorf("live USB control can change the device: ask the user first (USB-connected Quad Cortex, qcctl installed, Cortex Control quit), then call again with confirm: true")
	}
	cmd := qcctl.Command{
		Sub:     argString(args, "command"),
		Slot:    argString(args, "slot"),
		Scene:   argInt(args, "scene", -1),
		Setlist: argString(args, "setlist"),
	}
	argv, err := cmd.Argv()
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	out, err := qcctl.Run(ctx, cmd)
	if err != nil {
		return "", err
	}
	return marshal(map[string]any{
		"command": strings.Join(argv, " "),
		"output":  out,
	})
}

// qcDesign builds a serial preset (amp, then cab, then the effects in the
// order given) and writes an encrypted .pb file plus a setup card.
func (r *Registrar) qcDesign(args map[string]any) (string, error) {
	spec := qc.DesignSpec{
		Name:   argString(args, "name"),
		Author: argString(args, "author"),
		Volume: argFloat(args, "volume"),
	}
	blocks := []qc.BlockSpec{}
	for _, kind := range []string{"amp", "cab"} {
		if model := argString(args, kind); model != "" {
			blocks = append(blocks, qc.BlockSpec{
				Model:         model,
				Params:        argFloatMap(args, kind+"_params"),
				EncodedParams: argFloatMap(args, kind+"_encoded_params"),
			})
		}
	}
	for _, raw := range argList(args, "fx") {
		item, ok := raw.(map[string]any)
		if !ok {
			return "", fmt.Errorf("fx entries must be objects")
		}
		model := argString(item, "type")
		if model == "" {
			return "", fmt.Errorf("an fx entry needs a \"type\"")
		}
		blocks = append(blocks, qc.BlockSpec{
			Model:         model,
			Params:        argFloatMap(item, "params"),
			EncodedParams: argFloatMap(item, "encoded_params"),
		})
	}
	spec.Blocks = blocks

	outDir := argString(args, "output_dir")
	if outDir == "" {
		outDir = "."
	}
	pbPath, cardPath, jsonPath, err := qc.WritePresetWithCard(argString(args, "serial"), spec, outDir)
	if err != nil {
		return "", err
	}
	return marshal(map[string]any{
		"path":   pbPath,
		"card":   cardPath,
		"json":   jsonPath,
		"name":   spec.Name,
		"blocks": len(blocks),
		"caveat": qc.Caveat,
	})
}

// argList returns the array value of an argument, or nil.
func argList(args map[string]any, key string) []any {
	v, ok := args[key]
	if !ok {
		return nil
	}
	list, ok := v.([]any)
	if !ok {
		return nil
	}
	return list
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
	if err := os.WriteFile(path, []byte(d.SetupCardHTML(resolved)), 0o600); err != nil {
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
		"params":  mooerModuleParamsSchema(),
	})
}

// mooerAmpParamsSchema is the amp knob schema (raw 0-100, 50 = noon).
func mooerAmpParamsSchema() map[string]any {
	return objectSchema(map[string]any{
		"gain":     numberSchema("Amp gain, raw 0-100 (50 = noon)."),
		"bass":     numberSchema("Amp bass, raw 0-100 (50 = noon)."),
		"mid":      numberSchema("Amp middle, raw 0-100 (50 = noon)."),
		"treble":   numberSchema("Amp treble, raw 0-100 (50 = noon)."),
		"presence": numberSchema("Amp presence, raw 0-100 (50 = noon)."),
		"master":   numberSchema("Amp master volume, raw 0-100 (50 = noon)."),
	})
}

// mooerCabParamsSchema is the cab knob schema (raw 0-100, 50 = noon).
func mooerCabParamsSchema() map[string]any {
	return objectSchema(map[string]any{
		"mic":      numberSchema("Cab microphone index."),
		"center":   numberSchema("Mic position, raw 0-100 (50 = center)."),
		"distance": numberSchema("Mic distance, raw 0-100 (50 = noon)."),
		"tube":     numberSchema("Tube power-amp drive, raw 0-100 (50 = noon)."),
	})
}

// mooerModuleParamsSchema is the per-effect knob schema shared by every module.
// All values are the device's raw 0-100 scale (50 = noon) unless noted.
func mooerModuleParamsSchema() map[string]any {
	return objectSchema(map[string]any{
		"gain":        numberSchema("Drive/effect gain, raw 0-100 (50 = noon)."),
		"volume":      numberSchema("Output volume, raw 0-100 (50 = noon)."),
		"tone":        numberSchema("Tone, raw 0-100 (50 = noon)."),
		"level":       numberSchema("Effect level, raw 0-100 (50 = noon)."),
		"rate":        numberSchema("Modulation rate, raw 0-100 (50 = noon)."),
		"depth":       numberSchema("Modulation depth, raw 0-100 (50 = noon)."),
		"feedback":    numberSchema("Delay feedback, raw 0-100 (50 = noon)."),
		"time_ms":     numberSchema("Delay time in milliseconds."),
		"subdivision": numberSchema("Delay subdivision index."),
		"decay":       numberSchema("Reverb decay, raw 0-100 (50 = noon)."),
		"pre_delay":   numberSchema("Reverb pre-delay, raw 0-100 (50 = noon)."),
		"threshold":   numberSchema("Noise-gate threshold, raw 0-100."),
		"attack":      numberSchema("Attack, raw 0-100 (50 = noon)."),
		"release":     numberSchema("Release, raw 0-100 (50 = noon)."),
		"q":           numberSchema("Q, raw 0-100 (50 = noon)."),
		"position":    numberSchema("Position, raw 0-100 (50 = noon)."),
		"peak":        numberSchema("Peak, raw 0-100 (50 = noon)."),
		"band1":       numberSchema("EQ band 1, raw 0-100 (50 = flat)."),
		"band2":       numberSchema("EQ band 2, raw 0-100 (50 = flat)."),
		"band3":       numberSchema("EQ band 3, raw 0-100 (50 = flat)."),
		"band4":       numberSchema("EQ band 4, raw 0-100 (50 = flat)."),
		"band5":       numberSchema("EQ band 5, raw 0-100 (50 = flat)."),
		"band6":       numberSchema("EQ band 6, raw 0-100 (50 = flat)."),
		"band7":       numberSchema("EQ band 7, raw 0-100 (50 = flat)."),
		"band8":       numberSchema("EQ band 8, raw 0-100 (50 = flat)."),
		"band9":       numberSchema("EQ band 9, raw 0-100 (50 = flat)."),
		"band10":      numberSchema("EQ band 10, raw 0-100 (50 = flat)."),
		"band11":      numberSchema("EQ band 11, raw 0-100 (50 = flat)."),
		"band12":      numberSchema("EQ band 12, raw 0-100 (50 = flat)."),
	})
}
