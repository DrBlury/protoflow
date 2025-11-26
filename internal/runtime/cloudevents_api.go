package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"

	ce "github.com/drblury/protoflow/internal/runtime/cloudevents"
	errspkg "github.com/drblury/protoflow/internal/runtime/errors"
	idspkg "github.com/drblury/protoflow/internal/runtime/ids"
	transportpkg "github.com/drblury/protoflow/internal/runtime/transport"
)

// EventHandler is the callback signature for CloudEvents handlers.
// Return nil to acknowledge, or an error to control retry/DLQ behavior:
//   - nil: acknowledge the message
//   - ErrRetry: retry with default delay
//   - ErrRetryAfter(d): retry after specific delay
//   - ErrDeadLetter: send to dead letter queue
//   - any other error: treated as ErrRetry
type EventHandler func(ctx context.Context, evt ce.Event) error

// PublishOption configures event publishing behavior.
type PublishOption func(*publishOptions)

type publishOptions struct {
	subject         *string
	dataContentType *string
	dataSchema      *string
	extensions      map[string]any
	maxAttempts     int
	traceID         string
	parentID        string
	correlationID   string
}

// WithSubject sets the CloudEvents subject attribute.
func WithSubject(subject string) PublishOption {
	return func(o *publishOptions) {
		o.subject = &subject
	}
}

// WithDataContentType sets the data content type (e.g., "application/json").
func WithDataContentType(contentType string) PublishOption {
	return func(o *publishOptions) {
		o.dataContentType = &contentType
	}
}

// WithDataSchema sets the data schema URI.
func WithDataSchema(schema string) PublishOption {
	return func(o *publishOptions) {
		o.dataSchema = &schema
	}
}

// WithExtension adds a CloudEvents extension attribute.
func WithExtension(key string, value any) PublishOption {
	return func(o *publishOptions) {
		if o.extensions == nil {
			o.extensions = make(map[string]any)
		}
		o.extensions[key] = value
	}
}

// WithMaxAttempts sets the maximum retry attempts for the event.
func WithMaxAttempts(max int) PublishOption {
	return func(o *publishOptions) {
		o.maxAttempts = max
	}
}

// WithTracing sets tracing context for the event.
func WithTracing(traceID, parentID string) PublishOption {
	return func(o *publishOptions) {
		o.traceID = traceID
		o.parentID = parentID
	}
}

// WithCorrelationID sets the correlation ID for request tracing.
func WithCorrelationID(correlationID string) PublishOption {
	return func(o *publishOptions) {
		o.correlationID = correlationID
	}
}

// PublishEvent publishes a CloudEvent to the specified event type topic.
func (s *Service) PublishEvent(ctx context.Context, evt ce.Event) error {
	if s == nil {
		return errors.New("event service is nil")
	}
	if s.publisher == nil {
		return errspkg.ErrPublisherRequired
	}

	// Validate the event
	if err := evt.Validate(); err != nil {
		return fmt.Errorf("invalid CloudEvent: %w", err)
	}

	// Convert to Watermill message (internal implementation detail)
	msg, err := toWatermillMessage(evt)
	if err != nil {
		return err
	}

	if ctx != nil {
		msg.SetContext(ctx)
	}

	// Use event type as topic
	topic := evt.Type
	return s.publisher.Publish(topic, msg)
}

// PublishEventAfter publishes a CloudEvent with a delay before processing.
func (s *Service) PublishEventAfter(ctx context.Context, evt ce.Event, delay time.Duration) error {
	if delay > 0 {
		ce.SetDelay(&evt, delay)
	}
	return s.PublishEvent(ctx, evt)
}

// PublishData publishes data as a CloudEvent.
// This is a convenience method that constructs the event for you.
func (s *Service) PublishData(ctx context.Context, eventType, source string, data any, opts ...PublishOption) error {
	evt := ce.New(eventType, source, data)
	applyPublishOptions(&evt, opts)
	return s.PublishEvent(ctx, evt)
}

// PublishDataAfter publishes data as a CloudEvent with a delay.
func (s *Service) PublishDataAfter(ctx context.Context, eventType, source string, data any, delay time.Duration, opts ...PublishOption) error {
	evt := ce.New(eventType, source, data)
	applyPublishOptions(&evt, opts)
	return s.PublishEventAfter(ctx, evt, delay)
}

