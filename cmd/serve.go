package cmd

import (
	"context"
	"os"

	"github.com/spf13/cobra"

	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/mcp"
	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/tools"
)

func newServeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "serve",
		Short: "Run the MCP server over stdio",
		Long:  "Serve the Model Context Protocol over stdin/stdout so an agent can design and write rigs.",
		RunE: func(_ *cobra.Command, _ []string) error {
			a, err := newApp()
			if err != nil {
				return err
			}
			server := mcp.NewServer("headrush-gigboard-mcp", version)
			tools.NewRegistrar(a.cat, a.builder, a.design).Register(server)
			return server.Run(context.Background(), os.Stdin, os.Stdout)
		},
	}
}
