package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type DemoMessage struct {
	ID        string    `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"`
	Source    string    `json:"source"`
	Data      string    `json:"data"`
	Priority  int       `json:"priority"`
}

// Prometheus metrics
var (
	consumerDelivered = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nats_consumer_delivered_consumer_seq",
			Help: "Number of messages delivered to consumer",
		},
		[]string{"server_id", "stream_name", "consumer_name"},
	)

	consumerPending = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nats_consumer_num_pending",
			Help: "Number of pending messages for consumer",
		},
		[]string{"server_id", "stream_name", "consumer_name"},
	)

	consumerAckPending = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nats_consumer_num_ack_pending",
			Help: "Number of ack pending messages for consumer",
		},
		[]string{"server_id", "stream_name", "consumer_name"},
	)

	streamTotalBytes = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nats_stream_total_bytes",
			Help: "Total bytes stored in stream",
		},
		[]string{"server_id", "stream_name"},
	)

	streamTotalMessages = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nats_stream_total_messages",
			Help: "Total number of messages in stream",
		},
		[]string{"server_id", "stream_name"},
	)
)

// MetricsSimulator holds state for metrics simulation
type MetricsSimulator struct {
	streams       []string
	consumers     map[string]string // stream -> consumer name
	serverID      string
	mu            sync.RWMutex
	stopChan      chan struct{}
	messageRate   map[string]float64 // stream -> messages per second
	currentMsgs   map[string]float64 // stream -> current message count
	currentBytes  map[string]float64 // stream -> current bytes
	consumerState map[string]*ConsumerState
	history       map[string][]DataPoint // historical data for time-series queries
}

type DataPoint struct {
	Timestamp time.Time
	Value     float64
}

type ConsumerState struct {
	Delivered  float64
	Pending    float64
	AckPending float64
}

func main() {
	// Parse flags
	metricsMode := flag.Bool("metrics", false, "Run in metrics simulation mode (serves Prometheus metrics on :9090)")
	metricsPort := flag.String("metrics-port", "9090", "Port to serve Prometheus metrics on")
	flag.Parse()

	// Connect to NATS
	nc, err := nats.Connect("localhost:4222")
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	// Create JetStream context
	js, err := nc.JetStream()
	if err != nil {
		log.Fatalf("Failed to create JetStream context: %v", err)
	}

	log.Println("Connected to NATS successfully!")

	// Define demo streams with their configurations
	streams := []struct {
		name        string
		subject     string
		description string
		msgCount    int
	}{
		{"user-events", "events.user", "User activity events", 150},
		{"order-processing", "orders.process", "Order processing stream", 75},
		{"payment-transactions", "payments.tx", "Payment transaction events", 200},
		{"notification-service", "notifications.send", "Notification delivery service", 50},
		{"audit-logs", "audit.logs", "System audit logging", 300},
		{"metrics-collector", "metrics.data", "Application metrics collection", 500},
		{"webhook-events", "webhooks.incoming", "Incoming webhook events", 25},
		{"email-queue", "email.outbound", "Outbound email queue", 100},
		{"analytics-events", "analytics.track", "Analytics tracking events", 175},
	}

	for _, streamConfig := range streams {
		log.Printf("Creating stream: %s", streamConfig.name)

		// Create stream
		_, err := js.AddStream(&nats.StreamConfig{
			Name:        streamConfig.name,
			Description: streamConfig.description,
			Subjects:    []string{streamConfig.subject},
			Retention:   nats.LimitsPolicy,
			MaxAge:      24 * time.Hour,
			Storage:     nats.FileStorage,
			Replicas:    1,
		})
		if err != nil {
			// Stream might already exist
			log.Printf("Stream %s might already exist: %v", streamConfig.name, err)
		}

		// Create consumer for the stream
		consumerName := fmt.Sprintf("%s-consumer", streamConfig.name)
		log.Printf("Creating consumer: %s", consumerName)

		_, err = js.AddConsumer(streamConfig.name, &nats.ConsumerConfig{
			Durable:       consumerName,
			AckPolicy:     nats.AckExplicitPolicy,
			DeliverPolicy: nats.DeliverAllPolicy,
			MaxDeliver:    3,
			AckWait:       30 * time.Second,
		})
		if err != nil {
			log.Printf("Consumer %s might already exist: %v", consumerName, err)
		}

		// Publish messages to the stream
		log.Printf("Publishing %d messages to stream: %s", streamConfig.msgCount, streamConfig.name)
		publishMessages(js, streamConfig.subject, streamConfig.name, streamConfig.msgCount)
	}

	// Create some control plane streams (like in the screenshot)
	log.Println("\nCreating control plane streams...")
	controlStreams := createControlPlaneStreams(js, nc)

	log.Println("\n✅ Demo data population completed successfully!")

	// If metrics mode is enabled, start the metrics simulator
	if *metricsMode {
		log.Println("\n🔥 Starting metrics simulation mode...")
		log.Printf("Prometheus metrics will be available at http://localhost:%s/metrics\n", *metricsPort)
		log.Println("Press Ctrl+C to stop")

		// Collect all stream names
		allStreams := make([]string, 0)
		streamConsumers := make(map[string]string)

		for _, s := range streams {
			allStreams = append(allStreams, s.name)
			streamConsumers[s.name] = fmt.Sprintf("%s-consumer", s.name)
		}

		for _, s := range controlStreams {
			allStreams = append(allStreams, s.name)
			streamConsumers[s.name] = fmt.Sprintf("[%s]", s.name)
		}

		// Start metrics simulator
		sim := NewMetricsSimulator(allStreams, streamConsumers)
		sim.Start(*metricsPort)
	} else {
		log.Println("You can now use n9s to view and capture screenshots of the populated data.")
	}
}

