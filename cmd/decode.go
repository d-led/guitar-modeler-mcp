package cmd

import (
	"github.com/spf13/cobra"

	"github.com/d-led/guitar-modeler-mcp/internal/headrush/rig"
)

func newDecodeCmd() *cobra.Command {
	var raw bool
	cmd := &cobra.Command{
		Use:   "decode <file.rig>",
		Short: "Decode a .rig file into its signal chain and parameter values",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			file, err := readRigFile(args[0])
			if err != nil {
				return err
			}
			if raw {
				content, err := file.RawContent()
				if err != nil {
					return err
				}
				return printJSON(content)
			}
			summary, err := rig.Describe(file)
			if err != nil {
				return err
			}
			return printJSON(summary)
		},
	}
	cmd.Flags().BoolVar(&raw, "raw", false, "print the full raw document instead of the summarized chain")
	return cmd
}
