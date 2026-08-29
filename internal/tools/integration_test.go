package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/d-led/guitar-modeler-mcp/internal/catalog"
	"github.com/d-led/guitar-modeler-mcp/internal/design"
	"github.com/d-led/guitar-modeler-mcp/internal/mcp"
	"github.com/d-led/guitar-modeler-mcp/internal/mooer"
	"github.com/d-led/guitar-modeler-mcp/internal/presetmap"
	"github.com/d-led/guitar-modeler-mcp/internal/rig"
)

// newIntegrationServer builds the fully wired server exactly as the CLI does.
func newIntegrationServer(t *testing.T) *mcp.Server {
	t.Helper()
	cat := catalog.New()
	builder, err := rig.NewBuilder(cat)
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	server := mcp.NewServer("guitar-modeler-mcp", "test")
	NewRegistrar(cat, builder, design.NewDesigner(cat), presetmap.NewTable(cat, mooer.Default())).Register(server)
	return server
}

// rpc performs one JSON-RPC round trip over the stdio transport and returns the
// decoded response map.
func rpc(t *testing.T, s *mcp.Server, id int, method string, params map[string]any) map[string]any {
	t.Helper()
	var in strings.Builder
	in.WriteString(`{"jsonrpc":"2.0","id":` + strconv.Itoa(id) + `,"method":"` + method + `"`)
	if params != nil {
		pb, err := json.Marshal(params)
		if err != nil {
			t.Fatalf("marshal params: %v", err)
		}
		in.WriteString(`,"params":` + string(pb))
	}
	in.WriteString("}\n")

	var out strings.Builder
	if err := s.Run(context.Background(), strings.NewReader(in.String()), &out); err != nil {
		t.Fatalf("Run: %v", err)
	}
	var resp map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(out.String())), &resp); err != nil {
		t.Fatalf("parse response %q: %v", out.String(), err)
	}
	return resp
}

func resultText(t *testing.T, resp map[string]any) string {
	t.Helper()
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("no result in %v", resp)
	}
	content, ok := result["content"].([]any)
	if !ok || len(content) == 0 {
		t.Fatalf("no content in %v", result)
	}
	block := content[0].(map[string]any)
	return block["text"].(string)
}

func TestIntegrationInitializeAndToolList(t *testing.T) {
	s := newIntegrationServer(t)

	init := rpc(t, s, 1, "initialize", nil)
	info := init["result"].(map[string]any)["serverInfo"].(map[string]any)
	if info["name"] != "guitar-modeler-mcp" {
		t.Fatalf("serverInfo.name = %v", info["name"])
	}

	list := rpc(t, s, 2, "tools/list", nil)
	tools := list["result"].(map[string]any)["tools"].([]any)
	names := make(map[string]bool, len(tools))
	for _, tool := range tools {
		names[tool.(map[string]any)["name"].(string)] = true
	}
	for _, want := range []string{
		"get_guide", "get_fx_placement", "search_catalog",
		"catalog_list_amps", "catalog_list_cabs", "catalog_list_mics", "catalog_list_fx",
		"catalog_list_fx_categories", "catalog_list_fx_by_category",
		"catalog_list_block_presets", "catalog_list_module_params",
		"translate_amp", "translate_cab", "translate_mic",
		"design_rig", "render_report", "rig_decode", "estimate_rig_level",
		"create_setlist",
		"device_list", "mooer_catalog_list_amps", "mooer_catalog_list_cabs", "mooer_catalog_list_fx",
		"mooer_design", "render_setup_card", "map_preset",
		"waza_catalog_list_amps", "waza_catalog_list_fx", "waza_setup_card", "waza_write_tsl", "waza_read_tsl",
		"waza_catalog_list_modes",
		"thr_catalog_list_amps", "thr_catalog_list_fx", "thr_setup_card",
		"qc_catalog_list_amps", "qc_catalog_list_cabs", "qc_catalog_list_fx",
		"qc_translate_amp", "qc_translate_cab", "qc_list_model_params",
		"qc_decode_preset", "qc_design",
	} {
		if !names[want] {
			t.Errorf("missing tool %q in tools/list", want)
		}
	}
}

