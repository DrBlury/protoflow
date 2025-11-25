package runtime

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobHooks_OnJobStart(t *testing.T) {
	var called bool
	var capturedCtx JobContext

	hooks := JobHooks{
		OnJobStart: func(ctx JobContext) {
			called = true
			capturedCtx = ctx
		},
	}

	mw := jobHooksMiddleware(hooks)
	handler := mw(func(msg *message.Message) ([]*message.Message, error) {
		return nil, nil
	})

	msg := message.NewMessage("test-uuid", []byte("payload"))
	msg.SetContext(context.Background())

	_, err := handler(msg)
	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, "test-uuid", capturedCtx.MessageUUID)
	assert.False(t, capturedCtx.StartedAt.IsZero())
}

func TestJobHooks_OnJobDone(t *testing.T) {
	var called bool
	var capturedCtx JobContext

	hooks := JobHooks{
		OnJobDone: func(ctx JobContext) {
			called = true
			capturedCtx = ctx
		},
	}

	mw := jobHooksMiddleware(hooks)
	handler := mw(func(msg *message.Message) ([]*message.Message, error) {
		time.Sleep(10 * time.Millisecond)
		return nil, nil
	})

	msg := message.NewMessage("test-uuid", []byte("payload"))
	msg.SetContext(context.Background())

	_, err := handler(msg)
	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, "test-uuid", capturedCtx.MessageUUID)
	assert.True(t, capturedCtx.Duration >= 10*time.Millisecond)
}

func TestJobHooks_OnJobError(t *testing.T) {
	var called bool
	var capturedCtx JobContext
	var capturedErr error
	expectedErr := errors.New("handler error")

	hooks := JobHooks{
		OnJobError: func(ctx JobContext, err error) {
			called = true
			capturedCtx = ctx
			capturedErr = err
		},
	}

	mw := jobHooksMiddleware(hooks)
	handler := mw(func(msg *message.Message) ([]*message.Message, error) {
		return nil, expectedErr
	})

	msg := message.NewMessage("test-uuid", []byte("payload"))
	msg.SetContext(context.Background())

	_, err := handler(msg)
	assert.Error(t, err)
	assert.True(t, called)
	assert.Equal(t, "test-uuid", capturedCtx.MessageUUID)
	assert.Equal(t, expectedErr, capturedErr)
}

func TestJobHooks_MetadataExtraction(t *testing.T) {
	var capturedCtx JobContext

	hooks := JobHooks{
		OnJobStart: func(ctx JobContext) {
			capturedCtx = ctx
		},
	}

	mw := jobHooksMiddleware(hooks)
	handler := mw(func(msg *message.Message) ([]*message.Message, error) {
		return nil, nil
	})

	msg := message.NewMessage("test-uuid", []byte("payload"))
	msg.Metadata.Set("protoflow_handler", "TestHandler")
	msg.Metadata.Set("protoflow_topic", "test-topic")
	msg.Metadata.Set("protoflow_retry_count", "3")
	msg.SetContext(context.Background())

	_, err := handler(msg)
	require.NoError(t, err)
	assert.Equal(t, "TestHandler", capturedCtx.HandlerName)
	assert.Equal(t, "test-topic", capturedCtx.Topic)
	assert.Equal(t, 3, capturedCtx.RetryCount)
}

func TestJobHooks_Merge(t *testing.T) {
	var calls []string
	var mu sync.Mutex

	hooks1 := JobHooks{
		OnJobStart: func(ctx JobContext) {
			mu.Lock()
			calls = append(calls, "start1")
			mu.Unlock()
		},
		OnJobDone: func(ctx JobContext) {
			mu.Lock()
			calls = append(calls, "done1")
			mu.Unlock()
		},
		OnJobError: func(ctx JobContext, err error) {
			mu.Lock()
			calls = append(calls, "error1")
			mu.Unlock()
		},
	}

	hooks2 := JobHooks{
		OnJobStart: func(ctx JobContext) {
			mu.Lock()
			calls = append(calls, "start2")
			mu.Unlock()
		},
		OnJobDone: func(ctx JobContext) {
			mu.Lock()
			calls = append(calls, "done2")
			mu.Unlock()
		},
		OnJobError: func(ctx JobContext, err error) {
			mu.Lock()
			calls = append(calls, "error2")
			mu.Unlock()
		},
	}

	merged := hooks1.Merge(hooks2)

	// Test merged OnJobStart
	calls = nil
	mw := jobHooksMiddleware(merged)
	handler := mw(func(msg *message.Message) ([]*message.Message, error) {
		return nil, nil
	})

	msg := message.NewMessage("test-uuid", []byte("payload"))
	msg.SetContext(context.Background())
	_, _ = handler(msg)

	assert.Contains(t, calls, "start1")
	assert.Contains(t, calls, "start2")
	assert.Contains(t, calls, "done1")
	assert.Contains(t, calls, "done2")
}