func publishMessages(js nats.JetStreamContext, subject, source string, count int) {
	messageTypes := []string{"info", "warning", "error", "debug", "trace"}

	for i := 0; i < count; i++ {
		msg := DemoMessage{
			ID:        fmt.Sprintf("%s-%d", source, i+1),
			Timestamp: time.Now().Add(-time.Duration(rand.Intn(3600)) * time.Second),
			Type:      messageTypes[rand.Intn(len(messageTypes))],
			Source:    source,
			Data:      generateRandomData(),
			Priority:  rand.Intn(10),
		}

		msgBytes, err := json.Marshal(msg)
		if err != nil {
			log.Printf("Failed to marshal message: %v", err)
			continue
		}

		_, err = js.Publish(subject, msgBytes)
		if err != nil {
			log.Printf("Failed to publish message: %v", err)
		}

		// Add some delay to make it more realistic
		if i%100 == 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}
}

func createControlPlaneStreams(js nats.JetStreamContext, nc *nats.Conn) []struct {
	name     string
	subject  string
	msgCount int
} {
	controlStreams := []struct {
		name     string
		subject  string
		msgCount int
	}{
		{"control-d1de5rtkl4as7395dmkq", "control.d1de5rtkl4as7395dmkq", 0},
		{"control-d1f94i8844nc738ghdn0", "control.d1f94i8844nc738ghdn0", 5},
		{"control-d1ill5dhinsc73a79n80", "control.d1ill5dhinsc73a79n80", 0},
		{"control-d2csju8chooc73eg93n0", "control.d2csju8chooc73eg93n0", 0},
		{"control-d2cspn8chooc73eg93og", "control.d2cspn8chooc73eg93og", 0},
		{"control-d2cspt0chooc73eg93g0", "control.d2cspt0chooc73eg93g0", 0},
		{"control-d41rn928bvnc73fbmr40", "control.d41rn928bvnc73fbmr40", 0},
		{"control-d449q2vqkkqc7398ck50", "control.d449q2vqkkqc7398ck50", 230},
		{"control-d449q4nqkkqc7398ck60", "control.d449q4nqkkqc7398ck60", 5},
		{"control-d449q4pmm7pc7380fapg", "control.d449q4pmm7pc7380fapg", 3},
		{"control-d449q5pmm7pc7380farg", "control.d449q5pmm7pc7380farg", 1},
		{"control-d449q77qkkqc7398ck90", "control.d449q77qkkqc7398ck90", 2},
		{"control-d449q7pmm7pc7380fasg", "control.d449q7pmm7pc7380fasg", 2},
		{"control-d44c1l9ql8t9grejfqf0", "control.d44c1l9ql8t9grejfqf0", 57},
		{"control-d44c5mhql8tagh7gvueg", "control.d44c5mhql8tagh7gvueg", 57},
		{"control-d44c8npql8tqr2b19e0", "control.d44c8npql8tqr2b19e0", 56},
		{"control-d44ce51ql8tbh5lqp2j0", "control.d44ce51ql8tbh5lqp2j0", 56},
		{"control-d44cek1ql8tbh5lqp2k0", "control.d44cek1ql8tbh5lqp2k0", 55},
		{"control-d44cgpql8tbp7q0naqg", "control.d44cgpql8tbp7q0naqg", 55},
		{"control-d453ej1ql8t60n7qe7ug", "control.d453ej1ql8t60n7qe7ug", 2},
		{"control-d453qbhql8t60n7qe7vg", "control.d453qbhql8t60n7qe7vg", 2},
	}

	for _, streamConfig := range controlStreams {
		log.Printf("Creating control stream: %s", streamConfig.name)

		_, err := js.AddStream(&nats.StreamConfig{
			Name:      streamConfig.name,
			Subjects:  []string{streamConfig.subject},
			Retention: nats.LimitsPolicy,
			MaxAge:    24 * time.Hour,
			Storage:   nats.FileStorage,
			Replicas:  1,
		})
		if err != nil {
			log.Printf("Control stream %s might already exist: %v", streamConfig.name, err)
		}

		// Create consumer with matching name pattern from screenshot
		consumerName := fmt.Sprintf("[%s]", streamConfig.name)
		_, err = js.AddConsumer(streamConfig.name, &nats.ConsumerConfig{
			Durable:       consumerName,
			AckPolicy:     nats.AckExplicitPolicy,
			DeliverPolicy: nats.DeliverAllPolicy,
		})
		if err != nil {
			log.Printf("Consumer %s might already exist: %v", consumerName, err)
		}

		// Publish messages if needed
		if streamConfig.msgCount > 0 {
			publishMessages(js, streamConfig.subject, streamConfig.name, streamConfig.msgCount)
		}
	}

	return controlStreams
}

