package fileutil

import "testing"

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
