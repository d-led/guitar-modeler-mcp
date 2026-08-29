package presetmap

import (
	"testing"

	"github.com/d-led/guitar-modeler-mcp/internal/catalog"
	"github.com/d-led/guitar-modeler-mcp/internal/mooer"
	"github.com/d-led/guitar-modeler-mcp/internal/rig"
)

func newTable(t *testing.T) *Table {
	t.Helper()
	return NewTable(catalog.New(), mooer.Default())
}

func TestMapAmpGigboardToMooer(t *testing.T) {
	table := newTable(t)
	for _, tc := range []struct {
		gig  string
		want string
	}{
		{"82 Lead 800 100W", "Brit 800"},
		{"67 Black Duo", "Twin Reverb"},
		{"65 Black SR", "Super Reverb"},
		{"92 Treadplate Modern", "Dual Rect"},
	} {
		got, ok := table.MapAmp(DeviceGigboard, tc.gig, DeviceMooer)
		if !ok || got != tc.want {
			t.Fatalf("MapAmp(gigboard %q -> mooer) = %q, %v; want %q", tc.gig, got, ok, tc.want)
		}
	}
}

func TestMapAmpMooerToGigboard(t *testing.T) {
	table := newTable(t)
	for _, tc := range []struct {
		mooer string
		want  string
	}{
		{"Brit 800", "82 Lead 800 50W"},
		{"Twin Reverb", "67 Black Duo"},
		{"Dual Rect", "92 Treadplate Modern"},
	} {
		got, ok := table.MapAmp(DeviceMooer, tc.mooer, DeviceGigboard)
		if !ok || got != tc.want {
			t.Fatalf("MapAmp(mooer %q -> gigboard) = %q, %v; want %q", tc.mooer, got, ok, tc.want)
		}
	}
}

func TestMapAmpUnmapped(t *testing.T) {
	table := newTable(t)
	if _, ok := table.MapAmp(DeviceGigboard, "83 400R", DeviceMooer); ok {
		t.Fatal("expected no mapping for an undocumented gigboard amp")
	}
}

func TestMapCabGigboardToMooer(t *testing.T) {
	table := newTable(t)
	if got, ok := table.MapCab(DeviceGigboard, "4x12 Green 25W", DeviceMooer); !ok || got != "4x12 Green" {
		t.Fatalf("MapCab(gigboard 4x12 Green 25W -> mooer) = %q, %v; want 4x12 Green", got, ok)
	}
}

func TestMapFXGigboardToMooer(t *testing.T) {
	table := newTable(t)
	for _, tc := range []struct {
		gig    string
		module string
		name   string
	}{
		{"Green JRC-OD", "od", "TS808"},
		{"Tape Echo", "delay", "Tape"},
		{"Spring Reverb", "reverb", "Spring"},
		{"Tremolo", "mod", "Tremolo"},
		{"DC Distort", "od", "DS-1"},
	} {
		module, name, ok := table.MapFXGigboardToMooer(tc.gig)
		if !ok || module != tc.module || name != tc.name {
			t.Fatalf("MapFXGigboardToMooer(%q) = (%q, %q, %v); want (%q, %q)", tc.gig, module, name, ok, tc.module, tc.name)
		}
	}
}

func TestMapFXMooerToGigboard(t *testing.T) {
	table := newTable(t)
	if got, ok := table.MapFXMooerToGigboard("od", "TS808"); !ok || got != "Green JRC-OD" {
		t.Fatalf("MapFXMooerToGigboard(od, TS808) = %q, %v; want Green JRC-OD", got, ok)
	}
	if got, ok := table.MapFXMooerToGigboard("delay", "Tape"); !ok || got != "Tape Echo" {
		t.Fatalf("MapFXMooerToGigboard(delay, Tape) = %q, %v; want Tape Echo", got, ok)
	}
}

