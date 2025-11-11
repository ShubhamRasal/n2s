package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ConfigSource represents where the configuration was loaded from
type ConfigSource string

const (
	SourceCLI         ConfigSource = "cli"          // From command line argument
	SourceConfigFile  ConfigSource = "config-file"  // From ~/.config/n2s/config.yaml
	SourceNATSContext ConfigSource = "nats-context" // From NATS CLI contexts
	SourceDefault     ConfigSource = "default"      // Default configuration
)

// Config represents the application configuration
type Config struct {
	Contexts        []Context `yaml:"contexts"`
	DefaultContext  string    `yaml:"default_context"`
	RefreshInterval string    `yaml:"refresh_interval"`
	currentContext  *Context
	source          ConfigSource // Where this config was loaded from
	sourcePath      string       // Specific file path or context name
}

// Context represents a NATS server connection context
type Context struct {
	Name          string `yaml:"name"`
	Server        string `yaml:"server"`
	Token         string `yaml:"token,omitempty"`
	Creds         string `yaml:"creds,omitempty"`
	MetricsPlugin string `yaml:"metrics_plugin,omitempty"`
}

// natsContext represents the NATS CLI context JSON format
type natsContext struct {
	URL      string `json:"url"`
	Token    string `json:"token"`
	Creds    string `json:"creds"`
	User     string `json:"user"`
	Password string `json:"password"`
	NKey     string `json:"nkey"`
}

// expandPath expands environment variables, tilde, and relative paths
// Supports:
// - Environment variables: $HOME, ${HOME}, $VAR_NAME
// - Tilde expansion: ~/path or ~
// - Relative paths: ./creds/file.creds or ../creds/file.creds (relative to configDir)
func expandPath(path string, configDir string) (string, error) {
	if path == "" {
		return "", nil
	}

	// First, expand environment variables
	expanded := os.ExpandEnv(path)

	// Handle tilde expansion for home directory
	if strings.HasPrefix(expanded, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		expanded = filepath.Join(homeDir, expanded[2:])
	} else if expanded == "~" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		expanded = homeDir
	}

	// If the path is not absolute, make it relative to config directory
	if !filepath.IsAbs(expanded) && configDir != "" {
		expanded = filepath.Join(configDir, expanded)
	}

	// Clean the path to normalize it
	expanded = filepath.Clean(expanded)

	return expanded, nil
}

// getNATSContextDir returns the NATS CLI context directory path
func getNATSContextDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(homeDir, ".config", "nats", "context"), nil
}

// getCurrentNATSContext reads the current NATS CLI context name from context.txt
func getCurrentNATSContext() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	
	contextFile := filepath.Join(homeDir, ".config", "nats", "context.txt")
	data, err := os.ReadFile(contextFile)
	if err != nil {
		return "", err
	}
	
	return strings.TrimSpace(string(data)), nil
}

// readNATSContext reads a NATS CLI context JSON file and converts it to our Context format
func readNATSContext(name string) (*Context, error) {
	contextDir, err := getNATSContextDir()
	if err != nil {
		return nil, err
	}

	contextPath := filepath.Join(contextDir, name+".json")
	data, err := os.ReadFile(contextPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read NATS context '%s': %w", name, err)
	}

	var natsCtx natsContext
	if err := json.Unmarshal(data, &natsCtx); err != nil {
		return nil, fmt.Errorf("failed to parse NATS context '%s': %w", name, err)
	}

	// Expand paths in the NATS context
	creds := natsCtx.Creds
	if creds != "" {
		creds, err = expandPath(creds, contextDir)
		if err != nil {
			return nil, fmt.Errorf("failed to expand creds path: %w", err)
		}
	}

	// Expand environment variables in token if present
	token := natsCtx.Token
	if token != "" && strings.Contains(token, "$") {
		token = os.ExpandEnv(token)
	}

	return &Context{
		Name:   name,
		Server: natsCtx.URL,
		Token:  token,
		Creds:  creds,
	}, nil
}

