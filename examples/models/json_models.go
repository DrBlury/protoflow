package models

import "time"

// JSONIncoming represents a minimal JSON payload used across tests.
type JSONIncoming struct {
	ID int `json:"id"`
}

// JSONOutgoing captures the fields emitted by handlers in tests.
type JSONOutgoing struct {
	ID        int       `json:"id"`
	Processed time.Time `json:"processed"`
}

// IncomingMessage is an empty payload used when the body is irrelevant.
type IncomingMessage struct{}

// OutgoingMessage is an empty payload emitted when only metadata matters.
type OutgoingMessage struct{}
