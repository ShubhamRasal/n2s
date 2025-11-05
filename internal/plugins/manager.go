package plugins

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/shubhamrasal/n2s/internal/models"
	"github.com/shubhamrasal/n2s/internal/plugins/prometheus"
	"gopkg.in/yaml.v3"
)

// Manager manages metrics plugins
type Manager struct {
	plugins map[string]MetricsPlugin
}

// NewManager creates a new plugin manager
func NewManager() *Manager {
	return &Manager{
		plugins: make(map[string]MetricsPlugin),
	}
}

// LoadPlugins loads plugin configurations and initializes plugins
func (m *Manager) LoadPlugins() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	pluginPath := filepath.Join(homeDir, ".config", "n2s", "plugins.yaml")

	// Check if file exists
	if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
		// No plugins configured - that's OK
		return nil
	}

	// Load plugin config
	data, err := os.ReadFile(pluginPath)
	if err != nil {
		return fmt.Errorf("failed to read plugins file: %w", err)
	}

	var pluginsConfig models.PluginsConfig
	if err := yaml.Unmarshal(data, &pluginsConfig); err != nil {
		return fmt.Errorf("failed to parse plugins file: %w", err)
	}

	// Initialize each plugin
	for _, config := range pluginsConfig.Plugins {
		if err := m.loadPlugin(&config); err != nil {
			// Log error but continue loading other plugins
			continue
		}
	}

	return nil
}

// loadPlugin loads a single plugin
func (m *Manager) loadPlugin(config *models.PluginConfig) error {
	var plugin MetricsPlugin

	switch config.Type {
	case "prometheus":
		plugin = prometheus.NewPrometheusPlugin(config.Name)
	default:
		return fmt.Errorf("unknown plugin type: %s", config.Type)
	}

	// Configure the plugin
	if err := plugin.Configure(config); err != nil {
		return fmt.Errorf("failed to configure plugin %s: %w", config.Name, err)
	}

	// Store the plugin
	m.plugins[config.Name] = plugin

	return nil
}

// GetPlugin returns a plugin by name
func (m *Manager) GetPlugin(name string) (MetricsPlugin, error) {
	plugin, exists := m.plugins[name]
	if !exists {
		return nil, fmt.Errorf("plugin '%s' not found", name)
	}

	if !plugin.IsEnabled() {
		return nil, fmt.Errorf("plugin '%s' is not enabled", name)
	}

	return plugin, nil
}

// HasPlugin checks if a plugin exists and is enabled
func (m *Manager) HasPlugin(name string) bool {
	plugin, exists := m.plugins[name]
	return exists && plugin.IsEnabled()
}

