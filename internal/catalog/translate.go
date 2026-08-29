package catalog

import (
	"sort"
	"strings"
)

// Match is a translation result: the device model that best corresponds to a
// real-world hardware description, together with a relevance score and a short
// human readable reason for the match.
type Match struct {
	Amp    Amp    `json:"amp"`
	Score  int    `json:"score"`
	Reason string `json:"reason"`
}

// ampSynonyms maps common ways of describing real amplifiers to canonical
// tokens that appear in the catalog data. This is the core of the "translation
// layer": it lets the agent ask for "a JCM800" or "a blackface deluxe reverb"
// and get back the exact HeadRush model.
var ampSynonyms = map[string]string{
	"6505":           "5150",
	"evh":            "5150",
	"5150iii":        "5150",
	"dual rec":       "dual rectifier",
	"recto":          "rectifier",
	"mark ii":        "mark iic+",
	"mark 2":         "mark iic+",
	"mark iic":       "mark iic+",
	"iic+":           "mark iic+",
	"superlead":      "super lead",
	"super lead":     "super lead",
	"plexi":          "super lead",
	"slo100":         "slo-100",
	"slo":            "slo-100",
	"topboost":       "top boost",
	"top boost":      "top boost",
	"twin reverb":    "twin reverb",
	"deluxe reverb":  "deluxe reverb",
	"super reverb":   "super reverb",
	"vibroverb":      "vibroverb",
	"bassman":        "bassman",
	"jcm800":         "jcm800",
	"jcm 800":        "jcm800",
	"jtm45":          "jtm45",
	"jtm 45":         "jtm45",
	"jtm":            "jtm45",
	"powerball":      "powerball",
	"ecstasy":        "ecstasy",
	"dual rectifier": "dual rectifier",
	"rectifier":      "rectifier",
	"ad30":           "ad30",
	"dc30":           "dc30",
	"ac30":           "ac30",
	"elf":            "elf",
	"svt":            "svt",
	"portaflex":      "portaflex",
	"champ":          "champ",
}

// normalize lower-cases a string and collapses whitespace/punctuation.
func normalize(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		default:
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}

// expandSynonyms rewrites a normalized query so that common real-world names
// map onto the tokens used in the catalog.
func expandSynonyms(query string) string {
	out := query
	// Longest keys first so multi-word synonyms win over single words.
	keys := make([]string, 0, len(ampSynonyms))
	for k := range ampSynonyms {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return len(keys[i]) > len(keys[j]) })
	for _, k := range keys {
		out = strings.ReplaceAll(out, k, ampSynonyms[k])
	}
	return normalize(out)
}

func containsWord(haystack, word string) bool {
	if word == "" {
		return false
	}
	for _, h := range strings.Fields(haystack) {
		if h == word {
			return true
		}
	}
	return false
}

// TranslateAmp returns amp models that match a free-form hardware description,
// ranked by relevance. An empty or non-matching query returns no results.
func (c *Catalog) TranslateAmp(query string) []Match {
	q := expandSynonyms(query)
	if q == "" {
		return nil
	}
	words := strings.Fields(q)

	matches := make([]Match, 0)
	for _, a := range amps {
		brand := normalize(a.Brand)
		realModel := normalize(a.RealModel)
		model := normalize(a.Model)
		channel := normalize(a.Channel)
		wattage := normalize(a.Wattage)
		desc := normalize(a.Description)
		var styles []string
		for _, s := range a.Style {
			styles = append(styles, normalize(s))
		}

		score := 0
		var reasons []string

		if brand != "" && (brand == q || strings.Contains(q, brand) || strings.Contains(brand, q)) {
			score += 4
			reasons = append(reasons, "brand "+a.Brand)
		}
		if realModel != "" && (strings.Contains(q, realModel) || strings.Contains(realModel, q)) {
			score += 5
			reasons = append(reasons, "model "+a.RealModel)
		}

		for _, w := range words {
			if containsWord(realModel, w) {
				score += 2
			}
			if containsWord(brand, w) {
				score += 2
			}
			if containsWord(model, w) {
				score++
			}
			if containsWord(channel, w) {
				score++
			}
			if containsWord(wattage, w) {
				score++
			}
			for _, st := range styles {
				if st == w {
					score++
				}
			}
			if containsWord(desc, w) {
				// Description overlap is a weak signal only.
			}
		}

		if score == 0 {
			continue
		}

		reason := strings.Join(reasons, ", ")
		if reason == "" {
			reason = "close match"
		}
		matches = append(matches, Match{Amp: a, Score: score, Reason: reason})
	}

	sort.SliceStable(matches, func(i, j int) bool { return matches[i].Score > matches[j].Score })
	return matches
}

// TranslateCab returns cabinet models matching a free-form description.
func (c *Catalog) TranslateCab(query string) []Cab {
	q := normalize(query)
	if q == "" {
		return nil
	}
	type scored struct {
		cab   Cab
		score int
	}
	var found []scored
	for _, cb := range cabs {
		score := 0
		hay := normalize(cb.Model + " " + cb.Speakers + " " + cb.SpeakersRef + " " + cb.Description)
		if strings.Contains(hay, q) {
			score += 3
		}
		for _, w := range strings.Fields(q) {
			if containsWord(hay, w) {
				score++
			}
		}
		if score > 0 {
			found = append(found, scored{cb, score})
		}
	}
	sort.SliceStable(found, func(i, j int) bool { return found[i].score > found[j].score })
	out := make([]Cab, 0, len(found))
	for _, f := range found {
		out = append(out, f.cab)
	}
	return out
}

// TranslateMic returns microphone models matching a free-form description.
func (c *Catalog) TranslateMic(query string) []Mic {
	q := normalize(query)
	if q == "" {
		return nil
	}
	type scored struct {
		mic   Mic
		score int
	}
	var found []scored
	for _, m := range mics {
		hay := normalize(m.Model + " " + m.Kind + " " + m.RealModel + " " + m.Description)
		score := 0
		if strings.Contains(hay, q) {
			score += 3
		}
		for _, w := range strings.Fields(q) {
			if containsWord(hay, w) {
				score++
			}
		}
		if score > 0 {
			found = append(found, scored{m, score})
		}
	}
	sort.SliceStable(found, func(i, j int) bool { return found[i].score > found[j].score })
	out := make([]Mic, 0, len(found))
	for _, f := range found {
		out = append(out, f.mic)
	}
	return out
}
