package cmd

import (
	"strings"
	"testing"
)

func TestVersionFlagReportsStampedVersion(t *testing.T) {
	// given a version stamped by release ldflags
	version = "9.9.9-stamp"
	t.Cleanup(func() { version = "0.1.0" })

	var out strings.Builder
	root := newRootCmd()
	root.SetOut(&out)
	root.SetArgs([]string{"--version"})

	// when the user asks for the version
	err := root.Execute()

	// then the stamped version is reported
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(out.String(), "9.9.9-stamp") {
		t.Fatalf("expected stamped version in output, got %q", out.String())
	}
}
