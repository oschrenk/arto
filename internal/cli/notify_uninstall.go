package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/oschrenk/arto/internal/config"
)

func init() {
	notifyCmd.AddCommand(notifyUninstallCmd)
}

var notifyUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove hook and delete configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		settingsPath, err := resolveSettingsPath(cmd)
		if err != nil {
			return err
		}
		fmt.Printf("Removing hook from %s\n", settingsPath)

		if err := unregisterArtoHook(settingsPath); err != nil {
			return fmt.Errorf("removing hook: %w", err)
		}

		configPath := config.ConfigPath()
		fmt.Printf("Deleting %s\n", configPath)
		if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("deleting config: %w", err)
		}

		fmt.Println("Uninstalled.")
		return nil
	},
}
