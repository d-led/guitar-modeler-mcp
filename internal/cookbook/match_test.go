package cookbook

import (
	"strings"
	"testing"
)

func TestIngredientsCoversEveryDevice(t *testing.T) {
	for _, device := range []string{"gigboard", "quad-cortex", "wazaair", "ge200", "ge150pro", "ge100pro", "thr", "thr10"} {
		ingredients, err := Ingredients(device)
		if err != nil {
			t.Fatalf("Ingredients(%q): %v", device, err)
		}
		if len(ingredients) == 0 {
			t.Fatalf("Ingredients(%q) returned nothing", device)
		}
		kinds := map[string]bool{}
		for _, in := range ingredients {
			kinds[in.Kind] = true
		}
		// Every device has amps and effects; cabs are only present on the
		// full modelers.
		for _, kind := range []string{KindAmp, KindFX} {
			if !kinds[kind] {
				t.Errorf("%s: missing kind %q", device, kind)
			}
		}
	}
}

func TestUnknownDevice(t *testing.T) {
	if _, err := Ingredients("boss-katana"); err == nil {
		t.Fatal("unknown device accepted, want error")
	}
}

func TestMatchAmpByReference(t *testing.T) {
	src, _ := Ingredients("gigboard")
	tgt, _ := Ingredients("quad-cortex")

	plan, err := Map(src, tgt, "quad-cortex", []string{"82 Lead 800 100W"})
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if len(plan.Matches) != 1 || !plan.Matches[0].Matched {
		t.Fatalf("matches = %+v", plan.Matches)
	}
	if !strings.Contains(plan.Matches[0].Target, "JCM800") {
		t.Fatalf("JCM800 amp mapped to %q, want a JCM800", plan.Matches[0].Target)
	}
	if plan.Coverage != 1 {
		t.Fatalf("coverage = %g, want 1", plan.Coverage)
	}
}

func TestTagsEncodeSubFeatures(t *testing.T) {
	// A "fancy delay" that pitch-shifts carries both delay and pitch.
	fancy := newIngredient("x", KindFX, "Pitch Echo", "", "", "", "a delay with a built-in harmonizer")
	if !hasTag(fancy, "delay") || !hasTag(fancy, "pitch") {
		t.Fatalf("Pitch Echo tags = %v, want delay+pitch", fancy.Tags)
	}

	// A dedicated harmonizer carries pitch but not delay.
	harm := newIngredient("x", KindFX, "Smart Harm", "", "pitch", "", "intelligent harmonizer")
	if !hasTag(harm, "pitch") {
		t.Fatalf("Smart Harm tags = %v, want pitch", harm.Tags)
	}
	if hasTag(harm, "delay") {
		t.Fatalf("Smart Harm unexpectedly tagged delay: %v", harm.Tags)
	}
}

func TestMatchHarmonizerPrefersPitchThenFallsBackToDelayWithPitch(t *testing.T) {
	harm := newIngredient("src", KindFX, "Smart Harm", "", "pitch", "", "intelligent harmonizer")
	delayWithPitch := newIngredient("tgt", KindFX, "Pitch Echo", "", "", "", "delay with harmonizer")
	plainDelay := newIngredient("tgt", KindFX, "Digital Delay", "", "delay", "", "digital delay")
	pitch := newIngredient("tgt", KindFX, "Pitch Shifter", "", "pitch", "", "pitch shifter")

	// With a real pitch shifter available, it wins.
	best, score, ok := bestMatch(harm, []Ingredient{plainDelay, pitch, delayWithPitch})
	if !ok || best.Name != "Pitch Shifter" {
		t.Fatalf("best = %q (%.2f), want Pitch Shifter", best.Name, score)
	}

	// Without a dedicated pitch shifter, the delay-with-pitch still covers it.
	best, score, ok = bestMatch(harm, []Ingredient{plainDelay, delayWithPitch})
	if !ok || best.Name != "Pitch Echo" {
		t.Fatalf("best = %q (%.2f), want Pitch Echo", best.Name, score)
	}

	// A plain delay cannot cover a harmonizer.
	if _, _, ok := bestMatch(harm, []Ingredient{plainDelay}); ok {
		t.Fatal("a plain delay matched a harmonizer; it should refuse")
	}
}