func TestIntegrationCatalogAndTranslate(t *testing.T) {
	s := newIntegrationServer(t)

	amps := resultText(t, rpc(t, s, 1, "tools/call", map[string]any{
		"name":      "catalog_list_amps",
		"arguments": map[string]any{},
	}))
	if !strings.Contains(amps, "82 Lead 800 100W") {
		t.Fatal("catalog_list_amps should contain 82 Lead 800 100W")
	}
	if !strings.Contains(amps, "Marshall") {
		t.Fatal("catalog_list_amps should tag the Marshall brand")
	}

	translated := resultText(t, rpc(t, s, 2, "tools/call", map[string]any{
		"name":      "translate_amp",
		"arguments": map[string]any{"query": "Marshall JCM800"},
	}))
	if !strings.Contains(translated, "Marshall") || !strings.Contains(translated, "JCM800") {
		t.Fatalf("translate_amp result missing brand/model: %s", translated)
	}

	params := resultText(t, rpc(t, s, 3, "tools/call", map[string]any{
		"name":      "catalog_list_module_params",
		"arguments": map[string]any{"type": "Tape Echo"},
	}))
	if !strings.Contains(params, "\"kind\": \"range\"") || !strings.Contains(params, "Feedback") {
		t.Fatalf("catalog_list_module_params missing range/Feedback: %s", params)
	}
}

func TestIntegrationMooerCatalogDesignAndCard(t *testing.T) {
	s := newIntegrationServer(t)
	dir := t.TempDir()

	devices := resultText(t, rpc(t, s, 1, "tools/call", map[string]any{
		"name":      "device_list",
		"arguments": map[string]any{},
	}))
	if !strings.Contains(devices, "ge150") || !strings.Contains(devices, "file_exchange") {
		t.Fatalf("device_list missing ge150/file_exchange: %s", devices)
	}

	amps := resultText(t, rpc(t, s, 2, "tools/call", map[string]any{
		"name":      "mooer_catalog_list_amps",
		"arguments": map[string]any{"model": "ge200"},
	}))
	if !strings.Contains(amps, "800") || !strings.Contains(amps, "Marshall JCM800") {
		t.Fatalf("mooer_catalog_list_amps(ge200) missing 800/JCM800: %s", amps)
	}

	design := resultText(t, rpc(t, s, 3, "tools/call", map[string]any{
		"name": "mooer_design",
		"arguments": map[string]any{
			"model":      "ge200",
			"name":       "Mooer Test",
			"amp":        "Marshall JCM800",
			"cab":        "1960 412",
			"output_dir": dir,
			"fx": []any{
				map[string]any{"module": "od", "type": "808", "enabled": true},
			},
		},
	}))
	if !strings.Contains(design, "Setup card:") || !strings.Contains(design, ".mo") {
		t.Fatalf("mooer_design output missing card/.mo: %s", design)
	}

	mos, err := filepath.Glob(filepath.Join(dir, "*.mo"))
	if err != nil || len(mos) != 1 {
		t.Fatalf("expected one .mo in %s (got %v, err %v)", dir, mos, err)
	}
	cards, err := filepath.Glob(filepath.Join(dir, "*.html"))
	if err != nil || len(cards) != 1 {
		t.Fatalf("expected one setup card .html in %s (got %v, err %v)", dir, cards, err)
	}
	if got := filepath.Base(cards[0]); got != "Mooer Test.ge200.html" {
		t.Fatalf("setup card filename = %q, want Mooer Test.ge200.html", got)
	}
}