// listNATSContexts returns a list of all available NATS CLI contexts
func listNATSContexts() ([]Context, error) {
	contextDir, err := getNATSContextDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(contextDir)
	if err != nil {
		return nil, err
	}

	var contexts []Context
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}

		// Skip backup files
		if strings.HasSuffix(entry.Name(), ".bak") {
			continue
		}

		name := strings.TrimSuffix(entry.Name(), ".json")
		ctx, err := readNATSContext(name)
		if err != nil {
			// Skip contexts that can't be read
			continue
		}
		contexts = append(contexts, *ctx)
	}

	return contexts, nil
}

// loadFromNATSContexts creates a config from NATS CLI contexts
func loadFromNATSContexts() (*Config, error) {
	contexts, err := listNATSContexts()
	if err != nil || len(contexts) == 0 {
		return nil, fmt.Errorf("no NATS contexts found")
	}

	// Get the current context
	currentCtx, err := getCurrentNATSContext()
	if err != nil {
		// If no current context, use the first one
		currentCtx = contexts[0].Name
	}

	contextDir, _ := getNATSContextDir()
	
	cfg := &Config{
		Contexts:        contexts,
		DefaultContext:  currentCtx,
		RefreshInterval: "2s",
		source:          SourceNATSContext,
		sourcePath:      filepath.Join(contextDir, currentCtx+".json"),
	}

	// Set current context
	for i := range cfg.Contexts {
		if cfg.Contexts[i].Name == currentCtx {
			cfg.currentContext = &cfg.Contexts[i]
			break
		}
	}

	if cfg.currentContext == nil {
		cfg.currentContext = &cfg.Contexts[0]
	}

	return cfg, nil
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
		source:          SourceDefault,
		sourcePath:      "built-in default",
	}
}

// Load loads configuration from file, NATS contexts, or creates default
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
			source:          SourceCLI,
			sourcePath:      serverURL,
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
		// Try to load from NATS CLI contexts as fallback
		cfg, err = loadFromNATSContexts()
		if err == nil {
			return cfg, nil
		}

		// If NATS contexts also not found, create default config
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

		// Set source information
		cfg.source = SourceConfigFile
		cfg.sourcePath = configPath

		// Expand credential paths with env vars, tilde, and relative paths
		configDir := filepath.Dir(configPath)
		for i := range cfg.Contexts {
			if cfg.Contexts[i].Creds != "" {
				expanded, err := expandPath(cfg.Contexts[i].Creds, configDir)
				if err != nil {
					return nil, fmt.Errorf("failed to expand creds path for context '%s': %w", cfg.Contexts[i].Name, err)
				}
				cfg.Contexts[i].Creds = expanded
			}
			// Also expand token if it looks like it might be an env var reference
			if cfg.Contexts[i].Token != "" && strings.Contains(cfg.Contexts[i].Token, "$") {
				cfg.Contexts[i].Token = os.ExpandEnv(cfg.Contexts[i].Token)
			}
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

// GetConfigSource returns where the configuration was loaded from
func (c *Config) GetConfigSource() ConfigSource {
	return c.source
}

// GetConfigSourcePath returns the specific path or identifier for the config source
func (c *Config) GetConfigSourcePath() string {
	return c.sourcePath
}

// GetConfigSourceDescription returns a human-readable description of the config source
func (c *Config) GetConfigSourceDescription() string {
	switch c.source {
	case SourceCLI:
		return fmt.Sprintf("Command line: %s", c.sourcePath)
	case SourceConfigFile:
		// Show just the path, can be shortened by caller if needed
		return fmt.Sprintf("Config file: %s", c.sourcePath)
	case SourceNATSContext:
		// Extract just the context name from the path
		contextName := filepath.Base(c.sourcePath)
		contextName = strings.TrimSuffix(contextName, ".json")
		return fmt.Sprintf("NATS context: %s", contextName)
	case SourceDefault:
		return "Built-in default (no config found)"
	default:
		return "Unknown source"
	}
}

