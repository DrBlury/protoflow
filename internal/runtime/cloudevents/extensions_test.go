package cloudevents

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetSetAttempt(t *testing.T) {
	evt := New("test.event", "test-source", nil)

	// Initial attempt should be 0
	assert.Equal(t, 0, GetAttempt(evt))

	// Set attempt
	SetAttempt(&evt, 1)
	assert.Equal(t, 1, GetAttempt(evt))

	// Set higher attempt
	SetAttempt(&evt, 5)
	assert.Equal(t, 5, GetAttempt(evt))
}

func TestGetSetMaxAttempts(t *testing.T) {
	evt := New("test.event", "test-source", nil)

	// Should return default when not set
	assert.Equal(t, DefaultMaxAttempts, GetMaxAttempts(evt))

	// Set max attempts
	SetMaxAttempts(&evt, 10)
	assert.Equal(t, 10, GetMaxAttempts(evt))
}

func TestIncrementAttempt(t *testing.T) {
	evt := New("test.event", "test-source", nil)

	// First increment
	newAttempt := IncrementAttempt(&evt)
	assert.Equal(t, 1, newAttempt)
	assert.Equal(t, 1, GetAttempt(evt))

	// Second increment
	newAttempt = IncrementAttempt(&evt)
	assert.Equal(t, 2, newAttempt)
	assert.Equal(t, 2, GetAttempt(evt))
}

func TestExceedsMaxAttempts(t *testing.T) {
	evt := New("test.event", "test-source", nil)
	SetMaxAttempts(&evt, 3)

	// Not exceeded
	SetAttempt(&evt, 1)
	assert.False(t, ExceedsMaxAttempts(evt))

	SetAttempt(&evt, 2)
	assert.False(t, ExceedsMaxAttempts(evt))

	// At max
	SetAttempt(&evt, 3)
	assert.True(t, ExceedsMaxAttempts(evt))

	// Exceeded
	SetAttempt(&evt, 4)
	assert.True(t, ExceedsMaxAttempts(evt))
}

func TestGetSetNextAttemptAt(t *testing.T) {
	evt := New("test.event", "test-source", nil)

	// Should return zero time when not set
	assert.True(t, GetNextAttemptAt(evt).IsZero())

	// Set next attempt time
	futureTime := time.Now().Add(5 * time.Minute)
	SetNextAttemptAt(&evt, futureTime)
	retrieved := GetNextAttemptAt(evt)
	assert.WithinDuration(t, futureTime, retrieved, time.Second)
}

func TestSetNextAttemptAfter(t *testing.T) {
	evt := New("test.event", "test-source", nil)

	// Set next attempt 10 seconds from now
	before := time.Now()
	SetNextAttemptAfter(&evt, 10*time.Second)
	after := time.Now()

	retrieved := GetNextAttemptAt(evt)
	expectedEarliest := before.Add(10 * time.Second)
	expectedLatest := after.Add(10 * time.Second)

	// Check the time is in the expected range (with some tolerance for time parsing/formatting)
	assert.True(t, !retrieved.Before(expectedEarliest.Add(-time.Second)))
	assert.True(t, !retrieved.After(expectedLatest.Add(time.Second)))
}

func TestGetSetDeadLetter(t *testing.T) {
	evt := New("test.event", "test-source", nil)

	// Should be false by default
	assert.False(t, IsDeadLetter(evt))

	// Set to true
	SetDeadLetter(&evt, true)
	assert.True(t, IsDeadLetter(evt))

	// Set back to false
	SetDeadLetter(&evt, false)
	assert.False(t, IsDeadLetter(evt))
}

func TestGetSetOriginalTopic(t *testing.T) {
	evt := New("test.event", "test-source", nil)

	// Should be empty by default
	assert.Equal(t, "", GetOriginalTopic(evt))

	// Set original topic
	SetOriginalTopic(&evt, "orders.created")
	assert.Equal(t, "orders.created", GetOriginalTopic(evt))
}

