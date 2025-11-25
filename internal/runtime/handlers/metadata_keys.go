package handlers

// Metadata key constants used throughout protoflow.
// These keys are reserved and should not be used for custom metadata.
const (
	// MetadataKeyCorrelationID tracks related messages across services.
	MetadataKeyCorrelationID = "correlation_id"

	// MetadataKeyEventSchema identifies the proto message type.
	MetadataKeyEventSchema = "event_message_schema"

	// MetadataKeyQueueDepth indicates queue depth at time of enqueue.
	MetadataKeyQueueDepth = "protoflow_queue_depth"

	// MetadataKeyEnqueuedAt records when a message was enqueued.
	MetadataKeyEnqueuedAt = "protoflow_enqueued_at"

	// MetadataKeyTraceID stores distributed tracing ID.
	MetadataKeyTraceID = "trace_id"

	// MetadataKeySpanID stores distributed tracing span ID.
	MetadataKeySpanID = "span_id"
)
