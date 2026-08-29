package install

import (
	"encoding/json"
	"testing"
)

func server() Server {
	return Server{Name: "guitar-modeler-mcp", Command: "/opt/bin/hg", Args: []string{"serve"}}
}

func TestApplyInstallOnEmpty(t *testing.T) {
	out, changed, err := applyInstall(nil, "servers", TargetVSCodeUser, server())
	if err != nil {
		t.Fatalf("applyInstall: %v", err)
	}
	if !changed {
		t.Fatal("expected changed on first install")
	}
	var root map[string]any
	if err := json.Unmarshal(out, &root); err != nil {
		t.Fatalf("result is not valid JSON: %v", err)
	}
	servers := root["servers"].(map[string]any)
	entry := servers["guitar-modeler-mcp"].(map[string]any)
	if entry["type"] != "stdio" {
		t.Fatalf("entry type = %v, want stdio", entry["type"])
	}
	if entry["command"] != "/opt/bin/hg" {
		t.Fatalf("entry command = %v", entry["command"])
	}
}

func TestApplyInstallIsIdempotent(t *testing.T) {
	first, _, err := applyInstall(nil, "servers", TargetVSCodeUser, server())
	if err != nil {
		t.Fatalf("first install: %v", err)
	}
	_, changed, err := applyInstall(first, "servers", TargetVSCodeUser, server())
	if err != nil {
		t.Fatalf("second install: %v", err)
	}
	if changed {
		t.Fatal("expected unchanged on second identical install")
	}
}

func TestApplyInstallPreservesOtherServers(t *testing.T) {
	existing := []byte(`{"servers":{"other":{"type":"http","url":"https://example.com"}}}`)
	out, _, err := applyInstall(existing, "servers", TargetVSCodeUser, server())
	if err != nil {
		t.Fatalf("applyInstall: %v", err)
	}
	var root map[string]any
	if err := json.Unmarshal(out, &root); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	servers := root["servers"].(map[string]any)
	if _, ok := servers["other"]; !ok {
		t.Fatal("other server was dropped")
	}
	if _, ok := servers["guitar-modeler-mcp"]; !ok {
		t.Fatal("new server was not added")
	}
}

func TestApplyUninstall(t *testing.T) {
	existing := []byte(`{"servers":{"a":{},"guitar-modeler-mcp":{}}}`)
	out, changed, err := applyUninstall(existing, "servers", "guitar-modeler-mcp")
	if err != nil {
		t.Fatalf("applyUninstall: %v", err)
	}
	if !changed {
		t.Fatal("expected changed")
	}
	var root map[string]any
	if err := json.Unmarshal(out, &root); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	servers := root["servers"].(map[string]any)
	if _, ok := servers["guitar-modeler-mcp"]; ok {
		t.Fatal("entry not removed")
	}
	if _, ok := servers["a"]; !ok {
		t.Fatal("other server was dropped")
	}
}

func TestEntryShapeByTarget(t *testing.T) {
	vscode := entryMap(TargetVSCodeUser, server())
	if vscode["type"] != "stdio" {
		t.Fatalf("vscode entry should include type stdio: %v", vscode)
	}
	claude := entryMap(TargetClaudeDesktop, server())
	if _, ok := claude["type"]; ok {
		t.Fatalf("claude entry should omit type: %v", claude)
	}
	if serversKey(TargetClaudeDesktop) != "mcpServers" {
		t.Fatal("claude should use mcpServers key")
	}
	if serversKey(TargetVSCodeUser) != "servers" {
		t.Fatal("vscode should use servers key")
	}
}