func TestIntegrationMooerCardOnlyDevice(t *testing.T) {
	s := newIntegrationServer(t)
	dir := t.TempDir()

	// ge150 is card-only: no .mo is written, only the HTML card.
	out := resultText(t, rpc(t, s, 1, "tools/call", map[string]any{
		"name": "mooer_design",
		"arguments": map[string]any{
			"model":      "ge150",
			"name":       "Card Only",
			"amp":        "65 US TW",
			"output_dir": dir,
		},
	}))
	if !strings.Contains(out, "does not support preset file transfer") {
		t.Fatalf("ge150 design should say card-only: %s", out)
	}
	if mos, _ := filepath.Glob(filepath.Join(dir, "*.mo")); len(mos) != 0 {
		t.Fatalf("ge150 should not write a .mo file, got %v", mos)
	}
	if cards, _ := filepath.Glob(filepath.Join(dir, "*.html")); len(cards) != 1 {
		t.Fatalf("ge150 should write one setup card, got %v", cards)
	}
}

func TestIntegrationWazaTSLAndCard(t *testing.T) {
	s := newIntegrationServer(t)
	dir := t.TempDir()

	// The Waza Air is a .tsl file-exchange device.
	devices := resultText(t, rpc(t, s, 1, "tools/call", map[string]any{
		"name":      "device_list",
		"arguments": map[string]any{},
	}))
	if !strings.Contains(devices, "wazaair") || !strings.Contains(devices, ".tsl") {
		t.Fatalf("device_list missing wazaair/.tsl: %s", devices)
	}

	// Write a backup from the built-in template plus a full tone.
	out := resultText(t, rpc(t, s, 2, "tools/call", map[string]any{
		"name": "waza_write_tsl",
		"arguments": map[string]any{
			"name":         "Brown Practice",
			"output_dir":   dir,
			"amp":          "BROWN",
			"amp_gain":     55,
			"amp_volume":   68,
			"amp_bass":     42,
			"amp_middle":   50,
			"amp_treble":   60,
			"booster":      "T-SCREAM",
			"delay":        "TAPE ECHO",
			"delay_time":   380,
			"reverb":       "HALL REVERB",
			"reverb_level": 45,
		},
	}))
	if !strings.Contains(out, ".tsl") {
		t.Fatalf("waza_write_tsl output missing .tsl path: %s", out)
	}

	tsls, err := filepath.Glob(filepath.Join(dir, "*.tsl"))
	if err != nil || len(tsls) != 1 {
		t.Fatalf("expected one .tsl in %s (got %v, err %v)", dir, tsls, err)
	}

	// Read it back: the name and the applied parameter values must round-trip.
	read := resultText(t, rpc(t, s, 3, "tools/call", map[string]any{
		"name":      "waza_read_tsl",
		"arguments": map[string]any{"input_file": tsls[0]},
	}))
	for _, want := range []string{
		"\"device\": \"WAZA-AIR\"", "Brown Practice",
		"\"amp\": \"BROWN\"", "\"gain\": 55", "\"volume\": 68", "\"bass\": 42",
		"\"middle\": 50", "\"treble\": 60",
		"\"booster\": \"T-SCREAM\"", "\"delay\": \"TAPE ECHO\"", "\"delay_time_ms\": 380",
		"\"reverb\": \"HALL REVERB\"", "\"reverb_level\": 45",
	} {
		if !strings.Contains(read, want) {
			t.Fatalf("waza_read_tsl missing %q:\n%s", want, read)
		}
	}

	// The setup card is still produced.
	card := resultText(t, rpc(t, s, 4, "tools/call", map[string]any{
		"name": "waza_setup_card",
		"arguments": map[string]any{
			"name":       "Brown Practice",
			"amp":        "BROWN",
			"booster":    "T-SCREAM",
			"output_dir": dir,
		},
	}))
	if !strings.Contains(card, "setup card") || !strings.Contains(card, ".wazaair.html") {
		t.Fatalf("waza_setup_card unexpected output: %s", card)
	}

	// The AIRSTEP BW modes are listed and can be printed on the card.
	modes := resultText(t, rpc(t, s, 5, "tools/call", map[string]any{
		"name":      "waza_catalog_list_modes",
		"arguments": map[string]any{},
	}))
	if !strings.Contains(modes, "airstep-bw") || !strings.Contains(modes, "Toggle DELAY") || !strings.Contains(modes, "CH 6") {
		t.Fatalf("waza_catalog_list_modes missing AIRSTEP bindings: %s", modes)
	}

	cardWithMode := resultText(t, rpc(t, s, 6, "tools/call", map[string]any{
		"name": "waza_setup_card",
		"arguments": map[string]any{
			"name":         "Scene Brown",
			"amp":          "BROWN",
			"booster":      "T-SCREAM",
			"airstep_mode": 3,
			"output_dir":   dir,
		},
	}))
	if !strings.Contains(cardWithMode, "setup card") {
		t.Fatalf("waza_setup_card(airstep_mode) unexpected output: %s", cardWithMode)
	}

	// A bad mode is rejected rather than silently ignored.
	bad := resultText(t, rpc(t, s, 7, "tools/call", map[string]any{
		"name":      "waza_setup_card",
		"arguments": map[string]any{"name": "X", "amp": "BROWN", "airstep_mode": 9, "output_dir": dir},
	}))
	if !strings.Contains(bad, "unknown AIRSTEP BW mode 9") {
		t.Fatalf("airstep_mode 9 should be rejected, got %q", bad)
	}
}

