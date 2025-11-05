package nats

import (
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/shubhamrasal/n2s/internal/models"
)

// ListConsumers returns a list of all consumers for a stream
func (c *Client) ListConsumers(streamName string) ([]*models.Consumer, error) {
	var consumers []*models.Consumer

	for info := range c.js.ConsumersInfo(streamName) {
		consumer := convertConsumerInfo(info)
		consumers = append(consumers, consumer)
	}

	return consumers, nil
}

// GetConsumerInfo returns detailed information about a consumer
func (c *Client) GetConsumerInfo(streamName, consumerName string) (*models.Consumer, error) {
	info, err := c.js.ConsumerInfo(streamName, consumerName)
	if err != nil {
		return nil, fmt.Errorf("failed to get consumer info: %w", err)
	}

	return convertConsumerInfo(info), nil
}

// DeleteConsumer deletes a consumer from a stream
func (c *Client) DeleteConsumer(streamName, consumerName string) error {
	err := c.js.DeleteConsumer(streamName, consumerName)
	if err != nil {
		return fmt.Errorf("failed to delete consumer: %w", err)
	}
	return nil
}

// UpdateConsumer updates consumer configuration
func (c *Client) UpdateConsumer(streamName, consumerName string, maxDeliver, maxAckPending int, ackWait time.Duration) error {
	// Get current consumer config
	info, err := c.js.ConsumerInfo(streamName, consumerName)
	if err != nil {
		return fmt.Errorf("failed to get current consumer config: %w", err)
	}
	
	// Update the config with new values
	cfg := &info.Config
	cfg.MaxDeliver = maxDeliver
	cfg.MaxAckPending = maxAckPending
	cfg.AckWait = ackWait
	
	// Update the consumer
	_, err = c.js.UpdateConsumer(streamName, cfg)
	if err != nil {
		return fmt.Errorf("failed to update consumer: %w", err)
	}
	
	return nil
}

// convertConsumerInfo converts NATS ConsumerInfo to our models.Consumer
func convertConsumerInfo(info *nats.ConsumerInfo) *models.Consumer {
	deliverPolicy := "all"
	if info.Config.DeliverPolicy == nats.DeliverAllPolicy {
		deliverPolicy = "all"
	} else if info.Config.DeliverPolicy == nats.DeliverLastPolicy {
		deliverPolicy = "last"
	} else if info.Config.DeliverPolicy == nats.DeliverNewPolicy {
		deliverPolicy = "new"
	}

	ackPolicy := "explicit"
	if info.Config.AckPolicy == nats.AckNonePolicy {
		ackPolicy = "none"
	} else if info.Config.AckPolicy == nats.AckAllPolicy {
		ackPolicy = "all"
	}

	replayPolicy := "instant"
	if info.Config.ReplayPolicy == nats.ReplayOriginalPolicy {
		replayPolicy = "original"
	}

	// Handle nil pointers for Last time
	var deliveredLast time.Time
	if info.Delivered.Last != nil {
		deliveredLast = *info.Delivered.Last
	}
	
	var ackFloorLast time.Time
	if info.AckFloor.Last != nil {
		ackFloorLast = *info.AckFloor.Last
	}

	return &models.Consumer{
		Name:           info.Name,
		Stream:         info.Stream,
		NumPending:     info.NumPending,
		NumAckPending:  uint64(info.NumAckPending),
		NumRedelivered: uint64(info.NumRedelivered),
		NumWaiting:     info.NumWaiting,
		Delivered: models.ConsumerSeqInfo{
			Stream:   info.Delivered.Stream,
			Consumer: info.Delivered.Consumer,
			Last:     deliveredLast,
		},
		AckFloor: models.ConsumerSeqInfo{
			Stream:   info.AckFloor.Stream,
			Consumer: info.AckFloor.Consumer,
			Last:     ackFloorLast,
		},
		Config: models.ConsumerConfig{
			Name:          info.Config.Name,
			Durable:       info.Config.Durable,
			FilterSubject: info.Config.FilterSubject,
			DeliverPolicy: deliverPolicy,
			AckPolicy:     ackPolicy,
			AckWait:       info.Config.AckWait,
			MaxDeliver:    info.Config.MaxDeliver,
			ReplayPolicy:  replayPolicy,
			SampleFreq:    "",
			MaxAckPending: info.Config.MaxAckPending,
			FlowControl:   info.Config.FlowControl,
			Heartbeat:     info.Config.Heartbeat,
		},
	}
}

