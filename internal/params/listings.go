package params

import "github.com/dmitryledentsov/headrush-gigboard-mcp/internal/catalog"

// AmpListing enriches an amp with its capability keywords.
type AmpListing struct {
	catalog.Amp
	Capabilities []string `json:"capabilities"`
}

// FXListing enriches an effect with its capability keywords.
type FXListing struct {
	catalog.FX
	Capabilities []string `json:"capabilities"`
}

// AmpListings returns amps matching the query, each annotated with the
// capability keywords derived from the amp's parameter spec.
func AmpListings(cat *catalog.Catalog, query string) []AmpListing {
	caps := Capabilities(cat, "Amp")
	out := make([]AmpListing, 0)
	for _, a := range cat.AmpsMatching(query) {
		out = append(out, AmpListing{Amp: a, Capabilities: caps})
	}
	return out
}

// FXListings returns every effect module annotated with its capabilities.
func FXListings(cat *catalog.Catalog) []FXListing {
	out := make([]FXListing, 0)
	for _, f := range cat.FX() {
		out = append(out, FXListing{FX: f, Capabilities: Capabilities(cat, f.Name)})
	}
	return out
}

// FXCategories returns every effect category with its module count.
func FXCategories(cat *catalog.Catalog) []catalog.Category {
	return cat.Categories()
}

// FXListingsByCategory returns the effects in one category, annotated with
// capabilities. An empty result means the category name was not recognised.
func FXListingsByCategory(cat *catalog.Catalog, category string) []FXListing {
	out := make([]FXListing, 0)
	for _, f := range cat.FXByCategory(category) {
		out = append(out, FXListing{FX: f, Capabilities: Capabilities(cat, f.Name)})
	}
	return out
}
