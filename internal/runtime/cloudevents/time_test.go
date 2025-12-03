package cloudevents

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTime_RFC3339Nano(t *testing.T) {
	timeStr := "2024-01-01T12:30:45.123456789Z"
	parsed, err := ParseTime(timeStr)
	require.NoError(t, err)
	assert.False(t, parsed.IsZero())
	assert.Equal(t, 2024, parsed.Year())
	assert.Equal(t, time.January, parsed.Month())
	assert.Equal(t, 1, parsed.Day())
}

func TestParseTime_RFC3339(t *testing.T) {
	timeStr := "2024-01-01T12:30:45Z"
	parsed, err := ParseTime(timeStr)
	require.NoError(t, err)
	assert.False(t, parsed.IsZero())
	assert.Equal(t, 2024, parsed.Year())
	assert.Equal(t, 12, parsed.Hour())
	assert.Equal(t, 30, parsed.Minute())
	assert.Equal(t, 45, parsed.Second())
}

func TestParseTime_ISO8601(t *testing.T) {
	timeStr := "2024-01-01T12:30:45Z"
	parsed, err := ParseTime(timeStr)
	require.NoError(t, err)
	assert.False(t, parsed.IsZero())
}

func TestParseTime_DateOnly(t *testing.T) {
	timeStr := "2024-01-01"
	parsed, err := ParseTime(timeStr)
	require.NoError(t, err)
	assert.False(t, parsed.IsZero())
	assert.Equal(t, 2024, parsed.Year())
	assert.Equal(t, time.January, parsed.Month())
	assert.Equal(t, 1, parsed.Day())
}

func TestParseTime_WithoutTimezone(t *testing.T) {
	timeStr := "2024-01-01T12:30:45"
	parsed, err := ParseTime(timeStr)
	require.NoError(t, err)
	assert.False(t, parsed.IsZero())
}

func TestParseTime_SpaceSeparator(t *testing.T) {
	timeStr := "2024-01-01 12:30:45"
	parsed, err := ParseTime(timeStr)
	require.NoError(t, err)
	assert.False(t, parsed.IsZero())
	assert.Equal(t, 12, parsed.Hour())
}

func TestParseTime_InvalidFormat(t *testing.T) {
	tests := []struct {
		name    string
		timeStr string
	}{
		{
			name:    "completely invalid",
			timeStr: "not a time",
		},
		{
			name:    "invalid date",
			timeStr: "2024-13-45",
		},
		{
			name:    "empty string",
			timeStr: "",
		},
		{
			name:    "random numbers",
			timeStr: "12345",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseTime(tt.timeStr)
			assert.Error(t, err)
			var parseErr *time.ParseError
			assert.ErrorAs(t, err, &parseErr)
		})
	}
}

func TestFormatTime(t *testing.T) {
	// Test with a specific time
	testTime := time.Date(2024, 1, 1, 12, 30, 45, 0, time.UTC)
	formatted := FormatTime(testTime)
	assert.Equal(t, "2024-01-01T12:30:45Z", formatted)
}

func TestFormatTime_ZeroValue(t *testing.T) {
	// Zero time should return empty string
	formatted := FormatTime(time.Time{})
	assert.Equal(t, "", formatted)
}

func TestFormatTime_NonUTC(t *testing.T) {
	// Test that non-UTC times are converted to UTC
	loc, err := time.LoadLocation("America/New_York")
	require.NoError(t, err)
	testTime := time.Date(2024, 1, 1, 12, 30, 45, 0, loc)
	formatted := FormatTime(testTime)

	// Should be in UTC
	assert.Contains(t, formatted, "Z")

	// Parse it back to verify
	parsed, err := time.Parse(time.RFC3339, formatted)
	require.NoError(t, err)
	assert.Equal(t, time.UTC, parsed.Location())
}

func TestFormatTime_RoundTrip(t *testing.T) {
	// Test that formatting and parsing are consistent
	original := time.Date(2024, 1, 1, 12, 30, 45, 0, time.UTC)
	formatted := FormatTime(original)
	parsed, err := ParseTime(formatted)
	require.NoError(t, err)
	assert.True(t, original.Equal(parsed))
}

func TestNow(t *testing.T) {
	before := time.Now().UTC()
	result := Now()
	after := time.Now().UTC()

	// Result should be between before and after
	assert.True(t, result.After(before) || result.Equal(before))
	assert.True(t, result.Before(after) || result.Equal(after))

	// Result should be in UTC
	assert.Equal(t, time.UTC, result.Location())
}

func TestTimeConstants(t *testing.T) {
	assert.Equal(t, time.RFC3339, TimeFormat)
	assert.Equal(t, time.RFC3339Nano, TimeFormatNano)
}