func generateRandomData() string {
	dataTemplates := []string{
		"User action completed successfully",
		"Processing request from API gateway",
		"Database query executed in %dms",
		"Cache hit for key: %s",
		"External service called: response time %dms",
		"Validation error: field %s is required",
		"Authentication successful for user",
		"Rate limit check passed",
		"Message queued for processing",
		"Background job scheduled",
	}

	template := dataTemplates[rand.Intn(len(dataTemplates))]

	switch template {
	case "Database query executed in %dms":
		return fmt.Sprintf(template, rand.Intn(500))
	case "Cache hit for key: %s":
		return fmt.Sprintf(template, fmt.Sprintf("key_%d", rand.Intn(1000)))
	case "External service called: response time %dms":
		return fmt.Sprintf(template, rand.Intn(2000))
	case "Validation error: field %s is required":
		fields := []string{"email", "username", "password", "firstName", "lastName"}
		return fmt.Sprintf(template, fields[rand.Intn(len(fields))])
	default:
		return template
	}
}

// NewMetricsSimulator creates a new metrics simulator
func NewMetricsSimulator(streams []string, consumers map[string]string) *MetricsSimulator {
	// Register Prometheus collectors
	prometheus.MustRegister(consumerDelivered)
	prometheus.MustRegister(consumerPending)
	prometheus.MustRegister(consumerAckPending)
	prometheus.MustRegister(streamTotalBytes)
	prometheus.MustRegister(streamTotalMessages)

	sim := &MetricsSimulator{
		streams:       streams,
		consumers:     consumers,
		serverID:      "nats-server-demo",
		stopChan:      make(chan struct{}),
		messageRate:   make(map[string]float64),
		currentMsgs:   make(map[string]float64),
		currentBytes:  make(map[string]float64),
		consumerState: make(map[string]*ConsumerState),
		history:       make(map[string][]DataPoint),
	}

	// Initialize with starting values
	for _, stream := range streams {
		// Random initial message rate (1-50 msgs/sec)
		sim.messageRate[stream] = float64(rand.Intn(50) + 1)

		// Initial message count (1000-10000)
		sim.currentMsgs[stream] = float64(rand.Intn(9000) + 1000)

		// Initial bytes (500 bytes average per message)
		sim.currentBytes[stream] = sim.currentMsgs[stream] * 500

		// Initialize consumer state
		consumerName := consumers[stream]
		sim.consumerState[consumerName] = &ConsumerState{
			Delivered:  sim.currentMsgs[stream],
			Pending:    float64(rand.Intn(100)),
			AckPending: float64(rand.Intn(50)),
		}
	}

	return sim
}

