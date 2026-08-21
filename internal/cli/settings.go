package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

func globalSettingsPath() (string, error) {
	configDir := os.Getenv("CLAUDE_CONFIG_DIR")
	if configDir == "" {
		out, err := exec.Command("claude", "config", "dir").Output()
		if err == nil && len(out) > 0 {
			configDir = strings.TrimSpace(string(out))
		}
	}
	if configDir == "" {
		return "", fmt.Errorf("CLAUDE_CONFIG_DIR not set and could not be resolved")
	}
	return filepath.Join(configDir, "settings.json"), nil
}

func resolveSettingsPath(cmd *cobra.Command) (string, error) {
	isGlobal, _ := cmd.Flags().GetBool("global")
	if isGlobal {
		return globalSettingsPath()
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	return filepath.Join(cwd, ".claude", "settings.json"), nil
}

func artoHookCommand() string {
	path, err := os.Executable()
	if err != nil {
		return "arto notify hook"
	}
	return fmt.Sprintf("%s notify hook", path)
}

func isArtoEntry(entry map[string]interface{}) bool {
	hooks, ok := entry["hooks"].([]interface{})
	if ok {
		for _, h := range hooks {
			hm, ok := h.(map[string]interface{})
			if ok {
				if cmd, ok := hm["command"].(string); ok && strings.Contains(cmd, "arto") {
					return true
				}
			}
		}
	}
	if cmd, ok := entry["command"].(string); ok && strings.Contains(cmd, "arto") {
		return true
	}
	return false
}

func registerArtoHook(settingsPath string) error {
	settings := make(map[string]interface{})
	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("reading settings: %w", err)
		}
	} else {
		if err := json.Unmarshal(raw, &settings); err != nil {
			return fmt.Errorf("parsing settings: %w", err)
		}
	}

	hooks, _ := settings["hooks"].(map[string]interface{})
	if hooks == nil {
		hooks = make(map[string]interface{})
		settings["hooks"] = hooks
	}

	permReq, _ := hooks["PermissionRequest"].([]interface{})
	if permReq == nil {
		permReq = []interface{}{}
	}

	hookCommand := artoHookCommand()
	hookEntry := map[string]interface{}{
		"hooks": []interface{}{
			map[string]interface{}{
				"type":    "command",
				"command": hookCommand,
			},
		},
	}

	existingIdx := -1
	for i, entry := range permReq {
		em, ok := entry.(map[string]interface{})
		if ok && isArtoEntry(em) {
			existingIdx = i
			break
		}
	}

	if existingIdx >= 0 {
		permReq[existingIdx] = hookEntry
	} else {
		permReq = append(permReq, hookEntry)
	}

	hooks["PermissionRequest"] = permReq

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling settings: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
		return fmt.Errorf("creating settings dir: %w", err)
	}

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return fmt.Errorf("writing settings: %w", err)
	}

	return nil
}

func unregisterArtoHook(settingsPath string) error {
	raw, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading settings: %w", err)
	}

	settings := make(map[string]interface{})
	if err := json.Unmarshal(raw, &settings); err != nil {
		return fmt.Errorf("parsing settings: %w", err)
	}

	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		return nil
	}

	permReq, ok := hooks["PermissionRequest"].([]interface{})
	if !ok {
		return nil
	}

	filtered := []interface{}{}
	for _, entry := range permReq {
		em, ok := entry.(map[string]interface{})
		if ok && isArtoEntry(em) {
			continue
		}
		filtered = append(filtered, entry)
	}

	if len(filtered) == len(permReq) {
		return nil
	}

	if len(filtered) == 0 {
		delete(hooks, "PermissionRequest")
	} else {
		hooks["PermissionRequest"] = filtered
	}

	if len(hooks) == 0 {
		delete(settings, "hooks")
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling settings: %w", err)
	}

	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return fmt.Errorf("writing settings: %w", err)
	}

	return nil
}
