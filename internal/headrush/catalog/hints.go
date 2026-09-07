package catalog

import (
	_ "embed"
	"encoding/json"
	"strings"
)

//go:embed data/hints.json
var hintsJSON []byte

type hintEntry struct {
	HeadrushName string `json:"headrushName"`
	Brand        string `json:"brand"`
	Model        string `json:"model"`
	Confirmed    bool   `json:"confirmed"`
}

type hintsData struct {
	Amps []hintEntry `json:"amps"`
	Cabs []hintEntry `json:"cabs"`
	Mics []hintEntry `json:"mics"`
	FXs  []hintEntry `json:"fxs"`
}

// nameAliases maps hint headrush names (lower-cased) to the device model names
// used by the catalog, for entries the hints misspell or abbreviate.
var nameAliases = map[string]string{
	"autosweel": "auto swell",
	"2x12":      "2x12 b30",
}

// brandCorrections fixes obvious typos in the community-maintained brand names.
var brandCorrections = map[string]string{
	"Mashall":          "Marshall",
	"Sehnnheiser":      "Sennheiser",
	"Elektro-Harmonix": "Electro-Harmonix",
}

func cleanHint(s string) string {
	s = strings.TrimSpace(s)
	if s == "-" || s == "" {
		return ""
	}
	return s
}

// modeledAfter combines a brand and model into a human-readable emulation
// target, avoiding duplication when the model already names the brand.
func modeledAfter(brand, model string) string {
	b := cleanHint(brand)
	if corrected, ok := brandCorrections[b]; ok {
		b = corrected
	}
	m := cleanHint(model)
	switch {
	case b != "" && m != "":
		if strings.HasPrefix(strings.ToLower(m), strings.ToLower(b)) {
			return m
		}
		return b + " " + m
	case m != "":
		return m
	case b != "":
		return b
	}
	return ""
}

func indexHints(entries []hintEntry) map[string]hintEntry {
	m := make(map[string]hintEntry, len(entries))
	for _, e := range entries {
		key := strings.ToLower(e.HeadrushName)
		m[key] = e
		if alias, ok := nameAliases[key]; ok {
			m[alias] = e
		}
	}
	return m
}

func lookupHint(m map[string]hintEntry, name string) (hintEntry, bool) {
	e, ok := m[strings.ToLower(name)]
	return e, ok
}

// init enriches the static catalog with the Gigboard Hints knowledge: the
// emulation target (modeled_after) and whether the community confirmed it.
func init() {
	var h hintsData
	if err := json.Unmarshal(hintsJSON, &h); err != nil {
		return
	}
	enrichAmps(indexHints(h.Amps))
	enrichCabs(indexHints(h.Cabs))
	enrichMics(indexHints(h.Mics))
	enrichFX(indexHints(h.FXs))
}

func enrichAmps(by map[string]hintEntry) {
	for i := range amps {
		if e, ok := lookupHint(by, amps[i].Model); ok {
			amps[i].ModeledAfter = modeledAfter(e.Brand, e.Model)
			amps[i].Confirmed = e.Confirmed
		}
		if amps[i].ModeledAfter == "" {
			amps[i].ModeledAfter = modeledAfter(amps[i].Brand, amps[i].RealModel)
		}
	}
}

func enrichCabs(by map[string]hintEntry) {
	for i := range cabs {
		if e, ok := lookupHint(by, cabs[i].Model); ok {
			cabs[i].ModeledAfter = modeledAfter(e.Brand, e.Model)
			cabs[i].Confirmed = e.Confirmed
		}
	}
}

func enrichMics(by map[string]hintEntry) {
	for i := range mics {
		if e, ok := lookupHint(by, mics[i].Model); ok {
			mics[i].ModeledAfter = modeledAfter(e.Brand, e.Model)
			mics[i].Confirmed = e.Confirmed
		}
		if mics[i].ModeledAfter == "" {
			mics[i].ModeledAfter = modeledAfter("", mics[i].RealModel)
		}
	}
}

func enrichFX(by map[string]hintEntry) {
	for i := range fx {
		if e, ok := lookupHint(by, fx[i].Name); ok {
			fx[i].ModeledAfter = modeledAfter(e.Brand, e.Model)
			fx[i].Confirmed = e.Confirmed
		}
	}
}
