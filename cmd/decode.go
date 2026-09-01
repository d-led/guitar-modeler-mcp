package cmd

import (
	"github.com/spf13/cobra"

	"github.com/d-led/guitar-modeler-mcp/internal/rig"
)

func newDecodeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "decode <file.rig>",
		Short: "Decode a .rig file into its signal chain and parameter values",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			file, err := readRigFile(args[0])
			if err != nil {
				return err
			}
			summary, err := rig.Describe(file)
			if err != nil {
				return err
			}
			return printJSON(summary)
		},
	}
}
