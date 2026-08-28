package cmd

import "github.com/spf13/cobra"

func newTranslateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "translate",
		Short: "Translate real-world hardware into device models",
	}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "amp <query>",
			Short: "Translate an amplifier description",
			Args:  cobra.ExactArgs(1),
			RunE: func(_ *cobra.Command, args []string) error {
				a, err := newApp()
				if err != nil {
					return err
				}
				return printJSON(a.cat.TranslateAmp(args[0]))
			},
		},
		&cobra.Command{
			Use:   "cab <query>",
			Short: "Translate a cabinet description",
			Args:  cobra.ExactArgs(1),
			RunE: func(_ *cobra.Command, args []string) error {
				a, err := newApp()
				if err != nil {
					return err
				}
				return printJSON(a.cat.TranslateCab(args[0]))
			},
		},
		&cobra.Command{
			Use:   "mic <query>",
			Short: "Translate a microphone description",
			Args:  cobra.ExactArgs(1),
			RunE: func(_ *cobra.Command, args []string) error {
				a, err := newApp()
				if err != nil {
					return err
				}
				return printJSON(a.cat.TranslateMic(args[0]))
			},
		},
	)
	return cmd
}
