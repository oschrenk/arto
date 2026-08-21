package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/oschrenk/arto/internal/config"
	"github.com/oschrenk/arto/internal/ntfy"
)

func init() {
	notifyCmd.AddCommand(notifyTestCmd)
}

var notifyTestCmd = &cobra.Command{
	Use:   "test",
	Short: "Send a test notification",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("loading config: %w", err)
		}
		if cfg.Topic == "" {
			return fmt.Errorf("no topic configured. Run 'arto notify setup' first")
		}

		auth := config.ResolveAuth(cfg)

		err = ntfy.SendNotification(ntfy.NotificationParams{
			Server:    cfg.NtfyServer,
			Topic:     cfg.Topic,
			Title:     "arto",
			Message:   "Test notification - if you see this, setup is working!",
			Actions:   []ntfy.Action{},
			RequestID: "test",
			Auth:      auth,
		})
		if err != nil {
			return fmt.Errorf("sending notification: %w", err)
		}

		fmt.Println("Test notification sent successfully.")
		return nil
	},
}
