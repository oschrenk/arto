package cli

import (
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(notifyCmd)
	notifyCmd.PersistentFlags().Bool("local", true, "Use project-level .claude/settings.json (default)")
	notifyCmd.PersistentFlags().Bool("global", false, "Use global $CLAUDE_CONFIG_DIR/settings.json")
	notifyCmd.MarkFlagsMutuallyExclusive("local", "global")
}

var notifyCmd = &cobra.Command{
	Use:   "notify",
	Short: "Manage ntfy push notifications for Claude Code",
}
