package models

import "time"

// Consumer represents a NATS JetStream consumer
type Consumer struct {
	Name         string
	Stream       string
	Config       ConsumerConfig
	Delivered    ConsumerSeqInfo
	AckFloor     ConsumerSeqInfo
	NumPending   uint64
	NumAckPending uint64
	NumRedelivered uint64
	NumWaiting   int
	LastActivity time.Time
}

// ConsumerConfig holds consumer configuration
type ConsumerConfig struct {
	Name           string
	Durable        string
	FilterSubject  string
	DeliverPolicy  string // all, last, new, by_start_sequence, by_start_time
	AckPolicy      string // none, all, explicit
	AckWait        time.Duration
	MaxDeliver     int
	ReplayPolicy   string // instant, original
	SampleFreq     string
	MaxAckPending  int
	FlowControl    bool
	Heartbeat      time.Duration
}

// ConsumerSeqInfo holds sequence information
type ConsumerSeqInfo struct {
	Stream   uint64
	Consumer uint64
	Last     time.Time
}

// PendingMessage represents a pending message in a consumer
type PendingMessage struct {
	Sequence    uint64
	Subject     string
	Redelivered int
	Timestamp   time.Time
}

