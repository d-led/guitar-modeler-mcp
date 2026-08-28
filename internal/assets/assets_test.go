package assets

import (
	"strings"
	"testing"

	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/catalog"
)

// TestEveryCatalogFXHasBlockDefinition guards the single source of truth: every
// effect module advertised by the catalog must have an embedded block folder so
// the rig builder can produce a default node for it.
func TestEveryCatalogFXHasBlockDefinition(t *testing.T) {
	types := make(map[string]bool, len(ModuleTypes()))
	for _, mt := range ModuleTypes() {
		types[mt] = true
	}
	for _, f := range catalog.New().FX() {
		if !types[strings.ToUpper(f.Name)] {
			t.Errorf("FX %q has no embedded block folder %q", f.Name, strings.ToUpper(f.Name))
		}
	}
}

func TestDefaultBlockExistsForCommonModules(t *testing.T) {
	for _, mt := range []string{"AMP", "CAB", "TAPE ECHO", "WHITE BOOST", "SPRING REVERB"} {
		if _, err := DefaultBlock(mt); err != nil {
			t.Errorf("DefaultBlock(%q): %v", mt, err)
		}
	}
}
