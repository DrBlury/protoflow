package handlers

import (
	"io"
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"

	loggingpkg "github.com/drblury/protoflow/internal/runtime/logging"
	metadatapkg "github.com/drblury/protoflow/internal/runtime/metadata"
)

func TestMessageContextBase_Get(t *testing.T) {
	metadata := metadatapkg.Metadata{
		"key1": "value1",
		"key2": "value2",
	}

	logger := loggingpkg.NewSlogServiceLogger(slog.New(slog.NewTextHandler(io.Discard, nil)))

	ctx := MessageContextBase{
		Metadata: metadata,
		Logger:   logger,
	}

	assert.Equal(t, "value1", ctx.Get("key1"))
	assert.Equal(t, "value2", ctx.Get("key2"))
	assert.Equal(t, "", ctx.Get("nonexistent"))
}

func TestMessageContextBase_CorrelationID(t *testing.T) {
	tests := []struct {
		name     string
		metadata metadatapkg.Metadata
		want     string
	}{
		{
			name: "correlation ID present",
			metadata: metadatapkg.Metadata{
				MetadataKeyCorrelationID: "correlation-123",
			},
			want: "correlation-123",
		},
		{
			name:     "correlation ID absent",
			metadata: metadatapkg.Metadata{},
			want:     "",
		},
	}

	logger := loggingpkg.NewSlogServiceLogger(slog.New(slog.NewTextHandler(io.Discard, nil)))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := MessageContextBase{
				Metadata: tt.metadata,
				Logger:   logger,
			}
			assert.Equal(t, tt.want, ctx.CorrelationID())
		})
	}
}

func TestMessageContextBase_CloneMetadata(t *testing.T) {
	original := metadatapkg.Metadata{
		"key1": "value1",
		"key2": "value2",
	}

	logger := loggingpkg.NewSlogServiceLogger(slog.New(slog.NewTextHandler(io.Discard, nil)))

	ctx := MessageContextBase{
		Metadata: original,
		Logger:   logger,
	}

	cloned := ctx.CloneMetadata()

	// Verify clone has same values
	assert.Equal(t, "value1", cloned["key1"])
	assert.Equal(t, "value2", cloned["key2"])

	// Modify clone
	cloned["key1"] = "modified"
	cloned["key3"] = "new"

	// Verify original is unchanged
	assert.Equal(t, "value1", ctx.Metadata["key1"])
	assert.Equal(t, "", ctx.Metadata["key3"])
}