func TestMatchCoverageAndUnknownBlock(t *testing.T) {
	src, _ := Ingredients("gigboard")
	tgt, _ := Ingredients("quad-cortex")

	plan, err := Map(src, tgt, "quad-cortex", []string{"82 Lead 800 100W", "Green JRC-OD", "Not A Real Block"})
	if err != nil {
		t.Fatalf("Match: %v", err)
	}
	if plan.Coverage != 2.0/3.0 {
		t.Fatalf("coverage = %g, want 0.6667", plan.Coverage)
	}
	var unknown *Match
	for i := range plan.Matches {
		if plan.Matches[i].Source == "Not A Real Block" {
			unknown = &plan.Matches[i]
		}
	}
	if unknown == nil || unknown.Matched || unknown.Reason != "unknown source block" {
		t.Fatalf("unknown block = %+v", unknown)
	}
	if plan.ByKind[KindFX] != 1 {
		t.Fatalf("fx coverage = %g, want 1", plan.ByKind[KindFX])
	}
}

func hasTag(in Ingredient, tag string) bool {
	for _, t := range in.Tags {
		if t == tag {
			return true
		}
	}
	return false
}

func TestCanonicalParam(t *testing.T) {
	cases := map[string]string{
		"GAIN":       "gain",
		"DRIVE":      "gain",
		"OVERDRIVE":  "gain",
		"GainA":      "gain",
		"TONE":       "tone",
		"DIRECT MIX": "mix",
		"FLUTTER":    "flutter", // unknown falls back to itself
	}
	for raw, want := range cases {
		if got := canonicalParam(raw); got != want {
			t.Errorf("canonicalParam(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestLinkParams(t *testing.T) {
	src := Ingredient{Params: []string{"GAIN", "TONE", "FLUTTER"}}
	tgt := Ingredient{Params: []string{"DRIVE", "TREBLE", "FLUTTER"}}

	links := linkParams(src, tgt)
	got := map[string]ParamLink{}
	for _, l := range links {
		got[l.Canonical] = l
	}
	if l, ok := got["gain"]; !ok || l.Source != "GAIN" || l.Target != "DRIVE" {
		t.Fatalf("gain link = %+v, want GAIN -> DRIVE", l)
	}
	if l, ok := got["flutter"]; !ok || l.Target != "FLUTTER" {
		t.Fatalf("flutter link = %+v, want FLUTTER -> FLUTTER", l)
	}
	if _, ok := got["tone"]; ok {
		t.Fatal("TONE must not link to TREBLE")
	}
}

func TestMatchAmpMapsParameters(t *testing.T) {
	src, _ := Ingredients("gigboard")
	tgt, _ := Ingredients("quad-cortex")

	plan, err := Map(src, tgt, "quad-cortex", []string{"82 Lead 800 100W"})
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	if len(plan.Matches) != 1 || !plan.Matches[0].Matched {
		t.Fatalf("matches = %+v", plan.Matches)
	}
	byCanon := map[string]ParamLink{}
	for _, l := range plan.Matches[0].Params {
		byCanon[l.Canonical] = l
	}
	for _, want := range []string{"gain", "master", "bass", "mid", "treble", "presence", "level"} {
		if _, ok := byCanon[want]; !ok {
			t.Errorf("amp mapping missing canonical knob %q (links: %+v)", want, plan.Matches[0].Params)
		}
	}
	if byCanon["gain"].Target != "GAIN" {
		t.Fatalf("gain maps to %q, want the QC GAIN", byCanon["gain"].Target)
	}
}
