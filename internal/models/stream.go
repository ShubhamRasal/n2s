package models

import "time"

// Stream represents a NATS JetStream stream
type Stream struct {
	Name      string
	Subjects  []string
	Messages  uint64
	Bytes     uint64
	Consumers int
	Config    StreamConfig
	State     StreamState
}

// StreamConfig holds stream configuration
type StreamConfig struct {
	Name         string
	Subjects     []string
	Retention    string // limits, interest, workqueue
	Storage      string // file, memory
	Replicas     int
	MaxAge       time.Duration
	MaxMessages  int64
	MaxBytes     int64
	MaxMsgSize   int32
	Discard      string // old, new
}

// StreamState holds stream state information
type StreamState struct {
	Messages     uint64
	Bytes        uint64
	FirstSeq     uint64
	FirstTime    time.Time
	LastSeq      uint64
	LastTime     time.Time
	Consumers    int
	NumDeleted   uint64
}