// Start begins the metrics simulation
func (s *MetricsSimulator) Start(port string) {
	// Register Prometheus export endpoint (for real Prometheus scraping)
	http.Handle("/metrics", promhttp.Handler())

	// Register Prometheus Query API endpoints (for n2s to query)
	http.HandleFunc("/api/v1/query_range", s.handleQueryRange)
	http.HandleFunc("/api/v1/query", s.handleQuery)

	go func() {
		log.Printf("Starting Prometheus metrics server on :%s\n", port)
		log.Printf("  Metrics export: http://localhost:%s/metrics\n", port)
		log.Printf("  Query API: http://localhost:%s/api/v1/query_range\n", port)
		if err := http.ListenAndServe(":"+port, nil); err != nil {
			log.Fatalf("Failed to start metrics server: %v", err)
		}
	}()

	// Start metrics update goroutine
	go s.updateMetrics()

	// Wait for stop signal
	<-s.stopChan
}

// updateMetrics continuously updates metrics to simulate activity
func (s *MetricsSimulator) updateMetrics() {
	ticker := time.NewTicker(5 * time.Second) // Update every 5 seconds
	defer ticker.Stop()

	log.Println("📊 Metrics simulation started. Updating every 5 seconds...")

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			s.simulateActivity()
			s.publishMetrics()
			s.mu.Unlock()

		case <-s.stopChan:
			return
		}
	}
}

// simulateActivity simulates realistic stream and consumer activity
func (s *MetricsSimulator) simulateActivity() {
	for _, stream := range s.streams {
		// Randomly vary message rate (±20%)
		variation := 1.0 + (rand.Float64()-0.5)*0.4
		currentRate := s.messageRate[stream] * variation

		// Add messages based on rate (5 second interval)
		newMessages := currentRate * 5.0
		s.currentMsgs[stream] += newMessages
		s.currentBytes[stream] += newMessages * 500 // 500 bytes per message average

		// Simulate consumer activity
		consumerName := s.consumers[stream]
		state := s.consumerState[consumerName]

		// Consumer delivers messages
		deliveryRate := currentRate * 0.9 * 5.0 // 90% of incoming rate
		state.Delivered += deliveryRate

		// Pending messages fluctuate
		state.Pending = state.Pending + newMessages - deliveryRate
		if state.Pending < 0 {
			state.Pending = 0
		}
		if state.Pending > 500 {
			state.Pending = 500 // Cap at 500
		}

		// Ack pending fluctuates (usually less than pending)
		state.AckPending = state.Pending * (0.3 + rand.Float64()*0.3) // 30-60% of pending

		// Occasionally spike the message rate to simulate bursts
		if rand.Float64() < 0.1 { // 10% chance
			s.messageRate[stream] = s.messageRate[stream] * (1.5 + rand.Float64())
		}

		// Occasionally drop the rate to simulate quiet periods
		if rand.Float64() < 0.05 { // 5% chance
			s.messageRate[stream] = s.messageRate[stream] * 0.5
		}

		// Keep rate in reasonable bounds
		if s.messageRate[stream] < 1 {
			s.messageRate[stream] = 1
		}
		if s.messageRate[stream] > 100 {
			s.messageRate[stream] = 100
		}
	}
}

// publishMetrics updates Prometheus metrics
func (s *MetricsSimulator) publishMetrics() {
	for _, stream := range s.streams {
		consumerName := s.consumers[stream]
		state := s.consumerState[consumerName]

		// Update stream metrics
		streamTotalMessages.WithLabelValues(s.serverID, stream).Set(s.currentMsgs[stream])
		streamTotalBytes.WithLabelValues(s.serverID, stream).Set(s.currentBytes[stream])

		// Update consumer metrics
		consumerDelivered.WithLabelValues(s.serverID, stream, consumerName).Set(state.Delivered)
		consumerPending.WithLabelValues(s.serverID, stream, consumerName).Set(state.Pending)
		consumerAckPending.WithLabelValues(s.serverID, stream, consumerName).Set(state.AckPending)
	}
}

// Stop stops the metrics simulation
func (s *MetricsSimulator) Stop() {
	close(s.stopChan)
}

