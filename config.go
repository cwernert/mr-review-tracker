package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	appDirName        = "mr-review-tracker"
	configFileName    = "config.json"
	defaultChannelID = "fbdee107-eb6b-47d5-ba86-0aaa8a41b813"
	// defaultPollSecs is what new installs (and any config below minPollSecs)
	// snap to. 10 minutes is a sane MR-review-pacing default: not so frequent
	// it hammers the API, frequent enough to catch urgent reviews within a
	// stand-up window.
	defaultPollSecs = 600
	// minPollSecs is the floor accepted by SetPollSeconds + normalize. Matches
	// the smallest option in pollIntervalChoices (1 minute). Anything below
	// snaps up to defaultPollSecs.
	minPollSecs       = 60
	storageURLPattern = "https://store.zapier.com/api/records?secret=%s"
)

// Config is the on-disk shape persisted to ~/Library/Application Support/mr-review-tracker/config.json.
type Config struct {
	ChannelID   string `json:"channel_id"`
	PollSeconds int    `json:"poll_seconds"`

	mu   sync.Mutex
	path string
}

func defaultConfig() *Config {
	return &Config{
		ChannelID:   defaultChannelID,
		PollSeconds: defaultPollSecs,
	}
}

// supportDir returns the app's directory under ~/Library/Application Support, creating it on demand.
func supportDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, "Library", "Application Support", appDirName)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

// LoadConfig reads (or creates) the on-disk config and returns it.
func LoadConfig() (*Config, error) {
	dir, err := supportDir()
	if err != nil {
		return nil, fmt.Errorf("locate support dir: %w", err)
	}
	path := filepath.Join(dir, configFileName)

	cfg := defaultConfig()
	cfg.path = path

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		if err := cfg.save(); err != nil {
			return nil, fmt.Errorf("seed config: %w", err)
		}
		return cfg, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	cfg.path = path
	cfg.normalize()
	return cfg, nil
}

// normalize fills in any missing or invalid values with sensible defaults.
func (c *Config) normalize() {
	if c.ChannelID == "" {
		c.ChannelID = defaultChannelID
	}
	if c.PollSeconds < minPollSecs {
		c.PollSeconds = defaultPollSecs
	}
}

// Snapshot returns a copy of the current config values, safe to read concurrently.
func (c *Config) Snapshot() (channelID string, pollSeconds int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ChannelID, c.PollSeconds
}

// SetChannelID updates the channel ID and persists.
func (c *Config) SetChannelID(id string) error {
	c.mu.Lock()
	c.ChannelID = id
	c.mu.Unlock()
	return c.save()
}

// SetPollSeconds updates the polling interval and persists.
func (c *Config) SetPollSeconds(secs int) error {
	c.mu.Lock()
	if secs < minPollSecs {
		secs = minPollSecs
	}
	c.PollSeconds = secs
	c.mu.Unlock()
	return c.save()
}

// Path returns the path of the on-disk config file.
func (c *Config) Path() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.path
}

func (c *Config) save() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.path == "" {
		dir, err := supportDir()
		if err != nil {
			return err
		}
		c.path = filepath.Join(dir, configFileName)
	}

	data, err := json.MarshalIndent(struct {
		ChannelID   string `json:"channel_id"`
		PollSeconds int    `json:"poll_seconds"`
	}{
		ChannelID:   c.ChannelID,
		PollSeconds: c.PollSeconds,
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, append(data, '\n'), 0o644)
}
