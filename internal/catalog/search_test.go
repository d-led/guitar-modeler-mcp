package catalog

import (
	"strings"
	"testing"
)

func topName(t *testing.T, results []SearchResult) string {
	t.Helper()
	if len(results) == 0 {
		return ""
	}
	return results[0].Name
}

func TestSearchForwardByRealHardware(t *testing.T) {
	c := New()
	results := c.Search("JCM800", "")
	if topName(t, results) == "" {
		t.Fatal("expected results for JCM800")
	}
	if !strings.HasPrefix(results[0].Name, "82 Lead 800") {
		t.Fatalf("top result = %q, want a JCM800 (82 Lead 800...)", results[0].Name)
	}
}

func TestSearchInverseByDeviceName(t *testing.T) {
	c := New()
	results := c.Search("82 Lead 800 100W", "amp")
	if len(results) == 0 {
		t.Fatal("expected the amp by its device name")
	}
	if results[0].ModeledAfter == "" || !strings.Contains(results[0].ModeledAfter, "JCM800") {
		t.Fatalf("inverse info missing: modeled_after=%q", results[0].ModeledAfter)
	}
}

func TestSearchToleratesTypos(t *testing.T) {
	c := New()
	results := c.Search("mashall jcm800", "amp")
	if len(results) == 0 {
		t.Fatal("expected a Marshall result despite the typo")
	}
	if !strings.HasPrefix(results[0].Name, "82 Lead 800") {
		t.Fatalf("top result = %q, want a JCM800", results[0].Name)
	}
}

func TestSearchFXByModeledAfter(t *testing.T) {
	c := New()
	results := c.Search("Tube Screamer", "fx")
	if len(results) == 0 {
		t.Fatal("expected a Tube Screamer effect")
	}
	if results[0].Name != "Green JRC-OD" {
		t.Fatalf("top fx = %q, want Green JRC-OD", results[0].Name)
	}
}

func TestSearchKindFilter(t *testing.T) {
	c := New()
	fx := c.Search("reverb", "fx")
	for _, r := range fx {
		if r.Type != "fx" {
			t.Fatalf("kind filter leaked non-fx result: %+v", r)
		}
	}
	if len(fx) == 0 {
		t.Fatal("expected reverb effects")
	}
}

func TestSearchEmptyQuery(t *testing.T) {
	c := New()
	if got := c.Search("", ""); got != nil {
		t.Fatalf("expected nil for empty query, got %v", got)
	}
}
