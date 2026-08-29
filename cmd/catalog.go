package cmd

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/assets"
	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/params"
)

func newCatalogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "catalog",
		Short: "List the models available on the device",
	}

	fxCmd := &cobra.Command{
		Use:   "fx",
		Short: "List effect modules",
		RunE: func(cmd *cobra.Command, _ []string) error {
			a, err := newApp()
			if err != nil {
				return err
			}
			category, _ := cmd.Flags().GetString("category")
			if category != "" {
				return printJSON(params.FXListingsByCategory(a.cat, category))
			}
			return printJSON(params.FXListings(a.cat))
		},
	}
	fxCmd.Flags().String("category", "", "only list effects in this category (see `catalog fx-categories`)")

	cmd.AddCommand(
		&cobra.Command{
			Use:   "amps [query]",
			Short: "List amp models",
			Args:  cobra.MaximumNArgs(1),
			RunE: func(_ *cobra.Command, args []string) error {
				a, err := newApp()
				if err != nil {
					return err
				}
				return printJSON(params.AmpListings(a.cat, firstArg(args)))
			},
		},
		&cobra.Command{
			Use:   "cabs [query]",
			Short: "List cabinet models",
			Args:  cobra.MaximumNArgs(1),
			RunE: func(_ *cobra.Command, args []string) error {
				a, err := newApp()
				if err != nil {
					return err
				}
				return printJSON(a.cat.CabsMatching(firstArg(args)))
			},
		},
		&cobra.Command{
			Use:   "mics [query]",
			Short: "List microphone models",
			Args:  cobra.MaximumNArgs(1),
			RunE: func(_ *cobra.Command, args []string) error {
				a, err := newApp()
				if err != nil {
					return err
				}
				return printJSON(a.cat.MicsMatching(firstArg(args)))
			},
		},
		fxCmd,
		&cobra.Command{
			Use:   "fx-categories",
			Short: "List effect categories with module counts",
			RunE: func(_ *cobra.Command, _ []string) error {
				a, err := newApp()
				if err != nil {
					return err
				}
				return printJSON(params.FXCategories(a.cat))
			},
		},
		&cobra.Command{
			Use:   "presets <module>",
			Short: "List factory presets for an effect module",
			Args:  cobra.ExactArgs(1),
			RunE: func(_ *cobra.Command, args []string) error {
				a, err := newApp()
				if err != nil {
					return err
				}
				name := args[0]
				if f, ok := a.cat.FXByName(name); ok {
					name = f.Name
				}
				presets, err := assets.Presets(strings.ToUpper(name))
				if err != nil {
					return fmt.Errorf("no presets for module %q: %w", name, err)
				}
				return printJSON(presets)
			},
		},
		&cobra.Command{
			Use:   "params <module>",
			Short: "Describe a module's parameters: kinds, ranges, units and options",
			Args:  cobra.ExactArgs(1),
			RunE: func(_ *cobra.Command, args []string) error {
				a, err := newApp()
				if err != nil {
					return err
				}
				spec, err := params.Describe(a.cat, args[0])
				if err != nil {
					return err
				}
				return printJSON(spec)
			},
		},
	)
	return cmd
}

func firstArg(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return ""
}

func printJSON(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}
