package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
)

const (
	metadataKeyQueueDepth = "protoflow_queue_depth"
	metadataKeyEnqueuedAt = "protoflow_enqueued_at"

	latencySampleSize    = 256
	throughputWindowSize = time.Minute
)

// UnprocessableEventError wraps payloads that failed validation or unmarshalling.
type UnprocessableEventError struct {
	eventMessage string
	err          error
}

func (e *UnprocessableEventError) Error() string {
	return "unprocessable event: " + e.eventMessage + " error: " + e.err.Error()
}

type HandlerStats struct {
	mu sync.Mutex `json:"-"`

	handlerName  string `json:"-"`
	consumeQueue string `json:"-"`
	publishQueue string `json:"-"`

	MessagesProcessed   uint64    `json:"messages_processed"`
	MessagesFailed      uint64    `json:"messages_failed"`
	TotalProcessingTime int64     `json:"total_processing_time_ns"`
	LastProcessedAt     time.Time `json:"last_processed_at"`

	Latency      LatencyMetrics     `json:"latency"`
	Throughput   ThroughputMetrics  `json:"throughput"`
	Errors       ErrorBreakdown     `json:"errors"`
	Resource     ResourceUsage      `json:"resource"`
	Backlog      BacklogMetrics     `json:"backlog"`
	Dependencies []DependencyHealth `json:"dependencies"`

	latencyWindow    *latencyWindow    `json:"-"`
	throughputWindow *throughputWindow `json:"-"`
	resourceSampler  *resourceTracker  `json:"-"`
	dependencyIndex  map[string]int    `json:"-"`
}

type HandlerInfo struct {
	Name         string        `json:"name"`
	ConsumeQueue string        `json:"consume_queue"`
	PublishQueue string        `json:"publish_queue"`
	Stats        *HandlerStats `json:"stats"`
}

type LatencyMetrics struct {
	AverageNs  int64 `json:"average_ns"`
	P50Ns      int64 `json:"p50_ns"`
	P95Ns      int64 `json:"p95_ns"`
	P99Ns      int64 `json:"p99_ns"`
	LastNs     int64 `json:"last_ns"`
	SampleSize int   `json:"sample_size"`
}

type ThroughputMetrics struct {
	CurrentRPS       float64 `json:"current_rps"`
	WindowSeconds    float64 `json:"window_seconds"`
	MessagesInWindow uint64  `json:"messages_in_window"`
	TotalMessages    uint64  `json:"total_messages"`
}

type ErrorBreakdown struct {
	Validation uint64 `json:"validation"`
	Transport  uint64 `json:"transport"`
	Downstream uint64 `json:"downstream"`
	Other      uint64 `json:"other"`
	LastError  string `json:"last_error,omitempty"`
}

type ResourceUsage struct {
	CPUPercent  float64 `json:"cpu_percent"`
	MemoryBytes uint64  `json:"memory_bytes"`
	Goroutines  int     `json:"goroutines"`
}

type BacklogMetrics struct {
	InFlight           uint64 `json:"in_flight"`
	MaxInFlight        uint64 `json:"max_in_flight"`
	LastQueueDepth     int64  `json:"last_queue_depth"`
	EstimatedLagMillis int64  `json:"estimated_lag_millis"`
}

type DependencyHealth struct {
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	LastChecked time.Time `json:"last_checked"`
	Details     string    `json:"details,omitempty"`
}

const (
	DependencyStatusUnknown  = "unknown"
	DependencyStatusHealthy  = "healthy"
	DependencyStatusDegraded = "degraded"
)

type ErrorCategory string

const (
	ErrorCategoryNone       ErrorCategory = "none"
	ErrorCategoryValidation ErrorCategory = "validation"
	ErrorCategoryTransport  ErrorCategory = "transport"
	ErrorCategoryDownstream ErrorCategory = "downstream"
	ErrorCategoryOther      ErrorCategory = "other"
)

type ErrorClassifier func(error) ErrorCategory

func newHandlerStats(name, consumeQueue, publishQueue string, sampler *resourceTracker) *HandlerStats {
	stats := &HandlerStats{
		handlerName:      name,
		consumeQueue:     consumeQueue,
		publishQueue:     publishQueue,
		resourceSampler:  sampler,
		latencyWindow:    newLatencyWindow(latencySampleSize),
		throughputWindow: newThroughputWindow(throughputWindowSize),
		Backlog: BacklogMetrics{
			LastQueueDepth:     -1,
			EstimatedLagMillis: -1,
		},
		dependencyIndex: make(map[string]int),
	}

	if consumeQueue != "" {
		stats.addDependency(fmt.Sprintf("subscriber:%s", consumeQueue))
	}
	if publishQueue != "" {
		stats.addDependency(fmt.Sprintf("publisher:%s", publishQueue))
	}

	return stats
}

