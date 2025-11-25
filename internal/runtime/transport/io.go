package transport

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
	"github.com/drblury/protoflow/internal/runtime/config"
)

var (
	IOPublisherFactory = func(filePath string, logger watermill.LoggerAdapter) (message.Publisher, error) {
		return &ioPublisher{filePath: filePath, logger: logger}, nil
	}
	IOSubscriberFactory = func(filePath string, logger watermill.LoggerAdapter) (message.Subscriber, error) {
		return &ioSubscriber{filePath: filePath, logger: logger}, nil
	}
)

func ioTransport(conf *config.Config, logger watermill.LoggerAdapter) (Transport, error) {
	filePath := conf.IOFile
	if filePath == "" {
		filePath = "messages.log"
	}

	pub, err := IOPublisherFactory(filePath, logger)
	if err != nil {
		return Transport{}, err
	}
	sub, err := IOSubscriberFactory(filePath, logger)
	if err != nil {
		return Transport{}, err
	}

	return Transport{
		Publisher:  pub,
		Subscriber: sub,
	}, nil
}

type ioPublisher struct {
	filePath string
	logger   watermill.LoggerAdapter
	mu       sync.Mutex
}

type storedMessage struct {
	UUID     string            `json:"uuid"`
	Metadata map[string]string `json:"metadata"`
	Payload  []byte            `json:"payload"`
	Topic    string            `json:"topic"`
}

func (p *ioPublisher) Publish(topic string, messages ...*message.Message) error {
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

func (p *ioPublisher) Close() error {
	return nil
}

type ioSubscriber struct {
	filePath string
	logger   watermill.LoggerAdapter
}

// handleEOF handles end-of-file condition by waiting and seeking back to last position.
// Returns true if the goroutine should continue, false if it should exit.
func (s *ioSubscriber) handleEOF(f *os.File, reader *bufio.Reader, lastPos *int64) bool {
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

// processMessage parses and sends a message if it matches the topic.
// Returns true if processing should continue.
func (s *ioSubscriber) processMessage(ctx context.Context, out chan<- *message.Message, line []byte, topic string) bool {
	var sm storedMessage
	if err := json.Unmarshal(line, &sm); err != nil {
		s.logger.Error("Failed to unmarshal message", err, nil)
		return true // continue processing
	}

	if sm.Topic != topic {
		return true // continue processing
	}

	msg := message.NewMessage(sm.UUID, sm.Payload)
	msg.Metadata = sm.Metadata

	select {
	case out <- msg:
		select {
		case <-msg.Acked():
			// good
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

func (s *ioSubscriber) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
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

func (s *ioSubscriber) Close() error {
	return nil
}
