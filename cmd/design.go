package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/d-led/guitar-modeler-mcp/internal/design"
	"github.com/d-led/guitar-modeler-mcp/internal/htmlreport"
	"github.com/d-led/guitar-modeler-mcp/internal/rig"
)

func newDesignCmd() *cobra.Command {
	var (
		device       string
		name         string
		song         string
		amp          string
		cab          string
		mic          string
		routing      string
		amp2         string
		cab2         string
		mic2         string
		tempo        float64
		inputGain    float64
		outputLevel  float64
		out          string
		fxJSON       string
		pathAFX      string
		pathBFX      string
		para1Level   float64
		para2Level   float64
		para1Pan     float64
		para2Pan     float64
		paraDelay    float64
		footswitches string
	)
	cmd := &cobra.Command{
		Use:   "design",
		Short: "Dial in a tone and write a .rig patch plus an HTML report",
		Example: `  guitar-modeler-mcp design --name "Brown Sound" --song "Van Halen - Panama" \
      --amp "Marshall JCM800" --fx '[{"type":"Tape Echo","enabled":true}]'`,
		RunE: func(cmd *cobra.Command, _ []string) error {
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
			footswitches, err := parseFootswitchFlags(footswitches)
			if err != nil {
				return err
			}
			res, err := a.design.Design(design.Request{
				Device:       device,
				Name:         name,
				Song:         song,
				Amp:          amp,
				Cab:          cab,
				Mic:          mic,
				Routing:      rig.Routing(routing),
				Amp2:         amp2,
				Cab2:         cab2,
				Mic2:         mic2,
				Tempo:        tempo,
				InputGain:    inputGain,
				OutputLevel:  floatPtr(cmd, "output-level", outputLevel),
				FX:           fx,
				PathAFX:      pathAFXBlocks,
				PathBFX:      pathBFXBlocks,
				Para1Level:   floatPtr(cmd, "para1-level", para1Level),
				Para2Level:   floatPtr(cmd, "para2-level", para2Level),
				Para1Pan:     floatPtr(cmd, "para1-pan", para1Pan),
				Para2Pan:     floatPtr(cmd, "para2-pan", para2Pan),
				ParaDelay:    floatPtr(cmd, "para-delay", paraDelay),
				Footswitches: footswitches,
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
			if summary, err := rig.Describe(file); err == nil {
				fmt.Printf("Footswitches: %s\n", rig.FootswitchLine(summary.Footswitches))
				fmt.Printf("Levels: input %+g dB, output %+g dB\n", summary.InputGain, summary.OutputVolume)
			}
			fmt.Printf("Rig file: %s\n", rigPath)
			fmt.Printf("Report:  %s\n", htmlPath)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "New Rig", "rig name")
	cmd.Flags().StringVar(&device, "device", "gigboard", "target device (currently only gigboard is supported)")
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
	cmd.Flags().Float64Var(&outputLevel, "output-level", 6, "overall rig output level in dB (RigVolume, default +6 to compensate the amp master)")
	cmd.Flags().StringVar(&out, "out", ".", "output directory")
	cmd.Flags().StringVar(&fxJSON, "fx", "", "effects as a JSON array")
	cmd.Flags().StringVar(&pathAFX, "path-a-fx", "", "effects for parallel path A as a JSON array")
	cmd.Flags().StringVar(&pathBFX, "path-b-fx", "", "effects for parallel path B as a JSON array")
	cmd.Flags().Float64Var(&para1Level, "para1-level", -6, "level of path A in dB (default -6)")
	cmd.Flags().Float64Var(&para2Level, "para2-level", -6, "level of path B in dB (default -6)")
	cmd.Flags().Float64Var(&para1Pan, "para1-pan", 0, "pan of path A, -100..100 (default 0)")
	cmd.Flags().Float64Var(&para2Pan, "para2-pan", 0, "pan of path B, -100..100 (default 0)")
	cmd.Flags().Float64Var(&paraDelay, "para-delay", 0, "delay of path B in ms (default 0)")
	cmd.Flags().StringVar(&footswitches, "footswitches", "", "footswitch assignments as a JSON array, e.g. '[{\"module\":\"Wham\"}]'")
	_ = cmd.MarkFlagRequired("amp")
	return cmd
}

// floatPtr returns &v only when the flag was explicitly set, so unset optional
// floats keep the designer's defaults.
func floatPtr(cmd *cobra.Command, name string, v float64) *float64 {
	if !cmd.Flags().Changed(name) {
		return nil
	}
	return &v
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

func parseFootswitchFlags(jsonValue string) ([]rig.Footswitch, error) {
	if jsonValue == "" {
		return nil, nil
	}
	var switches []rig.Footswitch
	if err := json.Unmarshal([]byte(jsonValue), &switches); err != nil {
		return nil, fmt.Errorf("parse --footswitches: %w", err)
	}
	return switches, nil
}
