// Package cloudevents provides CloudEvents v1.0 compatible event types and utilities
// for use within protoflow. This package implements the CloudEvents specification
// with protoflow-specific extensions for reliability semantics.
package cloudevents

import (
	"encoding/json"
	"fmt"
	"time"

	idspkg "github.com/drblury/protoflow/internal/runtime/ids"
)

// SpecVersion is the CloudEvents specification version implemented.
const SpecVersion = "1.0"

// Event represents a CloudEvents v1.0 compliant event with protoflow extensions.
// See https://github.com/cloudevents/spec/blob/v1.0/spec.md for specification details.
type Event struct {
	// Required attributes

	// SpecVersion is the version of the CloudEvents specification.
	// MUST be set to "1.0" for CloudEvents v1.0.
	SpecVersion string `json:"specversion"`

	// Type describes the type of event related to the originating occurrence.
	// Recommended format: <resource>.<action>[.v<version>]
	// Examples: customer.created, order.reserved.v2
	Type string `json:"type"`

	// Source identifies the context in which an event happened.
	// Often contains the service name or URI of the producer.
	Source string `json:"source"`

	// ID uniquely identifies the event. If not set, a ULID will be generated.
	ID string `json:"id"`

	// Optional attributes

	// Time is the timestamp when the occurrence happened.
	// If not set, the current time is used.
	Time time.Time `json:"time,omitempty"`

	// DataContentType describes the content type of the data attribute.
	// Common values: "application/json", "application/protobuf"
	DataContentType *string `json:"datacontenttype,omitempty"`

	// DataSchema identifies the schema that data adheres to.
	DataSchema *string `json:"dataschema,omitempty"`

	// Subject describes the subject of the event in the context of the source.
	Subject *string `json:"subject,omitempty"`

	// Data is the event payload. Can be any type that is JSON-serializable.
	Data any `json:"data,omitempty"`

	// DataBase64 contains base64-encoded binary data when Data cannot be
	// directly serialized (e.g., binary protobuf).
	DataBase64 *string `json:"data_base64,omitempty"`

	// Extensions contains CloudEvents extension attributes.
	// Protoflow uses extensions prefixed with "pf_" for reliability semantics.
	Extensions map[string]any `json:"extensions,omitempty"`
}

// New creates a new CloudEvent with required fields populated.
// ID is auto-generated using ULID, Time is set to current time.
func New(eventType, source string, data any) Event {
	return Event{
		SpecVersion: SpecVersion,
		Type:        eventType,
		Source:      source,
		ID:          idspkg.CreateULID(),
		Time:        time.Now().UTC(),
		Data:        data,
		Extensions:  make(map[string]any),
	}
}

// NewWithID creates a new CloudEvent with a specific ID.
func NewWithID(id, eventType, source string, data any) Event {
	evt := New(eventType, source, data)
	evt.ID = id
	return evt
}

// WithSubject sets the subject field and returns the event.
func (e Event) WithSubject(subject string) Event {
	e.Subject = &subject
	return e
}

// WithDataContentType sets the data content type and returns the event.
func (e Event) WithDataContentType(contentType string) Event {
	e.DataContentType = &contentType
	return e
}

// WithDataSchema sets the data schema and returns the event.
func (e Event) WithDataSchema(schema string) Event {
	e.DataSchema = &schema
	return e
}

// WithExtension sets an extension attribute and returns the event.
func (e Event) WithExtension(key string, value any) Event {
	if e.Extensions == nil {
		e.Extensions = make(map[string]any)
	}
	e.Extensions[key] = value
	return e
}

// GetExtension retrieves an extension value by key.
// Returns nil if the extension does not exist.
func (e Event) GetExtension(key string) any {
	if e.Extensions == nil {
		return nil
	}
	return e.Extensions[key]
}

// GetExtensionString retrieves an extension value as a string.
// Returns empty string if the extension does not exist or is not a string.
func (e Event) GetExtensionString(key string) string {
	v := e.GetExtension(key)
	if v == nil {
		return ""
	}
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}

// GetExtensionInt retrieves an extension value as an int.
// Returns 0 if the extension does not exist or cannot be converted.
func (e Event) GetExtensionInt(key string) int {
	v := e.GetExtension(key)
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	default:
		return 0
	}
}

// GetExtensionInt64 retrieves an extension value as an int64.
// Returns 0 if the extension does not exist or cannot be converted.
func (e Event) GetExtensionInt64(key string) int64 {
	v := e.GetExtension(key)
	if v == nil {
		return 0
	}
	switch n := v.(type) {
	case int:
		return int64(n)
	case int64:
		return n
	case float64:
		return int64(n)
	case json.Number:
		i, _ := n.Int64()
		return i
	default:
		return 0
	}
}

// GetExtensionBool retrieves an extension value as a bool.
// Returns false if the extension does not exist or is not a bool.
func (e Event) GetExtensionBool(key string) bool {
	v := e.GetExtension(key)
	if v == nil {
		return false
	}
	if b, ok := v.(bool); ok {
		return b
	}
	return false
}

// GetExtensionTime retrieves an extension value as a time.Time.
// Supports RFC3339 strings and unix timestamps (seconds).
// Returns zero time if the extension does not exist or cannot be parsed.
func (e Event) GetExtensionTime(key string) time.Time {
	v := e.GetExtension(key)
	if v == nil {
		return time.Time{}
	}

	switch t := v.(type) {
	case time.Time:
		return t
	case string:
		parsed, err := time.Parse(time.RFC3339, t)
		if err != nil {
			return time.Time{}
		}
		return parsed
	case int64:
		return time.Unix(t, 0)
	case float64:
		return time.Unix(int64(t), 0)
	case json.Number:
		i, _ := t.Int64()
		return time.Unix(i, 0)
	default:
		return time.Time{}
	}
}

