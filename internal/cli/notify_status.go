package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/oschrenk/arto/internal/config"
)

func init() {
	notifyCmd.AddCommand(notifyStatusCmd)
}

var notifyStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}

		fmt.Printf("Topic:   %s\n", cfg.Topic)
		fmt.Printf("Server:  %s\n", cfg.NtfyServer)
		fmt.Printf("Timeout: %ds\n", cfg.Timeout)

		auth := config.ResolveAuth(cfg)
		if auth != nil {
			fmt.Printf("Auth:    configured (username: %s)\n", auth.Username)
		} else {
			fmt.Printf("Auth:    not configured\n")
		}

		return nil
	},
}
