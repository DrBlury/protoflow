// Package io provides a file-based I/O transport for protoflow.
package io

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"os"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"

	"github.com/drblury/protoflow/transport"
)

// TransportName is the name used to register this transport.
const TransportName = "io"

// DefaultFilePath is the default file path if none is specified.
const DefaultFilePath = "messages.log"

// PublisherFactory allows overriding the publisher creation for testing.
var PublisherFactory = func(filePath string, logger watermill.LoggerAdapter) (message.Publisher, error) {
	return &Publisher{filePath: filePath, logger: logger}, nil
}

// SubscriberFactory allows overriding the subscriber creation for testing.
var SubscriberFactory = func(filePath string, logger watermill.LoggerAdapter) (message.Subscriber, error) {
	return &Subscriber{filePath: filePath, logger: logger}, nil
}

// Register registers the I/O transport with the default registry.
// This should be called from an init() function in an importing package,
// or explicitly before using the transport.
func Register() {
	transport.RegisterWithCapabilities(TransportName, Build, transport.IOCapabilities)
}

// Build creates a new I/O transport.
func Build(ctx context.Context, cfg transport.Config, logger watermill.LoggerAdapter) (transport.Transport, error) {
	filePath := cfg.GetIOFile()
	if filePath == "" {
		filePath = DefaultFilePath
	}

	pub, err := PublisherFactory(filePath, logger)
	if err != nil {
		return transport.Transport{}, err
	}

	sub, err := SubscriberFactory(filePath, logger)
	if err != nil {
		return transport.Transport{}, err
	}

	return transport.Transport{
		Publisher:  pub,
		Subscriber: sub,
	}, nil
}

// Capabilities returns the capabilities of this transport.
func Capabilities() transport.Capabilities {
	return transport.IOCapabilities
}

// storedMessage is the JSON structure for persisted messages.
type storedMessage struct {
	UUID     string            `json:"uuid"`
	Metadata map[string]string `json:"metadata"`
	Payload  []byte            `json:"payload"`
	Topic    string            `json:"topic"`
}

// Publisher writes messages to a file.
type Publisher struct {
	filePath string
	logger   watermill.LoggerAdapter
	mu       sync.Mutex
}

// Publish writes messages to the file.
func (p *Publisher) Publish(topic string, messages ...*message.Message) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	f, err := os.OpenFile(p.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, msg := range messages {
		sm := storedMessage{
			UUID:     msg.UUID,
			Metadata: msg.Metadata,
			Payload:  msg.Payload,
			Topic:    topic,
		}

		b, err := json.Marshal(sm)
		if err != nil {
			return err
		}

		if _, err := f.Write(b); err != nil {
			return err
		}
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}
	return nil
}

// Close closes the publisher.
func (p *Publisher) Close() error {
	return nil
}

// Subscriber reads messages from a file.
type Subscriber struct {
	filePath string
	logger   watermill.LoggerAdapter
}

// Subscribe subscribes to messages from the file.
func (s *Subscriber) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	out := make(chan *message.Message)

	go func() {
		defer close(out)

		f, err := os.OpenFile(s.filePath, os.O_RDONLY|os.O_CREATE, 0600)
		if err != nil {
			s.logger.Error("Failed to open file", err, nil)
			return
		}
		defer f.Close()

		var lastPos int64
		reader := bufio.NewReader(f)

		for {
			select {
			case <-ctx.Done():
				return
			default:
				line, err := reader.ReadBytes('\n')
				if err != nil {
					if err == io.EOF {
						if !s.handleEOF(f, reader, &lastPos) {
							return
						}
						continue
					}
					s.logger.Error("Failed to read file", err, nil)
					return
				}

				// Update position after successful read
				currentPos, _ := f.Seek(0, io.SeekCurrent)
				lastPos = currentPos - int64(reader.Buffered())

				if !s.processMessage(ctx, out, line, topic) {
					return
				}
			}
		}
	}()

	return out, nil
}

// Close closes the subscriber.
func (s *Subscriber) Close() error {
	return nil
}

func (s *Subscriber) handleEOF(f *os.File, reader *bufio.Reader, lastPos *int64) bool {
	currentPos, _ := f.Seek(0, io.SeekCurrent)
	currentPos -= int64(reader.Buffered())

	if currentPos > *lastPos {
		*lastPos = currentPos
	}

	time.Sleep(50 * time.Millisecond)

	if _, err := f.Seek(*lastPos, io.SeekStart); err != nil {
		s.logger.Error("Failed to seek file", err, nil)
		return false
	}
	reader.Reset(f)
	return true
}

func (s *Subscriber) processMessage(ctx context.Context, out chan<- *message.Message, line []byte, topic string) bool {
	var sm storedMessage
	if err := json.Unmarshal(line, &sm); err != nil {
		s.logger.Error("Failed to unmarshal message", err, nil)
		return true
	}

	if sm.Topic != topic {
		return true
	}

	msg := message.NewMessage(sm.UUID, sm.Payload)
	msg.Metadata = sm.Metadata

	select {
	case out <- msg:
		select {
		case <-msg.Acked():
		case <-msg.Nacked():
			s.logger.Debug("Message nacked", watermill.LogFields{"uuid": msg.UUID})
		case <-ctx.Done():
			return false
		}
	case <-ctx.Done():
		return false
	}
	return true
}