func TestMooerToGigboard(t *testing.T) {
	table := newTable(t)
	m := mooer.Default()

	ampIndex, _ := m.AmpIndex("Brit 800")
	cabIndex, _ := m.CabIndex("4x12 Green")
	driveIndex, _ := m.EffectIndex("od", "TS808")
	delayIndex, _ := m.EffectIndex("delay", "Tape")
	reverbIndex, _ := m.EffectIndex("reverb", "Spring")

	p := mooer.New()
	p.Name = "Mapped Tone"
	p.Amp = mooer.Amp{Enabled: true, Type: ampIndex}
	p.Cab = mooer.Cab{Enabled: true, Type: cabIndex}
	p.Drive = mooer.Drive{Enabled: true, Type: driveIndex}
	p.Delay = mooer.Delay{Enabled: true, Type: delayIndex}
	p.Reverb = mooer.Reverb{Enabled: true, Type: reverbIndex}

	spec, err := table.MooerToGigboard(p)
	if err != nil {
		t.Fatalf("MooerToGigboard: %v", err)
	}

	types := make([]string, 0, len(spec.Blocks))
	for _, b := range spec.Blocks {
		types = append(types, b.Type)
	}
	expect := []string{"Green JRC-OD", "Amp", "Cab", "Tape Echo", "Spring Reverb"}
	if len(types) != len(expect) {
		t.Fatalf("blocks = %v, want %v", types, expect)
	}
	for i := range expect {
		if types[i] != expect[i] {
			t.Fatalf("block %d = %q, want %q (all: %v)", i, types[i], expect[i], types)
		}
	}

	// The amp block carries the translated Gigboard model.
	var ampParams map[string]any
	for _, b := range spec.Blocks {
		if b.Type == "Amp" {
			ampParams = b.Params
		}
	}
	if ampParams["Type"] != "82 Lead 800 50W" {
		t.Fatalf("mapped amp model = %v, want 82 Lead 800 50W", ampParams["Type"])
	}
}

func TestMooerToGigboardUnmappedAmp(t *testing.T) {
	table := newTable(t)
	p := mooer.New()
	p.Amp = mooer.Amp{Enabled: true, Type: 255} // invalid, unmapped

	if _, err := table.MooerToGigboard(p); err == nil {
		t.Fatal("expected an UnmappedError for an unmappable amp")
	}
}

func TestGigboardToMooer(t *testing.T) {
	cat := catalog.New()
	builder, err := rig.NewBuilder(cat)
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	file, err := builder.Build(rig.Spec{
		Name: "To Mooer",
		Blocks: []rig.Block{
			{Type: "Green JRC-OD", Enabled: true},
			{Type: "Amp", Enabled: true, Params: map[string]any{"Type": "82 Lead 800 100W"}},
			{Type: "Cab", Enabled: true, Params: map[string]any{"CabType": "4x12 Green 25W", "MicType": "Dyn 57"}},
			{Type: "Tape Echo", Enabled: true},
			{Type: "Spring Reverb", Enabled: true},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	table := newTable(t)
	got, err := table.GigboardToMooer(file)
	if err != nil {
		t.Fatalf("GigboardToMooer: %v", err)
	}

	if got.Name != "To Mooer" {
		t.Fatalf("name = %q, want To Mooer", got.Name)
	}
	m := mooer.Default()
	if !got.Amp.Enabled || m.EffectName("amp", got.Amp.Type) != "Brit 800" {
		t.Fatalf("amp = %+v, want enabled Brit 800", got.Amp)
	}
	if !got.Cab.Enabled || m.EffectName("cab", got.Cab.Type) != "4x12 Green" {
		t.Fatalf("cab = %+v, want enabled 4x12 Green", got.Cab)
	}
	if !got.Drive.Enabled || m.EffectName("od", got.Drive.Type) != "TS808" {
		t.Fatalf("drive = %+v, want enabled TS808", got.Drive)
	}
	if !got.Delay.Enabled || m.EffectName("delay", got.Delay.Type) != "Tape" {
		t.Fatalf("delay = %+v, want enabled Tape", got.Delay)
	}
	if !got.Reverb.Enabled || m.EffectName("reverb", got.Reverb.Type) != "Spring" {
		t.Fatalf("reverb = %+v, want enabled Spring", got.Reverb)
	}
}
