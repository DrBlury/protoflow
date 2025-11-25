package errors

import sterrors "errors"

var (
	ErrServiceRequired             = sterrors.New("protoflow: event service is required")
	ErrHandlerRequired             = sterrors.New("protoflow: handler function is required")
	ErrConsumeQueueRequired        = sterrors.New("protoflow: consume queue is required")
	ErrHandlerNameRequired         = sterrors.New("protoflow: handler name is required")
	ErrConsumeMessageTypeRequired  = sterrors.New("protoflow: consume message type is required")
	ErrConsumeMessagePointerNeeded = sterrors.New("protoflow: consume message type must be a pointer")
	ErrPublisherRequired           = sterrors.New("protoflow: publisher is required")
	ErrTopicRequired               = sterrors.New("protoflow: topic is required")
	ErrConfigRequired              = sterrors.New("protoflow: configuration is required")
	ErrLoggerRequired              = sterrors.New("protoflow: logger is required")
	ErrEventPayloadRequired        = sterrors.New("protoflow: event payload is required")
)

// ConfigValidationError wraps configuration validation errors with additional context.
type ConfigValidationError struct {
	Err error
}

func (e ConfigValidationError) Error() string {
	return "protoflow: invalid configuration: " + e.Err.Error()
}

func (e ConfigValidationError) Unwrap() error {
	return e.Err
}

// NewConfigValidationError creates a configuration validation error.
func NewConfigValidationError(err error) error {
	if err == nil {
		return nil
	}
	return ConfigValidationError{Err: err}
}
