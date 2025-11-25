package handlers

import (
	loggingpkg "github.com/drblury/protoflow/internal/runtime/logging"
	metadatapkg "github.com/drblury/protoflow/internal/runtime/metadata"
)

// MessageContextBase provides common functionality for all message context types.
// It holds the metadata and logger shared by JSON and Proto handlers.
type MessageContextBase struct {
	Metadata metadatapkg.Metadata
	Logger   loggingpkg.ServiceLogger
}

// CloneMetadata returns a copy of the current metadata map so handlers can safely
// mutate headers for outgoing events without touching the original map.
func (b MessageContextBase) CloneMetadata() metadatapkg.Metadata {
	return b.Metadata.Clone()
}

// Get retrieves a metadata value by key.
func (b MessageContextBase) Get(key string) string {
	return b.Metadata[key]
}

// CorrelationID returns the correlation ID from metadata, if present.
func (b MessageContextBase) CorrelationID() string {
	return b.Metadata[MetadataKeyCorrelationID]
}
