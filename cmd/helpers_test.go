package cmd

import "testing"

func TestInferDevice(t *testing.T) {
	for _, tc := range []struct {
		path string
		want string
	}{
		{"tone.mo", "mooer"},
		{"TONE.MO", "mooer"},
		{"tone.rig", "gigboard"},
		{"no-ext", "gigboard"},
	} {
		if got := inferDevice(tc.path); got != tc.want {
			t.Fatalf("inferDevice(%q) = %q, want %q", tc.path, got, tc.want)
		}
	}
}

func TestSanitizeName(t *testing.T) {
	if got := sanitizeName("Brown Sound"); got != "Brown Sound" {
		t.Fatalf("sanitizeName kept %q", got)
	}
	if got := sanitizeName("a/b"); got != "a-b" {
		t.Fatalf("sanitizeName = %q, want a-b", got)
	}
	if got := sanitizeName(""); got != "preset" {
		t.Fatalf("sanitizeName(\"\") = %q, want preset", got)
	}
}

func TestFirstArg(t *testing.T) {
	if got := firstArg([]string{"one", "two"}); got != "one" {
		t.Fatalf("firstArg = %q, want one", got)
	}
	if got := firstArg(nil); got != "" {
		t.Fatalf("firstArg(nil) = %q, want empty", got)
	}
}

func TestParseFXFlags(t *testing.T) {
	fx, err := parseFXFlags(`[{"type":"Tape Echo","enabled":true}]`)
	if err != nil {
		t.Fatalf("parseFXFlags: %v", err)
	}
	if len(fx) != 1 || fx[0].Type != "Tape Echo" || !fx[0].Enabled {
		t.Fatalf("parseFXFlags = %+v", fx)
	}
	if fx, err := parseFXFlags(""); err != nil || fx != nil {
		t.Fatalf("parseFXFlags(\"\") = %v, %v; want nil, nil", fx, err)
	}
	if _, err := parseFXFlags("not json"); err == nil {
		t.Fatal("parseFXFlags accepted invalid JSON")
	}
}

func TestParseFootswitchFlags(t *testing.T) {
	sw, err := parseFootswitchFlags(`[{"module":"Wham"}]`)
	if err != nil {
		t.Fatalf("parseFootswitchFlags: %v", err)
	}
	if len(sw) != 1 || sw[0].Module != "Wham" {
		t.Fatalf("parseFootswitchFlags = %+v", sw)
	}
	if _, err := parseFootswitchFlags("bad"); err == nil {
		t.Fatal("parseFootswitchFlags accepted invalid JSON")
	}
}

func TestSupportedDevices(t *testing.T) {
	byName := make(map[string]deviceInfo, 13)
	for _, d := range supportedDevices() {
		byName[d.Name] = d
	}
	if len(byName) != 13 {
		t.Fatalf("supportedDevices returned %d devices, want 13", len(byName))
	}

	tests := []struct {
		name   string
		fileEx bool
		ext    string
		desc   string
	}{
		{"gigboard", true, ".rig", ""},
		{"ge200", true, ".mo", ""},
		{"ge150", false, "", ""},
		{"ge100pro", true, ".mo", ""},
		{"wazaair", true, ".tsl", "BOSS Waza Air"},
		{"thr", false, "", "Yamaha THR-II"},
		{"quad-cortex", false, "", "Neural DSP Quad Cortex"},
		{"gp200", true, ".prst", "Valeton GP-200"},
		{"gp200lt", true, ".prst", "Valeton GP-200 LT"},
	}
	for _, tc := range tests {
		g, ok := byName[tc.name]
		if !ok {
			t.Errorf("device %q missing from supportedDevices", tc.name)
			continue
		}
		if g.FileExchange != tc.fileEx || g.FileExt != tc.ext {
			t.Errorf("%s = %+v, want FileExchange=%v FileExt=%q", tc.name, g, tc.fileEx, tc.ext)
		}
		if tc.desc != "" && g.Description != tc.desc {
			t.Errorf("%s Description = %q, want %q", tc.name, g.Description, tc.desc)
		}
	}
}
