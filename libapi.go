package protoflow

import (
	"time"

	runtimepkg "github.com/drblury/protoflow/internal/runtime"
	ce "github.com/drblury/protoflow/internal/runtime/cloudevents"
	configpkg "github.com/drblury/protoflow/internal/runtime/config"
	errspkg "github.com/drblury/protoflow/internal/runtime/errors"
	handlerpkg "github.com/drblury/protoflow/internal/runtime/handlers"
	idspkg "github.com/drblury/protoflow/internal/runtime/ids"
	jsoncodec "github.com/drblury/protoflow/internal/runtime/jsoncodec"
	loggingpkg "github.com/drblury/protoflow/internal/runtime/logging"
	metadatapkg "github.com/drblury/protoflow/internal/runtime/metadata"
	transportpkg "github.com/drblury/protoflow/internal/runtime/transport"
	newtransport "github.com/drblury/protoflow/transport"
	"google.golang.org/protobuf/proto"
)

type (
	Config              = configpkg.Config
	Service             = runtimepkg.Service
	ServiceDependencies = runtimepkg.ServiceDependencies
	ProtoValidator      = runtimepkg.ProtoValidator
	OutboxStore         = runtimepkg.OutboxStore
	Transport           = transportpkg.Transport
	TransportFactory    = transportpkg.Factory

	MessageHandlerRegistration                = runtimepkg.MessageHandlerRegistration
	JSONHandlerRegistration[T any, O any]     = handlerpkg.JSONHandlerRegistration[T, O]
	JSONMessageContext[T any]                 = handlerpkg.JSONMessageContext[T]
	JSONMessageOutput[T any]                  = handlerpkg.JSONMessageOutput[T]
	JSONMessageHandler[T any, O any]          = handlerpkg.JSONMessageHandler[T, O]
	ProtoHandlerRegistration[T proto.Message] = handlerpkg.ProtoHandlerRegistration[T]
	ProtoHandlerOption                        = handlerpkg.ProtoHandlerOption
	ProtoMessageContext[T proto.Message]      = handlerpkg.ProtoMessageContext[T]
	ProtoMessageOutput                        = handlerpkg.ProtoMessageOutput
	ProtoMessageHandler[T proto.Message]      = handlerpkg.ProtoMessageHandler[T]
	MessageContextBase                        = handlerpkg.MessageContextBase

	MiddlewareBuilder      = runtimepkg.MiddlewareBuilder
	MiddlewareRegistration = runtimepkg.MiddlewareRegistration
	RetryMiddlewareConfig  = runtimepkg.RetryMiddlewareConfig

	Producer = runtimepkg.Producer

	Metadata = metadatapkg.Metadata

	LogFields                 = loggingpkg.LogFields
	ServiceLogger             = loggingpkg.ServiceLogger
	EntryLogger               = loggingpkg.EntryLogger
	EntryLoggerAdapter[T any] = loggingpkg.EntryLoggerAdapter[T]

	UnprocessableEventError = runtimepkg.UnprocessableEventError

	HandlerInfo           = runtimepkg.HandlerInfo
	HandlerStats          = runtimepkg.HandlerStats
	ConfigValidationError = errspkg.ConfigValidationError

	// Job lifecycle hooks
	JobContext = runtimepkg.JobContext
	JobHooks   = runtimepkg.JobHooks

	// DLQ metrics
	DLQMetrics         = runtimepkg.DLQMetrics
	DLQTopicMetrics    = runtimepkg.DLQTopicMetrics
	DLQMetricsSnapshot = runtimepkg.DLQMetricsSnapshot

	// Error classification
	ErrorClassifier = runtimepkg.ErrorClassifier
	ErrorCategory   = runtimepkg.ErrorCategory

	// CloudEvents types
	Event                          = ce.Event
	EventHandler                   = runtimepkg.EventHandler
	PublishOption                  = runtimepkg.PublishOption
	CloudEventsHandlerRegistration = runtimepkg.CloudEventsHandlerRegistration

	// Transport capabilities
	Capabilities = transportpkg.Capabilities

	// Modular transport types (new package structure)
	TransportBuilder         = newtransport.Builder
	TransportConfig          = newtransport.Config
	TransportRegistry        = newtransport.Registry
	TransportCapabilities    = newtransport.Capabilities
	TransportDLQManager      = newtransport.DLQManager
	TransportQueueIntrospect = newtransport.QueueIntrospector
	TransportDelayedPub      = newtransport.DelayedPublisher
)

