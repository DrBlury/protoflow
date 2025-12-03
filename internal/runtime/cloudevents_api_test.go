package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	ce "github.com/drblury/protoflow/internal/runtime/cloudevents"
	configpkg "github.com/drblury/protoflow/internal/runtime/config"
	errspkg "github.com/drblury/protoflow/internal/runtime/errors"
	loggingpkg "github.com/drblury/protoflow/internal/runtime/logging"
	transportpkg "github.com/drblury/protoflow/internal/runtime/transport"
)

// Test PublishOption functions

func TestWithSubject(t *testing.T) {
	opts := &publishOptions{}
	WithSubject("test-subject")(opts)
	require.NotNil(t, opts.subject)
	assert.Equal(t, "test-subject", *opts.subject)
}

func TestWithDataContentType(t *testing.T) {
	opts := &publishOptions{}
	WithDataContentType("application/json")(opts)
	require.NotNil(t, opts.dataContentType)
	assert.Equal(t, "application/json", *opts.dataContentType)
}

func TestWithDataSchema(t *testing.T) {
	opts := &publishOptions{}
	WithDataSchema("https://example.com/schema")(opts)
	require.NotNil(t, opts.dataSchema)
	assert.Equal(t, "https://example.com/schema", *opts.dataSchema)
}

func TestWithExtension(t *testing.T) {
	opts := &publishOptions{}
	WithExtension("key1", "value1")(opts)
	WithExtension("key2", 123)(opts)

	require.NotNil(t, opts.extensions)
	assert.Equal(t, "value1", opts.extensions["key1"])
	assert.Equal(t, 123, opts.extensions["key2"])
}

func TestWithMaxAttempts(t *testing.T) {
	opts := &publishOptions{}
	WithMaxAttempts(5)(opts)
	assert.Equal(t, 5, opts.maxAttempts)
}

func TestWithTracing(t *testing.T) {
	opts := &publishOptions{}
	WithTracing("trace-123", "parent-456")(opts)
	assert.Equal(t, "trace-123", opts.traceID)
	assert.Equal(t, "parent-456", opts.parentID)
}

func TestWithCorrelationID(t *testing.T) {
	opts := &publishOptions{}
	WithCorrelationID("corr-789")(opts)
	assert.Equal(t, "corr-789", opts.correlationID)
}

// Test applyPublishOptions

func TestApplyPublishOptions(t *testing.T) {
	evt := ce.New("test.type", "test.source", map[string]string{"key": "value"})

	opts := []PublishOption{
		WithSubject("subject"),
		WithDataContentType("application/json"),
		WithDataSchema("schema-uri"),
		WithExtension("custom", "ext-value"),
		WithMaxAttempts(3),
		WithTracing("trace-id", "parent-id"),
		WithCorrelationID("corr-id"),
	}

	applyPublishOptions(&evt, opts)

	require.NotNil(t, evt.Subject)
	assert.Equal(t, "subject", *evt.Subject)

	require.NotNil(t, evt.DataContentType)
	assert.Equal(t, "application/json", *evt.DataContentType)

	require.NotNil(t, evt.DataSchema)
	assert.Equal(t, "schema-uri", *evt.DataSchema)

	assert.Equal(t, "ext-value", evt.Extensions["custom"])
}

// Test PublishEvent