func TestIntegrationWazaMultiPatchBackup(t *testing.T) {
	s := newIntegrationServer(t)
	dir := t.TempDir()

	// Pack two named scenes into one backup; the file is named after the backup.
	out := resultText(t, rpc(t, s, 1, "tools/call", map[string]any{
		"name": "waza_write_tsl",
		"arguments": map[string]any{
			"name":       "Always with Me",
			"output_dir": dir,
			"patches": []any{
				map[string]any{"name": "DRIVE", "amp": "CRUNCH", "amp_gain": 58, "booster": "T-SCREAM", "delay": "ANALOG DELAY", "delay_time": 380},
				map[string]any{"name": "CLEAN", "amp": "CLEAN", "amp_gain": 30, "mod": "CHORUS", "reverb": "HALL REVERB"},
			},
		},
	}))
	if !strings.Contains(out, "Always with Me.tsl") {
		t.Fatalf("waza_write_tsl output should use the backup name, got %q", out)
	}

	tsls, err := filepath.Glob(filepath.Join(dir, "*.tsl"))
	if err != nil || len(tsls) != 1 {
		t.Fatalf("expected exactly one .tsl in %s (got %v, err %v)", dir, tsls, err)
	}

	read := resultText(t, rpc(t, s, 2, "tools/call", map[string]any{
		"name":      "waza_read_tsl",
		"arguments": map[string]any{"input_file": tsls[0]},
	}))
	for _, want := range []string{
		"\"name\": \"Always with Me\"",
		"\"name\": \"DRIVE\"", "\"amp\": \"CRUNCH\"", "\"delay\": \"ANALOG DELAY\"",
		"\"name\": \"CLEAN\"", "\"amp\": \"CLEAN\"", "\"reverb\": \"HALL REVERB\"",
	} {
		if !strings.Contains(read, want) {
			t.Fatalf("waza_read_tsl missing %q:\n%s", want, read)
		}
	}
}