var (
	NewService     = runtimepkg.NewService
	TryNewService  = runtimepkg.TryNewService
	ValidateConfig = configpkg.ValidateConfig

	RegisterMessageHandler  = runtimepkg.RegisterMessageHandler
	WithPublishMessageTypes = handlerpkg.WithPublishMessageTypes

	DefaultMiddlewares      = runtimepkg.DefaultMiddlewares
	CorrelationIDMiddleware = runtimepkg.CorrelationIDMiddleware
	LogMessagesMiddleware   = runtimepkg.LogMessagesMiddleware
	ProtoValidateMiddleware = runtimepkg.ProtoValidateMiddleware
	OutboxMiddleware        = runtimepkg.OutboxMiddleware
	TracerMiddleware        = runtimepkg.TracerMiddleware
	MetricsMiddleware       = runtimepkg.MetricsMiddleware
	RetryMiddleware         = runtimepkg.RetryMiddleware
	PoisonQueueMiddleware   = runtimepkg.PoisonQueueMiddleware
	RecovererMiddleware     = runtimepkg.RecovererMiddleware

	// Job lifecycle hooks
	JobHooksMiddleware = runtimepkg.JobHooksMiddleware
	LoggingHooks       = runtimepkg.LoggingHooks
	MetricsHooks       = runtimepkg.MetricsHooks
	AlertingHooks      = runtimepkg.AlertingHooks

	// DLQ metrics
	NewDLQMetrics = runtimepkg.NewDLQMetrics

	// CloudEvents constructors and helpers
	NewCloudEvent       = ce.New
	NewCloudEventWithID = ce.NewWithID

	// CloudEvents extension helpers
	GetAttempt          = ce.GetAttempt
	SetAttempt          = ce.SetAttempt
	GetMaxAttempts      = ce.GetMaxAttempts
	SetMaxAttempts      = ce.SetMaxAttempts
	IncrementAttempt    = ce.IncrementAttempt
	ExceedsMaxAttempts  = ce.ExceedsMaxAttempts
	GetNextAttemptAt    = ce.GetNextAttemptAt
	SetNextAttemptAt    = ce.SetNextAttemptAt
	SetNextAttemptAfter = ce.SetNextAttemptAfter
	IsDeadLetter        = ce.IsDeadLetter
	SetDeadLetter       = ce.SetDeadLetter
	GetOriginalTopic    = ce.GetOriginalTopic
	SetOriginalTopic    = ce.SetOriginalTopic
	GetErrorMessage     = ce.GetErrorMessage
	SetErrorMessage     = ce.SetErrorMessage
	GetTraceID          = ce.GetTraceID
	SetTraceID          = ce.SetTraceID
	GetParentID         = ce.GetParentID
	SetParentID         = ce.SetParentID
	GetCorrelationID    = ce.GetCorrelationID
	SetCorrelationID    = ce.SetCorrelationID
	GetDelayMs          = ce.GetDelayMs
	SetDelayMs          = ce.SetDelayMs
	GetDelay            = ce.GetDelay
	SetDelay            = ce.SetDelay
	GetEventVersion     = ce.GetEventVersion
	SetEventVersion     = ce.SetEventVersion
	PrepareForRetry     = ce.PrepareForRetry
	PrepareForDLQ       = ce.PrepareForDLQ
	DLQTopic            = ce.DLQTopic
	CopyTracingContext  = ce.CopyTracingContext

	// CloudEvents error types
	ErrRetry                = ce.ErrRetry
	ErrDeadLetter           = ce.ErrDeadLetter
	ErrSkip                 = ce.ErrSkip
	ErrUnprocessable        = ce.ErrUnprocessable
	ErrRetryAfter           = ce.ErrRetryAfter
	ErrDeadLetterWithReason = ce.ErrDeadLetterWithReason
	ClassifyError           = ce.ClassifyError
	IsRetryable             = ce.IsRetryable
	ShouldDeadLetter        = ce.ShouldDeadLetter

	// CloudEvents API
	RegisterCloudEventsHandler = runtimepkg.RegisterCloudEventsHandler

	// Transport capabilities
	GetCapabilities = transportpkg.GetCapabilities

	// Modular transport registry (new package structure)
	// Use RegisterTransport and BuildTransport to work with the modular transport packages.
	// Import individual transports via: _ "github.com/drblury/protoflow/transport/kafka"
	DefaultTransportRegistry = newtransport.DefaultRegistry
	RegisterTransport        = newtransport.Register
	BuildTransport           = newtransport.Build

	// Publish options
	WithSubject         = runtimepkg.WithSubject
	WithDataContentType = runtimepkg.WithDataContentType
	WithDataSchema      = runtimepkg.WithDataSchema
	WithExtension       = runtimepkg.WithExtension
	WithMaxAttempts     = runtimepkg.WithMaxAttempts
	WithTracing         = runtimepkg.WithTracing
	WithCorrelationID   = runtimepkg.WithCorrelationID

	Marshal       = jsoncodec.Marshal
	MarshalIndent = jsoncodec.MarshalIndent
	Unmarshal     = jsoncodec.Unmarshal
	Encode        = jsoncodec.Encode
	Decode        = jsoncodec.Decode

	ErrServiceRequired             = errspkg.ErrServiceRequired
	ErrHandlerRequired             = errspkg.ErrHandlerRequired
	ErrConsumeQueueRequired        = errspkg.ErrConsumeQueueRequired
	ErrHandlerNameRequired         = errspkg.ErrHandlerNameRequired
	ErrConsumeMessageTypeRequired  = errspkg.ErrConsumeMessageTypeRequired
	ErrConsumeMessagePointerNeeded = errspkg.ErrConsumeMessagePointerNeeded
	ErrPublisherRequired           = errspkg.ErrPublisherRequired
	ErrTopicRequired               = errspkg.ErrTopicRequired
	ErrConfigRequired              = errspkg.ErrConfigRequired
	ErrLoggerRequired              = errspkg.ErrLoggerRequired
	ErrEventPayloadRequired        = errspkg.ErrEventPayloadRequired

	NewSlogServiceLogger = loggingpkg.NewSlogServiceLogger

	NewMetadata = metadatapkg.New

	CreateULID = idspkg.CreateULID

	// NewEventID generates a unique event ID using ULID.
	NewEventID = runtimepkg.NewEventID
)

