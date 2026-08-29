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
	devices := supportedDevices()
	if len(devices) != 11 {
		t.Fatalf("supportedDevices returned %d devices, want 11", len(devices))
	}

	byName := make(map[string]deviceInfo, len(devices))
	for _, d := range devices {
		byName[d.Name] = d
	}

	if g := byName["gigboard"]; !g.FileExchange || g.FileExt != ".rig" {
		t.Fatalf("gigboard entry = %+v", g)
	}
	if g := byName["ge200"]; !g.FileExchange || g.FileExt != ".mo" {
		t.Fatalf("ge200 entry = %+v", g)
	}
	if g := byName["ge150"]; g.FileExchange || g.FileExt != "" {
		t.Fatalf("ge150 entry = %+v, want card-only", g)
	}
	if g := byName["ge100pro"]; !g.FileExchange || g.FileExt != ".mo" {
		t.Fatalf("ge100pro entry = %+v", g)
	}
	if g := byName["wazaair"]; !g.FileExchange || g.FileExt != ".tsl" || g.Description != "BOSS Waza Air" {
		t.Fatalf("wazaair entry = %+v, want .tsl file exchange", g)
	}
	if g := byName["thr"]; g.FileExchange || g.Description != "Yamaha THR-II" {
		t.Fatalf("thr entry = %+v, want card-only", g)
	}
	if g := byName["quad-cortex"]; !g.FileExchange || g.FileExt != ".pb" || g.Description != "Neural DSP Quad Cortex" {
		t.Fatalf("quad-cortex entry = %+v, want .pb file exchange", g)
	}
}
