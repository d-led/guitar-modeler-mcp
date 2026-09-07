package rig

import (
	"bytes"
	"testing"
)

// wantEq fails when got differs from want, naming the checked field so each
// assertion reads as a sentence.
func wantEq[T comparable](t *testing.T, name string, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
}

// wantBytes fails unless got equals want, printing both slices when they differ.
func wantBytes(t *testing.T, name string, got, want []byte) {
	t.Helper()
	if !bytes.Equal(got, want) {
		t.Fatalf("%s = %v, want %v", name, got, want)
	}
}