func TestPublishEvent_NilService(t *testing.T) {
	var svc *Service
	evt := ce.New("test.type", "test.source", nil)
	err := svc.PublishEvent(context.Background(), evt)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestPublishEvent_NilPublisher(t *testing.T) {
	svc := &Service{}
	evt := ce.New("test.type", "test.source", nil)
	err := svc.PublishEvent(context.Background(), evt)
	assert.Error(t, err)
	assert.ErrorIs(t, err, errspkg.ErrPublisherRequired)
}

func TestPublishEvent_InvalidEvent(t *testing.T) {
	svc := &Service{
		publisher: &testPublisher{},
	}
	// Event without required Type field
	evt := ce.Event{SpecVersion: "1.0", Source: "test"}
	err := svc.PublishEvent(context.Background(), evt)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid CloudEvent")
}

func TestPublishEvent_Success(t *testing.T) {
	pub := &testPublisher{}
	svc := &Service{
		publisher: pub,
	}

	evt := ce.New("test.type", "test.source", map[string]string{"key": "value"})
	err := svc.PublishEvent(context.Background(), evt)
	require.NoError(t, err)

	topics := pub.Topics()
	assert.Equal(t, 1, len(topics))
	assert.Equal(t, "test.type", topics[0])
}

// Test PublishEventAfter

func TestPublishEventAfter_WithDelay(t *testing.T) {
	pub := &testPublisher{}
	svc := &Service{
		publisher: pub,
	}

	evt := ce.New("test.type", "test.source", nil)
	err := svc.PublishEventAfter(context.Background(), evt, 5*time.Second)
	require.NoError(t, err)

	// Verify message was published
	topics := pub.Topics()
	require.Equal(t, 1, len(topics))
}

func TestPublishEventAfter_ZeroDelay(t *testing.T) {
	pub := &testPublisher{}
	svc := &Service{
		publisher: pub,
	}

	evt := ce.New("test.type", "test.source", nil)
	err := svc.PublishEventAfter(context.Background(), evt, 0)
	require.NoError(t, err)
}

// Test PublishData

func TestPublishData_Success(t *testing.T) {
	pub := &testPublisher{}
	svc := &Service{
		publisher: pub,
	}

	err := svc.PublishData(context.Background(), "data.type", "data.source", map[string]int{"count": 42})
	require.NoError(t, err)

	topics := pub.Topics()
	assert.Equal(t, "data.type", topics[0])
	assert.Equal(t, 1, len(topics))
}

func TestPublishData_WithOptions(t *testing.T) {
	pub := &testPublisher{}
	svc := &Service{
		publisher: pub,
	}

	err := svc.PublishData(
		context.Background(),
		"data.type",
		"data.source",
		nil,
		WithSubject("data-subject"),
		WithMaxAttempts(5),
	)
	require.NoError(t, err)
}

// Test PublishDataAfter

func TestPublishDataAfter_Success(t *testing.T) {
	pub := &testPublisher{}
	svc := &Service{
		publisher: pub,
	}

	err := svc.PublishDataAfter(
		context.Background(),
		"delayed.type",
		"delayed.source",
		nil,
		10*time.Second,
	)
	require.NoError(t, err)
}

// Test ConsumeEvents

func TestConsumeEvents_NilService(t *testing.T) {
	var svc *Service
	err := svc.ConsumeEvents("test.type", func(ctx context.Context, evt ce.Event) error { return nil })
	assert.Error(t, err)
	assert.ErrorIs(t, err, errspkg.ErrServiceRequired)
}

func TestConsumeEvents_NilHandler(t *testing.T) {
	svc := newTestService(t)
	err := svc.ConsumeEvents("test.type", nil)
	assert.Error(t, err)
	assert.ErrorIs(t, err, errspkg.ErrHandlerRequired)
}

func TestConsumeEvents_EmptyEventType(t *testing.T) {
	svc := newTestService(t)
	err := svc.ConsumeEvents("", func(ctx context.Context, evt ce.Event) error { return nil })
	assert.Error(t, err)
	assert.ErrorIs(t, err, errspkg.ErrConsumeMessageTypeRequired)
}

func TestConsumeEvents_Success(t *testing.T) {
	svc := newTestService(t)

	err := svc.ConsumeEvents("test.event.type", func(ctx context.Context, evt ce.Event) error {
		return nil
	})
	require.NoError(t, err)

	// Verify handler was registered
	svc.handlersMu.Lock()
	defer svc.handlersMu.Unlock()
	found := false
	for _, h := range svc.handlers {
		if h.Name == "cloudevents-test.event.type" {
			found = true
			break
		}
	}
	assert.True(t, found, "handler should be registered")
}

// Test GetTransportCapabilities

func TestGetTransportCapabilities_NilService(t *testing.T) {
	var svc *Service
	caps := svc.GetTransportCapabilities()
	assert.Equal(t, transportpkg.Capabilities{}, caps)
}

func TestGetTransportCapabilities_NilConf(t *testing.T) {
	svc := &Service{Conf: nil}
	caps := svc.GetTransportCapabilities()
	assert.Equal(t, transportpkg.Capabilities{}, caps)
}

func TestGetTransportCapabilities_WithConf(t *testing.T) {
	svc := &Service{
		Conf: &configpkg.Config{PubSubSystem: "channel"},
	}
	caps := svc.GetTransportCapabilities()
	// Should return whatever is registered for "channel"
	assert.NotNil(t, caps)
}

// Test RegisterCloudEventsHandler

func TestRegisterCloudEventsHandler_NilService(t *testing.T) {
	err := RegisterCloudEventsHandler(nil, CloudEventsHandlerRegistration{
		EventType: "test",
		Handler:   func(ctx context.Context, evt ce.Event) error { return nil },
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, errspkg.ErrServiceRequired)
}

func TestRegisterCloudEventsHandler_NilHandler(t *testing.T) {
	svc := newTestService(t)
	err := RegisterCloudEventsHandler(svc, CloudEventsHandlerRegistration{
		EventType: "test",
		Handler:   nil,
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, errspkg.ErrHandlerRequired)
}

func TestRegisterCloudEventsHandler_EmptyEventType(t *testing.T) {
	svc := newTestService(t)
	err := RegisterCloudEventsHandler(svc, CloudEventsHandlerRegistration{
		EventType: "",
		Handler:   func(ctx context.Context, evt ce.Event) error { return nil },
	})
	assert.Error(t, err)
	assert.ErrorIs(t, err, errspkg.ErrConsumeMessageTypeRequired)
}

func TestRegisterCloudEventsHandler_AutoName(t *testing.T) {
	svc := newTestService(t)
	err := RegisterCloudEventsHandler(svc, CloudEventsHandlerRegistration{
		Name:      "",
		EventType: "auto.name.type",
		Handler:   func(ctx context.Context, evt ce.Event) error { return nil },
	})
	require.NoError(t, err)
}

func TestRegisterCloudEventsHandler_CustomMaxAttempts(t *testing.T) {
	svc := newTestService(t)
	err := RegisterCloudEventsHandler(svc, CloudEventsHandlerRegistration{
		Name:        "custom-handler",
		EventType:   "custom.type",
		Handler:     func(ctx context.Context, evt ce.Event) error { return nil },
		MaxAttempts: 10,
	})
	require.NoError(t, err)
}

// Test EventContext

func TestEventContext_UnmarshalData(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		ec := &EventContext{
			Event: ce.Event{
				Data: map[string]string{"key": "value"},
			},
		}

		var result map[string]string
		err := ec.UnmarshalData(&result)
		require.NoError(t, err)
		assert.Equal(t, "value", result["key"])
	})

	t.Run("nil data", func(t *testing.T) {
		ec := &EventContext{
			Event: ce.Event{Data: nil},
		}

		var result map[string]string
		err := ec.UnmarshalData(&result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no data")
	})

	t.Run("base64 data not supported", func(t *testing.T) {
		base64Data := "dGVzdA=="
		ec := &EventContext{
			Event: ce.Event{
				Data:       "test",
				DataBase64: &base64Data,
			},
		}

		var result string
		err := ec.UnmarshalData(&result)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "base64")
	})
}

func TestEventContext_Publish(t *testing.T) {
	t.Run("nil publisher", func(t *testing.T) {
		ec := &EventContext{
			Publisher: nil,
		}

		err := ec.Publish(context.Background(), "test.type", "test.source", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "publisher not available")
	})

	t.Run("success", func(t *testing.T) {
		pub := &testPublisher{}
		svc := &Service{publisher: pub}
		ec := &EventContext{
			Publisher: svc,
			Event: ce.Event{
				Extensions: map[string]any{
					ce.ExtTraceID:       "trace-123",
					ce.ExtCorrelationID: "corr-456",
				},
			},
		}

		err := ec.Publish(context.Background(), "response.type", "test.source", nil)
		require.NoError(t, err)
		topics := pub.Topics()
		assert.Equal(t, 1, len(topics))
	})
}

// Test NewEventID

func TestNewEventID(t *testing.T) {
	id1 := NewEventID()
	id2 := NewEventID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2)
	assert.Len(t, id1, 26) // ULID length
}

