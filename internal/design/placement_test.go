package design

import "testing"

func TestPlacementGuideCoversAllRoutings(t *testing.T) {
	g := PlacementGuide()
	if len(g.Routings) != 3 {
		t.Fatalf("expected 3 routings, got %d", len(g.Routings))
	}
	byRouting := map[string]int{}
	for _, r := range g.Routings {
		byRouting[r.Routing] = len(r.Sections)
	}
	if byRouting["S"] != 1 || byRouting["SPS-1"] != 4 || byRouting["PS-1"] != 3 {
		t.Fatalf("unexpected section counts: %v", byRouting)
	}
	if g.AlwaysLast != "Volume" {
		t.Fatalf("AlwaysLast = %q, want Volume", g.AlwaysLast)
	}
}

func TestPlacementCategoriesArePreOrPost(t *testing.T) {
	g := PlacementGuide()
	seen := map[string]string{}
	for _, c := range g.Categories {
		if c.Place != "pre-amp" && c.Place != "post-amp" {
			t.Fatalf("category %q has place %q", c.Category, c.Place)
		}
		seen[c.Category] = c.Place
	}
	// A drive goes pre, a reverb goes post.
	if seen["distortion"] != "pre-amp" {
		t.Fatalf("distortion placement = %q, want pre-amp", seen["distortion"])
	}
	if seen["reverb"] != "post-amp" {
		t.Fatalf("reverb placement = %q, want post-amp", seen["reverb"])
	}
}

func TestPlaceForCategoryUnknownDefaultsToPost(t *testing.T) {
	if got := placeForCategory("bogus"); got != "post-amp" {
		t.Fatalf("placeForCategory(bogus) = %q, want post-amp", got)
	}
	if got := placeForCategory("distortion"); got != "pre-amp" {
		t.Fatalf("placeForCategory(distortion) = %q, want pre-amp", got)
	}
}
