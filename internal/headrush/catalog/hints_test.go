package catalog

import "testing"

func TestModeledAfterEnrichedFromHints(t *testing.T) {
	c := New()

	amp, ok := c.Amp("82 Lead 800 100W")
	if !ok {
		t.Fatal("amp not found")
	}
	if amp.ModeledAfter != "Marshall JCM800 2203 Head" {
		t.Fatalf("amp modeled_after = %q", amp.ModeledAfter)
	}

	mic, ok := c.Mic("Dyn 57")
	if !ok {
		t.Fatal("mic not found")
	}
	if mic.ModeledAfter != "Shure SM57 Unidyne II" {
		t.Fatalf("mic modeled_after = %q", mic.ModeledAfter)
	}
	if !mic.Confirmed {
		t.Fatal("Dyn 57 should be confirmed")
	}

	fx, ok := c.FXByName("Green JRC-OD")
	if !ok {
		t.Fatal("fx not found")
	}
	if fx.ModeledAfter != "Ibanez TS-808 Tube Screamer" {
		t.Fatalf("fx modeled_after = %q", fx.ModeledAfter)
	}
	if !fx.Confirmed {
		t.Fatal("Green JRC-OD should be confirmed")
	}
}

func TestModeledAfterFallsBackToCatalog(t *testing.T) {
	c := New()
	// "05 Tangerine 30 Ch1" is not in the hints; it must fall back to the
	// catalog's own brand + real model.
	amp, ok := c.Amp("05 Tangerine 30 Ch1")
	if !ok {
		t.Fatal("amp not found")
	}
	if amp.ModeledAfter != "Orange AD30 Twin Channel" {
		t.Fatalf("amp modeled_after = %q", amp.ModeledAfter)
	}
}

func TestModeledAfterAvoidsBrandDuplication(t *testing.T) {
	c := New()
	// The hints list "Royer" + "Royer 121" for Ribbon 121; the result must not
	// repeat the brand.
	mic, ok := c.Mic("Ribbon 121")
	if !ok {
		t.Fatal("mic not found")
	}
	if mic.ModeledAfter != "Royer 121" {
		t.Fatalf("mic modeled_after = %q", mic.ModeledAfter)
	}
}
