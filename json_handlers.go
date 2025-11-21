package protoflow

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/ThreeDotsLabs/watermill/message"
)

// JSONHandlerRegistration wires a typed JSON handler to the router.
type JSONHandlerRegistration[T any, O any] struct {
	Name         string
	ConsumeQueue string
	PublishQueue string
	Handler      JSONMessageHandler[T, O]
}

// JSONMessageContext exposes the incoming payload and metadata for JSON handlers.
type JSONMessageContext[T any] struct {
	Payload  T
	Metadata Metadata
}

// CloneMetadata copies the current metadata map so handlers can mutate headers safely.
func (c JSONMessageContext[T]) CloneMetadata() Metadata {
	return c.Metadata.clone()
}

// JSONMessageOutput represents an event emitted by a JSON handler.
type JSONMessageOutput[T any] struct {
	Message  T
	Metadata Metadata
}

// JSONMessageHandler processes a JSON payload and returns the events to publish.
type JSONMessageHandler[T any, O any] func(ctx context.Context, event JSONMessageContext[T]) ([]JSONMessageOutput[O], error)

// RegisterJSONHandler converts the typed JSON handler into a Watermill handler and registers it.
func RegisterJSONHandler[T any, O any](svc *Service, cfg JSONHandlerRegistration[T, O]) error {
	if svc == nil {
		return errors.New("event service is required")
	}

	wrapped, err := buildJSONHandler(cfg.Handler)
	if err != nil {
		return err
	}

	return svc.registerHandler(handlerRegistration{
		Name:         cfg.Name,
		ConsumeQueue: cfg.ConsumeQueue,
		PublishQueue: cfg.PublishQueue,
		Handler:      wrapped,
	})
}

func buildJSONHandler[T any, O any](handler JSONMessageHandler[T, O]) (message.HandlerFunc, error) {
	if handler == nil {
		return nil, errors.New("json handler function is required")
	}

	prototypeFactory, err := jsonPrototypeFactory[T]()
	if err != nil {
		return nil, err
	}

	return func(msg *message.Message) ([]*message.Message, error) {
		typed := prototypeFactory()

		if err := Unmarshal(msg.Payload, typed); err != nil {
			return nil, fmt.Errorf("failed to unmarshal JSON payload: %w", err)
		}

		ctx := JSONMessageContext[T]{
			Payload:  typed,
			Metadata: metadataFromWatermill(msg.Metadata),
		}

		outgoing, err := handler(msg.Context(), ctx)
		if err != nil {
			return nil, err
		}

		return convertJSONOutputs(outgoing, ctx.Metadata)
	}, nil
}

func jsonPrototypeFactory[T any]() (func() T, error) {
	var zero T
	typ := reflect.TypeOf(zero)
	if typ == nil {
		return nil, errors.New("consume message type is required")
	}
	if typ.Kind() != reflect.Ptr {
		return nil, errors.New("consume message type must be a pointer")
	}
	elem := typ.Elem()
	return func() T {
		clone := reflect.New(elem).Interface()
		return clone.(T)
	}, nil
}

func convertJSONOutputs[T any](outputs []JSONMessageOutput[T], fallback Metadata) ([]*message.Message, error) {
	if len(outputs) == 0 {
		return nil, nil
	}

	result := make([]*message.Message, len(outputs))
	for i, out := range outputs {
		if reflect.ValueOf(out.Message).IsZero() {
			return nil, errors.New("json handler emitted zero-value message")
		}

		payload, err := Marshal(out.Message)
		if err != nil {
			return nil, err
		}

		metadata := out.Metadata
		if metadata == nil {
			metadata = fallback
		}
		if metadata == nil {
			metadata = Metadata{}
		}
		metadata = metadata.clone()
		metadata["event_message_schema"] = fmt.Sprintf("%T", out.Message)

		msg := message.NewMessage(CreateULID(), payload)
		msg.Metadata = metadataToWatermill(metadata)
		result[i] = msg
	}

	return result, nil
}
