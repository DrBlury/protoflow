package protoflow

import runtimepkg "github.com/drblury/protoflow/internal/runtime"

type (
	MiddlewareBuilder      = runtimepkg.MiddlewareBuilder
	MiddlewareRegistration = runtimepkg.MiddlewareRegistration
	RetryMiddlewareConfig  = runtimepkg.RetryMiddlewareConfig
)

var (
	DefaultMiddlewares      = runtimepkg.DefaultMiddlewares
	CorrelationIDMiddleware = runtimepkg.CorrelationIDMiddleware
	LogMessagesMiddleware   = runtimepkg.LogMessagesMiddleware
	ProtoValidateMiddleware = runtimepkg.ProtoValidateMiddleware
	OutboxMiddleware        = runtimepkg.OutboxMiddleware
	TracerMiddleware        = runtimepkg.TracerMiddleware
	RetryMiddleware         = runtimepkg.RetryMiddleware
	PoisonQueueMiddleware   = runtimepkg.PoisonQueueMiddleware
	RecovererMiddleware     = runtimepkg.RecovererMiddleware
)