func applyPublishOptions(evt *ce.Event, opts []PublishOption) {
	var po publishOptions
	for _, opt := range opts {
		opt(&po)
	}

	if po.subject != nil {
		evt.Subject = po.subject
	}
	if po.dataContentType != nil {
		evt.DataContentType = po.dataContentType
	}
	if po.dataSchema != nil {
		evt.DataSchema = po.dataSchema
	}
	if po.maxAttempts > 0 {
		ce.SetMaxAttempts(evt, po.maxAttempts)
	}
	if po.traceID != "" {
		ce.SetTraceID(evt, po.traceID)
	}
	if po.parentID != "" {
		ce.SetParentID(evt, po.parentID)
	}
	if po.correlationID != "" {
		ce.SetCorrelationID(evt, po.correlationID)
	}
	for k, v := range po.extensions {
		if evt.Extensions == nil {
			evt.Extensions = make(map[string]any)
		}
		evt.Extensions[k] = v
	}
}

// ConsumeEvents registers a CloudEvents handler for the specified event type.
// The handler receives CloudEvents and returns errors to control message lifecycle.
func (s *Service) ConsumeEvents(eventType string, handler EventHandler) error {
	if s == nil {
		return errspkg.ErrServiceRequired
	}
	if handler == nil {
		return errspkg.ErrHandlerRequired
	}
	if eventType == "" {
		return errspkg.ErrConsumeMessageTypeRequired
	}

	handlerName := fmt.Sprintf("cloudevents-%s", eventType)

	wmHandler := s.wrapCloudEventsHandler(eventType, handler)

	s.router.AddNoPublisherHandler(
		handlerName,
		eventType, // topic = event type
		s.subscriber,
		wmHandler,
	)

	// Track handler info
	s.handlersMu.Lock()
	s.handlers = append(s.handlers, &HandlerInfo{
		Name:         handlerName,
		ConsumeQueue: eventType,
		PublishQueue: "",
	})
	s.handlersMu.Unlock()

	return nil
}

// wrapCloudEventsHandler wraps an EventHandler to handle retry/DLQ semantics.
func (s *Service) wrapCloudEventsHandler(eventType string, handler EventHandler) message.NoPublishHandlerFunc {
	return func(msg *message.Message) error {
		ctx := msg.Context()
		if ctx == nil {
			ctx = context.Background()
		}

		// Parse the CloudEvent
		evt := tryFromWatermillMessage(msg)

		// Increment attempt counter
		ce.IncrementAttempt(&evt)

		// Execute the handler
		err := handler(ctx, evt)

		// Classify the result and handle accordingly
		return s.handleCloudEventsResult(ctx, eventType, evt, msg, err)
	}
}

// handleCloudEventsResult processes the handler result and routes the message.
func (s *Service) handleCloudEventsResult(ctx context.Context, eventType string, evt ce.Event, msg *message.Message, err error) error {
	result, delay := ce.ClassifyError(err)

	switch result {
	case ce.ResultAck:
		// Successful processing
		return nil

	case ce.ResultSkip:
		// Skip without error
		s.Logger.Debug("Skipping CloudEvent", map[string]any{
			"event_id":   evt.ID,
			"event_type": evt.Type,
		})
		return nil

	case ce.ResultRetry:
		// Check if we've exceeded max attempts
		if ce.ExceedsMaxAttempts(evt) {
			return s.sendToCloudEventsDLQ(ctx, eventType, evt, err)
		}
		// Return error to trigger retry middleware
		return err

	case ce.ResultRetryAfter:
		// Check if we've exceeded max attempts
		if ce.ExceedsMaxAttempts(evt) {
			return s.sendToCloudEventsDLQ(ctx, eventType, evt, err)
		}

		// Check if transport supports delay
		caps := transportpkg.GetCapabilities(s.Conf.PubSubSystem)
		if caps.SupportsDelay {
			// Republish with delay
			ce.SetDelay(&evt, delay)
			if pubErr := s.PublishEvent(ctx, evt); pubErr != nil {
				s.Logger.Error("Failed to republish delayed event", pubErr, map[string]any{
					"event_id": evt.ID,
					"delay":    delay,
				})
				return err
			}
			return nil // Ack the original
		}

		// Transport doesn't support delay, fall back to regular retry
		return err

	case ce.ResultDeadLetter:
		return s.sendToCloudEventsDLQ(ctx, eventType, evt, err)

	default:
		return err
	}
}

