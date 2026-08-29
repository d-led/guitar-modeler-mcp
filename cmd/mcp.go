package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dmitryledentsov/headrush-gigboard-mcp/internal/install"
)

const serverName = "guitar-modeler-mcp"

func newMcpCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Install or uninstall the MCP server in a client",
	}
	cmd.AddCommand(newMcpInstallCmd(), newMcpUninstallCmd())
	return cmd
}

func newMcpInstallCmd() *cobra.Command {
	var (
		target    string
		printOnly bool
		command   string
	)
	c := &cobra.Command{
		Use:   "install",
		Short: "Register this server in an MCP client's config",
		Long:  "Writes the server entry into the selected client config. Defaults to the VS Code user profile (available in all workspaces).",
		Example: `  guitar-modeler-mcp mcp install                  # VS Code user profile (global)
  guitar-modeler-mcp mcp install --target workspace   # .vscode/mcp.json here
  guitar-modeler-mcp mcp install --target claude      # Claude Desktop
  guitar-modeler-mcp mcp install --print              # show config without writing`,
		RunE: func(_ *cobra.Command, _ []string) error {
			t, ok := install.ValidTarget(target)
			if !ok {
				return fmt.Errorf("invalid --target %q (use vscode, workspace or claude)", target)
			}
			exe := command
			if exe == "" {
				var err error
				exe, err = os.Executable()
				if err != nil {
					return fmt.Errorf("resolve executable: %w", err)
				}
			}
			server := install.Server{Name: serverName, Command: exe, Args: []string{"serve"}}

			if printOnly {
				out, _, _, err := install.Render(t, server)
				if err != nil {
					return err
				}
				fmt.Print(string(out))
				return nil
			}

			path, changed, err := install.Install(t, server)
			if err != nil {
				return err
			}
			if !changed {
				fmt.Printf("Already installed in %s\n", path)
			} else {
				fmt.Printf("Installed in %s\n", path)
			}
			fmt.Println("Restart your client (or run 'MCP: List Servers' in VS Code) to start it.")
			return nil
		},
	}
	c.Flags().StringVar(&target, "target", "vscode", "where to install: vscode (user profile), workspace, or claude")
	c.Flags().BoolVar(&printOnly, "print", false, "print the config instead of writing it")
	c.Flags().StringVar(&command, "command", "", "override the server command (defaults to this executable)")
	return c
}

func newMcpUninstallCmd() *cobra.Command {
	var target string
	c := &cobra.Command{
		Use:   "uninstall",
		Short: "Remove this server from an MCP client's config",
		RunE: func(_ *cobra.Command, _ []string) error {
			t, ok := install.ValidTarget(target)
			if !ok {
				return fmt.Errorf("invalid --target %q (use vscode, workspace or claude)", target)
			}
			path, changed, err := install.Uninstall(t, serverName)
			if err != nil {
				return err
			}
			if !changed {
				fmt.Printf("Not installed in %s\n", path)
			} else {
				fmt.Printf("Removed from %s\n", path)
			}
			return nil
		},
	}
	c.Flags().StringVar(&target, "target", "vscode", "where to uninstall: vscode (user profile), workspace, or claude")
	return c
}
