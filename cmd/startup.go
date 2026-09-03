package cmd

import (
	"fmt"
	"os"
	"runtime"
	"runtime/debug"
	"strings"
)

// buildTime is stamped by release builds via -ldflags, e.g.
//
//	-ldflags "-X github.com/d-led/guitar-modeler-mcp/cmd.buildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
//
// It stays empty in development builds, which fall back to the commit time the
// Go toolchain embedded at build time (vcs.time).
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

// buildInfo returns the full identity line for the startup log: the semver plus
// build/commit time, runtime, platform, VCS revision and binary path.
func buildInfo() string {
	return "version " + versionDetails()
}

// versionDetails returns the semver plus build metadata, without the leading
// "version " — Cobra's --version template adds that prefix itself. Every part
// is optional, so the line stays correct for builds with no VCS metadata (built
// outside a checkout or with -buildvcs=false) and no stamped build time.
func versionDetails() string {
	vcs := readVCS()
	parts := []string{effectiveVersion()}
	switch {
	case buildTime != "":
		parts = append(parts, "built "+buildTime)
	case vcs.time != "":
		parts = append(parts, "commit "+vcs.time)
	}
	parts = append(parts, runtime.Version(), runtime.GOOS+"/"+runtime.GOARCH)
	if vcs.revision != "" {
		parts = append(parts, "rev "+vcs.revision)
	}
	if exe, err := os.Executable(); err == nil {
		parts = append(parts, "binary "+exe)
	}
	return strings.Join(parts, ", ")
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
