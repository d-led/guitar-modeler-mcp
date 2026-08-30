package cardchain
package cardchain

import (
	"strings"
	"testing"
)

func TestRenderNumberedChain(t *testing.T) {
	html := Render([]Step{
		{Slot: 1, Module: "BOOSTER", Effect: "T-SCREAM"},
		{Slot: 2, Module: "AMP", Effect: "CLEAN"},
		{Slot: 3, Effect: ""},
	})
	for _, want := range []string{"slotno\">1", "BOOSTER: T-SCREAM", "slotno\">2", "AMP: CLEAN", "slotno\">3", "→", "class=\"chain\""} {
		if !strings.Contains(html, want) {
			t.Errorf("chain missing %q: %s", want, html)
		}
	}
}

func TestRenderEmpty(t *testing.T) {
	if Render(nil) != "" {
		t.Fatal("Render(nil) should be empty")
	}
}

func TestRenderEscapes(t *testing.T) {
	html := Render([]Step{{Slot: 1, Effect: "A<B&C"}})
	if strings.Contains(html, "<B&") {
		t.Fatalf("effect not escaped: %s", html)
	}
}