// Test toWatermillMessage

func TestToWatermillMessage(t *testing.T) {
	t.Run("valid event", func(t *testing.T) {
		evt := ce.New("test.type", "test.source", map[string]string{"key": "value"})
		subject := "test-subject"
		evt.Subject = &subject
		evt.Extensions["custom_ext"] = "ext-value"
		evt.Extensions["int_ext"] = 42
		evt.Extensions["bool_ext"] = true

		msg, err := toWatermillMessage(evt)
		require.NoError(t, err)
		assert.Equal(t, evt.ID, msg.UUID)

		// Check metadata
		assert.Equal(t, "1.0", msg.Metadata.Get("ce_specversion"))
		assert.Equal(t, "test.type", msg.Metadata.Get("ce_type"))
		assert.Equal(t, "test.source", msg.Metadata.Get("ce_source"))
		assert.Equal(t, "test-subject", msg.Metadata.Get("ce_subject"))

		// Check payload is valid JSON
		var parsedEvt ce.Event
		err = json.Unmarshal(msg.Payload, &parsedEvt)
		require.NoError(t, err)
		assert.Equal(t, "test.type", parsedEvt.Type)
	})

	t.Run("invalid event", func(t *testing.T) {
		evt := ce.Event{} // Missing required fields
		_, err := toWatermillMessage(evt)
		assert.Error(t, err)
	})

	t.Run("extension types", func(t *testing.T) {
		evt := ce.New("test.type", "test.source", nil)
		evt.Extensions["str"] = "string-value"
		evt.Extensions["int"] = 123
		evt.Extensions["int64"] = int64(456)
		evt.Extensions["float"] = 1.5
		evt.Extensions["bool_true"] = true
		evt.Extensions["bool_false"] = false

		msg, err := toWatermillMessage(evt)
		require.NoError(t, err)

		assert.Equal(t, "string-value", msg.Metadata.Get("str"))
		assert.Equal(t, "123", msg.Metadata.Get("int"))
		assert.Equal(t, "456", msg.Metadata.Get("int64"))
		assert.Equal(t, "1.5", msg.Metadata.Get("float"))
		assert.Equal(t, "true", msg.Metadata.Get("bool_true"))
		assert.Equal(t, "false", msg.Metadata.Get("bool_false"))
	})
}

