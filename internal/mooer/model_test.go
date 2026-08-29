package mooer

import (
	"strings"
	"testing"
)

func TestModelsRegistry(t *testing.T) {
	models := Models()
	if len(models) != 4 {
		t.Fatalf("Models() returned %d models, want 4", len(models))
	}
	names := make(map[string]bool, len(models))
	for _, m := range models {
		names[m.Name] = true
	}
	for _, want := range []string{"ge150pro", "ge200", "ge150", "ge100pro"} {
		if !names[want] {
			t.Fatalf("Models() missing %q (got %v)", want, names)
		}
	}
}

func TestModelByNameCaseInsensitive(t *testing.T) {
	for _, q := range []string{"ge200", "GE200", "Mooer GE200"} {
		m, ok := ModelByName(q)
		if !ok || m.Name != "ge200" {
			t.Fatalf("ModelByName(%q) = %q, %v; want ge200", q, m.Name, ok)
		}
	}
	if _, ok := ModelByName("nope"); ok {
		t.Fatal("ModelByName found a model that does not exist")
	}
}

func TestGE200CatalogShape(t *testing.T) {
	m, ok := ModelByName("ge200")
	if !ok {
		t.Fatal("ge200 not registered")
	}
	if len(m.Amps) != 55 {
		t.Fatalf("ge200 has %d amps, want 55", len(m.Amps))
	}
	if len(m.Cabs) != 26 {
		t.Fatalf("ge200 has %d cabs, want 26", len(m.Cabs))
	}
	if !m.FileExchange || m.FileExt != ".mo" {
		t.Fatalf("ge200 FileExchange=%v FileExt=%q, want true/.mo", m.FileExchange, m.FileExt)
	}
}

func TestGE200AmpLookup(t *testing.T) {
	m, _ := ModelByName("ge200")

	if got := m.AmpName(6); got != "800" {
		t.Fatalf("ge200 amp[6] = %q, want 800", got)
	}
	if index, ok := m.AmpIndex("UK30 CL"); !ok || index != 32 {
		t.Fatalf("ge200 AmpIndex(UK30 CL) = %d, %v; want 32", index, ok)
	}
	if inspired, ok := m.InspiredAmp("800"); !ok || inspired != "Marshall JCM800" {
		t.Fatalf("ge200 InspiredAmp(800) = %q, %v; want Marshall JCM800", inspired, ok)
	}
}

func TestGE150IsCardOnly(t *testing.T) {
	m, ok := ModelByName("ge150")
	if !ok {
		t.Fatal("ge150 not registered")
	}
	if m.FileExchange {
		t.Fatal("ge150 should not support file exchange (card only)")
	}
}

func TestGE100ProCoreCatalog(t *testing.T) {
	m, ok := ModelByName("ge100pro")
	if !ok {
		t.Fatal("ge100pro not registered")
	}
	if len(m.Amps) != 15 {
		t.Fatalf("ge100pro has %d amps, want 15 core amps", len(m.Amps))
	}
	if len(m.Cabs) != 5 {
		t.Fatalf("ge100pro has %d cabs, want 5 core cabs", len(m.Cabs))
	}
	if !m.FileExchange {
		t.Fatal("ge100pro should support file exchange")
	}
}

func TestDescribeListsNineModulesInOrder(t *testing.T) {
	m, _ := ModelByName("ge200")
	p := New()
	p.Name = "My Card"

	descs := Describe(p, m)
	wantOrder := []string{"FX", "DS/OD", "AMP", "CAB", "NS", "EQ", "MOD", "DELAY", "REVERB"}
	if len(descs) != len(wantOrder) {
		t.Fatalf("Describe returned %d modules, want %d", len(descs), len(wantOrder))
	}
	for i, want := range wantOrder {
		if descs[i].Module != want {
			t.Fatalf("module %d = %q, want %q", i, descs[i].Module, want)
		}
	}
}

func TestDescribeResolvesAmpEffectName(t *testing.T) {
	m, _ := ModelByName("ge200")
	p := New()
	index, _ := m.AmpIndex("800")
	p.Amp = Amp{Enabled: true, Type: index}

	descs := Describe(p, m)
	if descs[2].Effect != "800" || descs[2].InspiredBy != "Marshall JCM800" {
		t.Fatalf("amp desc = %+v", descs[2])
	}
}

func TestSetupCardHTML(t *testing.T) {
	m, _ := ModelByName("ge200")
	p := New()
	p.Name = "Brown Sound"
	index, _ := m.AmpIndex("PLX 100")
	p.Amp = Amp{Enabled: true, Type: index}

	html := SetupCardHTML(m, p)
	for _, want := range []string{"Brown Sound", "Mooer GE200", "setup card", "PLX 100", "Marshall Super Lead Plexi 100", "OFF"} {
		if !strings.Contains(html, want) {
			t.Fatalf("setup card missing %q:\n%s", want, html)
		}
	}
}

func TestResolveAmpByInspiredBy(t *testing.T) {
	m, _ := ModelByName("ge200")
	index, err := m.ResolveAmp("Marshall JCM800")
	if err != nil {
		t.Fatalf("ResolveAmp: %v", err)
	}
	if m.AmpName(index) != "800" {
		t.Fatalf("ResolveAmp(JCM800) = %q, want 800", m.AmpName(index))
	}
}

func TestResolveAmpUnknown(t *testing.T) {
	m, _ := ModelByName("ge200")
	if _, err := m.ResolveAmp("not an amp"); err == nil {
		t.Fatal("expected an error for an unknown amp")
	}
}

func TestBuildPreset(t *testing.T) {
	m, _ := ModelByName("ge200")
	p, err := m.BuildPreset(Spec{
		Name: "Mooer Tone",
		Amp:  "Marshall JCM800",
		Cab:  "1960 412",
		FX: []FXSpec{
			{Module: "od", Type: "808", Enabled: true},
			{Module: "delay", Type: "TAPE", Enabled: true},
			{Module: "reverb", Type: "SPRING", Enabled: false},
		},
	})
	if err != nil {
		t.Fatalf("BuildPreset: %v", err)
	}
	if p.Name != "Mooer Tone" {
		t.Fatalf("name = %q", p.Name)
	}
	if !p.Amp.Enabled || m.AmpName(p.Amp.Type) != "800" {
		t.Fatalf("amp = %+v, want enabled 800", p.Amp)
	}
	if !p.Cab.Enabled || m.CabName(p.Cab.Type) != "1960 412" {
		t.Fatalf("cab = %+v, want enabled 1960 412", p.Cab)
	}
	if !p.Drive.Enabled || m.EffectName("od", p.Drive.Type) != "808" {
		t.Fatalf("drive = %+v, want enabled 808", p.Drive)
	}
	if !p.Delay.Enabled || m.EffectName("delay", p.Delay.Type) != "TAPE" {
		t.Fatalf("delay = %+v, want enabled TAPE", p.Delay)
	}
	if p.Reverb.Enabled {
		t.Fatal("reverb should be disabled")
	}
}

func TestBuildPresetUnknownEffect(t *testing.T) {
	m, _ := ModelByName("ge200")
	if _, err := m.BuildPreset(Spec{Amp: "800", FX: []FXSpec{{Module: "od", Type: "nope"}}}); err == nil {
		t.Fatal("expected an error for an unknown effect")
	}
}
