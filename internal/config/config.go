package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Contexts        []Context `yaml:"contexts"`
	DefaultContext  string    `yaml:"default_context"`
	RefreshInterval string    `yaml:"refresh_interval"`
	currentContext  *Context
}

// Context represents a NATS server connection context
type Context struct {
	Name          string `yaml:"name"`
	Server        string `yaml:"server"`
	Token         string `yaml:"token,omitempty"`
	Creds         string `yaml:"creds,omitempty"`
	MetricsPlugin string `yaml:"metrics_plugin,omitempty"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Contexts: []Context{
			{
				Name:   "local",
				Server: "nats://localhost:4222",
			},
		},
		DefaultContext:  "local",
		RefreshInterval: "2s",
	}
}

// Load loads configuration from file or creates default
func Load(configPath, serverURL string) (*Config, error) {
	var cfg *Config

	// If server URL is provided via command line, use it
	if serverURL != "" {
		cfg = &Config{
			Contexts: []Context{
				{
					Name:   "cli",
					Server: serverURL,
				},
			},
			DefaultContext:  "cli",
			RefreshInterval: "2s",
		}
		cfg.currentContext = &cfg.Contexts[0]
		return cfg, nil
	}

	// Try to load from config file
	if configPath == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = filepath.Join(homeDir, ".config", "n2s", "config.yaml")
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Create default config
		cfg = DefaultConfig()
		if err := cfg.Save(configPath); err != nil {
			return nil, fmt.Errorf("failed to save default config: %w", err)
		}
	} else {
		// Load from file
		data, err := os.ReadFile(configPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		cfg = &Config{}
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Set current context
	for i := range cfg.Contexts {
		if cfg.Contexts[i].Name == cfg.DefaultContext {
			cfg.currentContext = &cfg.Contexts[i]
			break
		}
	}

	if cfg.currentContext == nil && len(cfg.Contexts) > 0 {
		cfg.currentContext = &cfg.Contexts[0]
	}

	return cfg, nil
}

// Save saves the configuration to file
func (c *Config) Save(configPath string) error {
	// Create config directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// CurrentContext returns the current context
func (c *Config) CurrentContext() *Context {
	if c.currentContext != nil {
		return c.currentContext
	}
	// Return default context
	return &Context{
		Name:   "default",
		Server: "nats://localhost:4222",
	}
}

// CurrentContextName returns the current context name
func (c *Config) CurrentContextName() string {
	if c.currentContext != nil {
		return c.currentContext.Name
	}
	return "unknown"
}

// SetContext switches to a different context
func (c *Config) SetContext(name string) error {
	for i := range c.Contexts {
		if c.Contexts[i].Name == name {
			c.currentContext = &c.Contexts[i]
			c.DefaultContext = name
			return nil
		}
	}
	return fmt.Errorf("context '%s' not found", name)
}

// AddContext adds a new context
func (c *Config) AddContext(name, server string) error {
	// Check if context already exists
	for _, ctx := range c.Contexts {
		if ctx.Name == name {
			return fmt.Errorf("context '%s' already exists", name)
		}
	}

	c.Contexts = append(c.Contexts, Context{
		Name:   name,
		Server: server,
	})

	return nil
}

// RemoveContext removes a context
func (c *Config) RemoveContext(name string) error {
	for i, ctx := range c.Contexts {
		if ctx.Name == name {
			c.Contexts = append(c.Contexts[:i], c.Contexts[i+1:]...)
			// If we removed the current context, switch to first available
			if c.currentContext != nil && c.currentContext.Name == name {
				if len(c.Contexts) > 0 {
					c.currentContext = &c.Contexts[0]
					c.DefaultContext = c.Contexts[0].Name
				} else {
					c.currentContext = nil
					c.DefaultContext = ""
				}
			}
			return nil
		}
	}
	return fmt.Errorf("context '%s' not found", name)
}

// GetRefreshInterval returns the refresh interval as duration
func (c *Config) GetRefreshInterval() time.Duration {
	d, err := time.ParseDuration(c.RefreshInterval)
	if err != nil {
		return 2 * time.Second
	}
	return d
}

