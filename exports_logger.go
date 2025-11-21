package protoflow

import loggingpkg "github.com/drblury/protoflow/internal/runtime/logging"

type (
	LogFields                 = loggingpkg.LogFields
	ServiceLogger             = loggingpkg.ServiceLogger
	EntryLogger               = loggingpkg.EntryLogger
	EntryLoggerAdapter[T any] = loggingpkg.EntryLoggerAdapter[T]
)

var (
	NewSlogServiceLogger = loggingpkg.NewSlogServiceLogger
)

func NewEntryServiceLogger[T EntryLoggerAdapter[T]](entry T) ServiceLogger {
	return loggingpkg.NewEntryServiceLogger(entry)
}
