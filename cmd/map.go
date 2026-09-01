package cmd

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/d-led/guitar-modeler-mcp/internal/mooer"
	"github.com/d-led/guitar-modeler-mcp/internal/presetmap"
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
			return runMap(args[0], from, to, out)
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "source device (gigboard or mooer; inferred from the file extension when omitted)")
	cmd.Flags().StringVar(&to, "to", "", "target device (gigboard or mooer; the opposite of the source when omitted)")
	cmd.Flags().StringVar(&out, "out", ".", "output directory")
	return cmd
}

func runMap(path, from, to, out string) error {
	a, err := newApp()
	if err != nil {
		return err
	}
	src, dst, err := resolveMapDevices(path, from, to)
	if err != nil {
		return err
	}
	return dispatchMap(a, path, out, src, dst)
}

func resolveMapDevices(path, from, to string) (string, string, error) {
	src := from
	if src == "" {
		src = inferDevice(path)
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
		return "", "", fmt.Errorf("source and target device are both %q", src)
	}
	return src, dst, nil
}

func dispatchMap(a *app, path, out, src, dst string) error {
	switch {
	case src == presetmap.DeviceGigboard && dst == presetmap.DeviceMooer:
		return mapGigboardToMooer(a, path, out, src, dst)
	case src == presetmap.DeviceMooer && dst == presetmap.DeviceGigboard:
		return mapMooerToGigboard(a, path, out, src, dst)
	default:
		return fmt.Errorf("unsupported mapping %s -> %s", src, dst)
	}
}

func mapGigboardToMooer(a *app, path, out, src, dst string) error {
	file, err := readRigFile(path)
	if err != nil {
		return err
	}
	p, err := a.table.GigboardToMooer(file)
	if err != nil {
		return err
	}
	outPath := filepath.Join(out, sanitizeName(p.Name)+".mo")
	if err := mooer.WriteMOFile(outPath, p); err != nil {
		return err
	}
	fmt.Printf("Mapped %s -> %s: %s\n", src, dst, outPath)
	return nil
}

func mapMooerToGigboard(a *app, path, out, src, dst string) error {
	p, err := mooer.ReadMOFile(path)
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
	outPath, err := file.Write(out)
	if err != nil {
		return err
	}
	fmt.Printf("Mapped %s -> %s: %s\n", src, dst, outPath)
	return nil
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
