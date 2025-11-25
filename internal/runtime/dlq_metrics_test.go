package runtime

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDLQMetrics_RecordMessageToDLQ(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewDLQMetrics(reg)
	require.NoError(t, m.Register())

	m.RecordMessageToDLQ("orders", "OrderHandler", 3, 5*time.Second)
	m.RecordMessageToDLQ("orders", "OrderHandler", 5, 10*time.Second)

	metrics := m.GetTopicMetrics("orders")
	require.NotNil(t, metrics)
	assert.Equal(t, uint64(2), metrics.MessagesReceived)
	assert.Equal(t, uint64(2), metrics.MessagesCurrent)
	assert.Equal(t, 4.0, metrics.AvgRetryCount) // (3+5)/2
	assert.False(t, metrics.OldestMessageAt.IsZero())
	assert.False(t, metrics.NewestMessageAt.IsZero())
}

func TestDLQMetrics_RecordMessageReplayed(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewDLQMetrics(reg)
	require.NoError(t, m.Register())

	m.RecordMessageToDLQ("orders", "OrderHandler", 3, 5*time.Second)
	m.RecordMessageToDLQ("orders", "OrderHandler", 3, 5*time.Second)
	m.RecordMessageReplayed("orders")

	metrics := m.GetTopicMetrics("orders")
	require.NotNil(t, metrics)
	assert.Equal(t, uint64(2), metrics.MessagesReceived)
	assert.Equal(t, uint64(1), metrics.MessagesCurrent)
	assert.Equal(t, uint64(1), metrics.MessagesReplayed)
}

func TestDLQMetrics_RecordMessagesPurged(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewDLQMetrics(reg)
	require.NoError(t, m.Register())

	m.RecordMessageToDLQ("orders", "OrderHandler", 3, 5*time.Second)
	m.RecordMessageToDLQ("orders", "OrderHandler", 3, 5*time.Second)
	m.RecordMessageToDLQ("orders", "OrderHandler", 3, 5*time.Second)
	m.RecordMessagesPurged("orders", 2)

	metrics := m.GetTopicMetrics("orders")
	require.NotNil(t, metrics)
	assert.Equal(t, uint64(3), metrics.MessagesReceived)
	assert.Equal(t, uint64(1), metrics.MessagesCurrent)
	assert.Equal(t, uint64(2), metrics.MessagesPurged)
}

func TestDLQMetrics_SetCurrentCount(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewDLQMetrics(reg)
	require.NoError(t, m.Register())

	m.SetCurrentCount("orders", 42)

	metrics := m.GetTopicMetrics("orders")
	require.NotNil(t, metrics)
	assert.Equal(t, uint64(42), metrics.MessagesCurrent)
}

func TestDLQMetrics_GetSnapshot(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewDLQMetrics(reg)
	require.NoError(t, m.Register())

	m.RecordMessageToDLQ("orders", "OrderHandler", 3, 5*time.Second)
	m.RecordMessageToDLQ("payments", "PaymentHandler", 2, 3*time.Second)
	m.RecordMessageReplayed("orders")

	snapshot := m.GetSnapshot()
	assert.Equal(t, uint64(1), snapshot.TotalMessages) // 1 in orders (replayed 1), 1 in payments
	assert.Equal(t, uint64(1), snapshot.TotalReplayed)
	assert.Len(t, snapshot.TopicMetrics, 2)
	assert.False(t, snapshot.CollectedAt.IsZero())
}

func TestDLQMetrics_GetTopicMetrics_NonExistent(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewDLQMetrics(reg)

	metrics := m.GetTopicMetrics("nonexistent")
	assert.Nil(t, metrics)
}

func TestDLQMetrics_Reset(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewDLQMetrics(reg)
	require.NoError(t, m.Register())

	m.RecordMessageToDLQ("orders", "OrderHandler", 3, 5*time.Second)
	m.Reset()

	snapshot := m.GetSnapshot()
	assert.Empty(t, snapshot.TopicMetrics)
}

func TestDLQMetrics_Register_Idempotent(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewDLQMetrics(reg)

	require.NoError(t, m.Register())
	require.NoError(t, m.Register()) // Should not error on double registration
}

func TestDLQMetrics_NilRegisterer(t *testing.T) {
	m := NewDLQMetrics(nil)
	assert.NotNil(t, m)
	// Should use default registerer - don't actually register in test to avoid conflicts
}

func TestDLQMetrics_PurgeMoreThanCurrent(t *testing.T) {
	reg := prometheus.NewRegistry()
	m := NewDLQMetrics(reg)
	require.NoError(t, m.Register())

	m.RecordMessageToDLQ("orders", "OrderHandler", 3, 5*time.Second)
	m.RecordMessagesPurged("orders", 10) // Purge more than exists

	metrics := m.GetTopicMetrics("orders")
	require.NotNil(t, metrics)
	assert.Equal(t, uint64(0), metrics.MessagesCurrent) // Should not go negative
	assert.Equal(t, uint64(10), metrics.MessagesPurged)
}
