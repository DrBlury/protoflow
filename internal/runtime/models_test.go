package runtime

import (
	"errors"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"

	handlerpkg "github.com/drblury/protoflow/internal/runtime/handlers"
)

func TestHandlerStatsCollectsExtendedMetrics(t *testing.T) {
	stats := newHandlerStats("orders", "orders.created", "orders.audit", nil)
	instrumented := wrapHandlerWithStats(func(msg *message.Message) ([]*message.Message, error) {
		time.Sleep(5 * time.Millisecond)
		return nil, errors.New("publish failed")
	}, stats, nil)

	msg := message.NewMessage("id", []byte("demo"))
	msg.Metadata.Set(handlerpkg.MetadataKeyQueueDepth, "42")
	msg.Metadata.Set(handlerpkg.MetadataKeyEnqueuedAt, time.Now().Add(-1500*time.Millisecond).Format(time.RFC3339Nano))

	if _, err := instrumented(msg); err == nil {
		t.Fatalf("expected error from instrumented handler")
	}

	stats.mu.Lock()
	defer stats.mu.Unlock()

	if stats.MessagesProcessed != 1 {
		t.Fatalf("expected 1 processed message, got %d", stats.MessagesProcessed)
	}
	if stats.MessagesFailed != 1 {
		t.Fatalf("expected failure count to increment")
	}
	if stats.Backlog.LastQueueDepth != 42 {
		t.Fatalf("expected backlog depth to be recorded, got %d", stats.Backlog.LastQueueDepth)
	}
	if stats.Backlog.EstimatedLagMillis < 1400 {
		t.Fatalf("expected lag to be recorded, got %d", stats.Backlog.EstimatedLagMillis)
	}
	if stats.Errors.Other != 1 {
		t.Fatalf("expected error bucket to increment, got %+v", stats.Errors)
	}
	if len(stats.Dependencies) < 2 {
		t.Fatalf("expected subscriber and publisher dependency entries")
	}
	publisher := stats.Dependencies[1]
	if publisher.Status != DependencyStatusDegraded {
		t.Fatalf("expected publisher to be marked degraded, got %s", publisher.Status)
	}
	if stats.Throughput.TotalMessages != 1 {
		t.Fatalf("expected throughput total to track processed messages")
	}
	if stats.Latency.SampleSize == 0 {
		t.Fatalf("expected latency metrics to have samples")
	}
}
