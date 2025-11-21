package runtime

import (
	handlerpkg "github.com/drblury/protoflow/internal/runtime/handlers"
	"google.golang.org/protobuf/proto"
)

// ProtoHandlerRegistration configures a typed protobuf handler that automatically
// unmarshals incoming payloads and marshals emitted events.
type ProtoHandlerRegistration[T proto.Message] = handlerpkg.ProtoHandlerRegistration[T]

// ProtoHandlerOption customises handler registration.
type ProtoHandlerOption = handlerpkg.ProtoHandlerOption

// WithPublishMessageTypes registers extra proto schemas emitted by this handler.
// Use this when the handler may emit multiple message types.
var WithPublishMessageTypes = handlerpkg.WithPublishMessageTypes

// ProtoMessageContext provides strongly typed access to the incoming message payload.
type ProtoMessageContext[T proto.Message] = handlerpkg.ProtoMessageContext[T]

// ProtoMessageOutput describes an event that should be emitted after the handler succeeds.
type ProtoMessageOutput = handlerpkg.ProtoMessageOutput

// ProtoMessageHandler processes a typed protobuf payload and returns the events to emit.
type ProtoMessageHandler[T proto.Message] = handlerpkg.ProtoMessageHandler[T]

// RegisterProtoHandler converts the typed handler into a Watermill handler and registers it on the Service router.
func RegisterProtoHandler[T proto.Message](svc *Service, cfg ProtoHandlerRegistration[T]) error {
	if svc == nil {
		return ErrServiceRequired
	}

	var zero T
	prototype, err := handlerpkg.EnsureProtoPrototype(zero)
	if err != nil {
		return err
	}

	resolvedOpts := handlerpkg.ApplyProtoHandlerOptions(cfg.Options)

	var validate func(proto.Message) error
	if cfg.ValidateOutgoing && svc.validator != nil {
		validate = func(msg proto.Message) error {
			return svc.validator.Validate(msg)
		}
	}

	wrapped, err := handlerpkg.BuildProtoHandler(prototype, cfg.Handler, validate, NewMessageFromProto)
	if err != nil {
		return err
	}

	if err := svc.registerHandler(handlerRegistration{
		Name:               cfg.Name,
		ConsumeQueue:       cfg.ConsumeQueue,
		PublishQueue:       cfg.PublishQueue,
		Handler:            wrapped,
		consumeMessageType: prototype,
	}); err != nil {
		return err
	}

	for _, emitted := range resolvedOpts.AdditionalPublishTypes {
		svc.registerProtoType(emitted)
	}

	return nil
}
