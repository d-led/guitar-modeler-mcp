package catalog

import "testing"

func TestCatalogLookupExact(t *testing.T) {
	c := New()
	if _, ok := c.Amp("82 Lead 800 100W"); !ok {
		t.Fatal("expected exact amp lookup to succeed")
	}
	if _, ok := c.Cab("4x12 Green 25W"); !ok {
		t.Fatal("expected exact cab lookup to succeed")
	}
	if _, ok := c.Mic("Dyn 57"); !ok {
		t.Fatal("expected exact mic lookup to succeed")
	}
}

func TestTranslateAmpMarshall(t *testing.T) {
	c := New()
	matches := c.TranslateAmp("Marshall JCM800")
	if len(matches) == 0 {
		t.Fatal("expected matches for a Marshall JCM800")
	}
	if matches[0].Amp.Brand != "Marshall" {
		t.Fatalf("top match brand = %q, want Marshall", matches[0].Amp.Brand)
	}
}

func TestTranslateAmpDeluxeReverb(t *testing.T) {
	c := New()
	matches := c.TranslateAmp("blackface deluxe reverb")
	if len(matches) == 0 {
		t.Fatal("expected matches for a blackface deluxe reverb")
	}
	top := matches[0].Amp
	if top.Brand != "Fender" || top.RealModel != "Deluxe Reverb" {
		t.Fatalf("top match = %+v, want Fender Deluxe Reverb", top)
	}
}

func TestTranslateCab(t *testing.T) {
	c := New()
	matches := c.TranslateCab("greenback 4x12")
	if len(matches) == 0 {
		t.Fatal("expected cab matches")
	}
}

func TestTranslateMic(t *testing.T) {
	c := New()
	matches := c.TranslateMic("SM57")
	if len(matches) == 0 {
		t.Fatal("expected mic matches")
	}
	if matches[0].Model != "Dyn 57" {
		t.Fatalf("top mic = %q, want Dyn 57", matches[0].Model)
	}
}

func TestAmpsMatchingFilters(t *testing.T) {
	c := New()
	got := c.AmpsMatching("mesa")
	for _, a := range got {
		if a.Brand != "Mesa Boogie" {
			t.Fatalf("unexpected amp %q for filter mesa", a.Model)
		}
	}
	if len(got) == 0 {
		t.Fatal("expected Mesa Boogie amps")
	}
}

func TestCatalogCounts(t *testing.T) {
	c := New()
	if len(c.Amps()) < 50 {
		t.Fatalf("amps = %d, expected at least 50", len(c.Amps()))
	}
	if len(c.Cabs()) != 15 {
		t.Fatalf("cabs = %d, want 15", len(c.Cabs()))
	}
	if len(c.Mics()) != 9 {
		t.Fatalf("mics = %d, want 9", len(c.Mics()))
	}
}
