package nats

import (
	"context"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/shubhamrasal/n2s/internal/config"
)

// Client wraps NATS connection and JetStream context
type Client struct {
	conn *nats.Conn
	js   nats.JetStreamContext
}

// NewClient creates a new NATS client with JetStream enabled
func NewClient(ctx *config.Context) (*Client, error) {
	// Build connection options
	opts := []nats.Option{
		nats.Timeout(10 * time.Second),
		nats.MaxReconnects(5),
		nats.ReconnectWait(2 * time.Second),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				// Log disconnect error (in production, use proper logging)
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			// Log reconnect (in production, use proper logging)
		}),
	}

	// Add token authentication if provided
	if ctx.Token != "" {
		opts = append(opts, nats.Token(ctx.Token))
	}

	// Add credentials file if provided
	if ctx.Creds != "" {
		opts = append(opts, nats.UserCredentials(ctx.Creds))
	}

	// Connect with options
	nc, err := nats.Connect(ctx.Server, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	return &Client{
		conn: nc,
		js:   js,
	}, nil
}

// Close closes the NATS connection
func (c *Client) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// IsConnected returns true if the client is connected to NATS
func (c *Client) IsConnected() bool {
	return c.conn != nil && c.conn.IsConnected()
}

// Stats returns connection statistics
func (c *Client) Stats() nats.Statistics {
	if c.conn != nil {
		return c.conn.Stats()
	}
	return nats.Statistics{}
}

// ServerInfo returns NATS server information
func (c *Client) ServerInfo() (string, error) {
	if c.conn == nil {
		return "", fmt.Errorf("not connected")
	}

	servers := c.conn.Servers()
	if len(servers) > 0 {
		return servers[0], nil
	}

	return "unknown", nil
}

// Ping checks if the connection is alive
func (c *Client) Ping(ctx context.Context) error {
	if c.conn == nil {
		return fmt.Errorf("not connected")
	}

	// Create a channel for the result
	done := make(chan error, 1)

	go func() {
		err := c.conn.FlushTimeout(2 * time.Second)
		done <- err
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}
