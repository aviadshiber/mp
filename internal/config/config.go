// Package config manages persistent CLI configuration stored in ~/.config/mp/config.yaml.
// It provides read/write/list operations and masks sensitive values in output.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// Known configuration keys.
const (
	KeyProjectID     = "project_id"
	KeyRegion        = "region"
	KeyServiceAccount = "service_account"
	KeyServiceSecret = "service_secret"
)

// sensitiveKeys are masked in list output.
var sensitiveKeys = map[string]bool{
	KeyServiceSecret: true,
}

// knownKeys defines the valid configuration keys and their descriptions.
var knownKeys = map[string]string{
	KeyProjectID:      "Mixpanel project ID",
	KeyRegion:         "API region (us, eu, in)",
	KeyServiceAccount: "Service account username",
	KeyServiceSecret:  "Service account secret",
}

// Config wraps viper to manage mp configuration.
type Config struct {
	v        *viper.Viper
	filePath string
}

// New creates a Config that reads from ~/.config/mp/config.yaml.
// It creates the config directory if it does not exist.
func New() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("determining home directory: %w", err)
	}

	dir := filepath.Join(home, ".config", "mp")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("creating config directory %s: %w", dir, err)
	}

	filePath := filepath.Join(dir, "config.yaml")

	v := viper.New()
	v.SetConfigFile(filePath)
	v.SetConfigType("yaml")

	// Read existing config; ignore file-not-found since we create on first write.
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			// File may simply not exist yet.
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("reading config: %w", err)
			}
		}
	}

	return &Config{v: v, filePath: filePath}, nil
}

// Get returns the value for a configuration key.
func (c *Config) Get(key string) string {
	return c.v.GetString(key)
}

// Set writes a configuration key-value pair and persists to disk.
func (c *Config) Set(key, value string) error {
	if _, ok := knownKeys[key]; !ok {
		return fmt.Errorf("unknown config key %q; valid keys: %s", key, strings.Join(KnownKeyNames(), ", "))
	}

	if key == KeyRegion {
		value = strings.ToLower(value)
		if value != "us" && value != "eu" && value != "in" {
			return fmt.Errorf("invalid region %q; must be one of: us, eu, in", value)
		}
	}

	c.v.Set(key, value)
	return c.write()
}

// List returns all set configuration entries as key-value pairs.
// Sensitive values are masked.
func (c *Config) List() []Entry {
	var entries []Entry
	for _, key := range KnownKeyNames() {
		val := c.v.GetString(key)
		if val == "" {
			continue
		}
		if sensitiveKeys[key] {
			val = mask(val)
		}
		entries = append(entries, Entry{Key: key, Value: val})
	}
	return entries
}

// Entry is a single configuration key-value pair.
type Entry struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// KnownKeyNames returns sorted known key names.
func KnownKeyNames() []string {
	return []string{KeyProjectID, KeyRegion, KeyServiceAccount, KeyServiceSecret}
}

// FilePath returns the path to the configuration file.
func (c *Config) FilePath() string {
	return c.filePath
}

func (c *Config) write() error {
	return c.v.WriteConfigAs(c.filePath)
}

// mask shows the first 4 characters followed by "****".
func mask(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return s[:4] + "****"
}
