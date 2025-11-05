package prometheus

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"github.com/shubhamrasal/n2s/internal/models"
)

// PrometheusPlugin implements the MetricsPlugin interface for Prometheus
type PrometheusPlugin struct {
	name     string
	config   *models.PluginConfig
	client   api.Client
	queryAPI v1.API
	enabled  bool
}

// NewPrometheusPlugin creates a new Prometheus plugin
func NewPrometheusPlugin(name string) *PrometheusPlugin {
	return &PrometheusPlugin{
		name:    name,
		enabled: false,
	}
}

// Name returns the plugin name
func (p *PrometheusPlugin) Name() string {
	return p.name
}

// Configure initializes the plugin
func (p *PrometheusPlugin) Configure(config *models.PluginConfig) error {
	p.config = config
	p.enabled = config.Enabled

	if !config.Enabled {
		return nil
	}

	// Create HTTP client with basic auth if provided
	roundTripper := api.DefaultRoundTripper
	if config.Username != "" || config.Password != "" {
		roundTripper = &basicAuthRoundTripper{
			username: config.Username,
			password: config.Password,
			next:     api.DefaultRoundTripper,
		}
	}

	// Create Prometheus API client
	client, err := api.NewClient(api.Config{
		Address:      config.URL,
		RoundTripper: roundTripper,
	})
	if err != nil {
		return fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	p.client = client
	p.queryAPI = v1.NewAPI(client)

	return nil
}

// basicAuthRoundTripper implements HTTP basic authentication
type basicAuthRoundTripper struct {
	username string
	password string
	next     http.RoundTripper
}

func (rt *basicAuthRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if rt.username != "" || rt.password != "" {
		req.SetBasicAuth(rt.username, rt.password)
	}
	return rt.next.RoundTrip(req)
}

// GetConsumerMetrics fetches consumer metrics from Prometheus
func (p *PrometheusPlugin) GetConsumerMetrics(streamName, consumerName string, timeRange string) (*models.MetricsData, error) {
	if !p.enabled {
		return nil, fmt.Errorf("plugin not enabled")
	}

	// Parse time range
	duration, err := time.ParseDuration(timeRange)
	if err != nil {
		duration = time.Hour // Default 1 hour
	}

	end := time.Now()
	start := end.Add(-duration)

	metricsData := &models.MetricsData{
		StreamName:   streamName,
		ConsumerName: consumerName,
		FetchTime:    time.Now(),
		Metrics:      make(map[string][]models.MetricSeries),
	}

	// Build all queries from config
	queries := p.buildQueries(streamName, consumerName)

	// Execute ALL queries dynamically
	for queryName, queryString := range queries {
		series, err := p.queryRange(queryString, start, end, duration/60)
		if err == nil && len(series) > 0 {
			metricsData.Metrics[queryName] = series
		}
	}

	return metricsData, nil
}

// GetStreamMetrics fetches stream-level metrics
func (p *PrometheusPlugin) GetStreamMetrics(streamName string, timeRange string) (*models.MetricsData, error) {
	if !p.enabled {
		return nil, fmt.Errorf("plugin not enabled")
	}

	// Parse time range
	duration, err := time.ParseDuration(timeRange)
	if err != nil {
		duration = time.Hour
	}

	end := time.Now()
	start := end.Add(-duration)

	metricsData := &models.MetricsData{
		StreamName: streamName,
		FetchTime:  time.Now(),
		Metrics:    make(map[string][]models.MetricSeries),
	}

	// Build all queries from config
	queries := p.buildQueries(streamName, "")

	// Execute ALL queries dynamically (no hardcoding!)
	for queryName, queryString := range queries {
		series, err := p.queryRange(queryString, start, end, duration/60)
		if err == nil && len(series) > 0 {
			metricsData.Metrics[queryName] = series
		}
	}

	return metricsData, nil
}

// queryRange executes a range query
func (p *PrometheusPlugin) queryRange(query string, start, end time.Time, step time.Duration) ([]models.MetricSeries, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, warnings, err := p.queryAPI.QueryRange(ctx, query, v1.Range{
		Start: start,
		End:   end,
		Step:  step,
	})

	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}

	if len(warnings) > 0 {
		// Log warnings (in production, use proper logging)
	}

	// Convert Prometheus result to our MetricSeries
	return p.convertToMetricSeries(result), nil
}

