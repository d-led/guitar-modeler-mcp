package rig

import (
	"testing"

	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/catalog"
	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/golden"
)

// TestRigSnapshot approves the exact on-disk format the builder emits for a
// representative rig. The id and timestamp are pinned so the snapshot is stable.
func TestRigSnapshot(t *testing.T) {
	b, err := NewBuilder(catalog.New())
	if err != nil {
		t.Fatalf("NewBuilder: %v", err)
	}
	file, err := b.Build(Spec{
		Name:  "Approval Rig",
		Tempo: 120,
		Blocks: []Block{
			{Type: "Green JRC-OD", Enabled: true, Params: map[string]any{"Drive": 60.0}},
			{Type: "Amp", Params: map[string]any{"Type": "65 Black SR", "GainA": 70.0, "Bass": 40.0}},
			{Type: "Cab", Params: map[string]any{"CabType": "1x12 Black Panel Lux", "MicType": "Dyn 57"}},
			{Type: "Tape Echo", Enabled: true, Params: map[string]any{"Feedback": 55.0}},
			{Type: "Eleven Reverb", Enabled: true},
			{Type: "Volume", Enabled: true},
		},
	})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	file.ID = "00000000-0000-4000-8000-000000000000"
	file.CreatedAt = 0

	out, err := file.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	out = append(out, '\n')
	golden.Assert(t, "rig", out)
}
