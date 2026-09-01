package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/d-led/guitar-modeler-mcp/internal/htmlreport"
)

func newReportCmd() *cobra.Command {
	var (
		rigFile string
		note    string
		out     string
	)
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Render an HTML report for an existing .rig file",
		RunE: func(_ *cobra.Command, _ []string) error {
			a, err := newApp()
			if err != nil {
				return err
			}
			file, err := readRigFile(rigFile)
			if err != nil {
				return err
			}
			html, err := htmlreport.Render(file, note, a.cat)
			if err != nil {
				return err
			}
			dir := out
			if dir == "" {
				dir = filepath.Dir(rigFile)
			}
			htmlPath := filepath.Join(dir, file.Name()+".gigboard.html")
			if err := os.WriteFile(htmlPath, []byte(html), 0o600); err != nil {
				return err
			}
			fmt.Printf("Report: %s\n", htmlPath)
			return nil
		},
	}
	cmd.Flags().StringVar(&rigFile, "rig", "", "path to the .rig file (required)")
	cmd.Flags().StringVar(&note, "note", "", "note annotation")
	cmd.Flags().StringVar(&out, "out", "", "output directory (default: same as the rig file)")
	_ = cmd.MarkFlagRequired("rig")
	return cmd
}
