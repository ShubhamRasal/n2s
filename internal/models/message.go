package models

import "time"

// Message represents a NATS message
type Message struct {
	Sequence  uint64
	Subject   string
	Data      []byte
	Headers   map[string][]string
	Timestamp time.Time
	Size      int
}

// MessageDetail holds detailed message information for display
type MessageDetail struct {
	Sequence  uint64
	Subject   string
	Payload   string
	Headers   map[string][]string
	Timestamp time.Time
	Size      int
}

