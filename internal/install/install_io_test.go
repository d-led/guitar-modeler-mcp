package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidTarget(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want Target
		ok   bool
	}{
		{"vscode", TargetVSCodeUser, true},
		{"workspace", TargetVSCodeWorkspace, true},
		{"claude", TargetClaudeDesktop, true},
		{"nope", "", false},
		{"", "", false},
	} {
		got, ok := ValidTarget(tc.in)
		if got != tc.want || ok != tc.ok {
			t.Fatalf("ValidTarget(%q) = %q, %v; want %q, %v", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}

func TestVSCodeUserDirDarwin(t *testing.T) {
	got := vscodeUserDir("/home/u", "darwin")
	want := filepath.Join("/home/u", "Library", "Application Support", "Code", "User")
	if got != want {
		t.Fatalf("vscodeUserDir(darwin) = %q, want %q", got, want)
	}
}

func TestVSCodeUserDirLinux(t *testing.T) {
	got := vscodeUserDir("/home/u", "linux")
	want := filepath.Join("/home/u", ".config", "Code", "User")
	if got != want {
		t.Fatalf("vscodeUserDir(linux) = %q, want %q", got, want)
	}
}

func TestVSCodeUserDirWindowsWithAppData(t *testing.T) {
	t.Setenv("APPDATA", "C:\\Users\\u\\AppData\\Roaming")
	got := vscodeUserDir("", "windows")
	if !strings.HasSuffix(got, "User") || !strings.Contains(got, "Code") {
		t.Fatalf("vscodeUserDir(windows, APPDATA) = %q", got)
	}
}

func TestVSCodeUserDirWindowsFallback(t *testing.T) {
	t.Setenv("APPDATA", "")
	home := "C:\\Users\\u"
	got := vscodeUserDir(home, "windows")
	if !strings.HasPrefix(got, home) || !strings.Contains(got, "Roaming") {
		t.Fatalf("vscodeUserDir(windows fallback) = %q", got)
	}
}

func TestClaudeConfigPathDarwin(t *testing.T) {
	got := claudeConfigPath("/home/u", "darwin")
	want := filepath.Join("/home/u", "Library", "Application Support", "Claude", "claude_desktop_config.json")
	if got != want {
		t.Fatalf("claudeConfigPath(darwin) = %q, want %q", got, want)
	}
}

func TestClaudeConfigPathLinux(t *testing.T) {
	got := claudeConfigPath("/home/u", "linux")
	want := filepath.Join("/home/u", ".config", "Claude", "claude_desktop_config.json")
	if got != want {
		t.Fatalf("claudeConfigPath(linux) = %q, want %q", got, want)
	}
}

func TestClaudeConfigPathWindowsFallback(t *testing.T) {
	t.Setenv("APPDATA", "")
	home := "C:\\Users\\u"
	got := claudeConfigPath(home, "windows")
	if !strings.HasPrefix(got, home) || !strings.Contains(got, "claude_desktop_config.json") {
		t.Fatalf("claudeConfigPath(windows fallback) = %q", got)
	}
}

func TestConfigPathWorkspace(t *testing.T) {
	got, err := ConfigPath(TargetVSCodeWorkspace)
	if err != nil {
		t.Fatalf("ConfigPath(workspace): %v", err)
	}
	if got != filepath.Join(".vscode", "mcp.json") {
		t.Fatalf("ConfigPath(workspace) = %q", got)
	}
}

func TestConfigPathUserAndClaude(t *testing.T) {
	for _, target := range []Target{TargetVSCodeUser, TargetClaudeDesktop} {
		got, err := ConfigPath(target)
		if err != nil {
			t.Fatalf("ConfigPath(%q): %v", target, err)
		}
		if got == "" {
			t.Fatalf("ConfigPath(%q) returned an empty path", target)
		}
	}
}

func TestConfigPathUnknownTarget(t *testing.T) {
	if _, err := ConfigPath(Target("bogus")); err == nil {
		t.Fatal("expected an error for an unknown target")
	}
}

func TestApplyInstallRejectsInvalidExisting(t *testing.T) {
	if _, _, err := applyInstall([]byte("{invalid"), "servers", TargetVSCodeUser, server()); err == nil {
		t.Fatal("expected a parse error for invalid existing config")
	}
}

func TestApplyUninstallNoop(t *testing.T) {
	out, changed, err := applyUninstall([]byte(`{"servers":{"other":{}}}`), "servers", "missing")
	if err != nil {
		t.Fatalf("applyUninstall: %v", err)
	}
	if changed || out != nil {
		t.Fatalf("applyUninstall(missing) = changed=%v out=%v, want no-op", changed, out)
	}

	out, changed, err = applyUninstall(nil, "servers", "missing")
	if err != nil || changed || out != nil {
		t.Fatalf("applyUninstall(nil) = changed=%v out=%v err=%v, want no-op", changed, out, err)
	}
}

func TestInstallAndUninstallWorkspace(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	path, changed, err := Install(TargetVSCodeWorkspace, server())
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	if !changed {
		t.Fatal("Install reported unchanged on a fresh config")
	}
	if path != filepath.Join(".vscode", "mcp.json") {
		t.Fatalf("Install path = %q", path)
	}

	// The file lands under the changed working directory.
	abs := filepath.Join(dir, path)
	data, err := os.ReadFile(abs)
	if err != nil {
		t.Fatalf("read installed config: %v", err)
	}
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		t.Fatalf("installed config is not valid JSON: %v", err)
	}
	if root["servers"] == nil {
		t.Fatal("installed config missing servers key")
	}

	// Installing the identical server again is a no-op.
	if _, changed, err := Install(TargetVSCodeWorkspace, server()); err != nil || changed {
		t.Fatalf("second Install = changed=%v err=%v, want unchanged", changed, err)
	}

	// Uninstall removes it and reports a change.
	_, changed, err = Uninstall(TargetVSCodeWorkspace, server().Name)
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	if !changed {
		t.Fatal("Uninstall reported unchanged")
	}
	data, err = os.ReadFile(abs)
	if err != nil {
		t.Fatalf("read config after uninstall: %v", err)
	}
	if strings.Contains(string(data), server().Name) {
		t.Fatalf("server still present after uninstall: %s", data)
	}

	// Uninstalling again is a no-op.
	_, changed, err = Uninstall(TargetVSCodeWorkspace, server().Name)
	if err != nil || changed {
		t.Fatalf("second Uninstall = changed=%v err=%v, want unchanged", changed, err)
	}
}

func TestRenderReportsUnchanged(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	// Install writes the config; Render only computes the merged JSON, so a
	// second Render over the written file must report no change.
	if _, _, err := Install(TargetVSCodeWorkspace, server()); err != nil {
		t.Fatalf("Install: %v", err)
	}
	_, _, changed, err := Render(TargetVSCodeWorkspace, server())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if changed {
		t.Fatal("Render reported changed for an unchanged entry")
	}
}

func TestRenderProducesMergedJSON(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	out, path, changed, err := Render(TargetVSCodeWorkspace, server())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !changed {
		t.Fatal("Render reported unchanged for a missing config")
	}
	if path != filepath.Join(".vscode", "mcp.json") {
		t.Fatalf("Render path = %q", path)
	}
	var root map[string]any
	if err := json.Unmarshal(out, &root); err != nil {
		t.Fatalf("Render output is not valid JSON: %v", err)
	}
	if root["servers"] == nil {
		t.Fatal("Render output missing servers key")
	}
}
