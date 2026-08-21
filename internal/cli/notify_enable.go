package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/oschrenk/arto/internal/config"
)

func init() {
	notifyCmd.AddCommand(notifyEnableCmd)
}

var notifyEnableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Re-register the hook in Claude Code",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		if cfg.Topic == "" {
			return fmt.Errorf("no topic configured. Run 'arto notify setup' first")
		}

		settingsPath, err := resolveSettingsPath(cmd)
		if err != nil {
			return err
		}
		fmt.Printf("Registering hook in %s\n", settingsPath)

		if err := registerArtoHook(settingsPath); err != nil {
			return fmt.Errorf("enabling hook: %w", err)
		}

		fmt.Println("Hook enabled.")
		return nil
	},
}
