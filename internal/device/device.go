// Package device holds the shared catalog model and the name/resolver helpers
// used by every device-specific backend (Yamaha THR, Boss Waza Air, Mooer).
// Keeping them in one place means each backend normalises agent input and
// resolves model names identically.
package device

import (
	"fmt"
	"strings"
)

// Item is one named model on a device: the on-screen name and the real
// hardware it emulates (empty when not documented).
type Item struct {
	Name       string `json:"name"`
	InspiredBy string `json:"inspired_by,omitempty"`
}

// Items converts name/inspired-by pairs into Item values, preserving order.
func Items(pairs ...[2]string) []Item {
	out := make([]Item, len(pairs))
	for i, p := range pairs {
		out[i] = Item{Name: p[0], InspiredBy: p[1]}
	}
	return out
}

// CanonicalKey normalises an agent-supplied name to a canonical key:
// lower-case with every non-alphanumeric run collapsed to a single underscore,
// so "GAIN", "Time (ms)" and "time_ms" all match "time_ms".
func CanonicalKey(s string) string {
	var b strings.Builder
	lastUnderscore := false
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastUnderscore = false
		} else if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	return strings.Trim(b.String(), "_")
}

// Norm lower-cases, folds the middot separator and collapses whitespace, so
// "Fender Twin Reverb", "fender  twin reverb" and "Fender·Twin Reverb" all
// compare equal.
func Norm(s string) string {
	s = strings.ReplaceAll(s, "·", " ")
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(s))), " ")
}

// ResolveItem canonicalises a free-form selection to an on-device model name.
// An exact case-insensitive name match wins; otherwise the first item whose
// "inspired by" description contains the query (normalised) is returned. An
// empty query is left empty.
func ResolveItem(items []Item, query, label string) (string, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return "", nil
	}
	for _, it := range items {
		if strings.EqualFold(it.Name, q) {
			return it.Name, nil
		}
	}
	for _, it := range items {
		if it.InspiredBy != "" && strings.Contains(Norm(it.InspiredBy), Norm(q)) {
			return it.Name, nil
		}
	}
	return "", fmt.Errorf("no %s matches %q", label, query)
}

// FindByName returns the first model whose stable or display name matches the
// identifier case-insensitively.
func FindByName[T any](models []T, name string, key func(T) (stable, display string)) (T, bool) {
	q := strings.ToLower(strings.TrimSpace(name))
	for _, m := range models {
		stable, display := key(m)
		if strings.ToLower(stable) == q || strings.ToLower(display) == q {
			return m, true
		}
	}
	var zero T
	return zero, false
}
