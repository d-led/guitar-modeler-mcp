package fileutil

import (
	"os"
	"path/filepath"
	"testing"
)

// An output directory a caller names should not have to exist first: the writer
// makes it, so a card can be written into a folder the tool has not used before.
func TestWriteFileCreatesTheDirectoryItWritesInto(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cards", "song", "tone.mo")

	if err := WriteFile(path, []byte("preset")); err != nil {
		t.Fatalf("WriteFile into a directory that does not exist yet: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading the file back: %v", err)
	}
	if string(got) != "preset" {
		t.Fatalf("content = %q, want %q", got, "preset")
	}
}

func TestSanitizeName(t *testing.T) {
	cases := map[string]string{
		"Brown Sound":            "Brown Sound",
		"Rig v1.2 (final)":       "Rig v1.2 _final_",
		"café über-alles":        "caf_ _ber-alles",
		"Tone 大阪":                "Tone __",
		"no/slash\\back:colon":   "no_slash_back_colon",
		"  spaces  around  ":     "spaces  around",
		".dot..leading..trail..": ".dot..leading..trail",
		"sl/ash:chars":           "sl_ash_chars",
		"trailing...":            "trailing",
		"UPPER_lower-123.ok":     "UPPER_lower-123.ok",
	}
	for in, want := range cases {
		if got := SanitizeName(in); got != want {
			t.Errorf("SanitizeName(%q) = %q, want %q", in, got, want)
		}
	}

	// Every rune in the output must be printable ASCII.
	for _, r := range SanitizeName("café 大阪 ünïcode 中文") {
		if r < 0x20 || r > 0x7e {
			t.Fatalf("non-ASCII rune %q leaked into filename", r)
		}
	}
}