func TestIntegrationThrSetupCard(t *testing.T) {
	s := newIntegrationServer(t)
	dir := t.TempDir()

	// THR is card-only, listed alongside the other devices.
	devices := resultText(t, rpc(t, s, 1, "tools/call", map[string]any{
		"name":      "device_list",
		"arguments": map[string]any{},
	}))
	if !strings.Contains(devices, "Yamaha THR-II") || !strings.Contains(devices, "thr10") {
		t.Fatalf("device_list missing THR models: %s", devices)
	}

	// The amp catalog exposes the official 24-cell grid with descriptions.
	amps := resultText(t, rpc(t, s, 2, "tools/call", map[string]any{
		"name":      "thr_catalog_list_amps",
		"arguments": map[string]any{"query": "twin"},
	}))
	if !strings.Contains(amps, "CLEAN CLASSIC") || !strings.Contains(amps, "Fender Twin Reverb") {
		t.Fatalf("thr_catalog_list_amps missing CLEAN CLASSIC/Twin: %s", amps)
	}

	// The setup card resolves a description to an amp and writes the file.
	card := resultText(t, rpc(t, s, 3, "tools/call", map[string]any{
		"name": "thr_setup_card",
		"arguments": map[string]any{
			"name":          "THR Clean",
			"amp":           "Twin Reverb",
			"cab":           "California 1x12",
			"mod":           "CHORUS",
			"echo":          "Tape",
			"reverb":        "Hall",
			"gain":          42,
			"master":        68,
			"bass":          50,
			"mid":           45,
			"treble":        60,
			"echo_time":     380,
			"echo_feedback": 32,
			"echo_mix":      24,
			"reverb_level":  40,
			"reverb_decay":  55,
			"compressor":    true,
			"output_dir":    dir,
		},
	}))
	if !strings.Contains(card, ".thr.html") {
		t.Fatalf("thr_setup_card output missing .thr.html: %s", card)
	}
	cards, _ := filepath.Glob(filepath.Join(dir, "*.html"))
	if len(cards) != 1 {
		t.Fatalf("expected one .html card in %s, got %v", dir, cards)
	}
	body, err := os.ReadFile(cards[0])
	if err != nil {
		t.Fatalf("read setup card: %v", err)
	}
	for _, want := range []string{"Gain: 42", "Master: 68", "Time (ms): 380", "Level: 40", "Decay: 55"} {
		if !strings.Contains(string(body), want) {
			t.Fatalf("setup card missing %q:\n%s", want, body)
		}
	}

	// The effects catalog now includes cabinets, echo and reverb types.
	fx := resultText(t, rpc(t, s, 4, "tools/call", map[string]any{
		"name":      "thr_catalog_list_fx",
		"arguments": map[string]any{},
	}))
	if !strings.Contains(fx, "Brown 4x12") || !strings.Contains(fx, "Digital Delay") || !strings.Contains(fx, "Spring") {
		t.Fatalf("thr_catalog_list_fx missing cabs/echo/reverb: %s", fx)
	}
}

func TestIntegrationDesignDecodeReportRoundTrip(t *testing.T) {
	s := newIntegrationServer(t)
	dir := t.TempDir()

	design := resultText(t, rpc(t, s, 1, "tools/call", map[string]any{
		"name": "design_rig",
		"arguments": map[string]any{
			"name":       "Integration Rig",
			"song":       "Test Song",
			"amp":        "Fender Twin Reverb",
			"output_dir": dir,
			"fx": []any{
				map[string]any{"type": "Spring Reverb", "enabled": true},
			},
		},
	}))
	if !strings.Contains(design, "Rig file:") {
		t.Fatalf("design_rig output missing rig path: %s", design)
	}

	rigs, err := filepath.Glob(filepath.Join(dir, "*.rig"))
	if err != nil || len(rigs) != 1 {
		t.Fatalf("expected one .rig in %s (got %v, err %v)", dir, rigs, err)
	}
	if _, err := filepath.Glob(filepath.Join(dir, "*.html")); err != nil {
		t.Fatalf("expected .html report: %v", err)
	}

	decoded := resultText(t, rpc(t, s, 2, "tools/call", map[string]any{
		"name":      "rig_decode",
		"arguments": map[string]any{"rig_file": rigs[0]},
	}))
	if !strings.Contains(decoded, "\"Amp\"") || !strings.Contains(decoded, "\"Cab\"") {
		t.Fatalf("rig_decode missing Amp/Cab modules: %s", decoded)
	}
	if !strings.Contains(decoded, "\"name\": \"Integration Rig\"") {
		t.Fatalf("rig_decode missing rig name: %s", decoded)
	}

	report := resultText(t, rpc(t, s, 3, "tools/call", map[string]any{
		"name":      "render_report",
		"arguments": map[string]any{"rig_file": rigs[0]},
	}))
	if !strings.Contains(report, "Wrote report") {
		t.Fatalf("render_report unexpected output: %s", report)
	}
}

