package nats

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/shubhamrasal/n2s/internal/models"
)

// ListStreams returns a list of all streams
func (c *Client) ListStreams() ([]*models.Stream, error) {
	var streams []*models.Stream

	for info := range c.js.StreamsInfo() {
		stream := &models.Stream{
			Name:      info.Config.Name,
			Subjects:  info.Config.Subjects,
			Messages:  info.State.Msgs,
			Bytes:     info.State.Bytes,
			Consumers: info.State.Consumers,
			Config: models.StreamConfig{
				Name:        info.Config.Name,
				Subjects:    info.Config.Subjects,
				Retention:   info.Config.Retention.String(),
				Storage:     info.Config.Storage.String(),
				Replicas:    info.Config.Replicas,
				MaxAge:      info.Config.MaxAge,
				MaxMessages: info.Config.MaxMsgs,
				MaxBytes:    info.Config.MaxBytes,
				MaxMsgSize:  info.Config.MaxMsgSize,
				Discard:     info.Config.Discard.String(),
			},
			State: models.StreamState{
				Messages:   info.State.Msgs,
				Bytes:      info.State.Bytes,
				FirstSeq:   info.State.FirstSeq,
				FirstTime:  info.State.FirstTime,
				LastSeq:    info.State.LastSeq,
				LastTime:   info.State.LastTime,
				Consumers:  info.State.Consumers,
				NumDeleted: uint64(info.State.NumDeleted),
			},
		}
		streams = append(streams, stream)
	}

	return streams, nil
}

// GetStreamInfo returns detailed information about a stream
func (c *Client) GetStreamInfo(name string) (*models.Stream, error) {
	info, err := c.js.StreamInfo(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get stream info: %w", err)
	}

	stream := &models.Stream{
		Name:      info.Config.Name,
		Subjects:  info.Config.Subjects,
		Messages:  info.State.Msgs,
		Bytes:     info.State.Bytes,
		Consumers: info.State.Consumers,
		Config: models.StreamConfig{
			Name:        info.Config.Name,
			Subjects:    info.Config.Subjects,
			Retention:   info.Config.Retention.String(),
			Storage:     info.Config.Storage.String(),
			Replicas:    info.Config.Replicas,
			MaxAge:      info.Config.MaxAge,
			MaxMessages: info.Config.MaxMsgs,
			MaxBytes:    info.Config.MaxBytes,
			MaxMsgSize:  info.Config.MaxMsgSize,
			Discard:     info.Config.Discard.String(),
		},
		State: models.StreamState{
			Messages:   info.State.Msgs,
			Bytes:      info.State.Bytes,
			FirstSeq:   info.State.FirstSeq,
			FirstTime:  info.State.FirstTime,
			LastSeq:    info.State.LastSeq,
			LastTime:   info.State.LastTime,
			Consumers:  info.State.Consumers,
			NumDeleted: uint64(info.State.NumDeleted),
		},
	}

	return stream, nil
}

// DeleteStream deletes a stream
func (c *Client) DeleteStream(name string) error {
	err := c.js.DeleteStream(name)
	if err != nil {
		return fmt.Errorf("failed to delete stream: %w", err)
	}
	return nil
}

// PurgeStream purges all messages from a stream
func (c *Client) PurgeStream(name string) error {
	err := c.js.PurgeStream(name)
	if err != nil {
		return fmt.Errorf("failed to purge stream: %w", err)
	}
	return nil
}

// UpdateStream updates stream configuration
func (c *Client) UpdateStream(name string, maxMsgs int64, maxBytes int64, maxAge time.Duration, maxMsgSize int32, retention, discard string) error {
	// Get current stream config
	info, err := c.js.StreamInfo(name)
	if err != nil {
		return fmt.Errorf("failed to get current stream config: %w", err)
	}
	
	// Update the config with new values
	cfg := info.Config
	cfg.MaxMsgs = maxMsgs
	cfg.MaxBytes = maxBytes
	cfg.MaxAge = maxAge
	cfg.MaxMsgSize = maxMsgSize
	
	// Parse retention
	switch retention {
	case "limits":
		cfg.Retention = nats.LimitsPolicy
	case "interest":
		cfg.Retention = nats.InterestPolicy
	case "workqueue":
		cfg.Retention = nats.WorkQueuePolicy
	}
	
	// Parse discard
	switch discard {
	case "old":
		cfg.Discard = nats.DiscardOld
	case "new":
		cfg.Discard = nats.DiscardNew
	}
	
	// Update the stream
	_, err = c.js.UpdateStream(&cfg)
	if err != nil {
		return fmt.Errorf("failed to update stream: %w", err)
	}
	
	return nil
}

// GetMessage retrieves a specific message from a stream by sequence number
func (c *Client) GetMessage(streamName string, seq uint64) (*models.Message, error) {
	msg, err := c.js.GetMsg(streamName, seq)
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	return &models.Message{
		Sequence:  msg.Sequence,
		Subject:   msg.Subject,
		Data:      msg.Data,
		Headers:   msg.Header,
		Timestamp: msg.Time,
		Size:      len(msg.Data),
	}, nil
}
