package ids

import (
	"sync"
	"testing"

	"github.com/oklog/ulid/v2"
)

func TestCreateULIDSequentialOrdering(t *testing.T) {
	const total = 100
	ids := make([]string, total)
	for i := 0; i < total; i++ {
		ids[i] = CreateULID()
	}

	for i := 0; i < total; i++ {
		if len(ids[i]) != 26 {
			t.Fatalf("expected ULID length 26, got %d", len(ids[i]))
		}
		if _, err := ulid.Parse(ids[i]); err != nil {
			t.Fatalf("expected valid ULID, got %v", err)
		}
	}

	for i := 1; i < total; i++ {
		if ids[i-1] >= ids[i] {
			t.Fatalf("expected ULIDs to be strictly increasing, %s >= %s", ids[i-1], ids[i])
		}
	}
}

func TestCreateULIDConcurrentUniqueness(t *testing.T) {
	const goroutines = 10
	const perGoroutine = 20

	var (
		wg   sync.WaitGroup
		mu   sync.Mutex
		seen = make(map[string]struct{})
	)

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < perGoroutine; j++ {
				id := CreateULID()
				if len(id) != 26 {
					t.Errorf("expected ULID length 26, got %d", len(id))
				}
				mu.Lock()
				if _, ok := seen[id]; ok {
					t.Errorf("duplicate ULID generated: %s", id)
				} else {
					seen[id] = struct{}{}
				}
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	expected := goroutines * perGoroutine
	if len(seen) != expected {
		t.Fatalf("expected %d unique ULIDs, got %d", expected, len(seen))
	}
}