func TestIntegrationToolErrorReturnsIsError(t *testing.T) {
	s := newIntegrationServer(t)
	resp := rpc(t, s, 1, "tools/call", map[string]any{
		"name":      "translate_amp",
		"arguments": map[string]any{"query": "zzzz not an amp zzzz"},
	})
	// An empty match returns an empty JSON array, not an error — verify shape.
	if !strings.Contains(resultText(t, resp), "[]") {
		t.Fatalf("expected empty result for non-matching query")
	}
}

func TestIntegrationDesignDualAmpParallel(t *testing.T) {
	s := newIntegrationServer(t)
	dir := t.TempDir()

	out := resultText(t, rpc(t, s, 1, "tools/call", map[string]any{
		"name": "design_rig",
		"arguments": map[string]any{
			"name":       "Two Heads",
			"amp":        "65 Black SR",
			"amp2":       "67 Black Duo",
			"routing":    "SPS-1",
			"output_dir": dir,
		},
	}))
	if !strings.Contains(out, "Rig file:") {
		t.Fatalf("design_rig output missing rig path: %s", out)
	}

	rigs, err := filepath.Glob(filepath.Join(dir, "*.rig"))
	if err != nil || len(rigs) != 1 {
		t.Fatalf("expected one .rig, got %v (%v)", rigs, err)
	}

	decoded := resultText(t, rpc(t, s, 2, "tools/call", map[string]any{
		"name":      "rig_decode",
		"arguments": map[string]any{"rig_file": rigs[0]},
	}))
	if !strings.Contains(decoded, `"routing": "SPS-1"`) {
		t.Fatalf("rig_decode missing SPS-1 routing: %s", decoded)
	}
	if !strings.Contains(decoded, `"Amp 2"`) {
		t.Fatalf("rig_decode missing Amp 2 module: %s", decoded)
	}
}

func TestIntegrationDesignFootswitches(t *testing.T) {
	s := newIntegrationServer(t)
	dir := t.TempDir()

	out := resultText(t, rpc(t, s, 1, "tools/call", map[string]any{
		"name": "design_rig",
		"arguments": map[string]any{
			"name":       "Whammy Toe",
			"amp":        "65 Black SR",
			"output_dir": dir,
			"fx": []any{
				map[string]any{"type": "Wham", "enabled": true},
			},
			"footswitches": []any{
				map[string]any{"module": "Wham"},
			},
		},
	}))
	if !strings.Contains(out, "Rig file:") {
		t.Fatalf("design_rig output missing rig path: %s", out)
	}
	if !strings.Contains(out, "Footswitches: FS5=Wham (On)") {
		t.Fatalf("design_rig output missing footswitch summary: %s", out)
	}

	rigs, err := filepath.Glob(filepath.Join(dir, "*.rig"))
	if err != nil || len(rigs) != 1 {
		t.Fatalf("expected one .rig, got %v (%v)", rigs, err)
	}

	decoded := resultText(t, rpc(t, s, 2, "tools/call", map[string]any{
		"name":      "rig_decode",
		"arguments": map[string]any{"rig_file": rigs[0]},
	}))
	if !strings.Contains(decoded, `"switch": "FS5"`) || !strings.Contains(decoded, `"module": "Wham"`) {
		t.Fatalf("rig_decode missing FS5/Wham footswitch: %s", decoded)
	}
	if !strings.Contains(decoded, `"operation": "On"`) {
		t.Fatalf("rig_decode missing default On operation: %s", decoded)
	}
}

