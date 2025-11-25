package runtime

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// DLQMetrics tracks dead letter queue statistics.
type DLQMetrics struct {
	mu sync.RWMutex

	// Per-topic counts
	topicCounts map[string]*DLQTopicMetrics

	// Prometheus collectors
	messagesTotal   *prometheus.CounterVec
	messagesCurrent *prometheus.GaugeVec
	replayedTotal   *prometheus.CounterVec
	purgedTotal     *prometheus.CounterVec
	ageSecondsHist  *prometheus.HistogramVec
	retryCountHist  *prometheus.HistogramVec

	registerer prometheus.Registerer
	registered bool
}

// DLQTopicMetrics holds metrics for a specific topic's DLQ.
type DLQTopicMetrics struct {
	MessagesReceived uint64    `json:"messages_received"`
	MessagesCurrent  uint64    `json:"messages_current"`
	MessagesReplayed uint64    `json:"messages_replayed"`
	MessagesPurged   uint64    `json:"messages_purged"`
	OldestMessageAt  time.Time `json:"oldest_message_at,omitempty"`
	NewestMessageAt  time.Time `json:"newest_message_at,omitempty"`
	AvgRetryCount    float64   `json:"avg_retry_count"`
	LastUpdatedAt    time.Time `json:"last_updated_at"`
}

// DLQMetricsSnapshot provides a point-in-time view of DLQ metrics.
type DLQMetricsSnapshot struct {
	TotalMessages uint64                      `json:"total_messages"`
	TotalReplayed uint64                      `json:"total_replayed"`
	TotalPurged   uint64                      `json:"total_purged"`
	TopicMetrics  map[string]*DLQTopicMetrics `json:"topic_metrics"`
	CollectedAt   time.Time                   `json:"collected_at"`
}

// newDLQCounterVec creates a new counter vec with standard protoflow/dlq namespace.
func newDLQCounterVec(name, help string, labels []string) *prometheus.CounterVec {
	return prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "protoflow",
			Subsystem: "dlq",
			Name:      name,
			Help:      help,
		},
		labels,
	)
}

// newDLQGaugeVec creates a new gauge vec with standard protoflow/dlq namespace.
func newDLQGaugeVec(name, help string, labels []string) *prometheus.GaugeVec {
	return prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "protoflow",
			Subsystem: "dlq",
			Name:      name,
			Help:      help,
		},
		labels,
	)
}

// newDLQHistogramVec creates a new histogram vec with standard protoflow/dlq namespace.
func newDLQHistogramVec(name, help string, buckets []float64, labels []string) *prometheus.HistogramVec {
	return prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "protoflow",
			Subsystem: "dlq",
			Name:      name,
			Help:      help,
			Buckets:   buckets,
		},
		labels,
	)
}

// NewDLQMetrics creates a new DLQ metrics collector.
func NewDLQMetrics(registerer prometheus.Registerer) *DLQMetrics {
	if registerer == nil {
		registerer = prometheus.DefaultRegisterer
	}

	return &DLQMetrics{
		topicCounts:     make(map[string]*DLQTopicMetrics),
		registerer:      registerer,
		messagesTotal:   newDLQCounterVec("messages_total", "Total number of messages sent to the dead letter queue", []string{"topic", "handler"}),
		messagesCurrent: newDLQGaugeVec("messages_current", "Current number of messages in the dead letter queue", []string{"topic"}),
		replayedTotal:   newDLQCounterVec("replayed_total", "Total number of messages replayed from the dead letter queue", []string{"topic"}),
		purgedTotal:     newDLQCounterVec("purged_total", "Total number of messages purged from the dead letter queue", []string{"topic"}),
		ageSecondsHist:  newDLQHistogramVec("message_age_seconds", "Age of messages when moved to DLQ (time since first attempt)", []float64{1, 5, 10, 30, 60, 300, 600, 1800, 3600}, []string{"topic"}),
		retryCountHist:  newDLQHistogramVec("retry_count", "Number of retries before message was moved to DLQ", []float64{1, 2, 3, 5, 10, 20}, []string{"topic"}),
	}
}

// Register registers the Prometheus collectors. Safe to call multiple times.
func (m *DLQMetrics) Register() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.registered {
		return nil
	}

	collectors := []prometheus.Collector{
		m.messagesTotal,
		m.messagesCurrent,
		m.replayedTotal,
		m.purgedTotal,
		m.ageSecondsHist,
		m.retryCountHist,
	}

	for _, c := range collectors {
		if err := m.registerer.Register(c); err != nil {
			// Check if it's already registered (not an error)
			if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
				return err
			}
		}
	}

	m.registered = true
	return nil
}

