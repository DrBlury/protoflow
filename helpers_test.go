package protoflow

import (
	"context"
	"sync"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"google.golang.org/protobuf/proto"
)

type testPublisher struct {
	mu        sync.Mutex
	published []string
	err       error
}

func (p *testPublisher) Publish(topic string, messages ...*message.Message) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.err != nil {
		return p.err
	}
	p.published = append(p.published, topic)
	return nil
}

func (p *testPublisher) Close() error { return nil }

func (p *testPublisher) Topics() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	clone := make([]string, len(p.published))
	copy(clone, p.published)
	return clone
}

type testSubscriber struct {
	err error
}

func (s *testSubscriber) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	if s.err != nil {
		return nil, s.err
	}
	ch := make(chan *message.Message)
	close(ch)
	return ch, nil
}

func (s *testSubscriber) Close() error { return nil }

type logEntry struct {
	msg    string
	fields watermill.LogFields
	err    error
}

type testLogger struct {
	entries []logEntry
}

func (l *testLogger) Error(msg string, err error, fields watermill.LogFields) {
	l.entries = append(l.entries, logEntry{msg: msg, err: err, fields: fields})
}

func (l *testLogger) Info(msg string, fields watermill.LogFields) {
	l.entries = append(l.entries, logEntry{msg: msg, fields: fields})
}

func (l *testLogger) Debug(msg string, fields watermill.LogFields) {
	l.entries = append(l.entries, logEntry{msg: msg, fields: fields})
}

func (l *testLogger) Trace(msg string, fields watermill.LogFields) {
	l.entries = append(l.entries, logEntry{msg: msg, fields: fields})
}

func (l *testLogger) With(watermill.LogFields) watermill.LoggerAdapter { return l }

type testValidator struct{ err error }

func (v *testValidator) Validate(proto.Message) error { return v.err }

type testOutbox struct {
	mu      sync.Mutex
	records []outboxRecord
	err     error
}

type outboxRecord struct {
	eventType string
	uuid      string
	payload   string
}

func (o *testOutbox) StoreOutgoingMessage(ctx context.Context, eventType, uuid, payload string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.err != nil {
		return o.err
	}
	o.records = append(o.records, outboxRecord{eventType: eventType, uuid: uuid, payload: payload})
	return nil
}

func (o *testOutbox) Records() []outboxRecord {
	o.mu.Lock()
	defer o.mu.Unlock()
	clone := make([]outboxRecord, len(o.records))
	copy(clone, o.records)
	return clone
}