// Test tryFromWatermillMessage

func TestTryFromWatermillMessage(t *testing.T) {
	t.Run("valid CloudEvent payload", func(t *testing.T) {
		evt := ce.New("original.type", "original.source", map[string]string{"key": "value"})
		payload, _ := json.Marshal(evt)
		msg := message.NewMessage("msg-uuid", payload)

		result := tryFromWatermillMessage(msg)
		assert.Equal(t, "original.type", result.Type)
		assert.Equal(t, "original.source", result.Source)
		assert.Equal(t, evt.ID, result.ID)
	})

	t.Run("non-CloudEvent payload - uses metadata", func(t *testing.T) {
		msg := message.NewMessage("msg-uuid-2", []byte("plain text payload"))
		msg.Metadata.Set("ce_type", "meta.type")
		msg.Metadata.Set("ce_source", "meta.source")
		msg.Metadata.Set("ce_time", time.Now().Format(time.RFC3339))
		msg.Metadata.Set("custom_key", "custom_value")

		result := tryFromWatermillMessage(msg)
		assert.Equal(t, "meta.type", result.Type)
		assert.Equal(t, "meta.source", result.Source)
		assert.Equal(t, "msg-uuid-2", result.ID)
		assert.Equal(t, "custom_value", result.Extensions["custom_key"])
	})

	t.Run("fallback values for missing metadata", func(t *testing.T) {
		msg := message.NewMessage("fallback-uuid", []byte("some data"))

		result := tryFromWatermillMessage(msg)
		assert.Equal(t, "unknown", result.Type)
		assert.Equal(t, "unknown", result.Source)
		assert.Equal(t, "fallback-uuid", result.ID)
	})

	t.Run("legacy protoflow metadata", func(t *testing.T) {
		msg := message.NewMessage("legacy-uuid", []byte("legacy data"))
		msg.Metadata.Set("event_message_schema", "legacy.schema")
		msg.Metadata.Set("event_source", "legacy.source")

		result := tryFromWatermillMessage(msg)
		assert.Equal(t, "legacy.schema", result.Type)
		assert.Equal(t, "legacy.source", result.Source)
	})
}

// Test isCloudEventsMetadata

