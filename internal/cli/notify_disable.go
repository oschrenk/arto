package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func init() {
	notifyCmd.AddCommand(notifyDisableCmd)
}

var notifyDisableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Remove the hook from Claude Code, keep config",
	RunE: func(cmd *cobra.Command, args []string) error {
		settingsPath, err := resolveSettingsPath(cmd)
		if err != nil {
			return err
		}
		fmt.Printf("Removing hook from %s\n", settingsPath)

		if err := unregisterArtoHook(settingsPath); err != nil {
			return fmt.Errorf("disabling hook: %w", err)
		}

		fmt.Println("Hook disabled.")
		return nil
	},
}
