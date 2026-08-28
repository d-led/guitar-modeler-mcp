// Package install writes the MCP server registration into a client's config
// file (VS Code user profile, VS Code workspace, or Claude Desktop). It merges
// with any existing servers instead of clobbering the file.
package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

// Target selects which client config to modify.
type Target string

const (
	// TargetVSCodeUser is the VS Code user profile: available in all workspaces.
	TargetVSCodeUser Target = "vscode"
	// TargetVSCodeWorkspace is the local .vscode/mcp.json of the current folder.
	TargetVSCodeWorkspace Target = "workspace"
	// TargetClaudeDesktop is the Claude Desktop app config.
	TargetClaudeDesktop Target = "claude"
)

// ValidTarget maps a CLI flag value to a Target.
func ValidTarget(s string) (Target, bool) {
	switch s {
	case "vscode":
		return TargetVSCodeUser, true
	case "workspace":
		return TargetVSCodeWorkspace, true
	case "claude":
		return TargetClaudeDesktop, true
	}
	return "", false
}

// Server is the stdio server to register.
type Server struct {
	Name    string
	Command string
	Args    []string
}

// ConfigPath resolves the config file path for a target.
func ConfigPath(t Target) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	goos := runtime.GOOS
	switch t {
	case TargetVSCodeUser:
		return filepath.Join(vscodeUserDir(home, goos), "mcp.json"), nil
	case TargetVSCodeWorkspace:
		return filepath.Join(".vscode", "mcp.json"), nil
	case TargetClaudeDesktop:
		return claudeConfigPath(home, goos), nil
	}
	return "", fmt.Errorf("unknown target %q", t)
}

// Render produces the merged config JSON for a target without writing it.
// It returns the JSON bytes, the target path and whether the entry changed.
func Render(t Target, s Server) ([]byte, string, bool, error) {
	path, err := ConfigPath(t)
	if err != nil {
		return nil, "", false, err
	}
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, "", false, fmt.Errorf("read config: %w", err)
	}
	out, changed, err := applyInstall(existing, serversKey(t), t, s)
	if err != nil {
		return nil, "", false, err
	}
	return out, path, changed, nil
}

// Install merges the server into the target config file and returns the path.
func Install(t Target, s Server) (string, bool, error) {
	out, path, changed, err := Render(t, s)
	if err != nil {
		return "", false, err
	}
	if !changed {
		return path, false, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", false, err
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return "", false, fmt.Errorf("write config: %w", err)
	}
	return path, true, nil
}

// Uninstall removes the server entry from the target config file.
func Uninstall(t Target, name string) (string, bool, error) {
	path, err := ConfigPath(t)
	if err != nil {
		return "", false, err
	}
	existing, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return path, false, nil
		}
		return "", false, fmt.Errorf("read config: %w", err)
	}
	out, changed, err := applyUninstall(existing, serversKey(t), name)
	if err != nil {
		return "", false, err
	}
	if !changed {
		return path, false, nil
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return "", false, fmt.Errorf("write config: %w", err)
	}
	return path, true, nil
}

func serversKey(t Target) string {
	if t == TargetClaudeDesktop {
		return "mcpServers"
	}
	return "servers"
}

func applyInstall(existing []byte, key string, t Target, s Server) ([]byte, bool, error) {
	root := map[string]any{}
	if len(existing) > 0 {
		if err := json.Unmarshal(existing, &root); err != nil {
			return nil, false, fmt.Errorf("parse existing config: %w", err)
		}
	}
	servers, _ := root[key].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	entry := entryMap(t, s)
	if old, ok := servers[s.Name].(map[string]any); ok && jsonEqual(old, entry) {
		return nil, false, nil
	}
	servers[s.Name] = entry
	root[key] = servers
	out, err := marshal(root)
	if err != nil {
		return nil, false, err
	}
	return out, true, nil
}

func applyUninstall(existing []byte, key, name string) ([]byte, bool, error) {
	if len(existing) == 0 {
		return nil, false, nil
	}
	var root map[string]any
	if err := json.Unmarshal(existing, &root); err != nil {
		return nil, false, fmt.Errorf("parse existing config: %w", err)
	}
	servers, _ := root[key].(map[string]any)
	if _, ok := servers[name]; !ok {
		return nil, false, nil
	}
	delete(servers, name)
	if len(servers) == 0 {
		delete(root, key)
	} else {
		root[key] = servers
	}
	out, err := marshal(root)
	if err != nil {
		return nil, false, err
	}
	return out, true, nil
}

func entryMap(t Target, s Server) map[string]any {
	entry := map[string]any{"command": s.Command}
	if len(s.Args) > 0 {
		args := make([]any, len(s.Args))
		for i, a := range s.Args {
			args[i] = a
		}
		entry["args"] = args
	}
	if t != TargetClaudeDesktop {
		entry["type"] = "stdio"
	}
	return entry
}

func marshal(root map[string]any) ([]byte, error) {
	out, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

func jsonEqual(a, b any) bool {
	ab, _ := json.Marshal(a)
	bb, _ := json.Marshal(b)
	return string(ab) == string(bb)
}

func vscodeUserDir(home, goos string) string {
	switch goos {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Code", "User")
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "Code", "User")
		}
		return filepath.Join(home, "AppData", "Roaming", "Code", "User")
	default:
		return filepath.Join(home, ".config", "Code", "User")
	}
}

func claudeConfigPath(home, goos string) string {
	switch goos {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "Claude", "claude_desktop_config.json")
		}
		return filepath.Join(home, "AppData", "Roaming", "Claude", "claude_desktop_config.json")
	default:
		return filepath.Join(home, ".config", "Claude", "claude_desktop_config.json")
	}
}
