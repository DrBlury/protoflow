package cloudevents

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	data := map[string]string{"key": "value"}
	evt := New("test.event", "test-source", data)

	assert.Equal(t, SpecVersion, evt.SpecVersion)
	assert.Equal(t, "test.event", evt.Type)
	assert.Equal(t, "test-source", evt.Source)
	assert.NotEmpty(t, evt.ID)
	assert.False(t, evt.Time.IsZero())
	assert.Equal(t, data, evt.Data)
	assert.NotNil(t, evt.Extensions)
}

func TestNewWithID(t *testing.T) {
	data := "test-data"
	evt := NewWithID("custom-id", "test.event", "test-source", data)

	assert.Equal(t, "custom-id", evt.ID)
	assert.Equal(t, "test.event", evt.Type)
	assert.Equal(t, "test-source", evt.Source)
	assert.Equal(t, data, evt.Data)
}

func TestEventWithSubject(t *testing.T) {
	evt := New("test.event", "test-source", nil)
	evt = evt.WithSubject("test-subject")

	require.NotNil(t, evt.Subject)
	assert.Equal(t, "test-subject", *evt.Subject)
}

func TestEventWithDataContentType(t *testing.T) {
	evt := New("test.event", "test-source", nil)
	evt = evt.WithDataContentType("application/json")

	require.NotNil(t, evt.DataContentType)
	assert.Equal(t, "application/json", *evt.DataContentType)
}

func TestEventWithDataSchema(t *testing.T) {
	evt := New("test.event", "test-source", nil)
	evt = evt.WithDataSchema("https://example.com/schema")

	require.NotNil(t, evt.DataSchema)
	assert.Equal(t, "https://example.com/schema", *evt.DataSchema)
}

func TestEventWithExtension(t *testing.T) {
	evt := New("test.event", "test-source", nil)
	evt = evt.WithExtension("custom_key", "custom_value")

	value := evt.GetExtension("custom_key")
	assert.Equal(t, "custom_value", value)
}

func TestGetExtension(t *testing.T) {
	evt := New("test.event", "test-source", nil)
	evt.Extensions["test_key"] = "test_value"

	value := evt.GetExtension("test_key")
	assert.Equal(t, "test_value", value)

	// Test non-existent key
	value = evt.GetExtension("non_existent")
	assert.Nil(t, value)

	// Test with nil Extensions
	evt.Extensions = nil
	value = evt.GetExtension("test_key")
	assert.Nil(t, value)
}

func TestGetExtensionString(t *testing.T) {
	evt := New("test.event", "test-source", nil)
	evt.Extensions["string_key"] = "string_value"
	evt.Extensions["int_key"] = 123

	// Test string value
	value := evt.GetExtensionString("string_key")
	assert.Equal(t, "string_value", value)

	// Test non-string value - formats using %v
	value = evt.GetExtensionString("int_key")
	assert.Equal(t, "123", value)

	// Test non-existent key
	value = evt.GetExtensionString("non_existent")
	assert.Equal(t, "", value)
}

func TestGetExtensionInt(t *testing.T) {
	evt := New("test.event", "test-source", nil)
	evt.Extensions["int_key"] = 42
	evt.Extensions["float_key"] = 42.5
	evt.Extensions["string_key"] = "not_an_int"

	// Test int value
	value := evt.GetExtensionInt("int_key")
	assert.Equal(t, 42, value)

	// Test float64 value (from JSON unmarshaling)
	value = evt.GetExtensionInt("float_key")
	assert.Equal(t, 42, value)

	// Test non-numeric value
	value = evt.GetExtensionInt("string_key")
	assert.Equal(t, 0, value)

	// Test non-existent key
	value = evt.GetExtensionInt("non_existent")
	assert.Equal(t, 0, value)
}

