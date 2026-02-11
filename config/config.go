package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Finnhub  FinnhubConfig `yaml:"finnhub"`
	Bark     BarkConfig    `yaml:"bark"`
	Interval int           `yaml:"interval"` // Refresh interval in seconds
	Rules    []Rule        `yaml:"rules"`
	Holdings []Holding     `yaml:"holdings,omitempty"`
}

// FinnhubConfig holds Finnhub API configuration
type FinnhubConfig struct {
	APIKey string `yaml:"api_key"`
}

// BarkConfig holds Bark push notification configuration
type BarkConfig struct {
	ServerURL string `yaml:"server_url"` // e.g. https://api.day.app
	Key       string `yaml:"key"`
}

// Rule defines a stock monitoring rule
type Rule struct {
	Symbol      string   `yaml:"symbol"`
	Market      string   `yaml:"market,omitempty"`       // "US", "CN", "HK", "CRYPTO", "FOREX"
	Name        string   `yaml:"name,omitempty"`         // Optional display name
	PriceAbove  *float64 `yaml:"price_above,omitempty"`  // Trigger if price > threshold
	PriceBelow  *float64 `yaml:"price_below,omitempty"`  // Trigger if price < threshold
	ChangeAbove *float64 `yaml:"change_above,omitempty"` // Trigger if change% > threshold
	ChangeBelow *float64 `yaml:"change_below,omitempty"` // Trigger if change% < threshold
}

// Holding defines a user's stock position
type Holding struct {
	Symbol    string  `yaml:"symbol"`
	Quantity  float64 `yaml:"quantity"`
	CostPrice float64 `yaml:"cost_price"`
}

// DefaultConfigPath returns the default config file path
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".stock-ping.yaml"
	}
	return filepath.Join(home, ".stock-ping.yaml")
}

// Load loads configuration from the default path
func Load() (*Config, error) {
	return LoadFrom(DefaultConfigPath())
}

// LoadFrom loads configuration from a specific path
func LoadFrom(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			return &Config{
				Interval: 60,
				Rules:    []Rule{},
			}, nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Set defaults
	if cfg.Interval <= 0 {
		cfg.Interval = 60
	}
	if cfg.Bark.ServerURL == "" {
		cfg.Bark.ServerURL = "https://api.day.app"
	}

	return &cfg, nil
}

// Save saves configuration to the default path
func (c *Config) Save() error {
	return c.SaveTo(DefaultConfigPath())
}

// SaveTo saves configuration to a specific path
func (c *Config) SaveTo(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// AddRule adds a new rule to the configuration
func (c *Config) AddRule(rule Rule) error {
	// Check if symbol already exists
	for i, r := range c.Rules {
		if r.Symbol == rule.Symbol {
			// Update existing rule
			c.Rules[i] = rule
			return nil
		}
	}
	c.Rules = append(c.Rules, rule)
	return nil
}

// RemoveRule removes a rule by symbol
func (c *Config) RemoveRule(symbol string) bool {
	for i, r := range c.Rules {
		if r.Symbol == symbol {
			c.Rules = append(c.Rules[:i], c.Rules[i+1:]...)
			return true
		}
	}
	return false
}

// GetRule returns a rule by symbol
func (c *Config) GetRule(symbol string) *Rule {
	for i := range c.Rules {
		if c.Rules[i].Symbol == symbol {
			return &c.Rules[i]
		}
	}
	return nil
}

// AddHolding adds or updates a holding in the configuration
func (c *Config) AddHolding(holding Holding) {
	for i, h := range c.Holdings {
		if h.Symbol == holding.Symbol {
			c.Holdings[i] = holding
			return
		}
	}
	c.Holdings = append(c.Holdings, holding)
}

// RemoveHolding removes a holding by symbol
func (c *Config) RemoveHolding(symbol string) bool {
	for i, h := range c.Holdings {
		if h.Symbol == symbol {
			c.Holdings = append(c.Holdings[:i], c.Holdings[i+1:]...)
			return true
		}
	}
	return false
}

// GetHolding returns a holding by symbol
func (c *Config) GetHolding(symbol string) *Holding {
	for i := range c.Holdings {
		if c.Holdings[i].Symbol == symbol {
			return &c.Holdings[i]
		}
	}
	return nil
}
