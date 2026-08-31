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
	wantEq(t, "install changed", changed, true)
	wantEq(t, "install path", path, filepath.Join(".vscode", "mcp.json"))

	// The file lands under the changed working directory and carries the server.
	assertConfigHasServers(t, filepath.Join(dir, path))

	// Installing the identical server again is a no-op.
	assertInstallNoop(t)

	// Uninstall removes it and reports a change.
	_, changed, err = Uninstall(TargetVSCodeWorkspace, server().Name)
	if err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	wantEq(t, "uninstall changed", changed, true)
	assertServerAbsent(t, filepath.Join(dir, path))

	// Uninstalling again is a no-op.
	assertUninstallNoop(t)
}

func assertConfigHasServers(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
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
}

func assertServerAbsent(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config after uninstall: %v", err)
	}
	if strings.Contains(string(data), server().Name) {
		t.Fatalf("server still present after uninstall: %s", data)
	}
}

func assertInstallNoop(t *testing.T) {
	t.Helper()
	_, changed, err := Install(TargetVSCodeWorkspace, server())
	if err != nil {
		t.Fatalf("second Install: %v", err)
	}
	wantEq(t, "second install changed", changed, false)
}

func assertUninstallNoop(t *testing.T) {
	t.Helper()
	_, changed, err := Uninstall(TargetVSCodeWorkspace, server().Name)
	if err != nil {
		t.Fatalf("second Uninstall: %v", err)
	}
	wantEq(t, "second uninstall changed", changed, false)
}

func wantEq[T comparable](t *testing.T, name string, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %v, want %v", name, got, want)
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

// TestUninstallMissingConfigIsNoop guards the idempotency of Uninstall: with no
// config file on disk it must be a silent no-op, not an error.
func TestUninstallMissingConfigIsNoop(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	path, changed, err := Uninstall(TargetVSCodeWorkspace, server().Name)
	if err != nil {
		t.Fatalf("Uninstall on missing config: %v", err)
	}
	if changed {
		t.Fatal("Uninstall on a missing config reported a change")
	}
	if path != filepath.Join(".vscode", "mcp.json") {
		t.Fatalf("Uninstall path = %q", path)
	}
}

// TestInstallUninstallInstallCycle proves the operations round-trip cleanly:
// install, remove, then install again leaves the same entry as the first time.
func TestInstallUninstallInstallCycle(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if _, _, err := Install(TargetVSCodeWorkspace, server()); err != nil {
		t.Fatalf("first Install: %v", err)
	}
	if _, _, err := Uninstall(TargetVSCodeWorkspace, server().Name); err != nil {
		t.Fatalf("Uninstall: %v", err)
	}
	_, changed, err := Install(TargetVSCodeWorkspace, server())
	if err != nil {
		t.Fatalf("re-Install: %v", err)
	}
	if !changed {
		t.Fatal("re-Install after Uninstall reported no change")
	}

	data, err := os.ReadFile(filepath.Join(dir, ".vscode", "mcp.json"))
	if err != nil {
		t.Fatalf("read config after cycle: %v", err)
	}
	if !strings.Contains(string(data), server().Name) {
		t.Fatalf("server missing after install/uninstall/install cycle: %s", data)
	}
}
