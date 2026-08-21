package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/oschrenk/arto/internal/hook"
)

func init() {
	notifyCmd.AddCommand(notifyHookCmd)
}

var notifyHookCmd = &cobra.Command{
	Use:    "hook",
	Short:  "Process a Claude Code hook (internal)",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		inputData, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[arto] Failed to read stdin: %v\n", err)
			fmt.Println(`{"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"ask"}}}`)
			return nil
		}

		result, err := hook.ProcessHook(inputData)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[arto] Hook processing failed: %v\n", err)
			fmt.Println(`{"hookSpecificOutput":{"hookEventName":"PermissionRequest","decision":{"behavior":"ask"}}}`)
			return nil
		}

		fmt.Println(string(result))
		return nil
	},
}
