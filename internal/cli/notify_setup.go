package cli

import (
	"bufio"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/mdp/qrterminal/v3"
	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/oschrenk/arto/internal/config"
)

func init() {
	notifyCmd.AddCommand(notifySetupCmd)
}

var notifySetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Generate topic, configure auth, register hook",
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		cfg, err := config.Load()
		if err != nil {
			cfg = &config.Config{}
			defaults := config.Defaults()
			*cfg = defaults
		}

		topic := cfg.Topic
		if topic == "" {
			fmt.Println("Generating topic...")
			topic = config.GenerateTopic()
			cfg.Topic = topic
		}
		fmt.Printf("Topic: %s\n\n", topic)

		configPath := config.ConfigPath()
		fmt.Printf("Saving config to %s\n\n", configPath)

		fmt.Print("Use authenticated topics? (only for self-hosted ntfy servers) (y/n): ")
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(answer)
		if strings.ToLower(answer) == "y" {
			fmt.Print("Username: ")
			username, _ := reader.ReadString('\n')
			cfg.NtfyUser = strings.TrimSpace(username)

			fmt.Print("Password: ")
			passwordBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
			fmt.Println()
			if err != nil {
				return fmt.Errorf("reading password: %w", err)
			}
			cfg.NtfyPass = string(passwordBytes)
		}
		fmt.Println()

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("saving config: %w", err)
		}

		settingsPath, err := resolveSettingsPath(cmd)
		if err != nil {
			return err
		}
		fmt.Printf("Registering hook in %s\n\n", settingsPath)

		if err := registerArtoHook(settingsPath); err != nil {
			return fmt.Errorf("registering hook: %w", err)
		}

		serverURL, err := url.Parse(cfg.NtfyServer)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Invalid ntfyServer URL in config: %s\n", cfg.NtfyServer)
			fmt.Printf("Subscribe to topic \"%s\" in the ntfy app.\n", topic)
			return nil
		}

		var ntfyURL string
		if serverURL.Scheme == "https" {
			ntfyURL = fmt.Sprintf("ntfy://%s/%s", serverURL.Host, topic)
		} else {
			ntfyURL = fmt.Sprintf("ntfy://%s/%s?secure=false", serverURL.Host, topic)
		}
		subscribeURL := fmt.Sprintf("%s/%s", strings.TrimRight(cfg.NtfyServer, "/"), topic)

		fmt.Print("Scan this QR code in the ntfy app to subscribe:\n\n")
		qrterminal.GenerateWithConfig(ntfyURL, qrterminal.Config{
			Level:     qrterminal.L,
			Writer:    os.Stdout,
			QuietZone: 1,
		})
		fmt.Printf("\nSubscribe URL: %s\n", subscribeURL)

		return nil
	},
}
