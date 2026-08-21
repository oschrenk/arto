/*
Copyright © 2026 Oliver Schrenk
*/
package cli

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "arto",
	Short: "Remote approval for Claude Code via ntfy push notifications",
}

func Execute() {
	// hide (but not disable) "completion" feature
	rootCmd.CompletionOptions.HiddenDefaultCmd = true
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}
