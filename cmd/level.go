package cmd

import (
	"github.com/spf13/cobra"

	"github.com/d-led/guitar-modeler-mcp/internal/rig"
)

func newLevelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "level <file.rig>",
		Short: "Estimate a rig's output level and recommend a RigVolume",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			file, err := readRigFile(args[0])
			if err != nil {
				return err
			}
			target, _ := cmd.Flags().GetFloat64("target")
			est, err := rig.EstimateLevel(file, target)
			if err != nil {
				return err
			}
			return printJSON(est)
		},
	}
	cmd.Flags().Float64("target", 0, "target output level in dB (default 0 = unity)")
	return cmd
}
