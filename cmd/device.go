package cmd

import (
	"github.com/spf13/cobra"

	"github.com/d-led/guitar-modeler-mcp/internal/gp200"
	"github.com/d-led/guitar-modeler-mcp/internal/mooer"
	"github.com/d-led/guitar-modeler-mcp/internal/qc"
	"github.com/d-led/guitar-modeler-mcp/internal/thr"
	"github.com/d-led/guitar-modeler-mcp/internal/waza"
)

// deviceInfo describes one supported device for the `device list` command.
type deviceInfo struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	FileExchange bool   `json:"file_exchange"`
	FileExt      string `json:"file_ext,omitempty"`
}

// supportedDevices lists the Gigboard, every Mooer model and the Waza Air.
func supportedDevices() []deviceInfo {
	list := []deviceInfo{
		{Name: "gigboard", Description: "HeadRush Gigboard", FileExchange: true, FileExt: ".rig"},
	}
	for _, m := range mooer.Models() {
		ext := ""
		if m.FileExchange {
			ext = m.FileExt
		}
		list = append(list, deviceInfo{
			Name:         m.Name,
			Description:  m.Display,
			FileExchange: m.FileExchange,
			FileExt:      ext,
		})
	}
	w := waza.Default()
	list = append(list, deviceInfo{Name: w.Name, Description: w.Display, FileExchange: w.FileExchange, FileExt: w.FileExt})
	for _, t := range thr.Models() {
		list = append(list, deviceInfo{Name: t.Name, Description: t.Display, FileExchange: t.FileExchange, FileExt: t.FileExt})
	}
	q := qc.Default()
	list = append(list, deviceInfo{Name: q.Name, Description: q.Display, FileExchange: q.FileExchange, FileExt: q.FileExt})
	for _, g := range gp200.Models() {
		list = append(list, deviceInfo{Name: g.Name, Description: g.Display, FileExchange: g.FileExchange, FileExt: g.FileExt})
	}
	return list
}

func newDeviceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "device",
		Short: "List the devices the tool can target",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List supported devices",
		Args:  cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return printJSON(supportedDevices())
		},
	})
	return cmd
}
