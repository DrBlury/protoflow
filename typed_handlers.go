package protoflow

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/ThreeDotsLabs/watermill/message"
	"google.golang.org/protobuf/proto"
)

// ProtoHandlerRegistration configures a typed protobuf handler that automatically
// unmarshals incoming payloads and marshals emitted events.
type ProtoHandlerRegistration[T proto.Message] struct {
	Name               string
	ConsumeQueue       string
	PublishQueue       string
	ConsumeMessageType T
	Handler            ProtoMessageHandler[T]
	PublishMessageType proto.Message
	Options            []ProtoHandlerOption
	ValidateOutgoing   bool
}

// ProtoHandlerOption customises handler registration.
type ProtoHandlerOption func(*protoHandlerOptions)

type protoHandlerOptions struct {
	additionalPublishTypes []proto.Message
}

// WithPublishMessageTypes registers extra proto schemas emitted by this handler.
// Use this when the handler may emit multiple message types.
func WithPublishMessageTypes(msgs ...proto.Message) ProtoHandlerOption {
	return func(cfg *protoHandlerOptions) {
		cfg.additionalPublishTypes = append(cfg.additionalPublishTypes, msgs...)
	}
}

// ProtoMessageContext provides strongly typed access to the incoming message payload
type ProtoMessageContext[T proto.Message] struct {
	Payload  T
	Metadata Metadata
}

// CloneMetadata returns a copy of the current metadata map so handlers can safely
// mutate headers for outgoing events without touching the original map.
func (c ProtoMessageContext[T]) CloneMetadata() Metadata {
	return c.Metadata.clone()
}

// ProtoMessageOutput describes an event that should be emitted after the handler succeeds.
type ProtoMessageOutput struct {
	Message  proto.Message
	Metadata Metadata
}

// ProtoMessageHandler processes a typed protobuf payload and returns the events to emit.
type ProtoMessageHandler[T proto.Message] func(ctx context.Context, event ProtoMessageContext[T]) ([]ProtoMessageOutput, error)

// RegisterProtoHandler converts the typed handler into a Watermill handler and registers it on the Service router.
func RegisterProtoHandler[T proto.Message](svc *Service, cfg ProtoHandlerRegistration[T]) error {
	if svc == nil {
		return errors.New("event service is required")
	}

	extra := protoHandlerOptions{}
	for _, opt := range cfg.Options {
		if opt != nil {
			opt(&extra)
		}
	}

	var validate func(proto.Message) error
	if cfg.ValidateOutgoing && svc.validator != nil {
		validate = svc.validator.Validate
	}

	wrapped, err := buildProtoHandler(cfg.ConsumeMessageType, cfg.Handler, validate)
	if err != nil {
		return err
	}

	if err := svc.registerHandler(handlerRegistration{
		Name:               cfg.Name,
		ConsumeQueue:       cfg.ConsumeQueue,
		PublishQueue:       cfg.PublishQueue,
		Handler:            wrapped,
		consumeMessageType: cfg.ConsumeMessageType,
	}); err != nil {
		return err
	}

	if cfg.PublishMessageType != nil {
		svc.registerProtoType(cfg.PublishMessageType)
	}
	for _, emitted := range extra.additionalPublishTypes {
		svc.registerProtoType(emitted)
	}

	return nil
}

func buildProtoHandler[T proto.Message](prototype T, handler ProtoMessageHandler[T], validate func(proto.Message) error) (message.HandlerFunc, error) {
	if handler == nil {
		return nil, errors.New("proto handler function is required")
	}
	if isNilProto(prototype) {
		return nil, errors.New("consume message type is required")
	}

	return func(msg *message.Message) ([]*message.Message, error) {
		typed, err := clonePrototype(prototype)
		if err != nil {
			return nil, err
		}

		if err := Unmarshal(msg.Payload, typed); err != nil {
			return nil, fmt.Errorf("failed to unmarshal %T payload: %w", prototype, err)
		}

		ctx := ProtoMessageContext[T]{
			Payload:  typed,
			Metadata: metadataFromWatermill(msg.Metadata),
		}

		outgoing, err := handler(msg.Context(), ctx)
		if err != nil {
			return nil, err
		}

		if validate != nil {
			for _, out := range outgoing {
				if out.Message == nil {
					return nil, errors.New("proto handler emitted nil message")
				}
				if err := validate(out.Message); err != nil {
					return nil, err
				}
			}
		}

		return convertProtoOutputs(outgoing, ctx.Metadata)
	}, nil
}

func clonePrototype[T proto.Message](prototype T) (T, error) {
	if isNilProto(prototype) {
		var zero T
		return zero, errors.New("consume message type is required")
	}

	cloned := proto.Clone(prototype)
	proto.Reset(cloned)

	typed, ok := cloned.(T)
	if !ok {
		var zero T
		return zero, fmt.Errorf("unexpected prototype type %T", cloned)
	}

	return typed, nil
}

func convertProtoOutputs(outputs []ProtoMessageOutput, fallback Metadata) ([]*message.Message, error) {
	if len(outputs) == 0 {
		return nil, nil
	}

	result := make([]*message.Message, len(outputs))
	for i, out := range outputs {
		metadata := out.Metadata
		if metadata == nil {
			metadata = fallback
		}

		msg, err := NewMessageFromProto(out.Message, metadata)
		if err != nil {
			return nil, err
		}
		result[i] = msg
	}

	return result, nil
}

func isNilProto[T proto.Message](prototype T) bool {
	msg := proto.Message(prototype)
	if msg == nil {
		return true
	}

	val := reflect.ValueOf(msg)
	switch val.Kind() {
	case reflect.Interface, reflect.Ptr, reflect.Slice, reflect.Map, reflect.Func:
		return val.IsNil()
	default:
		return false
	}
}
