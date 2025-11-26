package cloudevents

import (
	"time"
)

// Protoflow extension keys for CloudEvents.
// These extensions provide reliability semantics for event processing.
const (
	// ExtAttempt is the current retry attempt number (1-based).
	ExtAttempt = "pf_attempt"

	// ExtMaxAttempts is the maximum number of retry attempts allowed.
	ExtMaxAttempts = "pf_max_attempts"

	// ExtNextAttemptAt is the RFC3339 timestamp or unix time for the next retry.
	ExtNextAttemptAt = "pf_next_attempt_at"

	// ExtDeadLetter indicates the event has been moved to DLQ.
	ExtDeadLetter = "pf_dead_letter"

	// ExtTraceID is the distributed trace ID (W3C traceparent compatible).
	ExtTraceID = "pf_trace_id"

	// ExtParentID is the parent span ID for trace correlation.
	ExtParentID = "pf_parent_id"

	// ExtDelayMs is the delay in milliseconds before processing.
	ExtDelayMs = "pf_delay_ms"

	// ExtEventVersion is an optional version number for the event schema.
	ExtEventVersion = "pf_event_version"

	// ExtOriginalTopic stores the original topic when moved to DLQ.
	ExtOriginalTopic = "pf_original_topic"

	// ExtErrorMessage stores the last error message when moved to DLQ.
	ExtErrorMessage = "pf_error_message"

	// ExtCorrelationID is a correlation identifier for request tracing.
	ExtCorrelationID = "pf_correlation_id"
)

// Default values for protoflow extensions.
const (
	DefaultMaxAttempts = 3
)

// --- Attempt Extensions ---

// GetAttempt returns the current attempt number (1-based).
// Returns 0 if not set.
func GetAttempt(evt Event) int {
	return evt.GetExtensionInt(ExtAttempt)
}

// SetAttempt sets the current attempt number.
func SetAttempt(evt *Event, n int) {
	if evt.Extensions == nil {
		evt.Extensions = make(map[string]any)
	}
	evt.Extensions[ExtAttempt] = n
}

// GetMaxAttempts returns the maximum number of attempts.
// Returns DefaultMaxAttempts if not set.
func GetMaxAttempts(evt Event) int {
	v := evt.GetExtensionInt(ExtMaxAttempts)
	if v == 0 {
		return DefaultMaxAttempts
	}
	return v
}

// SetMaxAttempts sets the maximum number of attempts.
func SetMaxAttempts(evt *Event, n int) {
	if evt.Extensions == nil {
		evt.Extensions = make(map[string]any)
	}
	evt.Extensions[ExtMaxAttempts] = n
}

// IncrementAttempt increments the attempt counter and returns the new value.
func IncrementAttempt(evt *Event) int {
	current := GetAttempt(*evt)
	newAttempt := current + 1
	SetAttempt(evt, newAttempt)
	return newAttempt
}

// ExceedsMaxAttempts returns true if the current attempt exceeds the max.
func ExceedsMaxAttempts(evt Event) bool {
	return GetAttempt(evt) >= GetMaxAttempts(evt)
}

// --- Next Attempt Time Extensions ---

// GetNextAttemptAt returns the scheduled time for the next retry attempt.
// Returns zero time if not set.
func GetNextAttemptAt(evt Event) time.Time {
	return evt.GetExtensionTime(ExtNextAttemptAt)
}

// SetNextAttemptAt sets the time for the next retry attempt.
func SetNextAttemptAt(evt *Event, t time.Time) {
	if evt.Extensions == nil {
		evt.Extensions = make(map[string]any)
	}
	evt.Extensions[ExtNextAttemptAt] = t.Format(time.RFC3339)
}

// SetNextAttemptAfter sets the next attempt time as a duration from now.
func SetNextAttemptAfter(evt *Event, d time.Duration) {
	SetNextAttemptAt(evt, time.Now().Add(d))
}

// --- Dead Letter Extensions ---

// IsDeadLetter returns true if the event has been marked for DLQ.
func IsDeadLetter(evt Event) bool {
	return evt.GetExtensionBool(ExtDeadLetter)
}

// SetDeadLetter marks the event as dead-lettered.
func SetDeadLetter(evt *Event, isDead bool) {
	if evt.Extensions == nil {
		evt.Extensions = make(map[string]any)
	}
	evt.Extensions[ExtDeadLetter] = isDead
}

// GetOriginalTopic returns the original topic before DLQ routing.
func GetOriginalTopic(evt Event) string {
	return evt.GetExtensionString(ExtOriginalTopic)
}

