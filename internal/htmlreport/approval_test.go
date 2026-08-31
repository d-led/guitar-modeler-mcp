package htmlreport

import (
	"regexp"
	"strings"
	"testing"

	"github.com/d-led/guitar-modeler-mcp/internal/catalog"
	"github.com/d-led/guitar-modeler-mcp/internal/golden"
	"github.com/d-led/guitar-modeler-mcp/internal/rig"
)

// TestReportSnapshot approves the HTML report for a representative rig. The
// generation timestamp is pinned so the snapshot is stable.
func TestReportSnapshot(t *testing.T) {
	b, err := rig.NewBuilder(catalog.New())
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	file, err := b.Build(rig.Spec{
		Name: "Report Rig",
		Blocks: []rig.Block{
			{Type: "Amp", Params: map[string]any{"Type": "82 Lead 800 100W"}},
			{Type: "Cab", Params: map[string]any{"CabType": "4x12 Green 25W", "MicType": "Dyn 57"}},
			{Type: "Spring Reverb", Enabled: true},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	html, err := Render(file, "Test Note", catalog.New())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	html = regexp.MustCompile(`Generated \d{4}-\d{2}-\d{2} \d{2}:\d{2}`).ReplaceAllString(html, "Generated FIXED")

	golden.Assert(t, "report", []byte(html))
}

// TestReportGreysOutBypassedModules guards the visual cue that a module is off
// by default: its card is greyed and carries an "off" badge.
func TestReportGreysOutBypassedModules(t *testing.T) {
	b, err := rig.NewBuilder(catalog.New())
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	file, err := b.Build(rig.Spec{
		Name: "Off Rig",
		Blocks: []rig.Block{
			{Type: "Amp", Params: map[string]any{"Type": "82 Lead 800 100W"}},
			{Type: "Cab", Params: map[string]any{"CabType": "4x12 Green 25W", "MicType": "Dyn 57"}},
			{Type: "Chorus", Enabled: false},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}

	html, err := Render(file, "", catalog.New())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	if !strings.Contains(html, `class="module off"`) {
		t.Fatal("expected a greyed module card for the bypassed Chorus")
	}
	if !strings.Contains(html, `<span class="offbadge">off</span>`) {
		t.Fatal("expected an 'off' badge on the bypassed Chorus")
	}
}