// convertToMetricSeries converts Prometheus results to our format
func (p *PrometheusPlugin) convertToMetricSeries(value model.Value) []models.MetricSeries {
	var series []models.MetricSeries

	if value.Type() != model.ValMatrix {
		return series
	}

	matrix := value.(model.Matrix)

	for _, sampleStream := range matrix {
		// Extract consumer name from labels
		consumerName := string(sampleStream.Metric["consumer_name"])

		points := make([]float64, len(sampleStream.Values))
		times := make([]time.Time, len(sampleStream.Values))

		for i, sample := range sampleStream.Values {
			points[i] = float64(sample.Value)
			times[i] = sample.Timestamp.Time()
		}

		series = append(series, models.MetricSeries{
			Name:   consumerName,
			Points: points,
			Times:  times,
		})
	}

	return series
}

// buildQueries builds all default queries with label filters applied
func (p *PrometheusPlugin) buildQueries(streamName, consumerName string) map[string]string {
	queries := make(map[string]string)

	// Build label filter string
	labelFilters := p.buildLabelFilters()
	if labelFilters != "" {
		labelFilters += ","
	}

	// Default queries - all include label filters from config

	// 1. Consumer Delivered Messages
	queries["consumer_delivered"] = fmt.Sprintf(
		`sum(nats_consumer_delivered_consumer_seq{%sserver_id=~".*",stream_name=~"%s",consumer_name=~".*"}) by (consumer_name)`,
		labelFilters, streamName)

	// 2. Consumer Pending Messages
	queries["consumer_pending"] = fmt.Sprintf(
		`sum(nats_consumer_num_pending{%sserver_id=~".*",stream_name=~"%s",consumer_name=~".*"}) by (consumer_name)`,
		labelFilters, streamName)

	// 3. Message Acks Pending
	queries["consumer_ack_pending"] = fmt.Sprintf(
		`sum(nats_consumer_num_ack_pending{%sserver_id=~".*",stream_name=~"%s",consumer_name=~".*"}) by (consumer_name)`,
		labelFilters, streamName)

	// 4. Stream Size (bytes)
	queries["stream_bytes"] = fmt.Sprintf(
		`sum(nats_stream_total_bytes{%sstream_name=~"%s"}) by (stream_name)`,
		labelFilters, streamName)

	// 5. Stream Message Count
	queries["stream_messages"] = fmt.Sprintf(
		`sum(nats_stream_total_messages{%sstream_name=~"%s"}) by (stream_name)`,
		labelFilters, streamName)

	// 6. Message Rate (per second)
	queries["message_rate"] = fmt.Sprintf(
		`sum(rate(nats_stream_total_messages{%sserver_id=~".*",stream_name=~"%s"}[5m])) by (stream_name)`,
		labelFilters, streamName)

	return queries
}

// buildLabelFilters creates label filter string from config
func (p *PrometheusPlugin) buildLabelFilters() string {
	if len(p.config.Labels) == 0 {
		return ""
	}

	filters := []string{}
	for key, value := range p.config.Labels {
		filters = append(filters, fmt.Sprintf(`%s="%s"`, key, value))
	}

	return strings.Join(filters, ",")
}

// HealthCheck verifies Prometheus is reachable
func (p *PrometheusPlugin) HealthCheck() error {
	if !p.enabled {
		return fmt.Errorf("plugin not enabled")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Simple query to check if Prometheus is alive
	_, _, err := p.queryAPI.Query(ctx, "up", time.Now())
	if err != nil {
		return fmt.Errorf("Prometheus health check failed: %s", err.Error())
	}

	return nil
}

// IsEnabled returns whether the plugin is enabled
func (p *PrometheusPlugin) IsEnabled() bool {
	return p.enabled
}
