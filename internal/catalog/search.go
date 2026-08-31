package catalog

import (
	"math"
	"sort"
	"strings"
)

// SearchResult is one fuzzy match across the amp, cab, mic and effect catalogs.
// ModeledAfter is the real-world hardware the device model emulates, so a query
// works in both directions: "JCM800" finds "82 Lead 800 100W" and
// "82 Lead 800" finds "Marshall JCM800 2203".
type SearchResult struct {
	Type         string  `json:"type"`                    // "amp", "cab", "mic" or "fx"
	Name         string  `json:"name"`                    // exact device name
	ModeledAfter string  `json:"modeled_after,omitempty"` // real hardware it emulates
	Category     string  `json:"category,omitempty"`      // effects only
	Gain         string  `json:"gain,omitempty"`          // distortion effects: drive strength
	Description  string  `json:"description,omitempty"`
	Score        float64 `json:"score"`      // 0..1 relevance
	MatchedOn    string  `json:"matched_on"` // which field matched best
}

// Search fuzzy-matches a query against the device name, the real hardware each
// model emulates and the description of every amp, cab, mic and effect. kind,
// when non-empty, restricts results to one of "amp", "cab", "mic" or "fx".
func (c *Catalog) Search(query, kind string) []SearchResult {
	q := normalize(query)
	if q == "" {
		return nil
	}
	k := strings.ToLower(strings.TrimSpace(kind))

	var results []SearchResult
	if k == "" || k == "amp" {
		for _, a := range amps {
			hardware := a.ModeledAfter
			if hardware == "" {
				hardware = a.Brand + " " + a.RealModel
			}
			if r, ok := matchEntry("amp", a.Model, hardware, "", a.Description, q); ok {
				results = append(results, r)
			}
		}
	}
	if k == "" || k == "cab" {
		for _, cb := range cabs {
			hardware := cb.ModeledAfter
			if hardware == "" {
				hardware = cb.Speakers + " " + cb.SpeakersRef
			}
			if r, ok := matchEntry("cab", cb.Model, hardware, "", cb.Description, q); ok {
				results = append(results, r)
			}
		}
	}
	if k == "" || k == "mic" {
		for _, m := range mics {
			hardware := m.ModeledAfter
			if hardware == "" {
				hardware = m.Kind + " " + m.RealModel
			}
			if r, ok := matchEntry("mic", m.Model, hardware, "", m.Description, q); ok {
				results = append(results, r)
			}
		}
	}
	if k == "" || k == "fx" {
		for _, f := range fx {
			if r, ok := matchEntry("fx", f.Name, f.ModeledAfter, f.Category, f.Description, q); ok {
				r.Gain = f.Gain
				results = append(results, r)
			}
		}
	}

	sort.SliceStable(results, func(i, j int) bool { return results[i].Score > results[j].Score })
	if len(results) > 20 {
		results = results[:20]
	}
	return results
}

// matchEntry scores a query against a single catalog entry and reports which
// field matched best. Entries below the relevance threshold are dropped.
func matchEntry(typ, name, modeledAfter, category, description, q string) (SearchResult, bool) {
	fields := []struct{ label, value string }{
		{"name", name},
		{"modeled after", modeledAfter},
		{"category", category},
		{"description", description},
	}
	bestScore, bestLabel := 0.0, ""
	for _, f := range fields {
		if strings.TrimSpace(f.value) == "" {
			continue
		}
		if s := fuzzyScore(q, f.value); s > bestScore {
			bestScore = s
			bestLabel = f.label
		}
	}
	if bestScore < 0.3 {
		return SearchResult{}, false
	}
	return SearchResult{
		Type:         typ,
		Name:         name,
		ModeledAfter: strings.TrimSpace(modeledAfter),
		Category:     category,
		Description:  description,
		Score:        math.Round(bestScore*100) / 100,
		MatchedOn:    bestLabel,
	}, true
}

// fuzzyScore rates how well a query matches a target string: 1 for an exact
// substring, otherwise the better of word-overlap and character-bigram
// similarity (which tolerates typos).
func fuzzyScore(query, target string) float64 {
	q := normalize(query)
	t := normalize(target)
	if q == "" || t == "" {
		return 0
	}
	if strings.Contains(t, q) {
		return 1
	}

	qWords := strings.Fields(q)
	tWords := strings.Fields(t)
	var hits float64
	for _, qw := range qWords {
		best := 0.0
		for _, tw := range tWords {
			switch {
			case tw == qw:
				best = 1
			case len(qw) >= 3 && strings.HasPrefix(tw, qw):
				if best < 0.7 {
					best = 0.7
				}
			}
		}
		hits += best
	}
	tokenScore := hits / float64(len(qWords))

	bigram := diceCoefficient(stripSpaces(q), stripSpaces(t))
	if tokenScore > bigram {
		return tokenScore
	}
	return bigram
}

// diceCoefficient is the Sørensen–Dice similarity of two strings' character
// bigrams, in 0..1.
func diceCoefficient(a, b string) float64 {
	if len(a) < 2 || len(b) < 2 {
		return 0
	}
	set := make(map[string]struct{}, len(a)-1)
	for i := 0; i+1 < len(a); i++ {
		set[a[i:i+2]] = struct{}{}
	}
	shared := 0
	for i := 0; i+1 < len(b); i++ {
		if _, ok := set[b[i:i+2]]; ok {
			shared++
		}
	}
	return 2 * float64(shared) / float64(len(a)+len(b)-2)
}

func stripSpaces(s string) string {
	return strings.ReplaceAll(s, " ", "")
}