// sendToCloudEventsDLQ sends an event to the dead letter queue.
func (s *Service) sendToCloudEventsDLQ(ctx context.Context, eventType string, evt ce.Event, err error) error {
	dlqTopic := ce.DLQTopic(eventType)

	// Prepare the event for DLQ
	ce.PrepareForDLQ(&evt, eventType, err)

	s.Logger.Info("Sending CloudEvent to DLQ", map[string]any{
		"event_id":   evt.ID,
		"event_type": evt.Type,
		"dlq_topic":  dlqTopic,
		"attempts":   ce.GetAttempt(evt),
		"error":      err.Error(),
	})

	// Publish to DLQ topic
	if pubErr := s.PublishEvent(ctx, evt); pubErr != nil {
		s.Logger.Error("Failed to publish to DLQ", pubErr, map[string]any{
			"event_id":  evt.ID,
			"dlq_topic": dlqTopic,
		})
		// Return nil to ack the original message and avoid infinite loops
		return nil
	}

	// Return nil to ack the original message
	return nil
}

// GetTransportCapabilities returns the capabilities of the configured transport.
func (s *Service) GetTransportCapabilities() transportpkg.Capabilities {
	if s == nil || s.Conf == nil {
		return transportpkg.Capabilities{}
	}
	return transportpkg.GetCapabilities(s.Conf.PubSubSystem)
}

// CloudEventsHandlerRegistration configures a CloudEvents message handler.
type CloudEventsHandlerRegistration struct {
	// Name is a unique identifier for this handler.
	Name string

	// EventType is the CloudEvents type to consume.
	EventType string

	// Handler processes incoming CloudEvents.
	Handler EventHandler

	// MaxAttempts overrides the default max retry attempts.
	MaxAttempts int
}

// RegisterCloudEventsHandler registers a CloudEvents handler with full configuration.
func RegisterCloudEventsHandler(s *Service, reg CloudEventsHandlerRegistration) error {
	if s == nil {
		return errspkg.ErrServiceRequired
	}
	if reg.Handler == nil {
		return errspkg.ErrHandlerRequired
	}
	if reg.EventType == "" {
		return errspkg.ErrConsumeMessageTypeRequired
	}
	if reg.Name == "" {
		reg.Name = fmt.Sprintf("cloudevents-%s", reg.EventType)
	}

	maxAttempts := reg.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = ce.DefaultMaxAttempts
	}

	// Wrap the handler to inject max attempts
	wrappedHandler := func(ctx context.Context, evt ce.Event) error {
		ce.SetMaxAttempts(&evt, maxAttempts)
		return reg.Handler(ctx, evt)
	}

	return s.ConsumeEvents(reg.EventType, wrappedHandler)
}

// EventContext provides helpers for working with CloudEvents in handlers.
type EventContext struct {
	Event     ce.Event
	RawData   json.RawMessage
	Publisher *Service
	Logger    interface {
		Info(msg string, fields map[string]any)
		Error(msg string, err error, fields map[string]any)
	}
}

// UnmarshalData unmarshals the event data into the provided struct.
func (ec *EventContext) UnmarshalData(v any) error {
	if ec.Event.Data == nil {
		return errors.New("event has no data")
	}

	// If data is already the right type
	if ec.Event.DataBase64 != nil {
		return errors.New("base64 data not supported, use DataBase64 directly")
	}

	// Marshal then unmarshal to handle type conversion
	data, err := json.Marshal(ec.Event.Data)
	if err != nil {
		return fmt.Errorf("failed to marshal event data: %w", err)
	}

	return json.Unmarshal(data, v)
}

// Publish publishes a response event, copying tracing context.
func (ec *EventContext) Publish(ctx context.Context, eventType, source string, data any) error {
	if ec.Publisher == nil {
		return errors.New("publisher not available in context")
	}

	evt := ce.New(eventType, source, data)
	ce.CopyTracingContext(ec.Event, &evt)

	return ec.Publisher.PublishEvent(ctx, evt)
}

