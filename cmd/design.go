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

type designFlags struct {
	device, name, note, amp, cab, mic string
	ampParams, cabParams, routing     string
	amp2, cab2, mic2                  string
	tempo, inputGain, outputLevel     float64
	out, fxJSON, pathAFX, pathBFX     string
	para1Level, para2Level            float64
	para1Pan, para2Pan                float64
	paraDelay                         float64
	footswitches                      string
}

func newDesignCmd() *cobra.Command {
	var f designFlags
	cmd := &cobra.Command{
		Use:   "design",
		Short: "Dial in a tone and write a .rig patch plus an HTML report",
		Example: `  guitar-modeler-mcp design --name "Brown Sound" --note "Van Halen - Panama" \
      --amp "Marshall JCM800" --fx '[{"type":"Tape Echo","enabled":true}]'`,
		RunE: f.run,
	}
	cmd.Flags().StringVar(&f.name, "name", "New Rig", "rig name")
	cmd.Flags().StringVar(&f.device, "device", "gigboard", "target device (currently only gigboard is supported)")
	cmd.Flags().StringVar(&f.note, "note", "", "note annotation shown on the report")
	cmd.Flags().StringVar(&f.amp, "amp", "", "amp: device model or real-hardware description (required)")
	cmd.Flags().StringVar(&f.cab, "cab", "", "cab: device model or description")
	cmd.Flags().StringVar(&f.mic, "mic", "", "mic: device model or description")
	cmd.Flags().StringVar(&f.ampParams, "amp-params", "", "amp knob overrides as a JSON object, e.g. '{\"GainA\":58,\"Master\":60}'")
	cmd.Flags().StringVar(&f.cabParams, "cab-params", "", "cab knob overrides as a JSON object")
	cmd.Flags().StringVar(&f.routing, "routing", "", "signal-chain topology: S (serial, default), SPS-1 (serial→parallel→serial) or PS-1 (parallel from input)")
	cmd.Flags().StringVar(&f.amp2, "amp2", "", "second amp for a dual-amp parallel rig (same model = same amp on both channels)")
	cmd.Flags().StringVar(&f.cab2, "cab2", "", "cab for the second amp path")
	cmd.Flags().StringVar(&f.mic2, "mic2", "", "mic for the second amp path")
	cmd.Flags().Float64Var(&f.tempo, "tempo", 0, "tempo in BPM")
	cmd.Flags().Float64Var(&f.inputGain, "input-gain", 0, "input gain in dB")
	cmd.Flags().Float64Var(&f.outputLevel, "output-level", 6, "overall rig output level in dB (RigVolume, default +6 to compensate the amp master)")
	cmd.Flags().StringVar(&f.out, "out", ".", "output directory")
	cmd.Flags().StringVar(&f.fxJSON, "fx", "", "effects as a JSON array")
	cmd.Flags().StringVar(&f.pathAFX, "path-a-fx", "", "effects for parallel path A as a JSON array")
	cmd.Flags().StringVar(&f.pathBFX, "path-b-fx", "", "effects for parallel path B as a JSON array")
	cmd.Flags().Float64Var(&f.para1Level, "para1-level", -6, "level of path A in dB (default -6)")
	cmd.Flags().Float64Var(&f.para2Level, "para2-level", -6, "level of path B in dB (default -6)")
	cmd.Flags().Float64Var(&f.para1Pan, "para1-pan", 0, "pan of path A, -100..100 (default 0)")
	cmd.Flags().Float64Var(&f.para2Pan, "para2-pan", 0, "pan of path B, -100..100 (default 0)")
	cmd.Flags().Float64Var(&f.paraDelay, "para-delay", 0, "delay of path B in ms (default 0)")
	cmd.Flags().StringVar(&f.footswitches, "footswitches", "", "footswitch assignments as a JSON array, e.g. '[{\"module\":\"Wham\"}]'")
	_ = cmd.MarkFlagRequired("amp")
	return cmd
}

func (f *designFlags) run(cmd *cobra.Command, _ []string) error {
	a, err := newApp()
	if err != nil {
		return err
	}
	in, err := parseDesignInputs(f)
	if err != nil {
		return err
	}
	res, err := a.design.Design(in.request(cmd, f))
	if err != nil {
		return err
	}
	return writeDesignOutput(a, res, f)
}

type designInputs struct {
	fx           []design.FXBlock
	pathA        []design.FXBlock
	pathB        []design.FXBlock
	footswitches []rig.Footswitch
	ampParams    map[string]any
	cabParams    map[string]any
}

func parseDesignInputs(f *designFlags) (designInputs, error) {
	var in designInputs
	var err error
	in.fx, err = parseFXFlags(f.fxJSON)
	if err != nil {
		return in, err
	}
	in.pathA, err = parseFXFlags(f.pathAFX)
	if err != nil {
		return in, err
	}
	in.pathB, err = parseFXFlags(f.pathBFX)
	if err != nil {
		return in, err
	}
	in.footswitches, err = parseFootswitchFlags(f.footswitches)
	if err != nil {
		return in, err
	}
	in.ampParams, err = parseParamsFlag(f.ampParams)
	if err != nil {
		return in, err
	}
	in.cabParams, err = parseParamsFlag(f.cabParams)
	if err != nil {
		return in, err
	}
	return in, nil
}

func (in designInputs) request(cmd *cobra.Command, f *designFlags) design.Request {
	return design.Request{
		Device:       f.device,
		Name:         f.name,
		Note:         f.note,
		Amp:          f.amp,
		Cab:          f.cab,
		Mic:          f.mic,
		AmpParams:    in.ampParams,
		CabParams:    in.cabParams,
		Routing:      rig.Routing(f.routing),
		Amp2:         f.amp2,
		Cab2:         f.cab2,
		Mic2:         f.mic2,
		Tempo:        f.tempo,
		InputGain:    f.inputGain,
		OutputLevel:  floatPtr(cmd, "output-level", f.outputLevel),
		FX:           in.fx,
		PathAFX:      in.pathA,
		PathBFX:      in.pathB,
		Para1Level:   floatPtr(cmd, "para1-level", f.para1Level),
		Para2Level:   floatPtr(cmd, "para2-level", f.para2Level),
		Para1Pan:     floatPtr(cmd, "para1-pan", f.para1Pan),
		Para2Pan:     floatPtr(cmd, "para2-pan", f.para2Pan),
		ParaDelay:    floatPtr(cmd, "para-delay", f.paraDelay),
		Footswitches: in.footswitches,
	}
}

func writeDesignOutput(a *app, res *design.Result, f *designFlags) error {
	file, err := a.builder.Build(res.Spec)
	if err != nil {
		return err
	}
	rigPath, err := file.Write(f.out)
	if err != nil {
		return err
	}
	html, err := htmlreport.Render(file, f.note, a.cat)
	if err != nil {
		return err
	}
	htmlPath := filepath.Join(f.out, file.Name()+".gigboard.html")
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

// parseParamsFlag parses a JSON object of knob overrides (--amp-params/
// --cab-params) into a map.
func parseParamsFlag(jsonValue string) (map[string]any, error) {
	if jsonValue == "" {
		return nil, nil
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(jsonValue), &m); err != nil {
		return nil, fmt.Errorf("parse --amp-params/--cab-params: %w", err)
	}
	return m, nil
}
