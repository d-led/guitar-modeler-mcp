package cmd

import (
	"github.com/spf13/cobra"
)

// deviceInfo describes one supported device for the `device list` command.
type deviceInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	FileExt     string `json:"file_ext"`
}

var supportedDevices = []deviceInfo{
	{Name: "gigboard", Description: "HeadRush Gigboard", FileExt: ".rig"},
	{Name: "mooer", Description: "Mooer GE150 Pro Li / GE150 Max", FileExt: ".mo"},
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
			return printJSON(supportedDevices)
		},
	})
	return cmd
}