func TestIntegrationDesignReportsUnassignedFootswitches(t *testing.T) {
	s := newIntegrationServer(t)
	dir := t.TempDir()

	out := resultText(t, rpc(t, s, 1, "tools/call", map[string]any{
		"name": "design_rig",
		"arguments": map[string]any{
			"name":       "No Switch",
			"amp":        "65 Black SR",
			"output_dir": dir,
			"fx": []any{
				map[string]any{"type": "Wham", "enabled": true},
			},
		},
	}))
	if !strings.Contains(out, "Footswitches: none assigned") {
		t.Fatalf("design_rig output should warn about unassigned footswitches: %s", out)
	}
	if !strings.Contains(out, "Wham has no footswitch") {
		t.Fatalf("design_rig output should hint that Wham needs a footswitch: %s", out)
	}
}

func TestIntegrationDesignFootswitchRejectsUnknownModule(t *testing.T) {
	s := newIntegrationServer(t)
	dir := t.TempDir()

	resp := rpc(t, s, 1, "tools/call", map[string]any{
		"name": "design_rig",
		"arguments": map[string]any{
			"name":         "Bad Switch",
			"amp":          "65 Black SR",
			"output_dir":   dir,
			"footswitches": []any{map[string]any{"module": "Not In Chain"}},
		},
	})
	result, ok := resp["result"].(map[string]any)
	if !ok {
		t.Fatalf("expected a result with isError, got: %v", resp)
	}
	if isErr, _ := result["isError"].(bool); !isErr {
		t.Fatalf("expected isError for an unknown footswitch module, got: %v", resp)
	}
}

func TestIntegrationCreateSetlist(t *testing.T) {
	s := newIntegrationServer(t)
	dir := t.TempDir()

	design := func(id int, name string) {
		out := resultText(t, rpc(t, s, id, "tools/call", map[string]any{
			"name": "design_rig",
			"arguments": map[string]any{
				"name":       name,
				"amp":        "65 Black SR",
				"output_dir": dir,
			},
		}))
		if !strings.Contains(out, "Rig file:") {
			t.Fatalf("design_rig output missing rig path: %s", out)
		}
	}
	design(1, "Clean")
	design(2, "Drive")

	rigs, err := filepath.Glob(filepath.Join(dir, "*.rig"))
	if err != nil || len(rigs) != 2 {
		t.Fatalf("expected two .rig files, got %v (%v)", rigs, err)
	}

	out := resultText(t, rpc(t, s, 3, "tools/call", map[string]any{
		"name": "create_setlist",
		"arguments": map[string]any{
			"name":       "Song",
			"rig_files":  []any{rigs[0], rigs[1]},
			"output_dir": dir,
		},
	}))
	if !strings.Contains(out, "Setlist \"Song\"") {
		t.Fatalf("create_setlist output missing setlist name: %s", out)
	}

	setlists, err := filepath.Glob(filepath.Join(dir, "*.setlist"))
	if err != nil || len(setlists) != 1 {
		t.Fatalf("expected one .setlist, got %v (%v)", setlists, err)
	}
	data, err := os.ReadFile(setlists[0])
	if err != nil {
		t.Fatalf("read setlist: %v", err)
	}
	var parsed struct {
		RigNames []string `json:"rig_names"`
		Rigs     []string `json:"rigs"`
	}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("parse setlist: %v", err)
	}
	if len(parsed.Rigs) != 2 || len(parsed.RigNames) != 2 {
		t.Fatalf("setlist should reference two rigs: %+v", parsed)
	}
}

