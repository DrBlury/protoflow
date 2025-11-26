// Package jetstream provides a NATS JetStream transport for protoflow.
package jetstream

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/nats-io/nats.go"

	"github.com/drblury/protoflow/transport"
)

// TransportName is the name used to register this transport.
const TransportName = "nats-jetstream"

const (
	// DefaultMaxDeliver is the default max delivery attempts.
	DefaultMaxDeliver = 3

	// DefaultAckWait is the default ack wait timeout.
	DefaultAckWait = 30 * time.Second

	// MetadataDelay is the metadata key for delay.
	MetadataDelay = "pf_delay_ms"
)

// Register registers the JetStream transport with the default registry.
// This should be called from an init() function in an importing package,
// or explicitly before using the transport.
func Register() {
	transport.RegisterWithCapabilities(TransportName, Build, transport.NATSJetStreamCapabilities)
}

// Build creates a new NATS JetStream transport.
func Build(ctx context.Context, cfg transport.Config, logger watermill.LoggerAdapter) (transport.Transport, error) {
	config := Config{
		URL: cfg.GetNATSURL(),
	}

	t, err := New(config, logger)
	if err != nil {
		return transport.Transport{}, err
	}

	return transport.Transport{
		Publisher:  t,
		Subscriber: t,
	}, nil
}

// Capabilities returns the capabilities of this transport.
func Capabilities() transport.Capabilities {
	return transport.NATSJetStreamCapabilities
}

// Config holds NATS JetStream-specific configuration.
type Config struct {
	// URL is the NATS server URL.
	URL string

	// StreamName is the name of the JetStream stream to use.
	// If empty, defaults to "PROTOFLOW".
	StreamName string

	// MaxDeliver is the maximum number of delivery attempts.
	MaxDeliver int

	// AckWait is the duration to wait for acknowledgment.
	AckWait time.Duration

	// Replicas is the number of stream replicas (for clustering).
	Replicas int

	// RetentionPolicy: "limits" (default), "interest", or "workqueue"
	RetentionPolicy string
}

func (c Config) withDefaults() Config {
	if c.StreamName == "" {
		c.StreamName = "PROTOFLOW"
	}
	if c.MaxDeliver <= 0 {
		c.MaxDeliver = DefaultMaxDeliver
	}
	if c.AckWait <= 0 {
		c.AckWait = DefaultAckWait
	}
	if c.Replicas <= 0 {
		c.Replicas = 1
	}
	return c
}

// Transport implements Publisher and Subscriber for NATS JetStream.
type Transport struct {
	nc     *nats.Conn
	js     nats.JetStreamContext
	config Config
	logger watermill.LoggerAdapter

	subscriptions map[string]*nats.Subscription
	subMu         sync.RWMutex

	closed     bool
	closedMu   sync.RWMutex
	closedChan chan struct{}
}

// New creates a new NATS JetStream transport.
func New(cfg Config, logger watermill.LoggerAdapter) (*Transport, error) {
	cfg = cfg.withDefaults()

	nc, err := nats.Connect(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	t := &Transport{
		nc:            nc,
		js:            js,
		config:        cfg,
		logger:        logger,
		subscriptions: make(map[string]*nats.Subscription),
		closedChan:    make(chan struct{}),
	}

	if err := t.ensureStream(); err != nil {
		nc.Close()
		return nil, fmt.Errorf("failed to ensure stream: %w", err)
	}

	return t, nil
}

func (t *Transport) ensureStream() error {
	streamCfg := &nats.StreamConfig{
		Name:     t.config.StreamName,
		Subjects: []string{t.config.StreamName + ".>"},
		MaxAge:   24 * time.Hour * 7,
		Replicas: t.config.Replicas,
	}

	switch t.config.RetentionPolicy {
	case "interest":
		streamCfg.Retention = nats.InterestPolicy
	case "workqueue":
		streamCfg.Retention = nats.WorkQueuePolicy
	default:
		streamCfg.Retention = nats.LimitsPolicy
	}

	_, err := t.js.AddStream(streamCfg)
	if err != nil {
		_, err = t.js.UpdateStream(streamCfg)
		if err != nil {
			if t.logger != nil {
				t.logger.Info("JetStream stream exists", watermill.LogFields{
					"stream": t.config.StreamName,
				})
			}
		}
	}

	return nil
}

// Publish publishes messages to the JetStream stream.
func (t *Transport) Publish(topic string, messages ...*message.Message) error {
	t.closedMu.RLock()
	if t.closed {
		t.closedMu.RUnlock()
		return fmt.Errorf("transport is closed")
	}
	t.closedMu.RUnlock()

	subject := t.topicToSubject(topic)

	for _, msg := range messages {
		headers := nats.Header{}
		for k, v := range msg.Metadata {
			headers.Set(k, v)
		}

		if delayStr := msg.Metadata.Get(MetadataDelay); delayStr != "" {
			delayMs, err := strconv.ParseInt(delayStr, 10, 64)
			if err == nil && delayMs > 0 {
				headers.Set("pf_publish_time", strconv.FormatInt(time.Now().UnixMilli(), 10))
				headers.Set("pf_delay_until", strconv.FormatInt(time.Now().Add(time.Duration(delayMs)*time.Millisecond).UnixMilli(), 10))
			}
		}

		natsMsg := &nats.Msg{
			Subject: subject,
			Data:    msg.Payload,
			Header:  headers,
		}

		_, err := t.js.PublishMsg(natsMsg)
		if err != nil {
			return fmt.Errorf("failed to publish to JetStream: %w", err)
		}
	}

	return nil
}

// Subscribe subscribes to a topic and returns a channel of messages.
func (t *Transport) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	t.closedMu.RLock()
	if t.closed {
		t.closedMu.RUnlock()
		return nil, fmt.Errorf("transport is closed")
	}
	t.closedMu.RUnlock()

	subject := t.topicToSubject(topic)
	consumerName := t.topicToConsumer(topic)
	output := make(chan *message.Message)

	consumerCfg := &nats.ConsumerConfig{
		Durable:       consumerName,
		FilterSubject: subject,
		AckPolicy:     nats.AckExplicitPolicy,
		MaxDeliver:    t.config.MaxDeliver,
		AckWait:       t.config.AckWait,
		DeliverPolicy: nats.DeliverAllPolicy,
	}

	_, err := t.js.AddConsumer(t.config.StreamName, consumerCfg)
	if err != nil {
		_, err = t.js.UpdateConsumer(t.config.StreamName, consumerCfg)
		if err != nil {
			return nil, fmt.Errorf("failed to create consumer: %w", err)
		}
	}

	sub, err := t.js.PullSubscribe(subject, consumerName)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	t.subMu.Lock()
	t.subscriptions[topic] = sub
	t.subMu.Unlock()

	go t.fetchMessages(ctx, sub, output, topic)

	return output, nil
}

