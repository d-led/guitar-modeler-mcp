package rig

import "testing"

func TestStoredNameUppercasesAndTruncates(t *testing.T) {
	cases := map[string]string{
		"Brown Sound":                           "BROWN SOUND",
		"Vai - For the Love of God":             "VAI - FOR THE LOVE OF",
		"SATCH - ALWAYS WITH ME - CLEAN":        "SATCH - ALWAYS WITH M",
		"A Very Long Preset Name Beyond Limits": "A VERY LONG PRESET NA",
	}
	for in, want := range cases {
		got, truncated := StoredName(in)
		if got != want {
			t.Errorf("StoredName(%q) = %q, want %q", in, got, want)
		}
		if truncated != (len(in) > NameLimit) {
			t.Errorf("StoredName(%q) truncated = %v, want %v", in, truncated, len(in) > NameLimit)
		}
	}
}
