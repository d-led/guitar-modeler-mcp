package catalog

import (
	"encoding/json"
	"testing"

	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/golden"
)

// TestCatalogSnapshot approves the full device catalog: every model with its
// tags (brand, real model, style, category, description).
func TestCatalogSnapshot(t *testing.T) {
	c := New()
	doc := struct {
		Amps []Amp `json:"amps"`
		Cabs []Cab `json:"cabs"`
		Mics []Mic `json:"mics"`
		FX   []FX  `json:"fx"`
	}{
		Amps: c.Amps(),
		Cabs: c.Cabs(),
		Mics: c.Mics(),
		FX:   c.FX(),
	}
	b, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatalf("marshal catalog: %v", err)
	}
	b = append(b, '\n')
	golden.Assert(t, "catalog", b)
}