// handleQueryRange handles Prometheus range query API
func (s *MetricsSimulator) handleQueryRange(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("query")

	// Parse query to determine what metric to return
	// Simple pattern matching for demo purposes
	var metricName string
	var streamName string
	var consumerName string

	if strings.Contains(query, "nats_consumer_delivered_consumer_seq") {
		metricName = "consumer_delivered"
		// Extract stream_name from query
		streamName = extractLabel(query, "stream_name")
		consumerName = extractLabel(query, "consumer_name")
	} else if strings.Contains(query, "nats_consumer_num_pending") {
		metricName = "consumer_pending"
		streamName = extractLabel(query, "stream_name")
	} else if strings.Contains(query, "nats_consumer_num_ack_pending") {
		metricName = "consumer_ack_pending"
		streamName = extractLabel(query, "stream_name")
	} else if strings.Contains(query, "nats_stream_total_bytes") {
		metricName = "stream_bytes"
		streamName = extractLabel(query, "stream_name")
	} else if strings.Contains(query, "nats_stream_total_messages") && strings.Contains(query, "rate") {
		metricName = "message_rate"
		streamName = extractLabel(query, "stream_name")
	} else if strings.Contains(query, "nats_stream_total_messages") {
		metricName = "stream_messages"
		streamName = extractLabel(query, "stream_name")
	}

	// Generate simulated time-series data
	response := s.generateQueryRangeResponse(metricName, streamName, consumerName)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleQuery handles instant query API
func (s *MetricsSimulator) handleQuery(w http.ResponseWriter, r *http.Request) {
	// Simple response for health checks
	response := map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"resultType": "vector",
			"result":     []interface{}{},
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// generateQueryRangeResponse creates fake Prometheus response
func (s *MetricsSimulator) generateQueryRangeResponse(metricName, streamName, consumerName string) map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now()
	step := 30 * time.Second // 30 second intervals
	points := 60             // Last 30 minutes

	result := []map[string]interface{}{}

	// Generate data based on metric type
	switch metricName {
	case "consumer_delivered", "consumer_pending", "consumer_ack_pending":
		// Get all consumers for matching streams
		for stream, consumer := range s.consumers {
			if streamName != "" && !strings.Contains(stream, streamName) {
				continue
			}

			state := s.consumerState[consumer]
			if state == nil {
				continue
			}

			// Generate time series
			values := make([][]interface{}, points)
			baseValue := state.Delivered
			if metricName == "consumer_pending" {
				baseValue = state.Pending
			} else if metricName == "consumer_ack_pending" {
				baseValue = state.AckPending
			}

			for i := 0; i < points; i++ {
				timestamp := now.Add(-time.Duration(points-i) * step).Unix()
				// Simulate gradual increase
				value := baseValue * float64(i) / float64(points)
				// Add some variation
				value += (rand.Float64() - 0.5) * baseValue * 0.1

				values[i] = []interface{}{timestamp, fmt.Sprintf("%.2f", value)}
			}

			result = append(result, map[string]interface{}{
				"metric": map[string]string{
					"consumer_name": consumer,
					"stream_name":   stream,
					"server_id":     s.serverID,
				},
				"values": values,
			})
		}

	case "stream_bytes", "stream_messages", "message_rate":
		// Stream-level metrics
		for stream := range s.currentMsgs {
			if streamName != "" && !strings.Contains(stream, streamName) {
				continue
			}

			values := make([][]interface{}, points)
			baseValue := s.currentMsgs[stream]
			if metricName == "stream_bytes" {
				baseValue = s.currentBytes[stream]
			} else if metricName == "message_rate" {
				baseValue = s.messageRate[stream]
			}

			for i := 0; i < points; i++ {
				timestamp := now.Add(-time.Duration(points-i) * step).Unix()
				value := baseValue * float64(i) / float64(points)
				value += (rand.Float64() - 0.5) * baseValue * 0.1

				values[i] = []interface{}{timestamp, fmt.Sprintf("%.2f", value)}
			}

			result = append(result, map[string]interface{}{
				"metric": map[string]string{
					"stream_name": stream,
					"server_id":   s.serverID,
				},
				"values": values,
			})
		}
	}

	return map[string]interface{}{
		"status": "success",
		"data": map[string]interface{}{
			"resultType": "matrix",
			"result":     result,
		},
	}
}

// extractLabel extracts a label value from a query string
func extractLabel(query, label string) string {
	// Simple extraction: stream_name=~"xyz" or stream_name="xyz"
	pattern := label + `=~"`
	idx := strings.Index(query, pattern)
	if idx == -1 {
		pattern = label + `="`
		idx = strings.Index(query, pattern)
	}
	if idx == -1 {
		return ""
	}

	start := idx + len(pattern)
	end := strings.Index(query[start:], `"`)
	if end == -1 {
		return ""
	}

	return query[start : start+end]
}
