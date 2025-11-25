package runtime

import (
	"testing"
	"time"
)

func TestResourceTracker_Snapshot(t *testing.T) {
	tracker := newResourceTracker()

	// First snapshot establishes baseline
	snap1 := tracker.Snapshot()

	// Initial CPU percent should be 0 (no previous sample)
	if snap1.CPUPercent != 0 {
		t.Errorf("expected 0 CPU percent on first snapshot, got %f", snap1.CPUPercent)
	}

	// Memory and goroutines should be non-zero
	if snap1.MemoryBytes == 0 {
		t.Error("expected non-zero memory bytes")
	}
	if snap1.Goroutines == 0 {
		t.Error("expected non-zero goroutine count")
	}

	// Allow some time to pass for CPU calculation
	time.Sleep(10 * time.Millisecond)

	// Second snapshot should have CPU data
	snap2 := tracker.Snapshot()

	// CPU percent should be >= 0 (might be 0 if idle)
	if snap2.CPUPercent < 0 {
		t.Errorf("expected non-negative CPU percent, got %f", snap2.CPUPercent)
	}
}

func TestResourceTracker_SnapshotNilTracker(t *testing.T) {
	var tracker *resourceTracker

	// Should return empty usage without panicking
	snap := tracker.Snapshot()

	if snap.CPUPercent != 0 || snap.MemoryBytes != 0 || snap.Goroutines != 0 {
		t.Errorf("expected zero ResourceUsage for nil tracker, got %+v", snap)
	}
}

func TestResourceTracker_SnapshotEmptySamples(t *testing.T) {
	tracker := &resourceTracker{
		samples: nil, // Empty samples slice
	}

	// Should handle empty samples gracefully
	snap := tracker.Snapshot()

	// Should still return memory and goroutine data
	if snap.MemoryBytes == 0 {
		t.Error("expected non-zero memory bytes even with empty samples")
	}
}