func TestJobHooks_MergePartial(t *testing.T) {
	var calls []string

	hooks1 := JobHooks{
		OnJobStart: func(ctx JobContext) {
			calls = append(calls, "start1")
		},
	}

	hooks2 := JobHooks{
		OnJobDone: func(ctx JobContext) {
			calls = append(calls, "done2")
		},
	}

	merged := hooks1.Merge(hooks2)

	mw := jobHooksMiddleware(merged)
	handler := mw(func(msg *message.Message) ([]*message.Message, error) {
		return nil, nil
	})

	msg := message.NewMessage("test-uuid", []byte("payload"))
	msg.SetContext(context.Background())
	_, _ = handler(msg)

	assert.Contains(t, calls, "start1")
	assert.Contains(t, calls, "done2")
}

func TestJobHooksMiddleware_Registration(t *testing.T) {
	hooks := JobHooks{
		OnJobStart: func(ctx JobContext) {},
	}

	reg := JobHooksMiddleware(hooks)
	assert.Equal(t, "job_hooks", reg.Name)
	assert.NotNil(t, reg.Builder)
}

func TestLoggingHooks(t *testing.T) {
	var infoCalls []string
	var errorCalls []string

	logger := &hooksTestLogger{
		infoFunc: func(msg string, fields map[string]interface{}) {
			infoCalls = append(infoCalls, msg)
		},
		errorFunc: func(msg string, err error, fields map[string]interface{}) {
			errorCalls = append(errorCalls, msg)
		},
	}

	hooks := LoggingHooks(logger)

	// Test start and done
	hooks.OnJobStart(JobContext{HandlerName: "test"})
	hooks.OnJobDone(JobContext{HandlerName: "test"})

	assert.Contains(t, infoCalls, "Job started")
	assert.Contains(t, infoCalls, "Job completed")

	// Test error
	hooks.OnJobError(JobContext{HandlerName: "test"}, errors.New("test error"))
	assert.Contains(t, errorCalls, "Job failed")
}

func TestMetricsHooks(t *testing.T) {
	var startCalls, doneCalls, errorCalls int

	hooks := MetricsHooks(
		func(handler, topic string) { startCalls++ },
		func(handler, topic string) { doneCalls++ },
		func(handler, topic string) { errorCalls++ },
	)

	hooks.OnJobStart(JobContext{})
	hooks.OnJobDone(JobContext{})
	hooks.OnJobError(JobContext{}, errors.New("test"))

	assert.Equal(t, 1, startCalls)
	assert.Equal(t, 1, doneCalls)
	assert.Equal(t, 1, errorCalls)
}

func TestAlertingHooks(t *testing.T) {
	var alertCalled bool
	var capturedErr error

	hooks := AlertingHooks(func(ctx JobContext, err error) {
		alertCalled = true
		capturedErr = err
	})

	expectedErr := errors.New("alert error")
	hooks.OnJobError(JobContext{}, expectedErr)

	assert.True(t, alertCalled)
	assert.Equal(t, expectedErr, capturedErr)
}

type hooksTestLogger struct {
	infoFunc  func(msg string, fields map[string]interface{})
	errorFunc func(msg string, err error, fields map[string]interface{})
}

func (m *hooksTestLogger) Info(msg string, fields map[string]interface{}) {
	if m.infoFunc != nil {
		m.infoFunc(msg, fields)
	}
}

func (m *hooksTestLogger) Error(msg string, err error, fields map[string]interface{}) {
	if m.errorFunc != nil {
		m.errorFunc(msg, err, fields)
	}
}