func TestGetExtensionInt64(t *testing.T) {
	evt := New("test.event", "test-source", nil)
	evt.Extensions["int_key"] = 42
	evt.Extensions["int64_key"] = int64(9223372036854775807)
	evt.Extensions["float_key"] = 42.5
	evt.Extensions["string_key"] = "not_an_int"

	// Test int value
	value := evt.GetExtensionInt64("int_key")
	assert.Equal(t, int64(42), value)

	// Test int64 value
	value = evt.GetExtensionInt64("int64_key")
	assert.Equal(t, int64(9223372036854775807), value)

	// Test float64 value
	value = evt.GetExtensionInt64("float_key")
	assert.Equal(t, int64(42), value)

	// Test non-numeric value
	value = evt.GetExtensionInt64("string_key")
	assert.Equal(t, int64(0), value)
}

func TestGetExtensionBool(t *testing.T) {
	evt := New("test.event", "test-source", nil)
	evt.Extensions["bool_true"] = true
	evt.Extensions["bool_false"] = false
	evt.Extensions["string_key"] = "not_a_bool"

	// Test true value
	value := evt.GetExtensionBool("bool_true")
	assert.True(t, value)

	// Test false value
	value = evt.GetExtensionBool("bool_false")
	assert.False(t, value)

	// Test non-bool value
	value = evt.GetExtensionBool("string_key")
	assert.False(t, value)

	// Test non-existent key
	value = evt.GetExtensionBool("non_existent")
	assert.False(t, value)
}

func TestGetExtensionTime(t *testing.T) {
	evt := New("test.event", "test-source", nil)
	testTime := time.Now().UTC()
	
	// Test with time.Time
	evt.Extensions["time_key"] = testTime
	value := evt.GetExtensionTime("time_key")
	assert.True(t, testTime.Equal(value))

	// Test with RFC3339 string
	rfc3339Str := testTime.Format(time.RFC3339Nano)
	evt.Extensions["time_string"] = rfc3339Str
	value = evt.GetExtensionTime("time_string")
	assert.WithinDuration(t, testTime, value, time.Second)

	// Test with invalid string
	evt.Extensions["invalid_time"] = "not_a_time"
	value = evt.GetExtensionTime("invalid_time")
	assert.True(t, value.IsZero())

	// Test with non-existent key
	value = evt.GetExtensionTime("non_existent")
	assert.True(t, value.IsZero())
}