// SetOriginalTopic stores the original topic when moving to DLQ.
func SetOriginalTopic(evt *Event, topic string) {
	if evt.Extensions == nil {
		evt.Extensions = make(map[string]any)
	}
	evt.Extensions[ExtOriginalTopic] = topic
}

// GetErrorMessage returns the last error message.
func GetErrorMessage(evt Event) string {
	return evt.GetExtensionString(ExtErrorMessage)
}

// SetErrorMessage stores an error message on the event.
func SetErrorMessage(evt *Event, msg string) {
	if evt.Extensions == nil {
		evt.Extensions = make(map[string]any)
	}
	evt.Extensions[ExtErrorMessage] = msg
}

// --- Tracing Extensions ---

// GetTraceID returns the distributed trace ID.
func GetTraceID(evt Event) string {
	return evt.GetExtensionString(ExtTraceID)
}

// SetTraceID sets the distributed trace ID.
func SetTraceID(evt *Event, traceID string) {
	if evt.Extensions == nil {
		evt.Extensions = make(map[string]any)
	}
	evt.Extensions[ExtTraceID] = traceID
}

// GetParentID returns the parent span ID.
func GetParentID(evt Event) string {
	return evt.GetExtensionString(ExtParentID)
}

// SetParentID sets the parent span ID.
func SetParentID(evt *Event, parentID string) {
	if evt.Extensions == nil {
		evt.Extensions = make(map[string]any)
	}
	evt.Extensions[ExtParentID] = parentID
}

// GetCorrelationID returns the correlation ID for request tracing.
func GetCorrelationID(evt Event) string {
	return evt.GetExtensionString(ExtCorrelationID)
}

// SetCorrelationID sets the correlation ID.
func SetCorrelationID(evt *Event, correlationID string) {
	if evt.Extensions == nil {
		evt.Extensions = make(map[string]any)
	}
	evt.Extensions[ExtCorrelationID] = correlationID
}

// --- Delay Extensions ---

// GetDelayMs returns the delay in milliseconds.
// Returns 0 if not set.
func GetDelayMs(evt Event) int64 {
	return evt.GetExtensionInt64(ExtDelayMs)
}

// SetDelayMs sets the delay in milliseconds.
func SetDelayMs(evt *Event, delayMs int64) {
	if evt.Extensions == nil {
		evt.Extensions = make(map[string]any)
	}
	evt.Extensions[ExtDelayMs] = delayMs
}

// GetDelay returns the delay as a time.Duration.
func GetDelay(evt Event) time.Duration {
	return time.Duration(GetDelayMs(evt)) * time.Millisecond
}

// SetDelay sets the delay from a time.Duration.
func SetDelay(evt *Event, d time.Duration) {
	SetDelayMs(evt, d.Milliseconds())
}

// --- Event Version Extensions ---

// GetEventVersion returns the event schema version.
func GetEventVersion(evt Event) string {
	return evt.GetExtensionString(ExtEventVersion)
}

// SetEventVersion sets the event schema version.
func SetEventVersion(evt *Event, version string) {
	if evt.Extensions == nil {
		evt.Extensions = make(map[string]any)
	}
	evt.Extensions[ExtEventVersion] = version
}

// --- Helper Functions ---

// PrepareForRetry prepares an event for retry by incrementing the attempt
// counter and setting the next attempt time.
func PrepareForRetry(evt *Event, delay time.Duration) {
	IncrementAttempt(evt)
	SetNextAttemptAfter(evt, delay)
}

// PrepareForDLQ prepares an event for dead letter queue by:
// - Setting the dead letter flag
// - Storing the original topic
// - Storing the error message
func PrepareForDLQ(evt *Event, originalTopic string, err error) {
	SetDeadLetter(evt, true)
	SetOriginalTopic(evt, originalTopic)
	if err != nil {
		SetErrorMessage(evt, err.Error())
	}
}

// DLQTopic returns the dead letter queue topic name for a given event type.
// Convention: <eventType>.dead
func DLQTopic(eventType string) string {
	return eventType + ".dead"
}

// CopyTracingContext copies tracing extensions from source to destination event.
func CopyTracingContext(src Event, dst *Event) {
	if traceID := GetTraceID(src); traceID != "" {
		SetTraceID(dst, traceID)
	}
	if parentID := GetParentID(src); parentID != "" {
		SetParentID(dst, parentID)
	}
	if correlationID := GetCorrelationID(src); correlationID != "" {
		SetCorrelationID(dst, correlationID)
	}
}
