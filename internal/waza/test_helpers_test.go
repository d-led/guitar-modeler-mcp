package waza

import "testing"

// wantEq fails when got differs from want, naming the checked field.
func wantEq[T comparable](t *testing.T, name string, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
}

// wantIndex fails when a type-index map has the wrong value for a key.
func wantIndex(t *testing.T, name string, m map[string]byte, key string, want byte) {
	t.Helper()
	if got := m[key]; got != want {
		t.Fatalf("%s[%q] = %d, want %d", name, key, got, want)
	}
}
