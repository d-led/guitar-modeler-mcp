package cmd

import (
	"os"
	"runtime"
	"strings"
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

// TestStampedBuildReportsBuildAndCommitTimes pins the point of the startup
// line: which build is running (the stamped build time) and which source it was
// built from (the commit time) are two different facts, so both are reported.
func TestStampedBuildReportsBuildAndCommitTimes(t *testing.T) {
	// given a release binary stamped with its build time, built from a commit
	facts := buildFacts{
		version:   "1.2.3",
		stamped:   "2026-09-25T18:30:36Z",
		commit:    "2026-09-24T10:00:00Z",
		revision:  "abc123",
		goVersion: "go1.27.1",
		platform:  "darwin/arm64",
	}

	// when the facts are rendered
	line := facts.describe()

	// then both times appear, each as its own instant
	assertStamp(t, line, "built ", "2026-09-25T18:30:36Z")
	assertStamp(t, line, "commit ", "2026-09-24T10:00:00Z")
}

// TestUnstampedBuildReportsTheBinaryFileTime covers a local build, for which
// the Go toolchain records no build timestamp: the executable's file time is
// the closest available answer, and is labelled as such so it is not mistaken
// for a time a release build stamped in.
func TestUnstampedBuildReportsTheBinaryFileTime(t *testing.T) {
	// given a binary built without a stamped build time
	facts := buildFacts{
		version:   "devel",
		commit:    "2026-09-24T10:00:00Z",
		goVersion: "go1.27.1",
		platform:  "darwin/arm64",
		fileTime:  time.Date(2026, 9, 25, 18, 30, 36, 0, time.UTC),
	}

	// when the facts are rendered
	line := facts.describe()

	// then the file time stands in for the build time, labelled as a file time
	assertStamp(t, line, "mtime ", "2026-09-25T18:30:36Z")
	if strings.Contains(line, "built ") {
		t.Fatalf("an unstamped build must not claim a stamped build time: %q", line)
	}
}

func TestStampedBuildTimeWinsOverTheFileTime(t *testing.T) {
	// given a release binary whose file was written again after the build
	facts := buildFacts{
		version:   "1.2.3",
		stamped:   "2026-09-25T18:30:36Z",
		goVersion: "go1.27.1",
		platform:  "darwin/arm64",
		fileTime:  time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC),
	}

	// when the facts are rendered
	line := facts.describe()

	// then the exact build time is reported, not the file time
	assertStamp(t, line, "built ", "2026-09-25T18:30:36Z")
	if strings.Contains(line, "mtime ") {
		t.Fatalf("the stamped build time should replace the file time: %q", line)
	}
}

func TestDescribeOmitsMetadataTheBinaryDoesNotCarry(t *testing.T) {
	// given a local build outside a checkout: no VCS metadata, no file time
	facts := buildFacts{version: "devel", goVersion: "go1.27.1", platform: "darwin/arm64"}

	// when the facts are rendered
	line := facts.describe()

	// then nothing is printed for the missing parts, not even a stray separator
	if want := "devel, go1.27.1, darwin/arm64"; line != want {
		t.Fatalf("describe() = %q, want %q", line, want)
	}
}

func TestProbeBuildDescribesTheRunningBinary(t *testing.T) {
	// when the facts are gathered for the running test binary
	facts := probeBuild()

	// then the executable on disk and the toolchain that produced it are described
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("locating the running executable: %v", err)
	}
	if facts.path != exe {
		t.Fatalf("path = %q, want %q", facts.path, exe)
	}
	if facts.fileTime.IsZero() {
		t.Fatal("expected the executable's modification time to be read")
	}
	if facts.goVersion != runtime.Version() {
		t.Fatalf("goVersion = %q, want %q", facts.goVersion, runtime.Version())
	}
	if want := runtime.GOOS + "/" + runtime.GOARCH; facts.platform != want {
		t.Fatalf("platform = %q, want %q", facts.platform, want)
	}
	if facts.version == "" {
		t.Fatal("expected a version, even for an unstamped build")
	}
}

// assertStamp checks that line carries a "<prefix><timestamp>" field holding the
// same instant as wantUTC, whichever timezone this machine renders it in.
func assertStamp(t *testing.T, line, prefix, wantUTC string) {
	t.Helper()
	want, err := time.Parse(time.RFC3339, wantUTC)
	if err != nil {
		t.Fatalf("bad test timestamp %q: %v", wantUTC, err)
	}
	for _, field := range strings.Split(line, ", ") {
		stamp, isStamp := strings.CutPrefix(field, prefix)
		if !isStamp {
			continue
		}
		got, err := time.Parse(time.RFC3339Nano, stamp)
		if err != nil {
			t.Fatalf("field %q is not a timestamp: %v", field, err)
		}
		if !got.Equal(want) {
			t.Fatalf("field %q holds %v, want the instant %v", field, got, want)
		}
		return
	}
	t.Fatalf("no %q field in %q", prefix, line)
}