func (t *Transport) fetchMessages(ctx context.Context, sub *nats.Subscription, output chan<- *message.Message, topic string) {
	defer close(output)

	for {
		if t.shouldStop(ctx) {
			return
		}

		msgs, err := sub.Fetch(10, nats.MaxWait(time.Second))
		if err != nil {
			t.handleFetchError(err, topic)
			continue
		}

		for _, natsMsg := range msgs {
			if t.handleDelayedMessage(natsMsg) {
				continue
			}
			if !t.processMessage(ctx, natsMsg, output) {
				return
			}
		}
	}
}

// shouldStop checks if the fetch loop should terminate.
func (t *Transport) shouldStop(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	case <-t.closedChan:
		return true
	default:
		return false
	}
}

// handleFetchError logs fetch errors (except timeout which is normal).
func (t *Transport) handleFetchError(err error, topic string) {
	if err == nats.ErrTimeout {
		return
	}
	if t.logger != nil {
		t.logger.Error("Failed to fetch messages", err, watermill.LogFields{
			"topic": topic,
		})
	}
}

// handleDelayedMessage checks if a message should be delayed and NAKs it if so.
// Returns true if the message was delayed (caller should skip processing).
func (t *Transport) handleDelayedMessage(natsMsg *nats.Msg) bool {
	delayUntilStr := natsMsg.Header.Get("pf_delay_until")
	if delayUntilStr == "" {
		return false
	}
	delayUntil, err := strconv.ParseInt(delayUntilStr, 10, 64)
	if err != nil || time.Now().UnixMilli() >= delayUntil {
		return false
	}
	remainingDelay := time.Duration(delayUntil-time.Now().UnixMilli()) * time.Millisecond
	if err := natsMsg.NakWithDelay(remainingDelay); err != nil && t.logger != nil {
		t.logger.Error("Failed to NAK delayed message", err, nil)
	}
	return true
}

// processMessage converts and sends a NATS message to the output channel.
// Returns false if context was cancelled (caller should return).
func (t *Transport) processMessage(ctx context.Context, natsMsg *nats.Msg, output chan<- *message.Message) bool {
	wmMsg := t.natsToWatermill(natsMsg)

	select {
	case output <- wmMsg:
		return t.waitForAck(ctx, natsMsg, wmMsg)
	case <-ctx.Done():
		return false
	}
}

// waitForAck waits for message acknowledgment and handles it.
// Returns false if context was cancelled.
func (t *Transport) waitForAck(ctx context.Context, natsMsg *nats.Msg, wmMsg *message.Message) bool {
	select {
	case <-wmMsg.Acked():
		if err := natsMsg.Ack(); err != nil && t.logger != nil {
			t.logger.Error("Failed to ack", err, nil)
		}
		return true
	case <-wmMsg.Nacked():
		if err := natsMsg.Nak(); err != nil && t.logger != nil {
			t.logger.Error("Failed to nak", err, nil)
		}
		return true
	case <-ctx.Done():
		return false
	}
}

func (t *Transport) natsToWatermill(natsMsg *nats.Msg) *message.Message {
	var msgID string
	var payload map[string]any
	if err := json.Unmarshal(natsMsg.Data, &payload); err == nil {
		if id, ok := payload["id"].(string); ok {
			msgID = id
		}
	}
	if msgID == "" {
		msgID = natsMsg.Header.Get("ce_id")
	}
	if msgID == "" {
		msgID = fmt.Sprintf("%d", time.Now().UnixNano())
	}

	wmMsg := message.NewMessage(msgID, natsMsg.Data)

	for k, v := range natsMsg.Header {
		if len(v) > 0 {
			wmMsg.Metadata.Set(k, v[0])
		}
	}

	return wmMsg
}

func (t *Transport) topicToSubject(topic string) string {
	return t.config.StreamName + "." + topic
}

func (t *Transport) topicToConsumer(topic string) string {
	return "consumer_" + topic
}

// Close closes the JetStream transport.
func (t *Transport) Close() error {
	t.closedMu.Lock()
	if t.closed {
		t.closedMu.Unlock()
		return nil
	}
	t.closed = true
	close(t.closedChan)
	t.closedMu.Unlock()

	t.subMu.Lock()
	for _, sub := range t.subscriptions {
		_ = sub.Unsubscribe() // Error intentionally ignored during cleanup
	}
	t.subscriptions = make(map[string]*nats.Subscription)
	t.subMu.Unlock()

	t.nc.Close()

	return nil
}

// GetCapabilities returns the JetStream transport capabilities.
func (t *Transport) GetCapabilities() transport.Capabilities {
	return transport.NATSJetStreamCapabilities
}