func TestIsCloudEventsMetadata(t *testing.T) {
	ceKeys := []string{
		"ce_specversion", "ce_type", "ce_source", "ce_id",
		"ce_time", "ce_datacontenttype", "ce_subject", "ce_dataschema",
	}

	for _, key := range ceKeys {
		assert.True(t, isCloudEventsMetadata(key), "expected %s to be CE metadata", key)
	}

	nonCEKeys := []string{
		"custom_key", "pf_attempt", "traceparent", "some_other_key",
	}

	for _, key := range nonCEKeys {
		assert.False(t, isCloudEventsMetadata(key), "expected %s to not be CE metadata", key)
	}
}

// Test handleCloudEventsResult

func TestHandleCloudEventsResult(t *testing.T) {
	t.Run("result ack", func(t *testing.T) {
		svc := newTestService(t)
		evt := ce.New("test.type", "test.source", nil)
		msg := message.NewMessage("test-uuid", nil)

		err := svc.handleCloudEventsResult(context.Background(), "test.type", evt, msg, nil)
		assert.NoError(t, err)
	})

	t.Run("result skip", func(t *testing.T) {
		svc := newTestService(t)
		evt := ce.New("test.type", "test.source", nil)
		msg := message.NewMessage("test-uuid", nil)

		err := svc.handleCloudEventsResult(context.Background(), "test.type", evt, msg, ce.ErrSkip)
		assert.NoError(t, err)
	})

	t.Run("result retry", func(t *testing.T) {
		svc := newTestService(t)
		evt := ce.New("test.type", "test.source", nil)
		msg := message.NewMessage("test-uuid", nil)

		retryErr := ce.ErrRetry
		err := svc.handleCloudEventsResult(context.Background(), "test.type", evt, msg, retryErr)
		assert.Error(t, err)
		assert.Equal(t, retryErr, err)
	})

	t.Run("result dead letter", func(t *testing.T) {
		pub := &testPublisher{}
		svc := &Service{
			publisher: pub,
			Logger:    newTestLogger(),
			Conf:      &configpkg.Config{PubSubSystem: "channel"},
		}

		evt := ce.New("test.type", "test.source", nil)
		msg := message.NewMessage("test-uuid", nil)

		err := svc.handleCloudEventsResult(context.Background(), "test.type", evt, msg, ce.ErrDeadLetter)
		assert.NoError(t, err) // DLQ publish acks original

		// Should have published to DLQ
		topics := pub.Topics()
		assert.Equal(t, 1, len(topics))
	})
}

// Helper to create test service for CloudEvents tests

func newCloudEventsTestService(t *testing.T) *Service {
	t.Helper()

	logger := loggingpkg.NewSlogServiceLogger(newTestSlogLogger())
	deps := ServiceDependencies{
		TransportFactory:          &mockTransportFactory{},
		DisableDefaultMiddlewares: true,
	}
	return NewService(&configpkg.Config{PubSubSystem: "channel"}, logger, context.Background(), deps)
}

// Test publisher that records calls for CloudEvents tests

type ceTestPublisher struct {
	topic    string
	messages []*message.Message
	err      error
}

func (p *ceTestPublisher) Publish(topic string, messages ...*message.Message) error {
	if p.err != nil {
		return p.err
	}
	p.topic = topic
	p.messages = append(p.messages, messages...)
	return nil
}

func (p *ceTestPublisher) Close() error { return nil }

// Test subscriber for CloudEvents tests

type ceTestSubscriber struct{}

func (s *ceTestSubscriber) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	ch := make(chan *message.Message)
	return ch, nil
}

func (s *ceTestSubscriber) Close() error { return nil }

// Test sendToCloudEventsDLQ

func TestSendToCloudEventsDLQ(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		pub := &testPublisher{}
		svc := &Service{
			publisher: pub,
			Logger:    newTestLogger(),
		}

		evt := ce.New("test.type", "test.source", nil)
		err := svc.sendToCloudEventsDLQ(context.Background(), "test.type", evt, errors.New("test error"))

		assert.NoError(t, err)
		topics := pub.Topics()
		assert.Equal(t, 1, len(topics))
	})

	t.Run("publish failure", func(t *testing.T) {
		pub := &testPublisher{err: errors.New("publish failed")}
		svc := &Service{
			publisher: pub,
			Logger:    newTestLogger(),
		}

		evt := ce.New("test.type", "test.source", nil)
		err := svc.sendToCloudEventsDLQ(context.Background(), "test.type", evt, errors.New("test error"))

		// Should return nil to ack original and avoid infinite loops
		assert.NoError(t, err)
	})
}
