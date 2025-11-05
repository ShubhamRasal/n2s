package plugins

import (
	"github.com/shubhamrasal/n2s/internal/models"
)

// MetricsPlugin is the interface all metrics plugins must implement
type MetricsPlugin interface {
	// Name returns the plugin name
	Name() string
	
	// Configure initializes the plugin with config
	Configure(config *models.PluginConfig) error
	
	// GetConsumerMetrics fetches consumer metrics
	GetConsumerMetrics(streamName, consumerName string, timeRange string) (*models.MetricsData, error)
	
	// GetStreamMetrics fetches stream-level metrics
	GetStreamMetrics(streamName string, timeRange string) (*models.MetricsData, error)
	
	// HealthCheck verifies the plugin is working
	HealthCheck() error
	
	// IsEnabled returns whether the plugin is enabled
	IsEnabled() bool
}