func (h *HandlerStats) addDependency(name string) {
	h.Dependencies = append(h.Dependencies, DependencyHealth{
		Name:   name,
		Status: DependencyStatusUnknown,
	})
	if h.dependencyIndex == nil {
		h.dependencyIndex = make(map[string]int)
	}
	h.dependencyIndex[name] = len(h.Dependencies) - 1
}

type handlerInvocationContext struct {
	queueDepth     int64
	queueLagMillis int64
}

func (h *HandlerStats) onMessageStart(msg *message.Message) handlerInvocationContext {
	depth, lag := extractBacklogHints(msg)

	h.mu.Lock()
	defer h.mu.Unlock()

	h.Backlog.InFlight++
	if h.Backlog.InFlight > h.Backlog.MaxInFlight {
		h.Backlog.MaxInFlight = h.Backlog.InFlight
	}

	return handlerInvocationContext{
		queueDepth:     depth,
		queueLagMillis: lag,
	}
}

func (h *HandlerStats) onMessageFinish(ctx handlerInvocationContext, duration time.Duration, err error, classifier ErrorClassifier) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.Backlog.InFlight > 0 {
		h.Backlog.InFlight--
	}
	if ctx.queueDepth >= 0 {
		h.Backlog.LastQueueDepth = ctx.queueDepth
	}
	if ctx.queueLagMillis >= 0 {
		h.Backlog.EstimatedLagMillis = ctx.queueLagMillis
	}

	h.MessagesProcessed++
	if err != nil {
		h.MessagesFailed++
	}
	h.TotalProcessingTime += int64(duration)
	h.LastProcessedAt = time.Now().UTC()

	if h.latencyWindow != nil {
		h.latencyWindow.Add(duration)
		snapshot := h.latencyWindow.Snapshot()
		snapshot.LastNs = int64(duration)
		if h.MessagesProcessed > 0 {
			snapshot.AverageNs = h.TotalProcessingTime / int64(h.MessagesProcessed)
		}
		h.Latency = snapshot
	}

	if h.throughputWindow != nil {
		snapshot := h.throughputWindow.AddAndSnapshot(time.Now())
		h.Throughput.CurrentRPS = snapshot.CurrentRPS
		h.Throughput.WindowSeconds = snapshot.WindowSeconds
		h.Throughput.MessagesInWindow = uint64(snapshot.Count)
	}
	h.Throughput.TotalMessages = h.MessagesProcessed

	if classifier == nil {
		classifier = defaultErrorClassifier
	}
	category := classifier(err)
	h.Errors.Record(category, err)

	if h.resourceSampler != nil {
		h.Resource = h.resourceSampler.Snapshot()
	}

	h.setDependencyStatusLocked(fmt.Sprintf("subscriber:%s", h.consumeQueue), DependencyStatusHealthy, "")
	if h.publishQueue != "" {
		status := DependencyStatusHealthy
		details := ""
		if err != nil {
			status = DependencyStatusDegraded
			details = err.Error()
		}
		h.setDependencyStatusLocked(fmt.Sprintf("publisher:%s", h.publishQueue), status, details)
	}
}

func (h *HandlerStats) setDependencyStatusLocked(name, status, details string) {
	if name == "" {
		return
	}
	idx, ok := h.dependencyIndex[name]
	if !ok {
		h.Dependencies = append(h.Dependencies, DependencyHealth{Name: name})
		idx = len(h.Dependencies) - 1
		h.dependencyIndex[name] = idx
	}
	dep := h.Dependencies[idx]
	dep.Status = status
	dep.Details = details
	dep.LastChecked = time.Now().UTC()
	h.Dependencies[idx] = dep
}

func extractBacklogHints(msg *message.Message) (int64, int64) {
	if msg == nil {
		return -1, -1
	}
	depth := parseInt64Metadata(msg.Metadata, metadataKeyQueueDepth)
	lag := parseLagMetadata(msg.Metadata, metadataKeyEnqueuedAt)
	return depth, lag
}

func parseInt64Metadata(meta message.Metadata, key string) int64 {
	if meta == nil {
		return -1
	}
	val := meta.Get(key)
	if val == "" {
		return -1
	}
	parsed, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return -1
	}
	return parsed
}

func parseLagMetadata(meta message.Metadata, key string) int64 {
	if meta == nil {
		return -1
	}
	raw := meta.Get(key)
	if raw == "" {
		return -1
	}
	ts, err := time.Parse(time.RFC3339Nano, raw)
	if err != nil {
		return -1
	}
	lag := time.Since(ts).Milliseconds()
	if lag < 0 {
		return 0
	}
	return lag
}

