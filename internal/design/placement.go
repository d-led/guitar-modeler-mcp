package design

// categoryPlacement is the conventional position of one effect category,
// together with the reason, so an agent can place effects sensibly.
type categoryPlacement struct {
	Category string
	Place    string // "pre-amp" or "post-amp"
	Why      string
}

// categoryPlacements is the single source of truth for where each effect
// category belongs in a chain. The designer's automatic ordering and the
// placement guidance tool both derive from it.
var categoryPlacements = []categoryPlacement{
	{"distortion", "pre-amp", "drives and fuzzes push the amp into breakup"},
	{"dynamics", "pre-amp", "compressors and gates shape the dry guitar signal"},
	{"eq", "pre-amp", "tone shaping before the amp's gain stages"},
	{"expression", "pre-amp", "wahs, whammy and volume pedals react to the dry signal"},
	{"modulation", "post-amp", "chorus, flanger and phaser sound best after distortion"},
	{"delay", "post-amp", "echoes stay clean after the cab"},
	{"reverb", "post-amp", "reverb is the room after the amp"},
	{"utility", "post-amp", "IR, FX-loop and acoustic simulation act after the cab"},
}

// placeForCategory returns where an effect category belongs ("pre-amp" or
// "post-amp"). Unknown categories default to post-amp.
func placeForCategory(category string) string {
	for _, cp := range categoryPlacements {
		if cp.Category == category {
			return cp.Place
		}
	}
	return "post-amp"
}

// PlacementCategory is one effect category with its conventional position.
type PlacementCategory struct {
	Category string `json:"category"`
	Place    string `json:"place"` // "pre-amp" or "post-amp"
	Why      string `json:"why"`
}

// PlacementSection is one segment of a routing with its slot budget and what
// belongs there.
type PlacementSection struct {
	Name  string `json:"name"`
	Slots int    `json:"slots"`
	Note  string `json:"note"`
}

// PlacementRouting is the placement guidance for one routing topology.
type PlacementRouting struct {
	Routing     string             `json:"routing"`
	Description string             `json:"description"`
	Sections    []PlacementSection `json:"sections"`
}

// Placement is the effect-placement guidance: which categories go before vs
// after the amp, and how each routing topology's sections map onto those slots.
type Placement struct {
	Categories []PlacementCategory `json:"categories"`
	AlwaysLast string              `json:"always_last"` // the Volume pedal
	Routings   []PlacementRouting  `json:"routings"`
}

// PlacementGuide returns the placement guidance for every routing topology.
func PlacementGuide() Placement {
	categories := make([]PlacementCategory, 0, len(categoryPlacements))
	for _, cp := range categoryPlacements {
		categories = append(categories, PlacementCategory(cp))
	}

	return Placement{
		Categories: categories,
		AlwaysLast: "Volume",
		Routings: []PlacementRouting{
			{
				Routing:     "S",
				Description: "serial — one 11-slot path",
				Sections: []PlacementSection{
					{Name: "chain", Slots: 11, Note: "pre-amp effects → amp → cab → post-amp effects → Volume"},
				},
			},
			{
				Routing:     "SPS-1",
				Description: "serial → parallel → serial",
				Sections: []PlacementSection{
					{Name: "prefix", Slots: 3, Note: "shared, before the split — pre-amp effects and/or the shared amp+cab"},
					{Name: "path A", Slots: 3, Note: "parallel path A — an amp+cab pair (dual-amp) or any effects (shared-amp)"},
					{Name: "path B", Slots: 3, Note: "parallel path B"},
					{Name: "suffix", Slots: 2, Note: "shared, after the merge — post-amp effects (only 2 slots)"},
				},
			},
			{
				Routing:     "PS-1",
				Description: "parallel from the input → serial",
				Sections: []PlacementSection{
					{Name: "path A", Slots: 3, Note: "split at the input — a full path: pre-amp effects → amp → cab"},
					{Name: "path B", Slots: 5, Note: "second full path: pre-amp effects → amp → cab"},
					{Name: "suffix", Slots: 3, Note: "shared, after the merge — post-amp effects"},
				},
			},
		},
	}
}
