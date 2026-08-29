package qc

import (
	"fmt"
	"strings"
)

// ResolveAmp finds an amp model for a query: exact name first, then a
// case-insensitive substring match against the "inspired by" hardware.
func (d Device) ResolveAmp(query string) (Item, error) {
	return resolve(query, append(append([]Item{}, d.Amps...), d.BassAmps...), "amp")
}

// ResolveCab finds a cabinet model for a query.
func (d Device) ResolveCab(query string) (Item, error) {
	return resolve(query, d.Cabs, "cab")
}

// ResolveFX finds an effect model in a category (drive, compressor,
// equalizer, delay, modulation, reverb, wah, pitch, filter or gate).
func (d Device) ResolveFX(category, query string) (Item, error) {
	cat, ok := d.Effects[strings.ToLower(strings.TrimSpace(category))]
	if !ok {
		return Item{}, fmt.Errorf("unknown effect category %q", category)
	}
	return resolve(query, cat, category)
}

func resolve(query string, items []Item, label string) (Item, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return Item{}, fmt.Errorf("a %s is required", label)
	}
	for _, it := range items {
		if strings.EqualFold(it.Name, q) {
			return it, nil
		}
	}
	for _, it := range items {
		if it.InspiredBy != "" && strings.Contains(strings.ToLower(it.InspiredBy), strings.ToLower(q)) {
			return it, nil
		}
	}
	return Item{}, fmt.Errorf("no %s matches %q", label, q)
}
