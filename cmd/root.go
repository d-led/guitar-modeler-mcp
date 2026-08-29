// Package cmd wires the domain packages into a Cobra CLI.
package cmd

import (
	"github.com/spf13/cobra"

	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/catalog"
	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/design"
	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/rig"
)

// version is reported by the MCP server and the --version flag.
const version = "0.1.0"

// app bundles the shared dependencies for all commands.
type app struct {
	cat     *catalog.Catalog
	builder *rig.Builder
	design  *design.Designer
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
	}, nil
}

// Execute runs the root command.
func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "headrush-gigboard-mcp",
		Short:         "Design and write HeadRush Gigboard sound presets",
		Long:          "headrush-gigboard-mcp exposes an MCP server and CLI for designing HeadRush Gigboard rigs, translating real-world hardware into device models and writing .rig patch files.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(
		newServeCmd(),
		newCatalogCmd(),
		newTranslateCmd(),
		newSearchCmd(),
		newDesignCmd(),
		newReportCmd(),
		newDecodeCmd(),
		newMcpCmd(),
	)
	return root
}
