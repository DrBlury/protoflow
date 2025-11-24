package ids

import (
	"crypto/rand"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

var (
	entropyMu sync.Mutex
	entropy   = ulid.Monotonic(rand.Reader, 0)
)

// CreateULID returns a time-sortable ULID encoded as a 26-character string.
func CreateULID() string {
	entropyMu.Lock()
	defer entropyMu.Unlock()

	id := ulid.MustNew(ulid.Timestamp(time.Now()), entropy)
	return id.String()
}
