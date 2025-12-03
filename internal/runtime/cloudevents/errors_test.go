package cloudevents

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestErrRetryAfter(t *testing.T) {
	// Test without cause
	err := ErrRetryAfter(5*time.Second, nil)
	assert.NotNil(t, err)
	assert.Equal(t, 5*time.Second, err.Delay)
	assert.Nil(t, err.Cause)
	assert.Contains(t, err.Error(), "retry after 5s")

	// Test with cause
	causeErr := errors.New("rate limited")
	err = ErrRetryAfter(time.Minute, causeErr)
	assert.Equal(t, time.Minute, err.Delay)
	assert.Equal(t, causeErr, err.Cause)
	assert.Contains(t, err.Error(), "retry after 1m0s")
	assert.Contains(t, err.Error(), "rate limited")
}

func TestRetryAfterError_Unwrap(t *testing.T) {
	causeErr := errors.New("underlying error")
	err := ErrRetryAfter(5*time.Second, causeErr)

	unwrapped := err.Unwrap()
	assert.Equal(t, causeErr, unwrapped)

	// Test without cause
	err = ErrRetryAfter(5*time.Second, nil)
	assert.Nil(t, err.Unwrap())
}

func TestRetryAfterError_Is(t *testing.T) {
	err := ErrRetryAfter(5*time.Second, nil)

	// Should match ErrRetry
	assert.True(t, errors.Is(err, ErrRetry))

	// Should match another RetryAfterError
	err2 := ErrRetryAfter(10*time.Second, nil)
	assert.True(t, err.Is(err2))

	// Should not match other errors
	assert.False(t, err.Is(ErrDeadLetter))
	assert.False(t, err.Is(ErrSkip))
}

func TestErrDeadLetterWithReason(t *testing.T) {
	// Test without cause
	err := ErrDeadLetterWithReason("payment already processed", nil)
	assert.NotNil(t, err)
	assert.Equal(t, "payment already processed", err.Reason)
	assert.Nil(t, err.Cause)
	assert.Contains(t, err.Error(), "dead letter")
	assert.Contains(t, err.Error(), "payment already processed")

	// Test with cause
	causeErr := errors.New("database error")
	err = ErrDeadLetterWithReason("failed validation", causeErr)
	assert.Equal(t, "failed validation", err.Reason)
	assert.Equal(t, causeErr, err.Cause)
	assert.Contains(t, err.Error(), "failed validation")
	assert.Contains(t, err.Error(), "database error")
}

func TestDeadLetterError_Unwrap(t *testing.T) {
	causeErr := errors.New("underlying error")
	err := ErrDeadLetterWithReason("test reason", causeErr)

	unwrapped := err.Unwrap()
	assert.Equal(t, causeErr, unwrapped)

	// Test without cause
	err = ErrDeadLetterWithReason("test reason", nil)
	assert.Nil(t, err.Unwrap())
}

func TestDeadLetterError_Is(t *testing.T) {
	err := ErrDeadLetterWithReason("test reason", nil)

	// Should match ErrDeadLetter
	assert.True(t, errors.Is(err, ErrDeadLetter))

	// Should match another DeadLetterError
	err2 := ErrDeadLetterWithReason("another reason", nil)
	assert.True(t, err.Is(err2))

	// Should not match other errors
	assert.False(t, err.Is(ErrRetry))
	assert.False(t, err.Is(ErrSkip))
}

func TestClassifyError_Nil(t *testing.T) {
	result, delay := ClassifyError(nil)
	assert.Equal(t, ResultAck, result)
	assert.Equal(t, time.Duration(0), delay)
}

func TestClassifyError_RetryAfter(t *testing.T) {
	err := ErrRetryAfter(30*time.Second, nil)
	result, delay := ClassifyError(err)
	assert.Equal(t, ResultRetryAfter, result)
	assert.Equal(t, 30*time.Second, delay)
}

func TestClassifyError_DeadLetter(t *testing.T) {
	result, delay := ClassifyError(ErrDeadLetter)
	assert.Equal(t, ResultDeadLetter, result)
	assert.Equal(t, time.Duration(0), delay)
}

func TestClassifyError_DeadLetterWithReason(t *testing.T) {
	err := ErrDeadLetterWithReason("test reason", nil)
	result, delay := ClassifyError(err)
	assert.Equal(t, ResultDeadLetter, result)
	assert.Equal(t, time.Duration(0), delay)
}

func TestClassifyError_Skip(t *testing.T) {
	result, delay := ClassifyError(ErrSkip)
	assert.Equal(t, ResultSkip, result)
	assert.Equal(t, time.Duration(0), delay)
}

func TestClassifyError_Unprocessable(t *testing.T) {
	result, delay := ClassifyError(ErrUnprocessable)
	assert.Equal(t, ResultDeadLetter, result)
	assert.Equal(t, time.Duration(0), delay)
}

