package runtime

import (
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"

	"github.com/drblury/protoflow/internal/runtime/logging"
)

type LogFields = logging.LogFields

type ServiceLogger = logging.ServiceLogger

type EntryLogger = logging.EntryLogger

type EntryLoggerAdapter[T any] = logging.EntryLoggerAdapter[T]

func NewSlogServiceLogger(log *slog.Logger) ServiceLogger {
	return logging.NewSlogServiceLogger(log)
}

func NewEntryServiceLogger[T EntryLoggerAdapter[T]](entry T) ServiceLogger {
	return logging.NewEntryServiceLogger(entry)
}

func newWatermillLogger(log ServiceLogger) watermill.LoggerAdapter {
	return logging.NewWatermillAdapter(log)
}
