package transport

import (
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/stretchr/testify/assert"
)

func TestTransport_Struct(t *testing.T) {
	// Test that Transport struct can be created and accessed
	transport := Transport{
		Publisher:  &mockPublisher{},
		Subscriber: &mockSubscriber{},
	}

	assert.NotNil(t, transport.Publisher)
	assert.NotNil(t, transport.Subscriber)
}

func TestDLQMessage_Struct(t *testing.T) {
	// Test DLQMessage structure
	msg := DLQMessage{
		ID:            1,
		UUID:          "test-uuid",
		OriginalTopic: "test.topic",
		Payload:       []byte("test payload"),
		Metadata:      map[string]string{"key": "value"},
		ErrorMessage:  "test error",
		FailedAt:      time.Now(),
		RetryCount:    3,
	}

	assert.Equal(t, int64(1), msg.ID)
	assert.Equal(t, "test-uuid", msg.UUID)
	assert.Equal(t, "test.topic", msg.OriginalTopic)
	assert.Equal(t, []byte("test payload"), msg.Payload)
	assert.Equal(t, "value", msg.Metadata["key"])
	assert.Equal(t, "test error", msg.ErrorMessage)
	assert.False(t, msg.FailedAt.IsZero())
	assert.Equal(t, 3, msg.RetryCount)
}

func TestConfig_Interface(t *testing.T) {
	// Test that mockConfig implements Config interface
	var _ Config = (*mockConfig)(nil)
	
	cfg := &mockConfig{pubSubSystem: "test"}
	assert.Equal(t, "test", cfg.GetPubSubSystem())
}

type testProvider struct{}

func (testProvider) Capabilities() Capabilities {
	return Capabilities{Name: "test"}
}

func TestCapabilitiesProvider_Interface(t *testing.T) {
	// Test CapabilitiesProvider interface
	var _ CapabilitiesProvider = testProvider{}
	
	provider := testProvider{}
	caps := provider.Capabilities()
	assert.Equal(t, "test", caps.Name)
}

// DLQManager interface impl
type testDLQManager struct{}

func (testDLQManager) GetDLQCount(topic string) (int64, error)         { return 0, nil }
func (testDLQManager) ReplayDLQMessage(dlqID int64) error              { return nil }
func (testDLQManager) ReplayAllDLQ(topic string) (int64, error)        { return 0, nil }
func (testDLQManager) PurgeDLQ(topic string) (int64, error)            { return 0, nil }

// DLQLister interface impl
type testDLQLister struct{}

func (testDLQLister) ListDLQMessages(topic string, limit, offset int) ([]DLQMessage, error) {
	return nil, nil
}

// QueueIntrospector interface impl
type testIntrospector struct{}

func (testIntrospector) GetPendingCount(topic string) (int64, error) { return 0, nil }

// DelayedPublisher interface impl
type testDelayedPub struct{ *mockPublisher }

func (testDelayedPub) PublishWithDelay(topic string, delay int64, messages ...*message.Message) error {
	return nil
}

func TestInterfaces_Documentation(t *testing.T) {
	// This test documents the interfaces defined in transport.go
	// and ensures they compile correctly
	var _ DLQManager = testDLQManager{}
	var _ DLQLister = testDLQLister{}
	var _ QueueIntrospector = testIntrospector{}
	var _ DelayedPublisher = testDelayedPub{}
}
