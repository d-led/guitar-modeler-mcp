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

// TestFileExchangeMatrix pins the GE150 Pro vs classic GE150 distinction: the
// .mo preset format belongs to the file-capable models only. The classic
// GE150 has no USB preset transfer, so only a setup card may be written for it.
func TestFileExchangeMatrix(t *testing.T) {
	for _, tc := range []struct {
		name string
		want bool
	}{
		{"ge150pro", true},
		{"ge200", true},
		{"ge150", false},
		{"ge100pro", true},
	} {
		m, ok := ModelByName(tc.name)
		if !ok {
			t.Fatalf("model %q not registered", tc.name)
		}
		if m.FileExchange != tc.want {
			t.Fatalf("%s FileExchange = %v, want %v", tc.name, m.FileExchange, tc.want)
		}
		if tc.want && m.FileExt != ".mo" {
			t.Fatalf("%s FileExt = %q, want .mo", tc.name, m.FileExt)
		}
		if !tc.want && m.FileExt != "" {
			t.Fatalf("%s should have no FileExt, got %q", tc.name, m.FileExt)
		}
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

// TestSetupCardHighlightsNonDefaultParams guards the cue that a knob was
// dialled away from its resting value: the dialled value is highlighted, a
// noon/neutral value is not.
func TestSetupCardHighlightsNonDefaultParams(t *testing.T) {
	m, _ := ModelByName("ge200")
	p := New()
	p.Name = "Dialed"
	index, _ := m.AmpIndex("PLX 100")
	p.Amp = Amp{Enabled: true, Type: index, Gain: 80, Bass: noon, Mid: noon, Treble: noon, Presence: noon, Master: noon}

	html := SetupCardHTML(m, p)
	if !strings.Contains(html, `<span class="hl">Gain: 80</span>`) {
		t.Fatal("expected the dialled Gain to be highlighted")
	}
	if strings.Contains(html, `<span class="hl">Bass:`) {
		t.Fatal("expected the noon Bass to stay unhighlighted")
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
	wantEq(t, "name", p.Name, "Mooer Tone")
	wantEq(t, "amp enabled", p.Amp.Enabled, true)
	wantEq(t, "amp name", m.AmpName(p.Amp.Type), "800")
	wantEq(t, "cab enabled", p.Cab.Enabled, true)
	wantEq(t, "cab name", m.CabName(p.Cab.Type), "1960 412")
	wantEq(t, "drive enabled", p.Drive.Enabled, true)
	wantEq(t, "drive name", m.EffectName("od", p.Drive.Type), "808")
	wantEq(t, "delay enabled", p.Delay.Enabled, true)
	wantEq(t, "delay name", m.EffectName("delay", p.Delay.Type), "TAPE")
	wantEq(t, "reverb enabled", p.Reverb.Enabled, false)
}

func TestBuildPresetUnknownEffect(t *testing.T) {
	m, _ := ModelByName("ge200")
	if _, err := m.BuildPreset(Spec{Amp: "800", FX: []FXSpec{{Module: "od", Type: "nope"}}}); err == nil {
		t.Fatal("expected an error for an unknown effect")
	}
}

func TestBuildPresetAppliesNeutralDefaults(t *testing.T) {
	m, _ := ModelByName("ge200")
	p, err := m.BuildPreset(Spec{Name: "Neutral", Amp: "800", Cab: "1960 412"})
	if err != nil {
		t.Fatalf("BuildPreset: %v", err)
	}

	// Amount knobs land on noon (50), selectors on the first option (0).
	if p.Amp.Gain != 50 || p.Amp.Master != 50 || p.Amp.Treble != 50 {
		t.Fatalf("amp values = %+v, want noon (50)", p.Amp)
	}
	if p.Cab.Mic != 0 || p.Cab.Center != 50 {
		t.Fatalf("cab values = %+v, want mic 0 / noon", p.Cab)
	}
}

// TestBuildPresetAppliesParams proves amp, cab and per-module raw knob values
// reach the correct module fields (the gap that previously forced every preset
// out at neutral 128).
func TestBuildPresetAppliesParams(t *testing.T) {
	m, _ := ModelByName("ge200")
	p, err := m.BuildPreset(Spec{
		Name:      "MoP Rhythm",
		Amp:       "MARK III DS",
		AmpParams: Params{"gain": 75, "bass": 70, "mid": 25, "treble": 75, "presence": 65, "master": 78},
		Cab:       "REC 412",
		CabParams: Params{"mic": 1, "distance": 90},
		FX: []FXSpec{
			{Module: "od", Type: "808", Enabled: true, Params: Params{"Gain": 20, "Tone": 70, "Volume": 85}},
			{Module: "ns", Type: "NOISE GATE", Enabled: true, Params: Params{"Threshold": 40}},
			{Module: "eq", Type: "EQ-G", Enabled: true, Params: Params{"band1": 60, "band3": 46}},
			{Module: "delay", Type: "DIGITAL", Enabled: true, Params: Params{"Time (ms)": 400, "feedback": 76}},
		},
	})
	if err != nil {
		t.Fatalf("BuildPreset: %v", err)
	}
	wantEq(t, "amp gain", p.Amp.Gain, 75)
	wantEq(t, "amp bass", p.Amp.Bass, 70)
	wantEq(t, "amp mid", p.Amp.Mid, 25)
	wantEq(t, "amp treble", p.Amp.Treble, 75)
	wantEq(t, "amp presence", p.Amp.Presence, 65)
	wantEq(t, "amp master", p.Amp.Master, 78)
	wantEq(t, "cab mic", p.Cab.Mic, 1)
	wantEq(t, "cab distance", p.Cab.Distance, 90)
	wantEq(t, "drive gain", p.Drive.Gain, 20)
	wantEq(t, "drive tone", p.Drive.Tone, 70)
	wantEq(t, "drive volume", p.Drive.Volume, 85)
	wantEq(t, "noise gate enabled", p.NoiseGate.Enabled, true)
	wantEq(t, "noise gate threshold", p.NoiseGate.Threshold, 40)
	wantEq(t, "eq enabled", p.EQ.Enabled, true)
	wantEq(t, "eq band1", p.EQ.Bands[0], 60)
	wantEq(t, "eq band3", p.EQ.Bands[2], 46)
	wantEq(t, "delay time", p.Delay.TimeMS, 400)
	wantEq(t, "delay feedback", p.Delay.Feedback, 76)
	// Unspecified knobs stay at their neutral value, not zero.
	wantEq(t, "cab center", p.Cab.Center, 50)
	wantEq(t, "cab tube", p.Cab.Tube, 50)
}

// wantEq fails the test when got differs from want, with the checked field
// named in the message so each assertion reads as a sentence.
func wantEq[T comparable](t *testing.T, name string, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
}

func TestSetModuleNeutralDelay(t *testing.T) {
	p := New()
	SetModule(&p, "delay", 2, true)

	if !p.Delay.Enabled || p.Delay.Type != 2 {
		t.Fatalf("delay = %+v, want enabled type 2", p.Delay)
	}
	if p.Delay.Level != 50 || p.Delay.Feedback != 50 || p.Delay.TimeMS != 400 || p.Delay.Subdivision != 0 {
		t.Fatalf("delay values = %+v, want noon/400ms/subdivision 0", p.Delay)
	}
}
