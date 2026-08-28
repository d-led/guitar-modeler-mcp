package htmlreport

import (
	"regexp"
	"testing"

	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/catalog"
	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/golden"
	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/rig"
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

	html, err := Render(file, "Test Song", catalog.New())
	if err != nil {
		t.Fatalf("Render: %v", err)
	}
	html = regexp.MustCompile(`Generated \d{4}-\d{2}-\d{2} \d{2}:\d{2}`).ReplaceAllString(html, "Generated FIXED")

	golden.Assert(t, "report", []byte(html))
}
