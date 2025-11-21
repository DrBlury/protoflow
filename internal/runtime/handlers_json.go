package runtime

import handlerpkg "github.com/drblury/protoflow/internal/runtime/handlers"

// JSONHandlerRegistration wires a typed JSON handler to the router.
type JSONHandlerRegistration[T any, O any] = handlerpkg.JSONHandlerRegistration[T, O]

// JSONMessageContext exposes the incoming payload and metadata for JSON handlers.
type JSONMessageContext[T any] = handlerpkg.JSONMessageContext[T]

// JSONMessageOutput represents an event emitted by a JSON handler.
type JSONMessageOutput[T any] = handlerpkg.JSONMessageOutput[T]

// JSONMessageHandler processes a JSON payload and returns the events to publish.
type JSONMessageHandler[T any, O any] = handlerpkg.JSONMessageHandler[T, O]

// RegisterJSONHandler converts the typed JSON handler into a Watermill handler and registers it.
func RegisterJSONHandler[T any, O any](svc *Service, cfg JSONHandlerRegistration[T, O]) error {
	if svc == nil {
		return ErrServiceRequired
	}

	wrapped, err := handlerpkg.BuildJSONHandler(cfg.Handler)
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
