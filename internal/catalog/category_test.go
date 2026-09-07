package catalog

import "testing"

func TestCategoriesListAllWithCounts(t *testing.T) {
	c := New()
	cats := c.Categories()

	want := []string{"distortion", "dynamics", "eq", "expression", "modulation", "delay", "reverb", "utility"}
	if len(cats) != len(want) {
		t.Fatalf("got %d categories, want %d", len(cats), len(want))
	}
	total := 0
	for i, cat := range cats {
		if cat.Name != want[i] {
			t.Fatalf("category %d = %q, want %q", i, cat.Name, want[i])
		}
		total += cat.Count
	}
	if total != len(c.FX()) {
		t.Fatalf("category counts sum to %d, want %d (every FX in one category)", total, len(c.FX()))
	}
}

func TestFXByCategorySeparatesDelayAndReverb(t *testing.T) {
	c := New()

	delays := c.FXByCategory("delay")
	if len(delays) == 0 {
		t.Fatal("expected delay effects")
	}
	for _, f := range delays {
		if f.Category != "delay" {
			t.Fatalf("%q is in category %q, want delay", f.Name, f.Category)
		}
	}

	reverbs := c.FXByCategory("reverbs") // plural must resolve
	if len(reverbs) == 0 {
		t.Fatal("expected reverb effects")
	}
	for _, f := range reverbs {
		if f.Category != "reverb" {
			t.Fatalf("%q is in category %q, want reverb", f.Name, f.Category)
		}
	}
}

func TestFXByCategoryIsCaseInsensitiveAndRejectsUnknown(t *testing.T) {
	c := New()
	if got := c.FXByCategory("DELAY"); len(got) == 0 {
		t.Fatal("expected case-insensitive match for DELAY")
	}
	if got := c.FXByCategory("bogus"); got != nil {
		t.Fatalf("expected nil for unknown category, got %v", got)
	}
}

func TestFXByCategoryDynamicsResolves(t *testing.T) {
	c := New()
	// "dynamics" ends in "s"; it must resolve to the canonical category rather
	// than being singularised into the unknown "dynamic".
	dynamics := c.FXByCategory("dynamics")
	if len(dynamics) == 0 {
		t.Fatal("expected dynamics effects (the category must resolve)")
	}
	for _, f := range dynamics {
		if f.Category != "dynamics" {
			t.Fatalf("%q is in category %q, want dynamics", f.Name, f.Category)
		}
	}
}

func TestFXByCategoryEQContainsGraphicEQ(t *testing.T) {
	c := New()
	eq := c.FXByCategory("eq")
	if len(eq) == 0 {
		t.Fatal("expected EQ effects")
	}
	found := false
	for _, f := range eq {
		if f.Name == "Graphic EQ" {
			found = true
		}
	}
	if !found {
		t.Fatalf("eq category should contain Graphic EQ, got %v", names(eq))
	}
}

func names(fx []FX) []string {
	out := make([]string, len(fx))
	for i, f := range fx {
		out[i] = f.Name
	}
	return out
}
