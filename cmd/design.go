package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/design"
	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/htmlreport"
	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/rig"
)

func newDesignCmd() *cobra.Command {
	var (
		name      string
		song      string
		amp       string
		cab       string
		mic       string
		routing   string
		amp2      string
		cab2      string
		mic2      string
		tempo     float64
		inputGain float64
		out       string
		fxJSON    string
		pathAFX   string
		pathBFX   string
	)
	cmd := &cobra.Command{
		Use:   "design",
		Short: "Dial in a tone and write a .rig patch plus an HTML report",
		Example: `  headrush-gigboard-mcp design --name "Brown Sound" --song "Van Halen - Panama" \
      --amp "Marshall JCM800" --fx '[{"type":"Tape Echo","enabled":true}]'`,
		RunE: func(_ *cobra.Command, _ []string) error {
			a, err := newApp()
			if err != nil {
				return err
			}
			fx, err := parseFXFlags(fxJSON)
			if err != nil {
				return err
			}
			pathAFXBlocks, err := parseFXFlags(pathAFX)
			if err != nil {
				return err
			}
			pathBFXBlocks, err := parseFXFlags(pathBFX)
			if err != nil {
				return err
			}
			res, err := a.design.Design(design.Request{
				Name:      name,
				Song:      song,
				Amp:       amp,
				Cab:       cab,
				Mic:       mic,
				Routing:   rig.Routing(routing),
				Amp2:      amp2,
				Cab2:      cab2,
				Mic2:      mic2,
				Tempo:     tempo,
				InputGain: inputGain,
				FX:        fx,
				PathAFX:   pathAFXBlocks,
				PathBFX:   pathBFXBlocks,
			})
			if err != nil {
				return err
			}
			file, err := a.builder.Build(res.Spec)
			if err != nil {
				return err
			}
			rigPath, err := file.Write(out)
			if err != nil {
				return err
			}
			html, err := htmlreport.Render(file, song, a.cat)
			if err != nil {
				return err
			}
			htmlPath := filepath.Join(out, file.Name()+".html")
			if err := os.WriteFile(htmlPath, []byte(html), 0o644); err != nil {
				return err
			}

			fmt.Printf("Rig %q written\n", file.Name())
			for _, n := range res.Notes {
				fmt.Printf("  - %s\n", n)
			}
			fmt.Printf("Rig file: %s\n", rigPath)
			fmt.Printf("Report:  %s\n", htmlPath)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "New Rig", "rig name")
	cmd.Flags().StringVar(&song, "song", "", "song the tone is for")
	cmd.Flags().StringVar(&amp, "amp", "", "amp: device model or real-hardware description (required)")
	cmd.Flags().StringVar(&cab, "cab", "", "cab: device model or description")
	cmd.Flags().StringVar(&mic, "mic", "", "mic: device model or description")
	cmd.Flags().StringVar(&routing, "routing", "", "signal-chain topology: S (serial, default), SPS-1 (serial→parallel→serial) or PS-1 (parallel from input)")
	cmd.Flags().StringVar(&amp2, "amp2", "", "second amp for a dual-amp parallel rig (same model = same amp on both channels)")
	cmd.Flags().StringVar(&cab2, "cab2", "", "cab for the second amp path")
	cmd.Flags().StringVar(&mic2, "mic2", "", "mic for the second amp path")
	cmd.Flags().Float64Var(&tempo, "tempo", 0, "tempo in BPM")
	cmd.Flags().Float64Var(&inputGain, "input-gain", 0, "input gain in dB")
	cmd.Flags().StringVar(&out, "out", ".", "output directory")
	cmd.Flags().StringVar(&fxJSON, "fx", "", "effects as a JSON array")
	cmd.Flags().StringVar(&pathAFX, "path-a-fx", "", "effects for parallel path A as a JSON array")
	cmd.Flags().StringVar(&pathBFX, "path-b-fx", "", "effects for parallel path B as a JSON array")
	_ = cmd.MarkFlagRequired("amp")
	return cmd
}

func parseFXFlags(fxJSON string) ([]design.FXBlock, error) {
	if fxJSON == "" {
		return nil, nil
	}
	var fx []design.FXBlock
	if err := json.Unmarshal([]byte(fxJSON), &fx); err != nil {
		return nil, fmt.Errorf("parse --fx: %w", err)
	}
	return fx, nil
}
