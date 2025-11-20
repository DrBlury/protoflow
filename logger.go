package protoflow

import (
	"log/slog"

	"github.com/ThreeDotsLabs/watermill"
)

// LogFields represents structured logging key/value pairs used by Protoflow.
type LogFields map[string]any

// ServiceLogger is the minimal logging contract required by Protoflow services.
// It maps directly onto Watermill's logging needs so applications can adapt
// their existing loggers without depending on slog.
type ServiceLogger interface {
	With(fields LogFields) ServiceLogger
	Debug(msg string, fields LogFields)
	Info(msg string, fields LogFields)
	Error(msg string, err error, fields LogFields)
	Trace(msg string, fields LogFields)
}

var logLevelMapping = map[slog.Level]slog.Level{
	slog.LevelDebug: slog.LevelDebug,
	slog.LevelInfo:  slog.LevelInfo,
	slog.LevelWarn:  slog.LevelWarn,
	slog.LevelError: slog.LevelError,
}

// NewSlogServiceLogger wraps a slog.Logger so it satisfies the ServiceLogger
// interface. This matches the previous behaviour of Protoflow services.
func NewSlogServiceLogger(log *slog.Logger) ServiceLogger {
	if log == nil {
		panic("protoflow: slog logger cannot be nil")
	}
	return NewWatermillServiceLogger(watermill.NewSlogLoggerWithLevelMapping(log, logLevelMapping))
}

// NewWatermillServiceLogger wraps an existing Watermill LoggerAdapter so it can
// be supplied to NewService.
func NewWatermillServiceLogger(logger watermill.LoggerAdapter) ServiceLogger {
	if logger == nil {
		panic("protoflow: watermill logger cannot be nil")
	}
	return &watermillServiceLogger{inner: logger}
}

type watermillServiceLogger struct {
	inner watermill.LoggerAdapter
}

func (w *watermillServiceLogger) With(fields LogFields) ServiceLogger {
	return &watermillServiceLogger{inner: w.inner.With(toWatermillFields(fields))}
}

func (w *watermillServiceLogger) Debug(msg string, fields LogFields) {
	w.inner.Debug(msg, toWatermillFields(fields))
}

func (w *watermillServiceLogger) Info(msg string, fields LogFields) {
	w.inner.Info(msg, toWatermillFields(fields))
}

func (w *watermillServiceLogger) Error(msg string, err error, fields LogFields) {
	w.inner.Error(msg, err, toWatermillFields(fields))
}

func (w *watermillServiceLogger) Trace(msg string, fields LogFields) {
	w.inner.Trace(msg, toWatermillFields(fields))
}

type serviceLoggerAdapter struct {
	base ServiceLogger
}

func newWatermillLogger(log ServiceLogger) watermill.LoggerAdapter {
	if log == nil {
		panic("protoflow: ServiceLogger cannot be nil")
	}
	return &serviceLoggerAdapter{base: log}
}

func (s *serviceLoggerAdapter) Error(msg string, err error, fields watermill.LogFields) {
	s.base.Error(msg, err, fromWatermillFields(fields))
}

func (s *serviceLoggerAdapter) Info(msg string, fields watermill.LogFields) {
	s.base.Info(msg, fromWatermillFields(fields))
}

func (s *serviceLoggerAdapter) Debug(msg string, fields watermill.LogFields) {
	s.base.Debug(msg, fromWatermillFields(fields))
}

func (s *serviceLoggerAdapter) Trace(msg string, fields watermill.LogFields) {
	s.base.Trace(msg, fromWatermillFields(fields))
}

func (s *serviceLoggerAdapter) With(fields watermill.LogFields) watermill.LoggerAdapter {
	return &serviceLoggerAdapter{base: s.base.With(fromWatermillFields(fields))}
}

func toWatermillFields(fields LogFields) watermill.LogFields {
	if len(fields) == 0 {
		return nil
	}
	return watermill.LogFields(fields)
}

func fromWatermillFields(fields watermill.LogFields) LogFields {
	if len(fields) == 0 {
		return nil
	}
	return LogFields(fields)
}
