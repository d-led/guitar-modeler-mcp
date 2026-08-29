package setlist

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestName(t *testing.T) {
	s, err := New("  My Song  ", []Entry{{ID: "11111111-1111-4111-8111-111111111111", Name: "Clean"}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if s.Name() != "My Song" {
		t.Fatalf("Name() = %q, want trimmed %q", s.Name(), "My Song")
	}
}

func TestSanitizeFileName(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want string
	}{
		{"Brown Sound", "Brown Sound"},
		{"sl/ash:chars", "sl_ash_chars"},
		{"trailing...", "trailing"},
		{"  padded  ", "padded"},
		{"UPPER_lower-123.ok", "UPPER_lower-123.ok"},
	} {
		if got := sanitizeFileName(tc.in); got != tc.want {
			t.Fatalf("sanitizeFileName(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

func TestWrite(t *testing.T) {
	s, err := New("My Song", []Entry{{ID: "11111111-1111-4111-8111-111111111111", Name: "Clean"}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	dir := t.TempDir()
	path, err := s.Write(dir)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	want := filepath.Join(dir, "My Song.setlist")
	if path != want {
		t.Fatalf("Write path = %q, want %q", path, want)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read setlist: %v", err)
	}
	if !strings.Contains(string(data), `"rig_names":["Clean"]`) {
		t.Fatalf("setlist content missing rig names: %s", data)
	}
}

func TestWriteSanitizesFileName(t *testing.T) {
	s, err := New("Bad/Name: Here", []Entry{{ID: "11111111-1111-4111-8111-111111111111", Name: "Clean"}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	path, err := s.Write(t.TempDir())
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if filepath.Base(path) != "Bad_Name_ Here.setlist" {
		t.Fatalf("file name = %q, want sanitized", filepath.Base(path))
	}
}
