package models

import "time"

// PluginConfig represents a metrics plugin configuration
type PluginConfig struct {
	Name            string            `yaml:"name"`
	Type            string            `yaml:"type"` // "prometheus"
	Enabled         bool              `yaml:"enabled"`
	URL             string            `yaml:"url"`
	Username        string            `yaml:"username,omitempty"`
	Password        string            `yaml:"password,omitempty"`
	RefreshInterval string            `yaml:"refresh_interval"` // "1m"
	TimeRange       string            `yaml:"time_range"`       // "1h"
	Labels          map[string]string `yaml:"labels,omitempty"` // Extra labels for filtering
}

// PluginsConfig holds all plugin configurations
type PluginsConfig struct {
	Plugins []PluginConfig `yaml:"plugins"`
}

// MetricSeries represents a time-series metric
type MetricSeries struct {
	Name   string      `json:"name"`   // e.g., consumer name
	Points []float64   `json:"points"` // Metric values
	Times  []time.Time `json:"times"`  // Timestamps
}

// MetricsData holds all metrics for a stream/consumer
type MetricsData struct {
	StreamName  string
	ConsumerName string
	Metrics     map[string][]MetricSeries  // Key = query name, Value = series
	FetchTime   time.Time
}

