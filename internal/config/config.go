package config

import (
	"crypto/rand"
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

//go:embed default.toml
var defaultConfig []byte

type Config struct {
	Topic      string `mapstructure:"topic"         toml:"topic"`
	NtfyServer string `mapstructure:"ntfy_server"   toml:"ntfy_server"`
	Timeout    int    `mapstructure:"timeout"        toml:"timeout"`
	PlanTimeout int   `mapstructure:"plan_timeout"   toml:"plan_timeout"`
	NtfyUser   string `mapstructure:"ntfy_username"  toml:"ntfy_username"`
	NtfyPass   string `mapstructure:"ntfy_password"  toml:"ntfy_password"`
}

func Defaults() Config {
	return Config{
		Topic:       "",
		NtfyServer:  "https://ntfy.sh",
		Timeout:     120,
		PlanTimeout: 300,
		NtfyUser:    "",
		NtfyPass:    "",
	}
}

func ConfigDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "arto")
	}
	return filepath.Join(os.Getenv("HOME"), ".config", "arto")
}

func ConfigPath() string {
	return filepath.Join(ConfigDir(), "config.toml")
}

func Load() (*Config, error) {
	c := Defaults()
	path := ConfigPath()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		return &c, nil
	}

	if _, err := toml.DecodeFile(path, &c); err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	// Override with env vars if set
	if u := os.Getenv("NTFY_USERNAME"); u != "" {
		c.NtfyUser = u
	}
	if p := os.Getenv("NTFY_PASSWORD"); p != "" {
		c.NtfyPass = p
	}

	// Validate
	if c.Timeout <= 0 {
		c.Timeout = 120
	}
	if c.PlanTimeout <= 0 {
		c.PlanTimeout = 300
	}

	return &c, nil
}

func Save(c *Config) error {
	dir := ConfigDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	path := ConfigPath()
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("opening config file: %w", err)
	}
	defer f.Close()

	enc := toml.NewEncoder(f)
	if err := enc.Encode(c); err != nil {
		return fmt.Errorf("writing config: %w", err)
	}

	return nil
}

func GenerateTopic() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return fmt.Sprintf("arto-%x", b)
}

type Auth struct {
	Username string
	Password string
}

func ResolveAuth(c *Config) *Auth {
	username := c.NtfyUser
	password := c.NtfyPass
	if u := os.Getenv("NTFY_USERNAME"); u != "" {
		username = u
	}
	if p := os.Getenv("NTFY_PASSWORD"); p != "" {
		password = p
	}
	if username == "" || password == "" {
		return nil
	}
	return &Auth{Username: username, Password: password}
}
