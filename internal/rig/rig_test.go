package rig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/d-led/guitar-modeler-mcp/internal/catalog"
)

func newTestBuilder(t *testing.T) *Builder {
	t.Helper()
	b, err := NewBuilder(catalog.New())
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	return b
}

func TestBuildRoundTrips(t *testing.T) {
	b := newTestBuilder(t)
	spec := Spec{
		Name:  "Test Rig",
		Tempo: 120,
		Blocks: []Block{
			{Type: "Green JRC-OD", Enabled: true, Params: map[string]any{"Drive": 60.0}},
			{Type: "Amp", Enabled: true, Params: map[string]any{"Type": "65 Black SR", "GainA": 70.0}},
			{Type: "Cab", Enabled: true, Params: map[string]any{"CabType": "1x12 Black Panel Lux", "MicType": "Dyn 57"}},
			{Type: "Tape Echo", Enabled: true},
		},
	}

	file, err := b.Build(spec)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	content, err := file.Decode()
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	patch := content.Data.Patch
	if _, ok := patch.Children["Amp"]; !ok {
		t.Fatal("expected Amp module in patch")
	}
	if _, ok := patch.Children["Cab"]; !ok {
		t.Fatal("expected Cab module in patch")
	}
	if _, ok := patch.Children["Green JRC-OD"]; !ok {
		t.Fatal("expected Green JRC-OD module in patch")
	}

	// Chain slots should reflect the signal order.
	chain := patch.Children["Chain"]
	if got := *chain.Children["ModuleType1"].Str; got != "Green JRC-OD" {
		t.Fatalf("slot 1 = %q, want Green JRC-OD", got)
	}
	if got := *chain.Children["ModuleType2"].Str; got != "Amp" {
		t.Fatalf("slot 2 = %q, want Amp", got)
	}
	if got := *chain.Children["ModuleType3"].Str; got != "Cab" {
		t.Fatalf("slot 3 = %q, want Cab", got)
	}
	if got := *chain.Children["ModuleType4"].Str; got != "Tape Echo" {
		t.Fatalf("slot 4 = %q, want Tape Echo", got)
	}
	if got := *chain.Children["ModuleType5"].Str; got != "Empty Slot" {
		t.Fatalf("slot 5 = %q, want Empty Slot", got)
	}

	// Amp parameters override defaults.
	amp := patch.Children["Amp"]
	if got := *amp.Children["GainA"].Value; got != 70 {
		t.Fatalf("Amp GainA = %v, want 70", got)
	}
	if got := *amp.Children["Type"].Str; got != "65 Black SR" {
		t.Fatalf("Amp Type = %q, want 65 Black SR", got)
	}
}

func TestBuildRejectsMissingAmp(t *testing.T) {
	b := newTestBuilder(t)
	_, err := b.Build(Spec{Name: "x", Blocks: []Block{{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}}}})
	if err == nil {
		t.Fatal("expected error for rig without an Amp")
	}
}

func TestBuildRejectsUnknownModule(t *testing.T) {
	b := newTestBuilder(t)
	_, err := b.Build(Spec{Name: "x", Blocks: []Block{
		{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
		{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
		{Type: "Not A Real FX"},
	}})
	if err == nil {
		t.Fatal("expected error for unknown module")
	}
}

func TestWriteProducesDecodableFile(t *testing.T) {
	b := newTestBuilder(t)
	file, err := b.Build(Spec{
		Name: "Write Test",
		Blocks: []Block{
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	dir := t.TempDir()
	path, err := file.Write(dir)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if filepath.Base(path) != "Write Test.rig" {
		t.Fatalf("filename = %q, want Write Test.rig", filepath.Base(path))
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	var reread RigFile
	if err := json.Unmarshal(raw, &reread); err != nil {
		t.Fatalf("reparse: %v", err)
	}
	if reread.Name() != "Write Test" {
		t.Fatalf("name = %q", reread.Name())
	}
}

func TestDescribeReturnsChainAndParams(t *testing.T) {
	b := newTestBuilder(t)
	file, err := b.Build(Spec{
		Name:         "Describe Test",
		InputGain:    6,
		OutputVolume: 3,
		Blocks: []Block{
			{Type: "Amp", Params: map[string]any{"Type": "82 Lead 800 100W"}},
			{Type: "Cab", Params: map[string]any{"CabType": "4x12 Green 25W", "MicType": "Dyn 57"}},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	summary, err := Describe(file)
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	if summary.Name != "Describe Test" {
		t.Fatalf("name = %q", summary.Name)
	}
	if summary.InputGain != 6 || summary.OutputVolume != 3 {
		t.Fatalf("input_gain/output_volume = %v/%v, want 6/3", summary.InputGain, summary.OutputVolume)
	}
	if len(summary.Modules) != 2 {
		t.Fatalf("modules = %d, want 2", len(summary.Modules))
	}
	if summary.Modules[0].Name != "Amp" || summary.Modules[1].Name != "Cab" {
		t.Fatalf("unexpected module order: %v", summary.Modules)
	}
	if summary.Modules[0].Params["Type"] != "82 Lead 800 100W" {
		t.Fatalf("amp type = %v", summary.Modules[0].Params["Type"])
	}
}

func TestDescribeDecodesSceneSnapshot(t *testing.T) {
	b := newTestBuilder(t)
	file, err := b.Build(Spec{
		Name: "Scene Describe",
		Blocks: []Block{
			{Type: "Green JRC-OD", Enabled: true},
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR"}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux"}},
			{Type: "BBD Delay", Enabled: true},
		},
		Footswitches: []Footswitch{{
			Module: "Green JRC-OD",
			Mode:   "Scene",
			Label:  "DRIVE",
			Scene:  &SceneSnapshot{On: []string{"Green JRC-OD"}, Off: []string{"BBD Delay"}},
		}},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	summary, err := Describe(file)
	if err != nil {
		t.Fatalf("Describe: %v", err)
	}
	if len(summary.Footswitches) != 1 {
		t.Fatalf("footswitches = %d, want 1", len(summary.Footswitches))
	}
	fs := summary.Footswitches[0]
	if fs.Label != "DRIVE" || fs.Mode != "Scene" {
		t.Fatalf("footswitch = %+v, want label DRIVE mode Scene", fs)
	}
	if fs.Scene == nil {
		t.Fatal("scene snapshot not decoded")
	}
	if len(fs.Scene.On) != 1 || fs.Scene.On[0] != "Green JRC-OD" {
		t.Fatalf("scene on = %v, want [Green JRC-OD]", fs.Scene.On)
	}
	if len(fs.Scene.Off) != 1 || fs.Scene.Off[0] != "BBD Delay" {
		t.Fatalf("scene off = %v, want [BBD Delay]", fs.Scene.Off)
	}
}
