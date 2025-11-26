package cloudevents

import (
	"errors"
	"fmt"
	"time"
)

// Handler return error types for CloudEvents processing.
// These errors control the message lifecycle after handler execution.

var (
	// ErrRetry signals that the message should be retried with the default delay.
	// The retry middleware will apply exponential backoff.
	ErrRetry = errors.New("protoflow: retry message")

	// ErrDeadLetter signals that the message should be sent to the dead letter queue
	// without further retry attempts.
	ErrDeadLetter = errors.New("protoflow: send to dead letter queue")

	// ErrSkip signals that the message should be acknowledged without processing.
	// Use this for messages that are intentionally ignored (e.g., duplicates).
	ErrSkip = errors.New("protoflow: skip message")

	// ErrUnprocessable signals that the message is permanently invalid and cannot
	// be processed. It will be sent to DLQ with the unprocessable flag.
	ErrUnprocessable = errors.New("protoflow: unprocessable message")
)

// RetryAfterError signals that a message should be retried after a specific delay.
type RetryAfterError struct {
	Delay time.Duration
	Cause error
}

// ErrRetryAfter creates a RetryAfterError with the specified delay.
// The handler will be retried after the delay expires.
//
// Example:
//
//	return nil, cloudevents.ErrRetryAfter(5*time.Second, nil)
//	return nil, cloudevents.ErrRetryAfter(time.Minute, fmt.Errorf("rate limited"))
func ErrRetryAfter(delay time.Duration, cause error) *RetryAfterError {
	return &RetryAfterError{
		Delay: delay,
		Cause: cause,
	}
}

func (e *RetryAfterError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("protoflow: retry after %v: %v", e.Delay, e.Cause)
	}
	return fmt.Sprintf("protoflow: retry after %v", e.Delay)
}

func (e *RetryAfterError) Unwrap() error {
	return e.Cause
}

// Is implements errors.Is for RetryAfterError.
func (e *RetryAfterError) Is(target error) bool {
	if target == ErrRetry {
		return true
	}
	_, ok := target.(*RetryAfterError)
	return ok
}

// DeadLetterError signals that a message should be sent to DLQ with a reason.
type DeadLetterError struct {
	Reason string
	Cause  error
}

// ErrDeadLetterWithReason creates a DeadLetterError with a specific reason.
//
// Example:
//
//	return nil, cloudevents.ErrDeadLetterWithReason("payment already processed", nil)
func ErrDeadLetterWithReason(reason string, cause error) *DeadLetterError {
	return &DeadLetterError{
		Reason: reason,
		Cause:  cause,
	}
}

func (e *DeadLetterError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("protoflow: dead letter (%s): %v", e.Reason, e.Cause)
	}
	return fmt.Sprintf("protoflow: dead letter (%s)", e.Reason)
}

func (e *DeadLetterError) Unwrap() error {
	return e.Cause
}

// Is implements errors.Is for DeadLetterError.
func (e *DeadLetterError) Is(target error) bool {
	if target == ErrDeadLetter {
		return true
	}
	_, ok := target.(*DeadLetterError)
	return ok
}

// HandlerResult represents the outcome of processing an event.
type HandlerResult int

const (
	// ResultAck indicates successful processing; message should be acknowledged.
	ResultAck HandlerResult = iota

	// ResultRetry indicates the message should be retried.
	ResultRetry

	// ResultRetryAfter indicates the message should be retried after a delay.
	ResultRetryAfter

	// ResultDeadLetter indicates the message should be sent to DLQ.
	ResultDeadLetter

	// ResultSkip indicates the message should be skipped (ack without processing).
	ResultSkip
)

// ClassifyError determines the appropriate action based on the handler error.
// Returns the result type and optional delay for RetryAfter errors.
func ClassifyError(err error) (HandlerResult, time.Duration) {
	if err == nil {
		return ResultAck, 0
	}

	// Check for specific error types
	var retryAfter *RetryAfterError
	if errors.As(err, &retryAfter) {
		return ResultRetryAfter, retryAfter.Delay
	}

	if errors.Is(err, ErrDeadLetter) {
		return ResultDeadLetter, 0
	}

	if errors.Is(err, ErrSkip) {
		return ResultSkip, 0
	}

	if errors.Is(err, ErrUnprocessable) {
		return ResultDeadLetter, 0
	}

	if errors.Is(err, ErrRetry) {
		return ResultRetry, 0
	}

	// Default: treat unknown errors as retry
	return ResultRetry, 0
}

// IsRetryable returns true if the error indicates the message should be retried.
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	result, _ := ClassifyError(err)
	return result == ResultRetry || result == ResultRetryAfter
}

// ShouldDeadLetter returns true if the error indicates the message should go to DLQ.
func ShouldDeadLetter(err error) bool {
	if err == nil {
		return false
	}

	result, _ := ClassifyError(err)
	return result == ResultDeadLetter
}
