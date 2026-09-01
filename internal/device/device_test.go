package device

import "testing"

func TestItemsPreserveOrder(t *testing.T) {
	got := Items(
		[2]string{"CHORUS", ""},
		[2]string{"T-SCREAM", "Ibanez Tube Screamer"},
	)
	if len(got) != 2 {
		t.Fatalf("Items() len = %d, want 2", len(got))
	}
	if got[0].Name != "CHORUS" || got[0].InspiredBy != "" {
		t.Errorf("Items()[0] = %+v", got[0])
	}
	if got[1].Name != "T-SCREAM" || got[1].InspiredBy != "Ibanez Tube Screamer" {
		t.Errorf("Items()[1] = %+v", got[1])
	}
}

func TestCanonicalKey(t *testing.T) {
	cases := map[string]string{
		"GAIN":          "gain",
		"Time (ms)":     "time_ms",
		"time_ms":       "time_ms",
		" EFFECT LEVEL": "effect_level",
		"low *":         "low",
	}
	for in, want := range cases {
		if got := CanonicalKey(in); got != want {
			t.Errorf("CanonicalKey(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNorm(t *testing.T) {
	if got := Norm("  Fender·Twin  Reverb "); got != "fender twin reverb" {
		t.Errorf("Norm() = %q, want %q", got, "fender twin reverb")
	}
}

func TestResolveItem(t *testing.T) {
	items := Items(
		[2]string{"CLEAN", "Fender Twin Reverb"},
		[2]string{"LEAD", "Peavey 5150"},
	)

	if got, err := ResolveItem(items, "clean", "amp"); err != nil || got != "CLEAN" {
		t.Errorf("ResolveItem(clean) = %q, %v; want CLEAN", got, err)
	}
	if got, err := ResolveItem(items, "5150", "amp"); err != nil || got != "LEAD" {
		t.Errorf("ResolveItem(5150) = %q, %v; want LEAD", got, err)
	}
	if got, err := ResolveItem(items, "  ", "amp"); err != nil || got != "" {
		t.Errorf("ResolveItem(blank) = %q, %v; want empty, nil", got, err)
	}
	if _, err := ResolveItem(items, "nope", "amp"); err == nil {
		t.Error("ResolveItem(nope) should error")
	}
}

func TestFindByName(t *testing.T) {
	type model struct{ Name, Display string }
	models := []model{{"ge200", "Mooer GE200"}, {"ge150", "Mooer GE150"}}

	got, ok := FindByName(models, "MOOER GE200", func(m model) (string, string) { return m.Name, m.Display })
	if !ok || got.Name != "ge200" {
		t.Errorf("FindByName(display) = %+v, %v; want ge200", got, ok)
	}
	if _, ok := FindByName(models, "nope", func(m model) (string, string) { return m.Name, m.Display }); ok {
		t.Error("FindByName(nope) should not match")
	}
}
