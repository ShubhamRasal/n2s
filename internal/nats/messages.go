package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/shubhamrasal/n2s/internal/models"
)

// ListMessages retrieves messages from a stream
// This gets the last N messages (non-destructive read, does NOT acknowledge)
func (c *Client) ListMessages(streamName string, limit int) ([]*models.Message, error) {
	// Create context with 60 second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Channel to receive results
	type result struct {
		messages []*models.Message
		err      error
	}
	resultChan := make(chan result, 1)

	// Fetch messages in goroutine
	go func() {
		messages, err := c.fetchMessages(streamName, limit)
		resultChan <- result{messages: messages, err: err}
	}()

	// Wait for result or timeout
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("timeout after 60 seconds fetching messages")
	case res := <-resultChan:
		return res.messages, res.err
	}
}

// fetchMessages does the actual message fetching
func (c *Client) fetchMessages(streamName string, limit int) ([]*models.Message, error) {
	// Get stream info to know the sequence range
	streamInfo, err := c.js.StreamInfo(streamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream info: %w", err)
	}

	if streamInfo.State.Msgs == 0 {
		return []*models.Message{}, nil
	}

	// Calculate start sequence - get last N messages
	startSeq := streamInfo.State.FirstSeq
	if streamInfo.State.Msgs > uint64(limit) {
		startSeq = streamInfo.State.LastSeq - uint64(limit) + 1
	}

	var messages []*models.Message
	failCount := 0
	maxFails := 10 // Stop after 10 consecutive failures to prevent hanging

	// Fetch messages with failure protection
	for seq := startSeq; seq <= streamInfo.State.LastSeq && len(messages) < limit; seq++ {
		// Prevent infinite loops on streams with many deleted messages
		if failCount >= maxFails {
			break
		}

		msg, err := c.js.GetMsg(streamName, seq)
		if err != nil {
			// Message might be deleted, skip it
			failCount++
			continue
		}

		// Reset fail counter on success
		failCount = 0

		messages = append(messages, &models.Message{
			Sequence:  msg.Sequence,
			Subject:   msg.Subject,
			Data:      msg.Data,
			Headers:   msg.Header,
			Timestamp: msg.Time,
			Size:      len(msg.Data),
		})
	}

	return messages, nil
}

// GetMessageDetail returns detailed information about a message
func (c *Client) GetMessageDetail(streamName string, seq uint64) (*models.MessageDetail, error) {
	msg, err := c.js.GetMsg(streamName, seq)
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	// Try to format payload as JSON if possible
	payload := string(msg.Data)
	var prettyJSON interface{}
	if json.Unmarshal(msg.Data, &prettyJSON) == nil {
		formatted, err := json.MarshalIndent(prettyJSON, "", "  ")
		if err == nil {
			payload = string(formatted)
		}
	}

	return &models.MessageDetail{
		Sequence:  msg.Sequence,
		Subject:   msg.Subject,
		Payload:   payload,
		Headers:   msg.Header,
		Timestamp: msg.Time,
		Size:      len(msg.Data),
	}, nil
}

