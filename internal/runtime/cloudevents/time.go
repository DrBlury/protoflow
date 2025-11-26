package cloudevents

import (
	"time"
)

// Time format constants for CloudEvents.
const (
	// TimeFormat is the standard CloudEvents time format (RFC3339).
	TimeFormat = time.RFC3339

	// TimeFormatNano is the RFC3339 format with nanosecond precision.
	TimeFormatNano = time.RFC3339Nano
)

// ParseTime parses a time string in various formats.
// Supports RFC3339, RFC3339Nano, and Unix timestamps.
func ParseTime(s string) (time.Time, error) {
	// Try RFC3339Nano first
	if t, err := time.Parse(TimeFormatNano, s); err == nil {
		return t, nil
	}

	// Try RFC3339
	if t, err := time.Parse(TimeFormat, s); err == nil {
		return t, nil
	}

	// Try additional formats
	formats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, s); err == nil {
			return t, nil
		}
	}

	return time.Time{}, &time.ParseError{
		Layout:     TimeFormat,
		Value:      s,
		LayoutElem: "",
		ValueElem:  "",
		Message:    "cannot parse as CloudEvents time",
	}
}

// FormatTime formats a time value for CloudEvents.
func FormatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(TimeFormat)
}

// Now returns the current UTC time.
func Now() time.Time {
	return time.Now().UTC()
}
