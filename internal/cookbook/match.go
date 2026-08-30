package cookbook

import (
	"fmt"
	"sort"
	"strings"
)

// ParamLink maps one source knob to the target's equivalent knob.
type ParamLink struct {
	Source    string `json:"source"`
	Target    string `json:"target"`
	Canonical string `json:"canonical"`
}

// Match is one source block mapped to its best target block.
type Match struct {
	Kind    string      `json:"kind"`
	Source  string      `json:"source"`
	Target  string      `json:"target,omitempty"`
	Score   float64     `json:"score"`
	Tags    []string    `json:"source_tags"`
	Matched bool        `json:"matched"`
	Reason  string      `json:"reason,omitempty"`
	Params  []ParamLink `json:"params,omitempty"`
}

// Plan is the mapping table plus coverage percentages.
type Plan struct {
	Source   string             `json:"source_device"`
	Target   string             `json:"target_device"`
	Matches  []Match            `json:"matches"`
	Coverage float64            `json:"coverage"`
	ByKind   map[string]float64 `json:"coverage_by_kind"`
}

// matchThreshold is the score a target must reach to count as a match. A same
// kind alone scores 0.5, so a block always maps to its own kind, but only
// shared feature tags or a shared reference push it above "best effort".
const matchThreshold = 0.45

// Map maps each named source block to its best target block and reports
// coverage. Unmatched blocks are listed, not silently dropped.
func Map(source []Ingredient, target []Ingredient, targetDevice string, blocks []string) (Plan, error) {
	byName := indexByName(source)
	plan := Plan{Source: deviceOf(source), Target: targetDevice, ByKind: map[string]float64{}}
	kindTotal := map[string]int{}
	kindMatched := map[string]int{}

	for _, block := range blocks {
		src, ok := byName[strings.ToLower(strings.TrimSpace(block))]
		if !ok {
			plan.Matches = append(plan.Matches, Match{Source: block, Matched: false, Reason: "unknown source block"})
			continue
		}
		best, score, found := bestMatch(src, target)
		m := Match{Kind: src.Kind, Source: src.Name, Tags: src.Tags}
		kindTotal[src.Kind]++
		if found {
			m.Target = best.Name
			m.Score = score
			m.Matched = true
			m.Reason = describe(src, best)
			m.Params = linkParams(src, best)
			kindMatched[src.Kind]++
		} else {
			m.Reason = "no target block close enough"
		}
		plan.Matches = append(plan.Matches, m)
	}

	matched := 0
	for _, m := range plan.Matches {
		if m.Matched {
			matched++
		}
	}
	if len(blocks) > 0 {
		plan.Coverage = float64(matched) / float64(len(blocks))
	}
	for kind, n := range kindTotal {
		plan.ByKind[kind] = float64(kindMatched[kind]) / float64(n)
	}
	return plan, nil
}

func indexByName(ingredients []Ingredient) map[string]Ingredient {
	out := make(map[string]Ingredient, len(ingredients))
	for _, in := range ingredients {
		key := strings.ToLower(strings.TrimSpace(in.Name))
		if _, exists := out[key]; !exists {
			out[key] = in
		}
	}
	return out
}

func deviceOf(ingredients []Ingredient) string {
	for _, in := range ingredients {
		if in.Device != "" {
			return in.Device
		}
	}
	return ""
}

// bestMatch returns the highest-scoring target and whether it clears the bar.
func bestMatch(src Ingredient, target []Ingredient) (Ingredient, float64, bool) {
	var best Ingredient
	bestScore := -1.0
	for _, t := range target {
		if s := score(src, t); s > bestScore {
			bestScore = s
			best = t
		}
	}
	if bestScore < matchThreshold {
		return Ingredient{}, bestScore, false
	}
	// Effects must share a feature tag or a real-hardware reference: a plain
	// delay does not "cover" a harmonizer just because both are effects. Amps
	// and cabs map by kind alone (any amp is a real substitution).
	if src.Kind == KindFX {
		if len(intersect(src.Tags, best.Tags)) == 0 && jaccard(tokenize(src.Ref), tokenize(best.Ref)) == 0 {
			return Ingredient{}, bestScore, false
		}
	}
	return best, bestScore, true
}

// score rates how well a target block covers a source block. Same kind is the
// dominant signal (0.5); shared feature tags (0.3) and a shared real-hardware
// reference (0.2) refine it. The maximum is 1.0.
func score(src, tgt Ingredient) float64 {
	s := 0.0
	if src.Kind == tgt.Kind {
		s += 0.5
	}
	s += 0.3 * jaccard(src.Tags, tgt.Tags)
	s += 0.2 * jaccard(tokenize(src.Ref), tokenize(tgt.Ref))
	return s
}

// describe explains a match in one line: the shared tags that carried it.
func describe(src, tgt Ingredient) string {
	shared := intersect(src.Tags, tgt.Tags)
	if len(shared) == 0 {
		if src.Kind == tgt.Kind {
			return fmt.Sprintf("same kind (%s)", src.Kind)
		}
		return "best effort"
	}
	return "shared tags: " + strings.Join(shared, ", ")
}

func intersect(a, b []string) []string {
	set := make(map[string]bool, len(a))
	for _, t := range a {
		set[t] = true
	}
	var out []string
	for _, t := range b {
		if set[t] {
			out = append(out, t)
			delete(set, t)
		}
	}
	return out
}

// linkParams maps a source block's knobs to the target block's knobs by their
// canonical names ("GAIN" and "DRIVE" both canonicalise to "gain"). Only knobs
// both blocks actually expose are linked; the result is a starting point for
// the agent to carry the values across, not a value conversion.
func linkParams(src, tgt Ingredient) []ParamLink {
	srcByCanon := map[string]string{}
	for _, p := range src.Params {
		if c := canonicalParam(p); srcByCanon[c] == "" {
			srcByCanon[c] = p
		}
	}
	tgtByCanon := map[string]string{}
	for _, p := range tgt.Params {
		if c := canonicalParam(p); tgtByCanon[c] == "" {
			tgtByCanon[c] = p
		}
	}

	var out []ParamLink
	for canon, sp := range srcByCanon {
		tp, ok := tgtByCanon[canon]
		if !ok {
			continue
		}
		out = append(out, ParamLink{Source: sp, Target: tp, Canonical: canon})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Canonical < out[j].Canonical })
	return out
}
