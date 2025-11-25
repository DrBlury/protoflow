package protoflow

import (
	"time"

	runtimepkg "github.com/drblury/protoflow/internal/runtime"
	configpkg "github.com/drblury/protoflow/internal/runtime/config"
	errspkg "github.com/drblury/protoflow/internal/runtime/errors"
	handlerpkg "github.com/drblury/protoflow/internal/runtime/handlers"
	idspkg "github.com/drblury/protoflow/internal/runtime/ids"
	jsoncodec "github.com/drblury/protoflow/internal/runtime/jsoncodec"
	loggingpkg "github.com/drblury/protoflow/internal/runtime/logging"
	metadatapkg "github.com/drblury/protoflow/internal/runtime/metadata"
	transportpkg "github.com/drblury/protoflow/internal/runtime/transport"
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

// Error category constants for ErrorClassifier.
const (
	ErrorCategoryNone       = runtimepkg.ErrorCategoryNone
	ErrorCategoryValidation = runtimepkg.ErrorCategoryValidation
	ErrorCategoryTransport  = runtimepkg.ErrorCategoryTransport
	ErrorCategoryDownstream = runtimepkg.ErrorCategoryDownstream
	ErrorCategoryOther      = runtimepkg.ErrorCategoryOther
)

func RegisterJSONHandler[T any, O any](svc *Service, cfg JSONHandlerRegistration[T, O]) error {
	return runtimepkg.RegisterJSONHandler[T, O](svc, cfg)
}

func RegisterProtoHandler[T proto.Message](svc *Service, cfg ProtoHandlerRegistration[T]) error {
	return runtimepkg.RegisterProtoHandler[T](svc, cfg)
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