// Metadata keys - use these constants for standard metadata fields.
const (
	MetadataKeyCorrelationID = handlerpkg.MetadataKeyCorrelationID
	MetadataKeyEventSchema   = handlerpkg.MetadataKeyEventSchema
	MetadataKeyQueueDepth    = handlerpkg.MetadataKeyQueueDepth
	MetadataKeyEnqueuedAt    = handlerpkg.MetadataKeyEnqueuedAt
	MetadataKeyTraceID       = handlerpkg.MetadataKeyTraceID
	MetadataKeySpanID        = handlerpkg.MetadataKeySpanID

	// MetadataKeyDelay is used by SQLite and PostgreSQL transports for delayed message processing.
	// Set to a duration string like "30s", "5m", "1h".
	MetadataKeyDelay = "protoflow_delay"
)

// CloudEvents extension keys for protoflow reliability semantics.
const (
	// ExtAttempt is the current retry attempt number (1-based).
	ExtAttempt = ce.ExtAttempt

	// ExtMaxAttempts is the maximum number of retry attempts allowed.
	ExtMaxAttempts = ce.ExtMaxAttempts

	// ExtNextAttemptAt is the RFC3339 timestamp for the next retry.
	ExtNextAttemptAt = ce.ExtNextAttemptAt

	// ExtDeadLetter indicates the event has been moved to DLQ.
	ExtDeadLetter = ce.ExtDeadLetter

	// ExtTraceID is the distributed trace ID (W3C traceparent compatible).
	ExtTraceID = ce.ExtTraceID

	// ExtParentID is the parent span ID for trace correlation.
	ExtParentID = ce.ExtParentID

	// ExtDelayMs is the delay in milliseconds before processing.
	ExtDelayMs = ce.ExtDelayMs

	// ExtEventVersion is an optional version number for the event schema.
	ExtEventVersion = ce.ExtEventVersion

	// ExtOriginalTopic stores the original topic when moved to DLQ.
	ExtOriginalTopic = ce.ExtOriginalTopic

	// ExtErrorMessage stores the last error message when moved to DLQ.
	ExtErrorMessage = ce.ExtErrorMessage

	// ExtCorrelationID is a correlation identifier for request tracing.
	ExtCorrelationID = ce.ExtCorrelationID
)

// Error category constants for ErrorClassifier.
const (
	ErrorCategoryNone       = runtimepkg.ErrorCategoryNone
	ErrorCategoryValidation = runtimepkg.ErrorCategoryValidation
	ErrorCategoryTransport  = runtimepkg.ErrorCategoryTransport
	ErrorCategoryDownstream = runtimepkg.ErrorCategoryDownstream
	ErrorCategoryOther      = runtimepkg.ErrorCategoryOther
)

func RegisterJSONHandler[T any, O any](svc *Service, cfg JSONHandlerRegistration[T, O]) error {
	return runtimepkg.RegisterJSONHandler(svc, cfg)
}

func RegisterProtoHandler[T proto.Message](svc *Service, cfg ProtoHandlerRegistration[T]) error {
	return runtimepkg.RegisterProtoHandler(svc, cfg)
}

func NewProtoMessage[T proto.Message]() (T, error) {
	return runtimepkg.NewProtoMessage[T]()
}

func MustProtoMessage[T proto.Message]() T {
	return runtimepkg.MustProtoMessage[T]()
}

func NewEntryServiceLogger[T EntryLoggerAdapter[T]](entry T) ServiceLogger {
	return loggingpkg.NewEntryServiceLogger(entry)
}

// WithDelay returns a Metadata with the protoflow_delay key set for delayed message processing.
// This is a convenience wrapper for SQLite and PostgreSQL transports' delayed message feature.
// Example: protoflow.NewMetadata().Merge(protoflow.WithDelay(30 * time.Second))
func WithDelay(delay time.Duration) Metadata {
	return Metadata{MetadataKeyDelay: delay.String()}
}