// Validate checks that the event has all required CloudEvents attributes.
func (e Event) Validate() error {
	if e.SpecVersion == "" {
		return fmt.Errorf("specversion is required")
	}
	if e.SpecVersion != SpecVersion {
		return fmt.Errorf("specversion must be %q, got %q", SpecVersion, e.SpecVersion)
	}
	if e.Type == "" {
		return fmt.Errorf("type is required")
	}
	if e.Source == "" {
		return fmt.Errorf("source is required")
	}
	if e.ID == "" {
		return fmt.Errorf("id is required")
	}
	return nil
}

// Clone creates a deep copy of the event.
func (e Event) Clone() Event {
	cloned := e

	// Clone optional string pointers
	if e.DataContentType != nil {
		v := *e.DataContentType
		cloned.DataContentType = &v
	}
	if e.DataSchema != nil {
		v := *e.DataSchema
		cloned.DataSchema = &v
	}
	if e.Subject != nil {
		v := *e.Subject
		cloned.Subject = &v
	}
	if e.DataBase64 != nil {
		v := *e.DataBase64
		cloned.DataBase64 = &v
	}

	// Clone extensions map
	if e.Extensions != nil {
		cloned.Extensions = make(map[string]any, len(e.Extensions))
		for k, v := range e.Extensions {
			cloned.Extensions[k] = v
		}
	}

	return cloned
}

// MarshalJSON implements json.Marshaler for CloudEvents JSON format.
func (e Event) MarshalJSON() ([]byte, error) {
	// Create a map for the flattened CloudEvents JSON format
	m := make(map[string]any)

	// Required attributes
	m["specversion"] = e.SpecVersion
	m["type"] = e.Type
	m["source"] = e.Source
	m["id"] = e.ID

	// Optional attributes
	if !e.Time.IsZero() {
		m["time"] = e.Time.Format(time.RFC3339Nano)
	}
	if e.DataContentType != nil {
		m["datacontenttype"] = *e.DataContentType
	}
	if e.DataSchema != nil {
		m["dataschema"] = *e.DataSchema
	}
	if e.Subject != nil {
		m["subject"] = *e.Subject
	}
	if e.Data != nil {
		m["data"] = e.Data
	}
	if e.DataBase64 != nil {
		m["data_base64"] = *e.DataBase64
	}

	// Extensions are flattened into the top-level object
	for k, v := range e.Extensions {
		m[k] = v
	}

	return json.Marshal(m)
}

// UnmarshalJSON implements json.Unmarshaler for CloudEvents JSON format.
func (e *Event) UnmarshalJSON(data []byte) error {
	// First, unmarshal into a generic map
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}

	// Known CloudEvents attributes
	knownAttrs := map[string]bool{
		"specversion":     true,
		"type":            true,
		"source":          true,
		"id":              true,
		"time":            true,
		"datacontenttype": true,
		"dataschema":      true,
		"subject":         true,
		"data":            true,
		"data_base64":     true,
	}

	// Parse required attributes
	if raw, ok := m["specversion"]; ok {
		if err := json.Unmarshal(raw, &e.SpecVersion); err != nil {
			return fmt.Errorf("invalid specversion: %w", err)
		}
	}
	if raw, ok := m["type"]; ok {
		if err := json.Unmarshal(raw, &e.Type); err != nil {
			return fmt.Errorf("invalid type: %w", err)
		}
	}
	if raw, ok := m["source"]; ok {
		if err := json.Unmarshal(raw, &e.Source); err != nil {
			return fmt.Errorf("invalid source: %w", err)
		}
	}
	if raw, ok := m["id"]; ok {
		if err := json.Unmarshal(raw, &e.ID); err != nil {
			return fmt.Errorf("invalid id: %w", err)
		}
	}

	// Parse optional attributes
	if raw, ok := m["time"]; ok {
		var timeStr string
		if err := json.Unmarshal(raw, &timeStr); err != nil {
			return fmt.Errorf("invalid time: %w", err)
		}
		t, err := time.Parse(time.RFC3339Nano, timeStr)
		if err != nil {
			// Try RFC3339
			t, err = time.Parse(time.RFC3339, timeStr)
			if err != nil {
				return fmt.Errorf("invalid time format: %w", err)
			}
		}
		e.Time = t
	}
	if raw, ok := m["datacontenttype"]; ok {
		var v string
		if err := json.Unmarshal(raw, &v); err != nil {
			return fmt.Errorf("invalid datacontenttype: %w", err)
		}
		e.DataContentType = &v
	}
	if raw, ok := m["dataschema"]; ok {
		var v string
		if err := json.Unmarshal(raw, &v); err != nil {
			return fmt.Errorf("invalid dataschema: %w", err)
		}
		e.DataSchema = &v
	}
	if raw, ok := m["subject"]; ok {
		var v string
		if err := json.Unmarshal(raw, &v); err != nil {
			return fmt.Errorf("invalid subject: %w", err)
		}
		e.Subject = &v
	}
	if raw, ok := m["data"]; ok {
		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
			return fmt.Errorf("invalid data: %w", err)
		}
		e.Data = v
	}
	if raw, ok := m["data_base64"]; ok {
		var v string
		if err := json.Unmarshal(raw, &v); err != nil {
			return fmt.Errorf("invalid data_base64: %w", err)
		}
		e.DataBase64 = &v
	}

	// Everything else goes into extensions
	e.Extensions = make(map[string]any)
	for k, raw := range m {
		if knownAttrs[k] {
			continue
		}
		var v any
		if err := json.Unmarshal(raw, &v); err != nil {
			return fmt.Errorf("invalid extension %q: %w", k, err)
		}
		e.Extensions[k] = v
	}

	return nil
}
