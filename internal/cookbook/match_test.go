package cookbook

import (
	"strings"
	"testing"
)

func TestIngredientsCoversEveryDevice(t *testing.T) {
	for _, device := range []string{"gigboard", "quad-cortex", "wazaair", "ge200", "ge150", "ge150pro", "ge100pro", "thr", "thr10"} {
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

// TestWazaFlatIsNeutralGuitarAmp documents that FLAT is the Waza Air's neutral
// guitar/acoustic amp, not a dedicated bass amp — bass tones use FLAT with a
// booster low-end lift instead (see the agent guide).
func TestWazaFlatIsNeutralGuitarAmp(t *testing.T) {
	ing, err := Ingredients("wazaair")
	if err != nil {
		t.Fatalf("Ingredients: %v", err)
	}
	for _, in := range ing {
		if in.Name == "FLAT" {
			if in.Kind != KindAmp {
				t.Fatalf("FLAT kind = %q, want %q (neutral guitar amp, not bassamp)", in.Kind, KindAmp)
			}
			return
		}
	}
	t.Fatal("FLAT amp not found in wazaair ingredients")
}

// TestTHRBassSelectorPositionsAreBassIngredients documents that the THR's BASS
// group is a real bass amp family (Eden/Markbass, Mesa Subway, Marshall Bass):
// it carries the bassamp kind, as the Gigboard's and the Cortex's bass amps do.
func TestTHRBassSelectorPositionsAreBassIngredients(t *testing.T) {
	want := map[string]string{
		"BASS CLASSIC":  KindBassAmp,
		"BASS BOUTIQUE": KindBassAmp,
		"BASS MODERN":   KindBassAmp,
		"FLAT CLASSIC":  KindAmp,
	}

	for _, in := range mustIngredients(t, "thr") {
		kind, ok := want[in.Name]
		if !ok {
			continue
		}
		if in.Kind != kind {
			t.Errorf("%s kind = %q, want %q", in.Name, in.Kind, kind)
		}
		delete(want, in.Name)
	}
	for name := range want {
		t.Errorf("%s not found in the thr ingredients", name)
	}
}

// TestWazaFlatHintNamesTheTargetsBassVoices guards the case that motivated the
// hints: the Waza Air has no bass amp, so a bass tone stands in with FLAT.
// Porting that stand-in must not quietly land on the target's own neutral
// position — the plan says which of the target's amps to use instead.
func TestWazaFlatHintNamesTheTargetsBassVoices(t *testing.T) {
	target := mustIngredients(t, "thr")

	plan, err := Map(mustIngredients(t, "wazaair"), target, "thr", []string{"FLAT"})
	if err != nil {
		t.Fatalf("Map: %v", err)
	}

	hint := onlyHint(t, plan)
	for _, voice := range namesOfKind(target, KindBassAmp) {
		if !strings.Contains(hint, voice) {
			t.Errorf("hint does not name the target's bass amp %q: %s", voice, hint)
		}
	}
}

// TestWazaFlatHintWithoutABassAmpOnTheTarget keeps the advice usable when the
// target has no bass amp: no dangling placeholder, and the low-end lift stays
// the agent's own job.
func TestWazaFlatHintWithoutABassAmpOnTheTarget(t *testing.T) {
	plan, err := Map(mustIngredients(t, "wazaair"), mustIngredients(t, "wazaair"), "wazaair", []string{"FLAT"})
	if err != nil {
		t.Fatalf("Map: %v", err)
	}

	hint := onlyHint(t, plan)
	if strings.ContainsAny(hint, "{}") {
		t.Errorf("hint left a placeholder unexpanded: %s", hint)
	}
	if !strings.Contains(hint, "low end") {
		t.Errorf("hint gives no fallback for a target without a bass amp: %s", hint)
	}
}

// TestOnlyTheFlatsRoleNeedsAHint: FLAT is the Air's one cross-device stand-in,
// so the amp models it stands in for carry no caveat.
func TestOnlyTheFlatsRoleNeedsAHint(t *testing.T) {
	plan, err := Map(mustIngredients(t, "wazaair"), mustIngredients(t, "thr"), "thr", []string{"CLEAN"})
	if err != nil {
		t.Fatalf("Map: %v", err)
	}

	if hints := plan.Matches[0].Hints; len(hints) != 0 {
		t.Errorf("CLEAN carries hints = %v, want none", hints)
	}
}

// TestLostLevelBlockIsCalledOut: the Waza Air's bass tone gets its loudness and
// low end from the CLEAN BOOST, which the THR has no block for. Dropping it
// silently is how a ported tone ends up quiet, so the plan says to fold it in.
func TestLostLevelBlockIsCalledOut(t *testing.T) {
	plan, err := Map(mustIngredients(t, "wazaair"), mustIngredients(t, "thr"), "thr", []string{"CLEAN BOOST"})
	if err != nil {
		t.Fatalf("Map: %v", err)
	}

	m := plan.Matches[0]
	if m.Matched {
		t.Fatalf("CLEAN BOOST matched a THR block: %+v", m)
	}
	hint := strings.Join(m.Hints, " ")
	if !strings.Contains(hint, "fold it into the target's own amp") {
		t.Errorf("unmatched level block carries no fold-in hint: %v", m.Hints)
	}
}

// TestMatchedBlocksCarryNoLostContributionHint: a block with a counterpart is
// re-dialled by value, not recreated by hand, so it needs no such caveat.
func TestMatchedBlocksCarryNoLostContributionHint(t *testing.T) {
	plan, err := Map(mustIngredients(t, "wazaair"), mustIngredients(t, "thr"), "thr", []string{"DIGITAL DELAY"})
	if err != nil {
		t.Fatalf("Map: %v", err)
	}

	if hints := plan.Matches[0].Hints; len(hints) != 0 {
		t.Errorf("matched delay carries hints = %v, want none", hints)
	}
}

// TestUnmatchedPitchBlockIsNotALevelProblem keeps the fold-in hint for the
// blocks whose loss is audible as level: a dropped harmonizer is a missing
// effect, not a quiet tone.
func TestUnmatchedPitchBlockIsNotALevelProblem(t *testing.T) {
	src := []Ingredient{{Device: "x", Kind: KindFX, Name: "Smart Harm", Tags: []string{"fx", "pitch"}}}
	tgt := []Ingredient{{Device: "y", Kind: KindFX, Name: "Digital Delay", Tags: []string{"fx", "delay"}}}

	plan, err := Map(src, tgt, "y", []string{"Smart Harm"})
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	if hints := plan.Matches[0].Hints; len(hints) != 0 {
		t.Errorf("unmatched harmonizer hints = %v, want none", hints)
	}
}

func mustIngredients(t *testing.T, device string) []Ingredient {
	t.Helper()
	ingredients, err := Ingredients(device)
	if err != nil {
		t.Fatalf("Ingredients(%q): %v", device, err)
	}
	return ingredients
}

// onlyHint returns the hint of a one-block plan, failing when the block carries
// none — a port that silently drops the caveat is the bug being guarded.
func onlyHint(t *testing.T, plan Plan) string {
	t.Helper()
	if len(plan.Matches) != 1 || len(plan.Matches[0].Hints) == 0 {
		t.Fatalf("matches = %+v, want one match carrying a hint", plan.Matches)
	}
	return strings.Join(plan.Matches[0].Hints, " ")
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

// The classic GE150 and the GE200 number their mod and delay types differently
// (the GE150 has no MONO PITCH and has a MOD delay). The mapping must still be
// informed: a GE200-only MONO PITCH lands on the GE150's pitch shifter via the
// shared pitch tag, and the two devices' mod lists stay the right length.
func TestGE150GE200CrossMappingIsGapSafe(t *testing.T) {
	ge200, err := Ingredients("ge200")
	if err != nil {
		t.Fatal(err)
	}
	ge150, err := Ingredients("ge150")
	if err != nil {
		t.Fatal(err)
	}

	plan, err := Map(ge200, ge150, "ge150", []string{"MONO PITCH"})
	if err != nil {
		t.Fatalf("Map: %v", err)
	}
	if len(plan.Matches) != 1 || !plan.Matches[0].Matched {
		t.Fatalf("MONO PITCH mapping = %+v, want a matched pitch target", plan.Matches)
	}
	if plan.Matches[0].Target != "PITCH SHIFT" {
		t.Fatalf("MONO PITCH mapped to %q, want PITCH SHIFT", plan.Matches[0].Target)
	}
}
