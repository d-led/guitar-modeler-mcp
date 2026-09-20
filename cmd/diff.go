package cmd

import (
	"github.com/spf13/cobra"

	"github.com/d-led/guitar-modeler-mcp/internal/headrush/rig"
)

func newDiffCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "diff <a.rig> <b.rig>",
		Short: "Diff two .rig files field by field, keyed by JSON path",
		Args:  cobra.ExactArgs(2),
		RunE: func(_ *cobra.Command, args []string) error {
			a, err := readRigFile(args[0])
			if err != nil {
				return err
			}
			b, err := readRigFile(args[1])
			if err != nil {
				return err
			}
			diff, err := rig.DiffRigs(a, b)
			if err != nil {
				return err
			}
			return printJSON(diff)
		},
	}
}