func TestIntegrationGuideAndFxCategories(t *testing.T) {
	s := newIntegrationServer(t)

	guide := resultText(t, rpc(t, s, 1, "tools/call", map[string]any{
		"name":      "get_guide",
		"arguments": map[string]any{},
	}))
	if !strings.Contains(guide, "SPS-1") || !strings.Contains(guide, "parallel") {
		t.Fatalf("get_guide missing routing guidance: %s", guide)
	}

	categories := resultText(t, rpc(t, s, 2, "tools/call", map[string]any{
		"name":      "catalog_list_fx_categories",
		"arguments": map[string]any{},
	}))
	for _, cat := range []string{"distortion", "delay", "reverb", "modulation", "utility"} {
		if !strings.Contains(categories, cat) {
			t.Fatalf("catalog_list_fx_categories missing %q: %s", cat, categories)
		}
	}

	delays := resultText(t, rpc(t, s, 3, "tools/call", map[string]any{
		"name":      "catalog_list_fx_by_category",
		"arguments": map[string]any{"category": "delay"},
	}))
	if !strings.Contains(delays, "Tape Echo") || !strings.Contains(delays, "BBD Delay") {
		t.Fatalf("delay category missing Tape Echo/BBD Delay: %s", delays)
	}
	if strings.Contains(delays, "Spring Reverb") {
		t.Fatalf("delay category should not contain Spring Reverb: %s", delays)
	}

	// An unknown category is an error, not a silent empty list.
	resp := rpc(t, s, 4, "tools/call", map[string]any{
		"name":      "catalog_list_fx_by_category",
		"arguments": map[string]any{"category": "bogus"},
	})
	result := resp["result"].(map[string]any)
	if isErr, _ := result["isError"].(bool); !isErr {
		t.Fatalf("expected isError for unknown category, got: %v", resp)
	}
}

func TestIntegrationQuadCortexDesignAndDecode(t *testing.T) {
	s := newIntegrationServer(t)

	// Translate a real amp description to the exact QC model and its wire id.
	amp := resultText(t, rpc(t, s, 1, "tools/call", map[string]any{
		"name":      "qc_translate_amp",
		"arguments": map[string]any{"query": "JCM800"},
	}))
	if !strings.Contains(amp, "Marshall JCM800") || !strings.Contains(amp, `"id": 1001`) {
		t.Fatalf("qc_translate_amp(JCM800) = %s, want Marshall JCM800 id 1001", amp)
	}

	// The model's parameters expose the screen scale for the design step.
	params := resultText(t, rpc(t, s, 2, "tools/call", map[string]any{
		"name":      "qc_list_model_params",
		"arguments": map[string]any{"model": "JCM800"},
	}))
	if !strings.Contains(params, "GAIN") {
		t.Fatalf("qc_list_model_params(JCM800) missing GAIN: %s", params)
	}

	// Design a serial preset and read it back.
	dir := t.TempDir()
	design := resultText(t, rpc(t, s, 3, "tools/call", map[string]any{
		"name": "qc_design",
		"arguments": map[string]any{
			"name":       "QC Integration Tone",
			"serial":     "QA00XXXXX",
			"amp":        "JCM800",
			"amp_params": map[string]any{"GAIN": 5},
			"cab":        "Mesa Rectifier",
			"fx": []any{
				map[string]any{"type": "TS808", "params": map[string]any{"OVERDRIVE": 4}},
			},
			"output_dir": dir,
		},
	}))
	var designed map[string]any
	if err := json.Unmarshal([]byte(design), &designed); err != nil {
		t.Fatalf("qc_design output not JSON: %v", design)
	}
	path, _ := designed["path"].(string)
	if path == "" {
		t.Fatalf("qc_design returned no path: %s", design)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("qc_design file missing: %v", err)
	}

	decoded := resultText(t, rpc(t, s, 4, "tools/call", map[string]any{
		"name":      "qc_decode_preset",
		"arguments": map[string]any{"path": path, "serial": "QA00XXXXX"},
	}))
	for _, want := range []string{"QC Integration Tone", "Marshall JCM800", "TS808", "GAIN"} {
		if !strings.Contains(decoded, want) {
			t.Fatalf("qc_decode_preset missing %q: %s", want, decoded)
		}
	}

	// A wrong serial must refuse to decode.
	resp := rpc(t, s, 5, "tools/call", map[string]any{
		"name":      "qc_decode_preset",
		"arguments": map[string]any{"path": path, "serial": "QB99YYYYY"},
	})
	result := resp["result"].(map[string]any)
	if isErr, _ := result["isError"].(bool); !isErr {
		t.Fatalf("expected isError for wrong serial, got: %v", resp)
	}
}
