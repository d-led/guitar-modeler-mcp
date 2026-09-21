// Command guitar-modeler-mcp designs and writes guitar-modeler presets for
// multiple hardware modelers — HeadRush Gigboard, Mooer, BOSS Waza Air,
// Yamaha THR, Neural DSP Quad Cortex and Valeton GP-200. It exposes an MCP
// server (serve) and a CLI for cataloguing device models, translating real
// hardware into device models, and generating presets.
package main

import (
	"fmt"
	"os"

	"github.com/d-led/guitar-modeler-mcp/cmd"
)

func main() {
	err := cmd.Execute()
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, "error:", err)
	if cmd.IsUsageError(err) {
		// A wrong command or flag should be self-explanatory: print the help
		// so the correct invocation is discoverable.
		fmt.Fprintln(os.Stderr)
		cmd.PrintHelp(os.Stderr)
	}
	os.Exit(1)
}
