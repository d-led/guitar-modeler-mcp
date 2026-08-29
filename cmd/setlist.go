package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/rig"
	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/setlist"
)

func newSetlistCmd() *cobra.Command {
	var (
		name string
		out  string
	)
	cmd := &cobra.Command{
		Use:     "setlist <file.rig>...",
		Short:   "Write a .setlist binding the given rig files in order",
		Example: `  guitar-modeler-mcp setlist --name "Ballad" --out Setlists Rigs/clean.rig Rigs/drive.rig Rigs/solo.rig`,
		Args:    cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			entries := make([]setlist.Entry, 0, len(args))
			for _, p := range args {
				data, err := os.ReadFile(p)
				if err != nil {
					return err
				}
				var file rig.RigFile
				if err := json.Unmarshal(data, &file); err != nil {
					return fmt.Errorf("parse rig file %q: %w", p, err)
				}
				entries = append(entries, setlist.Entry{ID: file.ID, Name: file.Name()})
			}

			sl, err := setlist.New(name, entries)
			if err != nil {
				return err
			}
			path, err := sl.Write(out)
			if err != nil {
				return err
			}
			fmt.Printf("Setlist %q written to %s\n", sl.Name(), path)
			for i, e := range entries {
				fmt.Printf("  %d. %s\n", i+1, e.Name)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "setlist name (required)")
	cmd.Flags().StringVar(&out, "out", ".", "output directory")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}
