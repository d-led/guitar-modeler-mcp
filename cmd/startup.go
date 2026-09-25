package cmd

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
	"time"
)

// buildTime is stamped by release builds via -ldflags, e.g.
//
//	-ldflags "-X github.com/d-led/guitar-modeler-mcp/cmd.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
//
// The Go toolchain records the VCS commit time (vcs.time) in a binary but no
// build timestamp of its own, so an unstamped build reports the executable's
// file time instead — see buildFacts.builtTime. Stamped times are UTC strings,
// displayed in the local timezone (see localTime).
var buildTime = ""

// logStartup writes one diagnostic line to stderr describing this binary, so an
// operator can tell which build an MCP server is running and where it lives.
// stdout must stay reserved for the JSON-RPC protocol, so this goes to stderr
// like the server's own per-call log.
func logStartup() {
	fmt.Fprintf(os.Stderr, "guitar-modeler-mcp starting: %s\n", buildInfo())
}

// effectiveVersion resolves the version to report: the stamped release version
// when present, otherwise the module version the Go toolchain recorded in the
// binary (e.g. "v0.0.8" for `go install module@v0.0.8`, "(devel)" for a local
// build), and "devel" only when neither is available.
func effectiveVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "devel"
}

// buildInfo returns the full identity line for the startup log, the version
// plus the same build metadata Cobra's --version reports.
func buildInfo() string {
	return "version " + versionDetails()
}

// versionDetails returns the semver plus build metadata, without the leading
// "version " — Cobra's --version template adds that prefix itself.
func versionDetails() string {
	return probeBuild().describe()
}

// buildFacts is the identity of the running binary, assembled from the build
// info the Go toolchain embedded and from the executable on disk. Gathering it
// in one place keeps the rendering in describe free of environment lookups, so
// the reported line can be specified end to end by tests.
type buildFacts struct {
	version   string    // stamped release version, else module version, else "devel"
	stamped   string    // build time injected via -ldflags, "" when unstamped
	commit    string    // VCS commit time, "" when the toolchain recorded none
	revision  string    // VCS revision, "" when the toolchain recorded none
	goVersion string    // toolchain that compiled the binary
	platform  string    // GOOS/GOARCH the binary was compiled for
	path      string    // absolute path of the running executable, "" if unknown
	fileTime  time.Time // executable modification time, zero when unreadable
}

// probeBuild gathers the facts about the binary that is currently running.
func probeBuild() buildFacts {
	vcs := readVCS()
	facts := buildFacts{
		version:   effectiveVersion(),
		stamped:   buildTime,
		commit:    vcs.time,
		revision:  vcs.revision,
		goVersion: runtime.Version(),
		platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}
	exe, err := os.Executable()
	if err != nil {
		return facts
	}
	facts.path = exe
	if info, err := os.Stat(exe); err == nil {
		facts.fileTime = info.ModTime()
	}
	return facts
}

// describe renders the facts as one comma-separated line. Every part is
// optional, so the line stays correct for a binary built outside a checkout or
// with -buildvcs=false, for a stamped build, and for one whose file time cannot
// be read.
func (f buildFacts) describe() string {
	parts := []string{f.version}
	if built := f.builtTime(); built != "" {
		parts = append(parts, built)
	}
	if commit := localTime(f.commit); commit != "" {
		parts = append(parts, "commit "+commit)
	}
	parts = append(parts, f.goVersion, f.platform)
	if f.revision != "" {
		parts = append(parts, "rev "+f.revision)
	}
	if f.path != "" {
		parts = append(parts, "binary "+f.path)
	}
	return strings.Join(parts, ", ")
}

// builtTime reports when this binary was built. A release binary carries the
// exact time stamped into it at build time; a local build is left unstamped by
// the Go toolchain, so the executable's file time is the closest available
// answer — to the second, since it is an approximation anyway. The two are
// labelled apart, because a file time says only when the file was last written:
// a copy or download can move it.
func (f buildFacts) builtTime() string {
	if f.stamped != "" {
		return "built " + localTime(f.stamped)
	}
	if !f.fileTime.IsZero() {
		return "mtime " + f.fileTime.Local().Format(time.RFC3339)
	}
	return ""
}

// localTime converts a UTC RFC3339 timestamp into the local timezone, keeping
// the same instant (the offset changes; fractional seconds are preserved when
// present). A string that is not a recognised timestamp is returned unchanged,
// so a hand-stamped build time of another shape still prints as-is.
func localTime(s string) string {
	t, ok := parseTime(s)
	if !ok {
		return s
	}
	return t.Local().Format(time.RFC3339Nano)
}

// parseTime parses a timestamp with either full RFC3339 (with or without
// fractional seconds) precision.
func parseTime(s string) (time.Time, bool) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// vcsInfo is the version-control metadata the Go toolchain embedded at build
// time. Its fields are empty when the binary was built outside a checkout or
// with -buildvcs=false.
type vcsInfo struct {
	revision string
	time     string
	dirty    bool
}

func readVCS() vcsInfo {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return vcsInfo{}
	}
	var v vcsInfo
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			v.revision = s.Value
		case "vcs.time":
			v.time = s.Value
		case "vcs.modified":
			v.dirty = s.Value == "true"
		}
	}
	if v.dirty {
		v.revision += " (dirty)"
	}
	return v
}
