package cmd

import (
	"github.com/spf13/cobra"

	"github.com/d-led/guitar-modeler-mcp/internal/design"
)

func newFxPlacementCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fx-placement",
		Short: "Show where each effect category goes in each chain layout",
		RunE: func(_ *cobra.Command, _ []string) error {
			return printJSON(design.PlacementGuide())
		},
	}
}
