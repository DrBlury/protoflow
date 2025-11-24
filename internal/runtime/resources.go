package runtime

import (
	"runtime"
	"runtime/metrics"
	"sync"
	"time"
)

// resourceTracker samples coarse CPU/memory usage for inclusion in handler stats snapshots.
type resourceTracker struct {
	mu             sync.Mutex
	samples        []metrics.Sample
	lastCPUSeconds float64
	lastSample     time.Time
	numCPU         float64
}

func newResourceTracker() *resourceTracker {
	return &resourceTracker{
		samples: []metrics.Sample{{Name: "/sched/cpu:seconds"}},
		numCPU:  float64(runtime.NumCPU()),
	}
}

func (r *resourceTracker) Snapshot() ResourceUsage {
	if r == nil {
		return ResourceUsage{}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if len(r.samples) == 0 {
		r.samples = []metrics.Sample{{Name: "/sched/cpu:seconds"}}
	}

	metrics.Read(r.samples)
	sample := r.samples[0]
	haveCPU := sample.Value.Kind() == metrics.KindFloat64
	var cpuSeconds float64
	if haveCPU {
		cpuSeconds = sample.Value.Float64()
	}
	now := time.Now()

	var cpuPercent float64
	if haveCPU && !r.lastSample.IsZero() {
		deltaCPU := cpuSeconds - r.lastCPUSeconds
		deltaWall := now.Sub(r.lastSample).Seconds()
		if deltaWall > 0 && r.numCPU > 0 {
			cpuPercent = (deltaCPU / deltaWall) / r.numCPU * 100
		}
	}

	if haveCPU {
		r.lastCPUSeconds = cpuSeconds
	}
	r.lastSample = now

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	return ResourceUsage{
		CPUPercent:  cpuPercent,
		MemoryBytes: mem.Alloc,
		Goroutines:  runtime.NumGoroutine(),
	}
}