func TestClassifyError_Retry(t *testing.T) {
	result, delay := ClassifyError(ErrRetry)
	assert.Equal(t, ResultRetry, result)
	assert.Equal(t, time.Duration(0), delay)
}

func TestClassifyError_UnknownError(t *testing.T) {
	// Unknown errors should default to retry
	err := errors.New("unknown error")
	result, delay := ClassifyError(err)
	assert.Equal(t, ResultRetry, result)
	assert.Equal(t, time.Duration(0), delay)
}

func TestClassifyError_WrappedErrors(t *testing.T) {
	// Test wrapped RetryAfterError
	wrappedRetry := fmt.Errorf("wrapped: %w", ErrRetryAfter(10*time.Second, nil))
	result, delay := ClassifyError(wrappedRetry)
	assert.Equal(t, ResultRetryAfter, result)
	assert.Equal(t, 10*time.Second, delay)

	// Test wrapped DeadLetterError
	wrappedDLQ := fmt.Errorf("wrapped: %w", ErrDeadLetter)
	result, delay = ClassifyError(wrappedDLQ)
	assert.Equal(t, ResultDeadLetter, result)
	assert.Equal(t, time.Duration(0), delay)

	// Test wrapped Skip
	wrappedSkip := fmt.Errorf("wrapped: %w", ErrSkip)
	result, delay = ClassifyError(wrappedSkip)
	assert.Equal(t, ResultSkip, result)
	assert.Equal(t, time.Duration(0), delay)

	// Test wrapped Retry
	wrappedRetrySimple := fmt.Errorf("wrapped: %w", ErrRetry)
	result, delay = ClassifyError(wrappedRetrySimple)
	assert.Equal(t, ResultRetry, result)
	assert.Equal(t, time.Duration(0), delay)
}

func TestIsRetryable_Nil(t *testing.T) {
	assert.False(t, IsRetryable(nil))
}

func TestIsRetryable_RetryErrors(t *testing.T) {
	// ErrRetry should be retryable
	assert.True(t, IsRetryable(ErrRetry))

	// RetryAfterError should be retryable
	assert.True(t, IsRetryable(ErrRetryAfter(5*time.Second, nil)))

	// Wrapped retry errors should be retryable
	assert.True(t, IsRetryable(fmt.Errorf("wrapped: %w", ErrRetry)))
}

func TestIsRetryable_NonRetryErrors(t *testing.T) {
	// Dead letter errors should not be retryable
	assert.False(t, IsRetryable(ErrDeadLetter))
	assert.False(t, IsRetryable(ErrDeadLetterWithReason("test", nil)))

	// Skip should not be retryable
	assert.False(t, IsRetryable(ErrSkip))

	// Unprocessable should not be retryable
	assert.False(t, IsRetryable(ErrUnprocessable))
}

func TestIsRetryable_UnknownError(t *testing.T) {
	// Unknown errors default to retry, so should be retryable
	assert.True(t, IsRetryable(errors.New("unknown error")))
}

func TestShouldDeadLetter_Nil(t *testing.T) {
	assert.False(t, ShouldDeadLetter(nil))
}

func TestShouldDeadLetter_DeadLetterErrors(t *testing.T) {
	// ErrDeadLetter should go to DLQ
	assert.True(t, ShouldDeadLetter(ErrDeadLetter))

	// DeadLetterError with reason should go to DLQ
	assert.True(t, ShouldDeadLetter(ErrDeadLetterWithReason("test", nil)))

	// ErrUnprocessable should go to DLQ
	assert.True(t, ShouldDeadLetter(ErrUnprocessable))

	// Wrapped dead letter errors should go to DLQ
	assert.True(t, ShouldDeadLetter(fmt.Errorf("wrapped: %w", ErrDeadLetter)))
}

func TestShouldDeadLetter_NonDeadLetterErrors(t *testing.T) {
	// Retry errors should not go to DLQ
	assert.False(t, ShouldDeadLetter(ErrRetry))
	assert.False(t, ShouldDeadLetter(ErrRetryAfter(5*time.Second, nil)))

	// Skip should not go to DLQ
	assert.False(t, ShouldDeadLetter(ErrSkip))

	// Unknown errors should not go to DLQ (they retry)
	assert.False(t, ShouldDeadLetter(errors.New("unknown error")))
}

func TestStandardErrors(t *testing.T) {
	// Verify standard error messages
	assert.Contains(t, ErrRetry.Error(), "retry message")
	assert.Contains(t, ErrDeadLetter.Error(), "dead letter")
	assert.Contains(t, ErrSkip.Error(), "skip message")
	assert.Contains(t, ErrUnprocessable.Error(), "unprocessable")
}
