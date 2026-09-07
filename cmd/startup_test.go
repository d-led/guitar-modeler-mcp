package cmd

import (
	"testing"
	"time"
)

func TestLocalTimePreservesInstant(t *testing.T) {
	for _, in := range []string{
		"2026-09-07T12:45:11Z",
		"2026-09-07T12:45:11.123Z",
	} {
		want, err := time.Parse(time.RFC3339Nano, in)
		if err != nil {
			t.Fatalf("parse %q: %v", in, err)
		}
		got := localTime(in)
		parsed, err := time.Parse(time.RFC3339Nano, got)
		if err != nil {
			t.Fatalf("localTime(%q) = %q, not a valid RFC3339 timestamp: %v", in, got, err)
		}
		if !parsed.Equal(want) {
			t.Fatalf("localTime(%q) = %q, changed the instant to %v", in, got, parsed)
		}
	}
}

func TestLocalTimePassesThroughNonTimestamps(t *testing.T) {
	for _, in := range []string{"", "not a timestamp", "2026/09/07 12:45:11"} {
		if got := localTime(in); got != in {
			t.Fatalf("localTime(%q) = %q, want unchanged", in, got)
		}
	}
}