// NewEventID generates a new event ID.
func NewEventID() string {
	return idspkg.CreateULID()
}

// toWatermillMessage converts a CloudEvent to a Watermill message (internal use only).
// The entire event is serialized as JSON in the message payload.
func toWatermillMessage(evt ce.Event) (*message.Message, error) {
	if err := evt.Validate(); err != nil {
		return nil, fmt.Errorf("invalid CloudEvent: %w", err)
	}

	payload, err := json.Marshal(evt)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal CloudEvent: %w", err)
	}

	msg := message.NewMessage(evt.ID, payload)

	// Map core CloudEvents attributes to Watermill metadata
	msg.Metadata.Set("ce_specversion", evt.SpecVersion)
	msg.Metadata.Set("ce_type", evt.Type)
	msg.Metadata.Set("ce_source", evt.Source)
	msg.Metadata.Set("ce_id", evt.ID)

	if !evt.Time.IsZero() {
		msg.Metadata.Set("ce_time", evt.Time.Format(time.RFC3339Nano))
	}
	if evt.DataContentType != nil {
		msg.Metadata.Set("ce_datacontenttype", *evt.DataContentType)
	}
	if evt.Subject != nil {
		msg.Metadata.Set("ce_subject", *evt.Subject)
	}
	if evt.DataSchema != nil {
		msg.Metadata.Set("ce_dataschema", *evt.DataSchema)
	}

	// Map protoflow extensions to metadata for transport-level access
	for k, v := range evt.Extensions {
		switch val := v.(type) {
		case string:
			msg.Metadata.Set(k, val)
		case int:
			msg.Metadata.Set(k, fmt.Sprintf("%d", val))
		case int64:
			msg.Metadata.Set(k, fmt.Sprintf("%d", val))
		case float64:
			msg.Metadata.Set(k, fmt.Sprintf("%v", val))
		case bool:
			if val {
				msg.Metadata.Set(k, "true")
			} else {
				msg.Metadata.Set(k, "false")
			}
		default:
			if val != nil {
				msg.Metadata.Set(k, fmt.Sprintf("%v", val))
			}
		}
	}

	return msg, nil
}

// tryFromWatermillMessage attempts to parse a Watermill message as a CloudEvent (internal use only).
// If the payload is not a valid CloudEvent, it creates a synthetic event
// from the message metadata and raw payload.
func tryFromWatermillMessage(msg *message.Message) ce.Event {
	// Try parsing as CloudEvent first
	var evt ce.Event
	if err := json.Unmarshal(msg.Payload, &evt); err == nil && evt.Validate() == nil {
		if evt.ID == "" {
			evt.ID = msg.UUID
		}
		return evt
	}

	// Create a synthetic event from message metadata
	evt = ce.Event{
		SpecVersion: ce.SpecVersion,
		ID:          msg.UUID,
		Extensions:  make(map[string]any),
	}

	// Try to recover attributes from metadata
	if v := msg.Metadata.Get("ce_type"); v != "" {
		evt.Type = v
	} else if v := msg.Metadata.Get("event_message_schema"); v != "" {
		evt.Type = v
	} else {
		evt.Type = "unknown"
	}

	if v := msg.Metadata.Get("ce_source"); v != "" {
		evt.Source = v
	} else if v := msg.Metadata.Get("event_source"); v != "" {
		evt.Source = v
	} else {
		evt.Source = "unknown"
	}

	// Parse time from metadata
	if v := msg.Metadata.Get("ce_time"); v != "" {
		if t, err := time.Parse(time.RFC3339Nano, v); err == nil {
			evt.Time = t
		} else if t, err := time.Parse(time.RFC3339, v); err == nil {
			evt.Time = t
		}
	}

	// Set data as raw payload
	evt.Data = string(msg.Payload)

	// Copy other metadata as extensions
	for k, v := range msg.Metadata {
		if !isCloudEventsMetadata(k) {
			evt.Extensions[k] = v
		}
	}

	return evt
}

// isCloudEventsMetadata returns true if the key is a CloudEvents metadata key.
func isCloudEventsMetadata(key string) bool {
	switch key {
	case "ce_specversion", "ce_type", "ce_source", "ce_id", "ce_time",
		"ce_datacontenttype", "ce_subject", "ce_dataschema":
		return true
	default:
		return false
	}
}
