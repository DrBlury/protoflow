package errors

import (
	"errors"
	"testing"
)

func TestSentinelErrors(t *testing.T) {
	// Verify sentinel errors have expected messages
	tests := []struct {
		name    string
		err     error
		wantMsg string
	}{
		{"ErrServiceRequired", ErrServiceRequired, "protoflow: event service is required"},
		{"ErrHandlerRequired", ErrHandlerRequired, "protoflow: handler function is required"},
		{"ErrConsumeQueueRequired", ErrConsumeQueueRequired, "protoflow: consume queue is required"},
		{"ErrHandlerNameRequired", ErrHandlerNameRequired, "protoflow: handler name is required"},
		{"ErrConsumeMessageTypeRequired", ErrConsumeMessageTypeRequired, "protoflow: consume message type is required"},
		{"ErrConsumeMessagePointerNeeded", ErrConsumeMessagePointerNeeded, "protoflow: consume message type must be a pointer"},
		{"ErrPublisherRequired", ErrPublisherRequired, "protoflow: publisher is required"},
		{"ErrTopicRequired", ErrTopicRequired, "protoflow: topic is required"},
		{"ErrConfigRequired", ErrConfigRequired, "protoflow: configuration is required"},
		{"ErrLoggerRequired", ErrLoggerRequired, "protoflow: logger is required"},
		{"ErrEventPayloadRequired", ErrEventPayloadRequired, "protoflow: event payload is required"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", got, tt.wantMsg)
			}
		})
	}
}

func TestConfigValidationError(t *testing.T) {
	inner := errors.New("invalid port")
	err := ConfigValidationError{Err: inner}

	// Test Error()
	want := "protoflow: invalid configuration: invalid port"
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}

	// Test Unwrap()
	if unwrapped := err.Unwrap(); unwrapped != inner {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, inner)
	}
}

func TestNewConfigValidationError(t *testing.T) {
	t.Run("nil error returns nil", func(t *testing.T) {
		err := NewConfigValidationError(nil)
		if err != nil {
			t.Errorf("NewConfigValidationError(nil) = %v, want nil", err)
		}
	})

	t.Run("wraps error correctly", func(t *testing.T) {
		inner := errors.New("bad config")
		err := NewConfigValidationError(inner)

		var cfgErr ConfigValidationError
		if !errors.As(err, &cfgErr) {
			t.Fatalf("expected ConfigValidationError, got %T", err)
		}
		if cfgErr.Err != inner {
			t.Errorf("wrapped error = %v, want %v", cfgErr.Err, inner)
		}
	})

	t.Run("errors.Is works with wrapped error", func(t *testing.T) {
		inner := errors.New("specific error")
		err := NewConfigValidationError(inner)

		if !errors.Is(err, inner) {
			t.Error("errors.Is should match wrapped error")
		}
	})
}