func (h *HandlerStats) MarshalJSON() ([]byte, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	type Alias HandlerStats
	return json.Marshal((*Alias)(h))
}

func (e *ErrorBreakdown) Record(category ErrorCategory, err error) {
	switch category {
	case ErrorCategoryNone:
		if err == nil {
			return
		}
		e.Other++
	case ErrorCategoryValidation:
		e.Validation++
	case ErrorCategoryTransport:
		e.Transport++
	case ErrorCategoryDownstream:
		e.Downstream++
	default:
		e.Other++
	}
	if err != nil {
		e.LastError = err.Error()
	}
}

type latencyWindow struct {
	samples []int64
	next    int
	filled  int
	last    int64
}

func newLatencyWindow(size int) *latencyWindow {
	if size <= 0 {
		size = latencySampleSize
	}
	return &latencyWindow{samples: make([]int64, size)}
}

func (lw *latencyWindow) Add(d time.Duration) {
	if lw == nil || len(lw.samples) == 0 {
		return
	}
	lw.samples[lw.next] = int64(d)
	lw.last = int64(d)
	lw.next = (lw.next + 1) % len(lw.samples)
	if lw.filled < len(lw.samples) {
		lw.filled++
	}
}

func (lw *latencyWindow) Snapshot() LatencyMetrics {
	var metrics LatencyMetrics
	if lw == nil {
		return metrics
	}
	if lw.filled == 0 {
		metrics.LastNs = lw.last
		return metrics
	}
	samples := make([]int64, lw.filled)
	for i := 0; i < lw.filled; i++ {
		idx := lw.next - lw.filled + i
		if idx < 0 {
			idx += len(lw.samples)
		}
		samples[i] = lw.samples[idx]
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	metrics.SampleSize = lw.filled
	metrics.P50Ns = percentile(samples, 0.50)
	metrics.P95Ns = percentile(samples, 0.95)
	metrics.P99Ns = percentile(samples, 0.99)
	var sum int64
	for _, v := range samples {
		sum += v
	}
	metrics.AverageNs = sum / int64(len(samples))
	metrics.LastNs = lw.last
	return metrics
}

func percentile(samples []int64, quantile float64) int64 {
	if len(samples) == 0 {
		return 0
	}
	if quantile <= 0 {
		return samples[0]
	}
	if quantile >= 1 {
		return samples[len(samples)-1]
	}
	pos := quantile * float64(len(samples)-1)
	lower := int(math.Floor(pos))
	upper := int(math.Ceil(pos))
	if lower == upper {
		return samples[lower]
	}
	frac := pos - float64(lower)
	return samples[lower] + int64(float64(samples[upper]-samples[lower])*frac)
}

type throughputWindow struct {
	horizon time.Duration
	samples []time.Time
}

type throughputSnapshot struct {
	Count         int
	WindowSeconds float64
	CurrentRPS    float64
}

func newThroughputWindow(horizon time.Duration) *throughputWindow {
	return &throughputWindow{
		horizon: horizon,
		samples: make([]time.Time, 0, 64),
	}
}

func (tw *throughputWindow) AddAndSnapshot(now time.Time) throughputSnapshot {
	if tw == nil {
		return throughputSnapshot{}
	}
	tw.samples = append(tw.samples, now)
	tw.cleanup(now)
	return tw.snapshot(now)
}

func (tw *throughputWindow) cleanup(now time.Time) {
	if tw == nil || len(tw.samples) == 0 {
		return
	}
	cutoff := now.Add(-tw.horizon)
	idx := 0
	for idx < len(tw.samples) && tw.samples[idx].Before(cutoff) {
		idx++
	}
	if idx > 0 {
		copy(tw.samples, tw.samples[idx:])
		tw.samples = tw.samples[:len(tw.samples)-idx]
	}
}

func (tw *throughputWindow) snapshot(now time.Time) throughputSnapshot {
	if tw == nil || len(tw.samples) == 0 {
		return throughputSnapshot{}
	}
	span := now.Sub(tw.samples[0])
	if span <= 0 {
		span = time.Nanosecond
	}
	count := len(tw.samples)
	return throughputSnapshot{
		Count:         count,
		WindowSeconds: span.Seconds(),
		CurrentRPS:    float64(count) / span.Seconds(),
	}
}

func defaultErrorClassifier(err error) ErrorCategory {
	if err == nil {
		return ErrorCategoryNone
	}
	var unprocessable *UnprocessableEventError
	if errors.As(err, &unprocessable) {
		return ErrorCategoryValidation
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return ErrorCategoryDownstream
	}
	return ErrorCategoryOther
}
