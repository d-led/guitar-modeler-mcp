package rig

import (
	"strconv"
	"strings"
	"testing"
)

func ampBlock(model string) Block {
	return Block{Type: "Amp", Enabled: true, Params: map[string]any{"Type": model}}
}

func cabBlock(cab string) Block {
	return Block{Type: "Cab", Enabled: true, Params: map[string]any{"CabType": cab, "MicType": "Dyn 57"}}
}

func TestBuildSPSDualAmpLaysOutFixedSlots(t *testing.T) {
	b := newTestBuilder(t)
	file, err := b.Build(Spec{
		Name:    "Dual Amp",
		Routing: RoutingSPS,
		Prefix: []Block{
			{Type: "Green JRC-OD", Enabled: true},
		},
		PathA: []Block{ampBlock("67 Black Duo"), cabBlock("1x12 Black Panel Lux")},
		PathB: []Block{ampBlock("85 M-2 Lead"), cabBlock("4x12 Green 25W")},
		Suffix: []Block{
			{Type: "Eleven Reverb", Enabled: true},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	content, err := file.Decode()
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	chain := content.Data.Patch.Children["Chain"]

	// SPS-1: slots 1-3 prefix, 4-6 path A, 7-9 path B, 10-11 suffix.
	want := []string{
		"Green JRC-OD", "Empty Slot", "Empty Slot", // prefix (1 block + 2 spare)
		"Amp", "Cab", "Empty Slot", // path A (2 blocks + 1 spare)
		"Amp 2", "Cab 2", "Empty Slot", // path B
		"Eleven Reverb", "Empty Slot", // suffix
	}
	for i := 1; i <= 11; i++ {
		got := *chain.Children["ModuleType"+strconv.Itoa(i)].Str
		if got != want[i-1] {
			t.Fatalf("slot %d = %q, want %q (full: %v)", i, got, want[i-1], want)
		}
	}

	if got := *chain.Children["Routing"].Str; got != "SPS-1" {
		t.Fatalf("Routing = %q, want SPS-1", got)
	}

	// The two amps must be distinct nodes with their own models.
	if _, ok := content.Data.Patch.Children["Amp 2"]; !ok {
		t.Fatal("expected an Amp 2 node for the dual-amp path")
	}
	ampA := content.Data.Patch.Children["Amp"]
	ampB := content.Data.Patch.Children["Amp 2"]
	if got := *ampA.Children["Type"].Str; got != "67 Black Duo" {
		t.Fatalf("Amp type = %q", got)
	}
	if got := *ampB.Children["Type"].Str; got != "85 M-2 Lead" {
		t.Fatalf("Amp 2 type = %q", got)
	}
}

func TestBuildSPSSharedAmpSplitsIntoParallelPaths(t *testing.T) {
	b := newTestBuilder(t)
	file, err := b.Build(Spec{
		Name:    "Wet Dry",
		Routing: RoutingSPS,
		Prefix: []Block{
			ampBlock("65 Black SR"),
			cabBlock("1x12 Black Panel Lux"),
		},
		PathA: []Block{{Type: "Tape Echo", Enabled: true}},
		PathB: []Block{{Type: "Eleven Reverb", Enabled: true}},
		Suffix: []Block{
			{Type: "Volume", Enabled: true},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	content, err := file.Decode()
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	chain := content.Data.Patch.Children["Chain"]

	want := []string{
		"Amp", "Cab", "Empty Slot", // shared amp+cab before the split
		"Tape Echo", "Empty Slot", "Empty Slot", // path A
		"Eleven Reverb", "Empty Slot", "Empty Slot", // path B
		"Volume", "Empty Slot", // suffix
	}
	for i := 1; i <= 11; i++ {
		got := *chain.Children["ModuleType"+strconv.Itoa(i)].Str
		if got != want[i-1] {
			t.Fatalf("slot %d = %q, want %q", i, got, want[i-1])
		}
	}

	// A single Amp node: the same amp feeds both paths.
	if _, ok := content.Data.Patch.Children["Amp 2"]; ok {
		t.Fatal("shared-amp rig must not create an Amp 2 node")
	}
}

func TestBuildPSParallelFromInput(t *testing.T) {
	b := newTestBuilder(t)
	file, err := b.Build(Spec{
		Name:    "Parallel In",
		Routing: RoutingPS,
		PathA: []Block{
			{Type: "Green JRC-OD", Enabled: true},
			ampBlock("67 Black Duo"),
			cabBlock("1x12 Black Panel Lux"),
		},
		PathB: []Block{ampBlock("85 M-2 Lead"), cabBlock("4x12 Green 25W")},
		Suffix: []Block{
			{Type: "Eleven Reverb", Enabled: true},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	content, err := file.Decode()
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	chain := content.Data.Patch.Children["Chain"]
	if got := *chain.Children["Routing"].Str; got != "PS-1" {
		t.Fatalf("Routing = %q, want PS-1", got)
	}
	want := []string{
		"Green JRC-OD", "Amp", "Cab", // path A
		"Amp 2", "Cab 2", "Empty Slot", "Empty Slot", "Empty Slot", // path B (2 blocks + 3 spare)
		"Eleven Reverb", "Empty Slot", "Empty Slot", // suffix
	}
	for i := 1; i <= 11; i++ {
		got := *chain.Children["ModuleType"+strconv.Itoa(i)].Str
		if got != want[i-1] {
			t.Fatalf("slot %d = %q, want %q", i, got, want[i-1])
		}
	}
}

func TestParallelPathMixControlsAreWritten(t *testing.T) {
	b := newTestBuilder(t)
	level := -8.0
	pan := -100.0
	delay := 12.0
	file, err := b.Build(Spec{
		Name:       "Panned",
		Routing:    RoutingSPS,
		Prefix:     []Block{ampBlock("65 Black SR"), cabBlock("1x12 Black Panel Lux")},
		PathA:      []Block{{Type: "Tape Echo", Enabled: true}},
		PathB:      []Block{{Type: "Eleven Reverb", Enabled: true}},
		Para1Level: &level,
		Para1Pan:   &pan,
		ParaDelay:  &delay,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	content, err := file.Decode()
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	chain := content.Data.Patch.Children["Chain"]
	if got := *chain.Children["Para1Level"].Value; got != -8 {
		t.Fatalf("Para1Level = %v", got)
	}
	if got := *chain.Children["Para1Pan"].Value; got != -100 {
		t.Fatalf("Para1Pan = %v", got)
	}
	if got := *chain.Children["ParaDelay"].Value; got != 12 {
		t.Fatalf("ParaDelay = %v", got)
	}
	// The Mix node mirrors the path mix values.
	mix := content.Data.Patch.Children["Mix"]
	if got := *mix.Children["Para1Level"].Value; got != -8 {
		t.Fatalf("Mix Para1Level = %v", got)
	}
}

func TestParallelValidationRejectsOverfullPath(t *testing.T) {
	b := newTestBuilder(t)
	_, err := b.Build(Spec{
		Name:    "Too Many",
		Routing: RoutingSPS,
		Prefix:  []Block{ampBlock("65 Black SR"), cabBlock("1x12 Black Panel Lux")},
		PathA: []Block{
			{Type: "Tape Echo"}, {Type: "Chorus"}, {Type: "Flanger"}, {Type: "Tremolo"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "path A has 4 blocks") {
		t.Fatalf("expected path A overfull error, got %v", err)
	}
}

func TestParallelValidationRejectsUnknownRouting(t *testing.T) {
	b := newTestBuilder(t)
	_, err := b.Build(Spec{Name: "x", Routing: "banana", Blocks: validBase()})
	if err == nil || !strings.Contains(err.Error(), "unknown routing") {
		t.Fatalf("expected unknown routing error, got %v", err)
	}
}

func TestParallelValidationRejectsOutOfRangePan(t *testing.T) {
	b := newTestBuilder(t)
	pan := 150.0
	_, err := b.Build(Spec{
		Name:     "Bad Pan",
		Routing:  RoutingSPS,
		Prefix:   []Block{ampBlock("65 Black SR"), cabBlock("1x12 Black Panel Lux")},
		PathA:    []Block{{Type: "Tape Echo"}},
		Para1Pan: &pan,
	})
	if err == nil || !strings.Contains(err.Error(), "Para1Pan") {
		t.Fatalf("expected Para1Pan range error, got %v", err)
	}
}

func TestParallelValidationRequiresPrefixAmpOrPathAmp(t *testing.T) {
	b := newTestBuilder(t)
	_, err := b.Build(Spec{
		Name:    "No Amp",
		Routing: RoutingSPS,
		Prefix:  []Block{{Type: "Tape Echo"}},
		PathA:   []Block{{Type: "Chorus"}},
	})
	if err == nil || !strings.Contains(err.Error(), "Amp") {
		t.Fatalf("expected missing Amp error, got %v", err)
	}
}

func TestDescribeIncludesMixerAndRouting(t *testing.T) {
	b := newTestBuilder(t)
	level := -8.0
	panA := -100.0
	panB := 100.0
	delay := 12.0
	file, err := b.Build(Spec{
		Name:       "Panned",
		Routing:    RoutingSPS,
		Prefix:     []Block{ampBlock("65 Black SR"), cabBlock("1x12 Black Panel Lux")},
		PathA:      []Block{{Type: "Tape Echo", Enabled: true}},
		PathB:      []Block{{Type: "Eleven Reverb", Enabled: true}},
		Para1Level: &level,
		Para1Pan:   &panA,
		Para2Pan:   &panB,
		ParaDelay:  &delay,
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	sum, err := Describe(file)
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	if sum.Routing != "SPS-1" {
		t.Fatalf("Routing = %q, want SPS-1", sum.Routing)
	}
	if sum.Mixer.Para1Pan != -100 || sum.Mixer.Para2Pan != 100 {
		t.Fatalf("mixer pans = %v/%v, want -100/100", sum.Mixer.Para1Pan, sum.Mixer.Para2Pan)
	}
	if sum.Mixer.Para1Level != -8 {
		t.Fatalf("mixer Para1Level = %v, want -8", sum.Mixer.Para1Level)
	}
	if sum.Mixer.ParaDelay != 12 {
		t.Fatalf("mixer ParaDelay = %v, want 12", sum.Mixer.ParaDelay)
	}
}
