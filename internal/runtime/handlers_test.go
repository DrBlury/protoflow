package runtime

import (
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
)

func TestRegisterMessageHandlerRequiresService(t *testing.T) {
	err := RegisterMessageHandler(nil, MessageHandlerRegistration{})
	if err == nil {
		t.Fatal("expected error when service is nil")
	}
}

func TestRegisterMessageHandlerRegistersHandler(t *testing.T) {
	svc := newTestService(t)
	err := RegisterMessageHandler(svc, MessageHandlerRegistration{
		Name:         "raw",
		ConsumeQueue: "input",
		PublishQueue: "output",
		Handler: func(msg *message.Message) ([]*message.Message, error) {
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := svc.router.Handlers()["raw"]; !ok {
		t.Fatal("handler not registered")
	}
}