// RecordMessageToDLQ records a message being added to the DLQ.
func (m *DLQMetrics) RecordMessageToDLQ(topic, handler string, retryCount int, messageAge time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Update internal metrics
	metrics := m.getOrCreateTopicMetrics(topic)
	metrics.MessagesReceived++
	metrics.MessagesCurrent++
	metrics.LastUpdatedAt = time.Now()
	if metrics.OldestMessageAt.IsZero() {
		metrics.OldestMessageAt = time.Now()
	}
	metrics.NewestMessageAt = time.Now()

	// Update average retry count (rolling average)
	total := metrics.MessagesReceived
	metrics.AvgRetryCount = ((metrics.AvgRetryCount * float64(total-1)) + float64(retryCount)) / float64(total)

	// Update Prometheus metrics
	m.messagesTotal.WithLabelValues(topic, handler).Inc()
	m.messagesCurrent.WithLabelValues(topic).Set(float64(metrics.MessagesCurrent))
	m.ageSecondsHist.WithLabelValues(topic).Observe(messageAge.Seconds())
	m.retryCountHist.WithLabelValues(topic).Observe(float64(retryCount))
}

// RecordMessageReplayed records a message being replayed from the DLQ.
func (m *DLQMetrics) RecordMessageReplayed(topic string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	metrics := m.getOrCreateTopicMetrics(topic)
	metrics.MessagesReplayed++
	if metrics.MessagesCurrent > 0 {
		metrics.MessagesCurrent--
	}
	metrics.LastUpdatedAt = time.Now()

	m.replayedTotal.WithLabelValues(topic).Inc()
	m.messagesCurrent.WithLabelValues(topic).Set(float64(metrics.MessagesCurrent))
}

// RecordMessagesPurged records messages being purged from the DLQ.
func (m *DLQMetrics) RecordMessagesPurged(topic string, count int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	metrics := m.getOrCreateTopicMetrics(topic)
	metrics.MessagesPurged += uint64(count)
	if metrics.MessagesCurrent >= uint64(count) {
		metrics.MessagesCurrent -= uint64(count)
	} else {
		metrics.MessagesCurrent = 0
	}
	metrics.LastUpdatedAt = time.Now()

	m.purgedTotal.WithLabelValues(topic).Add(float64(count))
	m.messagesCurrent.WithLabelValues(topic).Set(float64(metrics.MessagesCurrent))
}

// SetCurrentCount directly sets the current message count (for sync with external systems).
func (m *DLQMetrics) SetCurrentCount(topic string, count uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	metrics := m.getOrCreateTopicMetrics(topic)
	metrics.MessagesCurrent = count
	metrics.LastUpdatedAt = time.Now()

	m.messagesCurrent.WithLabelValues(topic).Set(float64(count))
}

// GetSnapshot returns a point-in-time snapshot of all DLQ metrics.
func (m *DLQMetrics) GetSnapshot() DLQMetricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := DLQMetricsSnapshot{
		TopicMetrics: make(map[string]*DLQTopicMetrics),
		CollectedAt:  time.Now(),
	}

	for topic, metrics := range m.topicCounts {
		// Create a copy
		metricsCopy := &DLQTopicMetrics{
			MessagesReceived: metrics.MessagesReceived,
			MessagesCurrent:  metrics.MessagesCurrent,
			MessagesReplayed: metrics.MessagesReplayed,
			MessagesPurged:   metrics.MessagesPurged,
			OldestMessageAt:  metrics.OldestMessageAt,
			NewestMessageAt:  metrics.NewestMessageAt,
			AvgRetryCount:    metrics.AvgRetryCount,
			LastUpdatedAt:    metrics.LastUpdatedAt,
		}
		snapshot.TopicMetrics[topic] = metricsCopy
		snapshot.TotalMessages += metrics.MessagesCurrent
		snapshot.TotalReplayed += metrics.MessagesReplayed
		snapshot.TotalPurged += metrics.MessagesPurged
	}

	return snapshot
}

// GetTopicMetrics returns metrics for a specific topic.
func (m *DLQMetrics) GetTopicMetrics(topic string) *DLQTopicMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if metrics, ok := m.topicCounts[topic]; ok {
		// Return a copy
		return &DLQTopicMetrics{
			MessagesReceived: metrics.MessagesReceived,
			MessagesCurrent:  metrics.MessagesCurrent,
			MessagesReplayed: metrics.MessagesReplayed,
			MessagesPurged:   metrics.MessagesPurged,
			OldestMessageAt:  metrics.OldestMessageAt,
			NewestMessageAt:  metrics.NewestMessageAt,
			AvgRetryCount:    metrics.AvgRetryCount,
			LastUpdatedAt:    metrics.LastUpdatedAt,
		}
	}
	return nil
}

func (m *DLQMetrics) getOrCreateTopicMetrics(topic string) *DLQTopicMetrics {
	if metrics, ok := m.topicCounts[topic]; ok {
		return metrics
	}
	metrics := &DLQTopicMetrics{}
	m.topicCounts[topic] = metrics
	return metrics
}

// Reset resets all metrics (useful for testing).
func (m *DLQMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.topicCounts = make(map[string]*DLQTopicMetrics)
	m.messagesTotal.Reset()
	m.messagesCurrent.Reset()
	m.replayedTotal.Reset()
	m.purgedTotal.Reset()
	m.ageSecondsHist.Reset()
	m.retryCountHist.Reset()
}
