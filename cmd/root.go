// Package cmd wires the domain packages into a Cobra CLI.
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/d-led/guitar-modeler-mcp/internal/catalog"
	"github.com/d-led/guitar-modeler-mcp/internal/design"
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
		table:   presetmap.NewTable(cat),
	}, nil
}

// Execute runs the root command.
func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "guitar-modeler-mcp",
		Short:         "Design and write guitar-modeler presets",
		Long:          "guitar-modeler-mcp exposes an MCP server and CLI for designing guitar presets: translate real-world hardware into device models and write preset files. The first supported device is the HeadRush Gigboard.",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
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
