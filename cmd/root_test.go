package cmd

import (
	"errors"
	"io"
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

func TestUnknownCommandIsUsageError(t *testing.T) {
	root := newRootCmd()
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	root.SetArgs([]string{"install"})

	err := root.Execute()

	if err == nil {
		t.Fatal("expected an error for an unknown command")
	}
	if !strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !IsUsageError(err) {
		t.Fatalf("expected unknown-command error to be a usage error: %v", err)
	}
}

func TestRuntimeErrorIsNotUsageError(t *testing.T) {
	if IsUsageError(nil) {
		t.Fatal("nil should not be a usage error")
	}
	if IsUsageError(errors.New("no amp matches \"nope\"")) {
		t.Fatal("a runtime error should not be a usage error")
	}
}

func TestPrintHelpListsCommands(t *testing.T) {
	var out strings.Builder
	PrintHelp(&out)
	for _, want := range []string{"Usage:", "catalog", "design", "mcp", "translate"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("help missing %q:\n%s", want, out.String())
		}
	}
}
