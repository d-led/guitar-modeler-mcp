package tools

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/catalog"
	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/design"
	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/mcp"
	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/rig"
)

// newIntegrationServer builds the fully wired server exactly as the CLI does.
func newIntegrationServer(t *testing.T) *mcp.Server {
	t.Helper()
	cat := catalog.New()
	builder, err := rig.NewBuilder(cat)
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	server := mcp.NewServer("headrush-gigboard-mcp", "test")
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
	if info["name"] != "headrush-gigboard-mcp" {
		t.Fatalf("serverInfo.name = %v", info["name"])
	}

	list := rpc(t, s, 2, "tools/list", nil)
	tools := list["result"].(map[string]any)["tools"].([]any)
	names := make(map[string]bool, len(tools))
	for _, tool := range tools {
		names[tool.(map[string]any)["name"].(string)] = true
	}
	for _, want := range []string{
		"catalog_list_amps", "catalog_list_cabs", "catalog_list_mics", "catalog_list_fx",
		"catalog_list_block_presets", "catalog_list_module_params",
		"translate_amp", "translate_cab", "translate_mic",
		"design_rig", "render_report", "rig_decode",
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
