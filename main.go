// Command headrush-gigboard-mcp designs and writes HeadRush Gigboard sound
// presets. It exposes an MCP server (serve) and a CLI for cataloguing device
// models, translating real hardware into device models, and generating rigs.
package main

import (
	"fmt"
	"os"

	"github.com/dmitryledentsov/headrush-gigboard-mcp/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
