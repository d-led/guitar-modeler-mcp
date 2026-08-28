package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/rig"
)

func newDecodeCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "decode <file.rig>",
		Short: "Decode a .rig file into its signal chain and parameter values",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			data, err := os.ReadFile(args[0])
			if err != nil {
				return err
			}
			var file rig.RigFile
			if err := json.Unmarshal(data, &file); err != nil {
				return fmt.Errorf("parse rig file: %w", err)
			}
			summary, err := rig.Describe(&file)
			if err != nil {
				return err
			}
			return printJSON(summary)
		},
	}
}