func TestEventValidate(t *testing.T) {
	tests := []struct {
		name    string
		event   Event
		wantErr bool
	}{
		{
			name: "valid event",
			event: Event{
				SpecVersion: SpecVersion,
				Type:        "test.event",
				Source:      "test-source",
				ID:          "test-id",
			},
			wantErr: false,
		},
		{
			name: "missing specversion",
			event: Event{
				Type:   "test.event",
				Source: "test-source",
				ID:     "test-id",
			},
			wantErr: true,
		},
		{
			name: "missing type",
			event: Event{
				SpecVersion: SpecVersion,
				Source:      "test-source",
				ID:          "test-id",
			},
			wantErr: true,
		},
		{
			name: "missing source",
			event: Event{
				SpecVersion: SpecVersion,
				Type:        "test.event",
				ID:          "test-id",
			},
			wantErr: true,
		},
		{
			name: "missing id",
			event: Event{
				SpecVersion: SpecVersion,
				Type:        "test.event",
				Source:      "test-source",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestEventClone(t *testing.T) {
	subject := "test-subject"
	contentType := "application/json"
	schema := "https://example.com/schema"

	original := Event{
		SpecVersion:     SpecVersion,
		Type:            "test.event",
		Source:          "test-source",
		ID:              "test-id",
		Time:            time.Now().UTC(),
		DataContentType: &contentType,
		DataSchema:      &schema,
		Subject:         &subject,
		Data:            map[string]string{"key": "value"},
		Extensions:      map[string]any{"custom": "value"},
	}

	cloned := original.Clone()

	// Verify all fields are equal
	assert.Equal(t, original.SpecVersion, cloned.SpecVersion)
	assert.Equal(t, original.Type, cloned.Type)
	assert.Equal(t, original.Source, cloned.Source)
	assert.Equal(t, original.ID, cloned.ID)
	assert.True(t, original.Time.Equal(cloned.Time))
	assert.Equal(t, *original.DataContentType, *cloned.DataContentType)
	assert.Equal(t, *original.DataSchema, *cloned.DataSchema)
	assert.Equal(t, *original.Subject, *cloned.Subject)
	assert.Equal(t, original.Data, cloned.Data)
	assert.Equal(t, original.Extensions, cloned.Extensions)

	// Verify modifications don't affect original
	cloned.ID = "modified-id"
	assert.NotEqual(t, original.ID, cloned.ID)

	// Modify extension in clone
	cloned.Extensions["custom"] = "modified"
	assert.NotEqual(t, original.Extensions["custom"], cloned.Extensions["custom"])
}

func TestEventMarshalJSON(t *testing.T) {
	subject := "test-subject"
	contentType := "application/json"
	
	evt := Event{
		SpecVersion:     SpecVersion,
		Type:            "test.event",
		Source:          "test-source",
		ID:              "test-id",
		Time:            time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		DataContentType: &contentType,
		Subject:         &subject,
		Data:            map[string]string{"key": "value"},
		Extensions:      map[string]any{"custom": "value"},
	}

	data, err := evt.MarshalJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Verify it's valid JSON
	var result map[string]any
	err = json.Unmarshal(data, &result)
	require.NoError(t, err)
	assert.Equal(t, SpecVersion, result["specversion"])
	assert.Equal(t, "test.event", result["type"])
	assert.Equal(t, "test-source", result["source"])
	assert.Equal(t, "test-id", result["id"])
}

func TestEventUnmarshalJSON(t *testing.T) {
	jsonData := `{
		"specversion": "1.0",
		"type": "test.event",
		"source": "test-source",
		"id": "test-id",
		"time": "2024-01-01T12:00:00Z",
		"datacontenttype": "application/json",
		"subject": "test-subject",
		"data": {"key": "value"},
		"custom_ext": "custom_value"
	}`

	var evt Event
	err := evt.UnmarshalJSON([]byte(jsonData))
	require.NoError(t, err)

	assert.Equal(t, SpecVersion, evt.SpecVersion)
	assert.Equal(t, "test.event", evt.Type)
	assert.Equal(t, "test-source", evt.Source)
	assert.Equal(t, "test-id", evt.ID)
	assert.False(t, evt.Time.IsZero())
	require.NotNil(t, evt.DataContentType)
	assert.Equal(t, "application/json", *evt.DataContentType)
	require.NotNil(t, evt.Subject)
	assert.Equal(t, "test-subject", *evt.Subject)
	assert.NotNil(t, evt.Data)
	assert.Equal(t, "custom_value", evt.Extensions["custom_ext"])
}

func TestEventUnmarshalJSON_InvalidJSON(t *testing.T) {
	var evt Event
	err := evt.UnmarshalJSON([]byte("invalid json"))
	assert.Error(t, err)
}

func TestEventUnmarshalJSON_MissingRequired(t *testing.T) {
	jsonData := `{
		"specversion": "1.0",
		"type": "test.event"
	}`

	var evt Event
	err := evt.UnmarshalJSON([]byte(jsonData))
	// UnmarshalJSON doesn't validate required fields, so no error
	assert.NoError(t, err)
	
	// But Validate should catch missing required fields
	err = evt.Validate()
	assert.Error(t, err)
}

func TestEventRoundTrip(t *testing.T) {
	subject := "test-subject"
	contentType := "application/json"
	
	original := Event{
		SpecVersion:     SpecVersion,
		Type:            "test.event",
		Source:          "test-source",
		ID:              "test-id",
		Time:            time.Now().UTC().Truncate(time.Second),
		DataContentType: &contentType,
		Subject:         &subject,
		Data:            map[string]any{"key": "value"},
		Extensions:      map[string]any{"custom": "value", "number": float64(42)},
	}

	// Marshal to JSON
	data, err := original.MarshalJSON()
	require.NoError(t, err)

	// Unmarshal back
	var decoded Event
	err = decoded.UnmarshalJSON(data)
	require.NoError(t, err)

	// Compare
	assert.Equal(t, original.SpecVersion, decoded.SpecVersion)
	assert.Equal(t, original.Type, decoded.Type)
	assert.Equal(t, original.Source, decoded.Source)
	assert.Equal(t, original.ID, decoded.ID)
	assert.WithinDuration(t, original.Time, decoded.Time, time.Second)
}
