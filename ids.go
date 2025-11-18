package protoflow

import (
	mathrand "math/rand"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

var (
	entropyMu sync.Mutex
	entropy   = ulid.Monotonic(mathrand.New(mathrand.NewSource(time.Now().UnixNano())), 0)
)

// CreateULID returns a time-sortable ULID encoded as a 26-character string.
func CreateULID() string {
	entropyMu.Lock()
	defer entropyMu.Unlock()

	id := ulid.MustNew(ulid.Timestamp(time.Now()), entropy)
	return id.String()
}
