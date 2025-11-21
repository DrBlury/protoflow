package protoflow

import runtimepkg "github.com/drblury/protoflow/internal/runtime"

type (
	LogFields                 = runtimepkg.LogFields
	ServiceLogger             = runtimepkg.ServiceLogger
	EntryLogger               = runtimepkg.EntryLogger
	EntryLoggerAdapter[T any] = runtimepkg.EntryLoggerAdapter[T]
)

var (
	NewSlogServiceLogger = runtimepkg.NewSlogServiceLogger
)

func NewEntryServiceLogger[T EntryLoggerAdapter[T]](entry T) ServiceLogger {
	return runtimepkg.NewEntryServiceLogger[T](entry)
}
