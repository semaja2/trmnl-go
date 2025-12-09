package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the application configuration
type Config struct {
	// APIKey for usetrmnl.com authentication (if using cloud service)
	APIKey string `json:"api_key,omitempty"`

	// DeviceID for self-hosted server authentication (MAC address or unique ID)
	// If not set, will auto-detect primary network interface MAC address
	DeviceID string `json:"device_id,omitempty"`

	// FriendlyID is the human-readable device name from setup
	FriendlyID string `json:"friendly_id,omitempty"`

	// BaseURL for the TRMNL API (default: https://trmnl.app)
	BaseURL string `json:"base_url,omitempty"`

	// Model name for the device (e.g., "TRMNL", "virtual", "virtual-hd")
	Model string `json:"model,omitempty"`

	// WindowWidth for the display window
	WindowWidth int `json:"window_width,omitempty"`

	// WindowHeight for the display window
	WindowHeight int `json:"window_height,omitempty"`

	// DarkMode inverts image colors
	DarkMode bool `json:"dark_mode,omitempty"`

	// AlwaysOnTop keeps the window above all others
	AlwaysOnTop bool `json:"always_on_top,omitempty"`

	// MirrorMode uses /api/current_screen instead of device-specific display
	MirrorMode bool `json:"mirror_mode,omitempty"`

	// Verbose enables detailed logging
	Verbose bool `json:"verbose,omitempty"`
}

const (
	DefaultBaseURL      = "https://trmnl.app"
	DefaultWindowWidth  = 800
	DefaultWindowHeight = 480
	ConfigFileName      = "config.json"
)

// Load reads configuration from file and environment variables
// Priority: CLI flags > Environment variables > Config file > Defaults
func Load() (*Config, error) {
	cfg := &Config{
		BaseURL:      DefaultBaseURL,
		WindowWidth:  DefaultWindowWidth,
		WindowHeight: DefaultWindowHeight,
	}

	// Get config directory path
	configDir, err := getConfigDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get config directory: %w", err)
	}

	// Read from config file if it exists
	configPath := filepath.Join(configDir, ConfigFileName)
	if data, err := os.ReadFile(configPath); err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Override with environment variables
	if apiKey := os.Getenv("TRMNL_API_KEY"); apiKey != "" {
		cfg.APIKey = apiKey
	}
	if deviceID := os.Getenv("TRMNL_DEVICE_ID"); deviceID != "" {
		cfg.DeviceID = deviceID
	}
	if baseURL := os.Getenv("TRMNL_BASE_URL"); baseURL != "" {
		cfg.BaseURL = baseURL
	}

	return cfg, nil
}

// Save writes the configuration to disk
func (c *Config) Save() error {
	configDir, err := getConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, ConfigFileName)
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// getConfigDir returns the configuration directory path
// Uses XDG Base Directory specification on Unix-like systems
func getConfigDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Use XDG_CONFIG_HOME if set, otherwise use ~/.config
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(homeDir, ".config")
	}

	return filepath.Join(configHome, "trmnl"), nil
}

// Validate checks if the configuration is valid
func (c *Config) Validate() error {
	// Must have either API key or Device ID
	if c.APIKey == "" && c.DeviceID == "" {
		return fmt.Errorf("either API key or Device ID must be provided")
	}

	if c.BaseURL == "" {
		return fmt.Errorf("base URL cannot be empty")
	}

	if c.WindowWidth <= 0 || c.WindowHeight <= 0 {
		return fmt.Errorf("window dimensions must be positive")
	}

	return nil
}

// GetAuthHeader returns the appropriate authentication header name and value
func (c *Config) GetAuthHeader() (string, string) {
	if c.APIKey != "" {
		return "access-token", c.APIKey
	}
	return "ID", c.DeviceID
}
