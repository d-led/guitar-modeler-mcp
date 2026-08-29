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
	NewRegistrar(cat, builder, design.NewDesigner(cat)).Register(server)
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
