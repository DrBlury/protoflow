package runtime

import (
	"context"
	"errors"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var protoJSONMarshalOptions = protojson.MarshalOptions{
	EmitUnpopulated: true,
}

// Producer emits proto-based events onto the configured transport.
type Producer interface {
	PublishProto(ctx context.Context, topic string, event proto.Message, metadata Metadata) error
}

// NewMessageFromProto converts the provided proto message into a Watermill message with
// the standard metadata required by the event pipeline.
func NewMessageFromProto(event proto.Message, metadata Metadata) (*message.Message, error) {
	if event == nil {
		return nil, errors.New("event payload is required")
	}

	payload, err := protoJSONMarshalOptions.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event payload: %w", err)
	}

	msg := message.NewMessage(CreateULID(), payload)
	msg.Metadata = metadataToWatermill(metadata)
	msg.Metadata["event_message_schema"] = fmt.Sprintf("%T", event)
	return msg, nil
}

// PublishProto marshals the proto payload and publishes it to the provided topic.
func PublishProto(ctx context.Context, publisher message.Publisher, topic string, event proto.Message, metadata Metadata) error {
	if publisher == nil {
		return ErrPublisherRequired
	}
	if topic == "" {
		return ErrTopicRequired
	}

	msg, err := NewMessageFromProto(event, metadata)
	if err != nil {
		return err
	}

	if ctx != nil {
		msg.SetContext(ctx)
	}

	return publisher.Publish(topic, msg)
}

// PublishProto emits the event using the Service publisher so HTTP handlers can
// create events without touching the internal Watermill APIs directly.
func (s *Service) PublishProto(ctx context.Context, topic string, event proto.Message, metadata Metadata) error {
	if s == nil {
		return errors.New("event service is nil")
	}
	return PublishProto(ctx, s.publisher, topic, event, metadata)
}
