// Package golden implements a minimal approval-testing helper: actual output is
// compared against a committed golden file under testdata/. Run tests with
// UPDATE_GOLDEN=1 to (re)write the golden files when intentional changes occur.
package golden

import (
	"os"
	"path/filepath"
	"testing"
)

// Assert compares got with the golden file testdata/<name>.golden relative to
// the caller's package. It fails if the file is missing or the bytes differ.
func Assert(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("testdata", name+".golden")

	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("mkdir for golden file: %v", err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatalf("write golden file: %v", err)
		}
		return
	}

	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("golden file %s missing (run UPDATE_GOLDEN=1 go test ./... to create it): %v", path, err)
	}
	if string(want) != string(got) {
		t.Fatalf("output differs from %s.\n--- got ---\n%s\n--- want ---\n%s", path, got, want)
	}
}
