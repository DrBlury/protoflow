package protoflow

import (
	"context"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
)

func TestBuildJSONHandlerProcessesPayload(t *testing.T) {
	type incoming struct {
		ID int `json:"id"`
	}
	type outgoing struct {
		ID        int       `json:"id"`
		Processed time.Time `json:"processed"`
	}

	handler, err := buildJSONHandler(&incoming{}, func(ctx context.Context, evt JSONMessageContext[*incoming]) ([]JSONMessageOutput[*outgoing], error) {
		if ctx == nil {
			t.Fatalf("context should not be nil")
		}
		if evt.Payload == nil || evt.Payload.ID != 42 {
			t.Fatalf("unexpected payload: %#v", evt.Payload)
		}
		md := evt.CloneMetadata()
		md["processed"] = "true"
		return []JSONMessageOutput[*outgoing]{
			{
				Message:  &outgoing{ID: evt.Payload.ID, Processed: time.Unix(100, 0)},
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
		ConsumeQueue:       "queue",
		PublishQueue:       "out",
		ConsumeMessageType: nil,
		Handler: func(context.Context, JSONMessageContext[*struct{}]) ([]JSONMessageOutput[*struct{}], error) {
			return nil, nil
		},
	})
	if err == nil {
		t.Fatalf("expected error when prototype missing")
	}

	err = RegisterJSONHandler(svc, JSONHandlerRegistration[*struct{}, *struct{}]{
		ConsumeQueue:       "queue",
		PublishQueue:       "out",
		ConsumeMessageType: &struct{}{},
		Handler:            nil,
	})
	if err == nil {
		t.Fatalf("expected error when handler missing")
	}

	err = RegisterJSONHandler(svc, JSONHandlerRegistration[*incomingMessage, *outgoingMessage]{
		Name:               "json_handler",
		ConsumeQueue:       "queue",
		PublishQueue:       "out",
		ConsumeMessageType: &incomingMessage{},
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
}

type incomingMessage struct{}
type outgoingMessage struct{}
