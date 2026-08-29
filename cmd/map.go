package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/d-led/guitar-modeler-mcp/internal/mooer"
	"github.com/d-led/guitar-modeler-mcp/internal/presetmap"
	"github.com/d-led/guitar-modeler-mcp/internal/rig"
)

func newMapCmd() *cobra.Command {
	var (
		from string
		to   string
		out  string
	)
	cmd := &cobra.Command{
		Use:   "map <file>",
		Short: "Map a preset from one device to another",
		Long:  "Read a preset file for one device and write the equivalent preset for the other, translating amp, cab and effect models through their shared \"inspired by\" hardware.",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			a, err := newApp()
			if err != nil {
				return err
			}

			src := from
			if src == "" {
				src = inferDevice(args[0])
			}
			dst := to
			if dst == "" {
				if src == presetmap.DeviceGigboard {
					dst = presetmap.DeviceMooer
				} else {
					dst = presetmap.DeviceGigboard
				}
			}
			if src == dst {
				return fmt.Errorf("source and target device are both %q", src)
			}

			switch {
			case src == presetmap.DeviceGigboard && dst == presetmap.DeviceMooer:
				data, err := os.ReadFile(args[0])
				if err != nil {
					return err
				}
				var file rig.RigFile
				if err := json.Unmarshal(data, &file); err != nil {
					return fmt.Errorf("parse rig file: %w", err)
				}
				p, err := a.table.GigboardToMooer(&file)
				if err != nil {
					return err
				}
				path := filepath.Join(out, sanitizeName(p.Name)+".mo")
				if err := mooer.WriteMOFile(path, p); err != nil {
					return err
				}
				fmt.Printf("Mapped %s -> %s: %s\n", src, dst, path)
			case src == presetmap.DeviceMooer && dst == presetmap.DeviceGigboard:
				p, err := mooer.ReadMOFile(args[0])
				if err != nil {
					return err
				}
				spec, err := a.table.MooerToGigboard(p)
				if err != nil {
					return err
				}
				file, err := a.builder.Build(spec)
				if err != nil {
					return err
				}
				path, err := file.Write(out)
				if err != nil {
					return err
				}
				fmt.Printf("Mapped %s -> %s: %s\n", src, dst, path)
			default:
				return fmt.Errorf("unsupported mapping %s -> %s", src, dst)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "source device (gigboard or mooer; inferred from the file extension when omitted)")
	cmd.Flags().StringVar(&to, "to", "", "target device (gigboard or mooer; the opposite of the source when omitted)")
	cmd.Flags().StringVar(&out, "out", ".", "output directory")
	return cmd
}

func inferDevice(path string) string {
	if strings.EqualFold(filepath.Ext(path), ".mo") {
		return presetmap.DeviceMooer
	}
	return presetmap.DeviceGigboard
}

func sanitizeName(name string) string {
	if name == "" {
		return "preset"
	}
	return strings.ReplaceAll(name, "/", "-")
}
