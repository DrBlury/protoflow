package runtime

import idspkg "github.com/drblury/protoflow/internal/runtime/ids"

// CreateULID returns a time-sortable ULID encoded as a 26-character string.
func CreateULID() string {
	return idspkg.CreateULID()
}
