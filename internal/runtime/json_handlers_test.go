package runtime

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	handlerpkg "github.com/drblury/protoflow/internal/runtime/handlers"
)

func TestBuildJSONHandlerProcessesPayload(t *testing.T) {
	handler, err := handlerpkg.BuildJSONHandler(func(ctx context.Context, evt JSONMessageContext[*jsonIncoming]) ([]JSONMessageOutput[*jsonOutgoing], error) {
		if ctx == nil {
			t.Fatalf("context should not be nil")
		}
		if evt.Payload == nil || evt.Payload.ID != 42 {
			t.Fatalf("unexpected payload: %#v", evt.Payload)
		}
		md := evt.CloneMetadata()
		md["processed"] = "true"
		return []JSONMessageOutput[*jsonOutgoing]{
			{
				Message:  &jsonOutgoing{ID: evt.Payload.ID, Processed: time.Unix(100, 0)},
				Metadata: md,
			},
		}, nil
	})
	if err != nil {
		t.Fatalf("unexpected error building handler: %v", err)
	}

	msg := message.NewMessage(CreateULID(), []byte(`{"id":42}`))
	msg.Metadata = message.Metadata{"origin": "test"}

	produced, err := handler(msg)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(produced) != 1 {
		t.Fatalf("expected single outgoing message, got %d", len(produced))
	}
	if produced[0].Metadata["processed"] != "true" {
		t.Fatalf("metadata not propagated: %#v", produced[0].Metadata)
	}
	if produced[0].Metadata["event_message_schema"] == "" {
		t.Fatal("expected schema metadata to be set")
	}
}

func TestRegisterJSONHandlerValidations(t *testing.T) {
	svc := newTestService(t)

	err := RegisterJSONHandler(nil, JSONHandlerRegistration[*struct{}, *struct{}]{})
	if err == nil {
		t.Fatalf("expected error when service nil")
	}

	err = RegisterJSONHandler(svc, JSONHandlerRegistration[*struct{}, *struct{}]{
		ConsumeQueue: "queue",
		PublishQueue: "out",
		Handler:      nil,
	})
	if err == nil {
		t.Fatalf("expected error when handler missing")
	}

	err = RegisterJSONHandler(svc, JSONHandlerRegistration[struct{}, *outgoingMessage]{
		ConsumeQueue: "queue",
		PublishQueue: "out",
		Handler: func(context.Context, JSONMessageContext[struct{}]) ([]JSONMessageOutput[*outgoingMessage], error) {
			return nil, nil
		},
	})
	if !errors.Is(err, ErrConsumeMessagePointerNeeded) {
		t.Fatalf("expected pointer error, got %v", err)
	}

	err = RegisterJSONHandler(svc, JSONHandlerRegistration[*incomingMessage, *outgoingMessage]{
		Name:         "json_handler",
		ConsumeQueue: "queue",
		PublishQueue: "out",
		Handler: func(context.Context, JSONMessageContext[*incomingMessage]) ([]JSONMessageOutput[*outgoingMessage], error) {
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}
	if _, ok := svc.router.Handlers()["json_handler"]; !ok {
		t.Fatalf("json handler not registered")
	}

	err = RegisterJSONHandler(svc, JSONHandlerRegistration[*incomingMessage, *outgoingMessage]{
		Name:         "json_handler_inferred",
		ConsumeQueue: "queue",
		PublishQueue: "out",
		Handler: func(context.Context, JSONMessageContext[*incomingMessage]) ([]JSONMessageOutput[*outgoingMessage], error) {
			return nil, nil
		},
	})
	if err != nil {
		t.Fatalf("expected handler to infer consume type: %v", err)
	}
}

type jsonIncoming struct {
	ID int `json:"id"`
}

type jsonOutgoing struct {
	ID        int       `json:"id"`
	Processed time.Time `json:"processed"`
}

type incomingMessage struct{}

type outgoingMessage struct{}