func TestGetSetErrorMessage(t *testing.T) {
	evt := New("test.event", "test-source", nil)

	// Should be empty by default
	assert.Equal(t, "", GetErrorMessage(evt))

	// Set error message
	SetErrorMessage(&evt, "Database connection failed")
	assert.Equal(t, "Database connection failed", GetErrorMessage(evt))
}

func TestGetSetTraceID(t *testing.T) {
	evt := New("test.event", "test-source", nil)

	// Should be empty by default
	assert.Equal(t, "", GetTraceID(evt))

	// Set trace ID
	SetTraceID(&evt, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01")
	assert.Equal(t, "00-4bf92f3577b34da6a3ce929d0e0e4736-00f067aa0ba902b7-01", GetTraceID(evt))
}

func TestGetSetParentID(t *testing.T) {
	evt := New("test.event", "test-source", nil)

	// Should be empty by default
	assert.Equal(t, "", GetParentID(evt))

	// Set parent ID
	SetParentID(&evt, "00f067aa0ba902b7")
	assert.Equal(t, "00f067aa0ba902b7", GetParentID(evt))
}

func TestGetSetCorrelationID(t *testing.T) {
	evt := New("test.event", "test-source", nil)

	// Should be empty by default
	assert.Equal(t, "", GetCorrelationID(evt))

	// Set correlation ID
	SetCorrelationID(&evt, "correlation-123")
	assert.Equal(t, "correlation-123", GetCorrelationID(evt))
}

func TestGetSetDelayMs(t *testing.T) {
	evt := New("test.event", "test-source", nil)

	// Should be 0 by default
	assert.Equal(t, int64(0), GetDelayMs(evt))

	// Set delay in milliseconds
	SetDelayMs(&evt, 5000)
	assert.Equal(t, int64(5000), GetDelayMs(evt))
}

func TestGetSetDelay(t *testing.T) {
	evt := New("test.event", "test-source", nil)

	// Should be 0 by default
	assert.Equal(t, time.Duration(0), GetDelay(evt))

	// Set delay as duration
	SetDelay(&evt, 5*time.Second)
	assert.Equal(t, 5*time.Second, GetDelay(evt))
	assert.Equal(t, int64(5000), GetDelayMs(evt))
}

func TestGetSetEventVersion(t *testing.T) {
	evt := New("test.event", "test-source", nil)

	// Should be empty by default
	assert.Equal(t, "", GetEventVersion(evt))

	// Set event version
	SetEventVersion(&evt, "v2")
	assert.Equal(t, "v2", GetEventVersion(evt))
}

func TestPrepareForRetry(t *testing.T) {
	evt := New("test.event", "test-source", nil)

	// Initial state
	assert.Equal(t, 0, GetAttempt(evt))
	assert.True(t, GetNextAttemptAt(evt).IsZero())

	// Prepare for first retry
	PrepareForRetry(&evt, 5*time.Second)
	assert.Equal(t, 1, GetAttempt(evt))
	assert.False(t, GetNextAttemptAt(evt).IsZero())
	assert.True(t, GetNextAttemptAt(evt).After(time.Now()))

	// Prepare for second retry
	time.Sleep(10 * time.Millisecond)
	PrepareForRetry(&evt, 10*time.Second)
	assert.Equal(t, 2, GetAttempt(evt))
}

func TestPrepareForDLQ(t *testing.T) {
	evt := New("test.event", "test-source", nil)

	// Prepare for DLQ with error
	err := errors.New("processing failed")
	PrepareForDLQ(&evt, "orders.created", err)

	assert.True(t, IsDeadLetter(evt))
	assert.Equal(t, "orders.created", GetOriginalTopic(evt))
	assert.Equal(t, "processing failed", GetErrorMessage(evt))
}

func TestPrepareForDLQ_NoError(t *testing.T) {
	evt := New("test.event", "test-source", nil)

	// Prepare for DLQ without error
	PrepareForDLQ(&evt, "orders.created", nil)

	assert.True(t, IsDeadLetter(evt))
	assert.Equal(t, "orders.created", GetOriginalTopic(evt))
	assert.Equal(t, "", GetErrorMessage(evt))
}

func TestDLQTopic(t *testing.T) {
	tests := []struct {
		name      string
		eventType string
		want      string
	}{
		{
			name:      "simple event type",
			eventType: "orders.created",
			want:      "orders.created.dead",
		},
		{
			name:      "versioned event type",
			eventType: "customer.updated.v2",
			want:      "customer.updated.v2.dead",
		},
		{
			name:      "empty event type",
			eventType: "",
			want:      ".dead",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DLQTopic(tt.eventType)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCopyTracingContext(t *testing.T) {
	src := New("test.event", "test-source", nil)
	SetTraceID(&src, "trace-123")
	SetParentID(&src, "parent-456")
	SetCorrelationID(&src, "correlation-789")

	dst := New("test.event2", "test-source", nil)

	// Copy tracing context
	CopyTracingContext(src, &dst)

	// Verify all tracing fields are copied
	assert.Equal(t, "trace-123", GetTraceID(dst))
	assert.Equal(t, "parent-456", GetParentID(dst))
	assert.Equal(t, "correlation-789", GetCorrelationID(dst))
}

func TestCopyTracingContext_EmptySource(t *testing.T) {
	src := New("test.event", "test-source", nil)
	dst := New("test.event2", "test-source", nil)
	SetTraceID(&dst, "existing-trace")

	// Copy from empty source should not overwrite
	CopyTracingContext(src, &dst)

	// Existing trace ID should remain
	assert.Equal(t, "existing-trace", GetTraceID(dst))
}

func TestCopyTracingContext_PartialSource(t *testing.T) {
	src := New("test.event", "test-source", nil)
	SetTraceID(&src, "trace-123")
	// ParentID and CorrelationID not set

	dst := New("test.event2", "test-source", nil)

	// Copy partial tracing context
	CopyTracingContext(src, &dst)

	// Only trace ID should be set
	assert.Equal(t, "trace-123", GetTraceID(dst))
	assert.Equal(t, "", GetParentID(dst))
	assert.Equal(t, "", GetCorrelationID(dst))
}

func TestExtensionsWithNilMap(t *testing.T) {
	evt := Event{
		SpecVersion: SpecVersion,
		Type:        "test.event",
		Source:      "test-source",
		ID:          "test-id",
		Extensions:  nil,
	}

	// All setters should initialize the map
	SetAttempt(&evt, 1)
	assert.NotNil(t, evt.Extensions)
	assert.Equal(t, 1, GetAttempt(evt))

	evt.Extensions = nil
	SetMaxAttempts(&evt, 5)
	assert.NotNil(t, evt.Extensions)

	evt.Extensions = nil
	SetDeadLetter(&evt, true)
	assert.NotNil(t, evt.Extensions)

	evt.Extensions = nil
	SetOriginalTopic(&evt, "test.topic")
	assert.NotNil(t, evt.Extensions)

	evt.Extensions = nil
	SetErrorMessage(&evt, "test error")
	assert.NotNil(t, evt.Extensions)

	evt.Extensions = nil
	SetTraceID(&evt, "trace-id")
	assert.NotNil(t, evt.Extensions)

	evt.Extensions = nil
	SetParentID(&evt, "parent-id")
	assert.NotNil(t, evt.Extensions)

	evt.Extensions = nil
	SetCorrelationID(&evt, "correlation-id")
	assert.NotNil(t, evt.Extensions)

	evt.Extensions = nil
	SetDelayMs(&evt, 1000)
	assert.NotNil(t, evt.Extensions)

	evt.Extensions = nil
	SetEventVersion(&evt, "v1")
	assert.NotNil(t, evt.Extensions)
}
