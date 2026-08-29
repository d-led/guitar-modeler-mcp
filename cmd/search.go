package cmd

import "github.com/spf13/cobra"

func newSearchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Fuzzy-search amps, cabs, mics and effects by name or the real hardware they emulate",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			a, err := newApp()
			if err != nil {
				return err
			}
			kind, _ := cmd.Flags().GetString("kind")
			return printJSON(a.cat.Search(args[0], kind))
		},
	}
	cmd.Flags().String("kind", "", "restrict to amp, cab, mic or fx")
	return cmd
}
