package runtime

import (
	"context"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
)

// JobContext provides information about a job execution to hooks.
type JobContext struct {
	// HandlerName is the name of the handler processing the job.
	HandlerName string
	// Topic is the topic/queue the message was received from.
	Topic string
	// MessageUUID is the unique identifier of the message.
	MessageUUID string
	// Metadata contains the message metadata.
	Metadata message.Metadata
	// Context is the context associated with the message.
	Context context.Context
	// StartedAt is when the job started processing.
	StartedAt time.Time
	// Duration is how long the job took (only set in OnJobDone and OnJobError).
	Duration time.Duration
	// RetryCount is the number of times this message has been retried.
	RetryCount int
}

// JobHooks defines callbacks for job lifecycle events.
// All hooks are optional - nil hooks are simply not called.
type JobHooks struct {
	// OnJobStart is called when a handler begins processing a message.
	// This is called before the handler function is invoked.
	OnJobStart func(ctx JobContext)

	// OnJobDone is called when a handler successfully completes processing.
	// Duration will be set to how long the handler took.
	OnJobDone func(ctx JobContext)

	// OnJobError is called when a handler returns an error.
	// The error is passed as the second argument.
	// Duration will be set to how long the handler took before failing.
	OnJobError func(ctx JobContext, err error)
}

// Merge combines two JobHooks, creating a new JobHooks that calls both.
// The hooks from 'other' are called after the hooks from 'h'.
func (h JobHooks) Merge(other JobHooks) JobHooks {
	return JobHooks{
		OnJobStart: chainStartHooks(h.OnJobStart, other.OnJobStart),
		OnJobDone:  chainDoneHooks(h.OnJobDone, other.OnJobDone),
		OnJobError: chainErrorHooks(h.OnJobError, other.OnJobError),
	}
}

func chainStartHooks(a, b func(JobContext)) func(JobContext) {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	return func(ctx JobContext) {
		a(ctx)
		b(ctx)
	}
}

func chainDoneHooks(a, b func(JobContext)) func(JobContext) {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	return func(ctx JobContext) {
		a(ctx)
		b(ctx)
	}
}

func chainErrorHooks(a, b func(JobContext, error)) func(JobContext, error) {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	return func(ctx JobContext, err error) {
		a(ctx, err)
		b(ctx, err)
	}
}

// JobHooksMiddleware creates a middleware that invokes the provided hooks
// at appropriate points in the message lifecycle.
func JobHooksMiddleware(hooks JobHooks) MiddlewareRegistration {
	return MiddlewareRegistration{
		Name: "job_hooks",
		Builder: func(s *Service) (message.HandlerMiddleware, error) {
			return jobHooksMiddleware(hooks), nil
		},
	}
}

func jobHooksMiddleware(hooks JobHooks) message.HandlerMiddleware {
	return func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			startTime := time.Now()

			// Extract retry count from metadata if available
			retryCount := 0
			if retryStr := msg.Metadata.Get("protoflow_retry_count"); retryStr != "" {
				// Parse retry count if present
				for i := 0; i < len(retryStr); i++ {
					if retryStr[i] >= '0' && retryStr[i] <= '9' {
						retryCount = retryCount*10 + int(retryStr[i]-'0')
					}
				}
			}

			jobCtx := JobContext{
				MessageUUID: msg.UUID,
				Metadata:    msg.Metadata,
				Context:     msg.Context(),
				StartedAt:   startTime,
				RetryCount:  retryCount,
			}

			// Extract handler name from metadata if available
			if handlerName := msg.Metadata.Get("protoflow_handler"); handlerName != "" {
				jobCtx.HandlerName = handlerName
			}
			if topic := msg.Metadata.Get("protoflow_topic"); topic != "" {
				jobCtx.Topic = topic
			}

			// Call OnJobStart hook
			if hooks.OnJobStart != nil {
				hooks.OnJobStart(jobCtx)
			}

			// Execute the actual handler
			msgs, err := h(msg)

			// Calculate duration
			jobCtx.Duration = time.Since(startTime)

			// Call appropriate completion hook
			if err != nil {
				if hooks.OnJobError != nil {
					hooks.OnJobError(jobCtx, err)
				}
			} else {
				if hooks.OnJobDone != nil {
					hooks.OnJobDone(jobCtx)
				}
			}

			return msgs, err
		}
	}
}

// LoggingHooks returns pre-built hooks that log job lifecycle events.
func LoggingHooks(logger interface {
	Info(msg string, fields map[string]interface{})
	Error(msg string, err error, fields map[string]interface{})
}) JobHooks {
	return JobHooks{
		OnJobStart: func(ctx JobContext) {
			logger.Info("Job started", map[string]interface{}{
				"handler":      ctx.HandlerName,
				"topic":        ctx.Topic,
				"message_uuid": ctx.MessageUUID,
				"retry_count":  ctx.RetryCount,
			})
		},
		OnJobDone: func(ctx JobContext) {
			logger.Info("Job completed", map[string]interface{}{
				"handler":      ctx.HandlerName,
				"topic":        ctx.Topic,
				"message_uuid": ctx.MessageUUID,
				"duration_ms":  ctx.Duration.Milliseconds(),
			})
		},
		OnJobError: func(ctx JobContext, err error) {
			logger.Error("Job failed", err, map[string]interface{}{
				"handler":      ctx.HandlerName,
				"topic":        ctx.Topic,
				"message_uuid": ctx.MessageUUID,
				"duration_ms":  ctx.Duration.Milliseconds(),
				"retry_count":  ctx.RetryCount,
			})
		},
	}
}

// MetricsHooks returns pre-built hooks that record job metrics.
func MetricsHooks(onStart, onDone, onError func(handlerName, topic string)) JobHooks {
	return JobHooks{
		OnJobStart: func(ctx JobContext) {
			if onStart != nil {
				onStart(ctx.HandlerName, ctx.Topic)
			}
		},
		OnJobDone: func(ctx JobContext) {
			if onDone != nil {
				onDone(ctx.HandlerName, ctx.Topic)
			}
		},
		OnJobError: func(ctx JobContext, err error) {
			if onError != nil {
				onError(ctx.HandlerName, ctx.Topic)
			}
		},
	}
}

// AlertingHooks returns pre-built hooks that trigger alerts on job errors.
func AlertingHooks(alertFunc func(ctx JobContext, err error)) JobHooks {
	return JobHooks{
		OnJobError: alertFunc,
	}
}
