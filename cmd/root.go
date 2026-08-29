// Package cmd wires the domain packages into a Cobra CLI.
package cmd

import (
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/d-led/guitar-modeler-mcp/internal/catalog"
	"github.com/d-led/guitar-modeler-mcp/internal/design"
	"github.com/d-led/guitar-modeler-mcp/internal/mooer"
	"github.com/d-led/guitar-modeler-mcp/internal/presetmap"
	"github.com/d-led/guitar-modeler-mcp/internal/rig"
)

// version is reported by the MCP server and the --version flag. It is a var
// (not a const) so release builds can stamp it via -ldflags "-X .../cmd.version=".
var version = "0.1.0"

// app bundles the shared dependencies for all commands.
type app struct {
	cat     *catalog.Catalog
	builder *rig.Builder
	design  *design.Designer
	table   *presetmap.Table
}

func newApp() (*app, error) {
	cat := catalog.New()
	builder, err := rig.NewBuilder(cat)
	if err != nil {
		return nil, err
	}
	return &app{
		cat:     cat,
		builder: builder,
		design:  design.NewDesigner(cat),
		table:   presetmap.NewTable(cat, mooer.Default()),
	}, nil
}

// Execute runs the root command.
func Execute() error {
	return newRootCmd().Execute()
}

// IsUsageError reports whether err is a command-line usage error (an unknown
// command or flag), for which showing the help is the useful response rather
// than a bare error line.
func IsUsageError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "unknown command") ||
		strings.Contains(msg, "unknown flag") ||
		strings.Contains(msg, "unknown shorthand flag") ||
		strings.Contains(msg, "flag needs an argument") ||
		strings.Contains(msg, "invalid argument")
}

// PrintHelp writes the root command's help to w.
func PrintHelp(w io.Writer) {
	root := newRootCmd()
	root.SetOut(w)
	_ = root.Help()
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:                        "guitar-modeler-mcp",
		Short:                      "Design and write guitar-modeler presets",
		Long:                       "guitar-modeler-mcp exposes an MCP server and CLI for designing guitar presets: translate real-world hardware into device models and write preset files. The first supported device is the HeadRush Gigboard.",
		Version:                    version,
		SilenceUsage:               true,
		SilenceErrors:              true,
		SuggestionsMinimumDistance: 2,
	}
	root.AddCommand(
		newServeCmd(),
		newCatalogCmd(),
		newTranslateCmd(),
		newSearchCmd(),
		newFxPlacementCmd(),
		newDesignCmd(),
		newReportCmd(),
		newDecodeCmd(),
		newLevelCmd(),
		newSetlistCmd(),
		newMcpCmd(),
		newDeviceCmd(),
		newMapCmd(),
	)
	return root
}
