// Package qcctl shells out to pyquadcortex's qcctl CLI for live Quad Cortex
// control over USB-HID. It is a thin, safe adapter: it looks the binary up on
// PATH, builds an argument vector (no shell interpolation) and captures the
// combined output. Because this talks to a physical unit, callers must
// confirm with the user first — see the confirm guard in the MCP tool.
package qcctl

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Command describes one qcctl invocation.
type Command struct {
	Sub     string // "version" | "recall" | "scene" | "dump-preset"
	Slot    string // for recall / dump-preset
	Scene   int    // for scene; -1 means unset
	Setlist string // optional setlist path for recall / dump-preset
}

// Argv builds the argument vector for the command, validating its inputs.
func (c Command) Argv() ([]string, error) {
	args := []string{c.Sub}
	switch c.Sub {
	case "version":
	case "recall", "dump-preset":
		if strings.TrimSpace(c.Slot) == "" {
			return nil, fmt.Errorf("%s needs a --slot", c.Sub)
		}
		if c.Setlist != "" {
			args = append(args, "--setlist", c.Setlist)
		}
		args = append(args, "--slot", c.Slot)
	case "scene":
		if c.Scene < 0 {
			return nil, fmt.Errorf("scene needs a scene number")
		}
		args = append(args, "--index", strconv.Itoa(c.Scene))
	default:
		return nil, fmt.Errorf("unknown qcctl command %q (version, recall, scene, dump-preset)", c.Sub)
	}
	return args, nil
}

// String renders the command for reporting, quoting the slot.
func (c Command) String() string {
	argv, err := c.Argv()
	if err != nil {
		return c.Sub
	}
	return "qcctl " + strings.Join(argv[1:], " ")
}

// Available returns the path to the qcctl binary, or an error explaining how
// to install it.
func Available() (string, error) {
	path, err := exec.LookPath("qcctl")
	if err != nil {
		return "", fmt.Errorf("qcctl is not on PATH: pip install pyquadcortex (macOS also needs `brew install hidapi`)")
	}
	return path, nil
}

// Run executes the command and returns its combined stdout+stderr.
func Run(ctx context.Context, cmd Command) (string, error) {
	path, err := Available()
	if err != nil {
		return "", err
	}
	argv, err := cmd.Argv()
	if err != nil {
		return "", err
	}
	out, err := exec.CommandContext(ctx, path, argv...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("qcctl %s failed: %v\n%s", cmd.String(), err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}
