package protoflow

import (
	"context"
	"fmt"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestBuildProtoHandlerUnmarshalsPayload(t *testing.T) {
	prototype := &structpb.Struct{}
	handler, err := buildProtoHandler(prototype, func(ctx context.Context, evt ProtoMessageContext[*structpb.Struct]) ([]ProtoMessageOutput, error) {
		if ctx == nil {
			t.Fatalf("context should not be nil")
		}
		if evt.Payload == nil {
			t.Fatalf("expected payload instance")
		}
		if evt.Metadata == nil {
			t.Fatalf("expected metadata")
		}
		evt.Metadata["processed_by"] = "typed_handler"
		return []ProtoMessageOutput{{
			Message: exampleStruct("processed"),
		}}, nil
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error building handler: %v", err)
	}

	payload, err := protojson.Marshal(exampleStruct("cust-123"))
	if err != nil {
		t.Fatalf("failed to marshal proto: %v", err)
	}
	msg := message.NewMessage(CreateULID(), payload)
	msg.Metadata = message.Metadata{"origin": "test"}

	produced, err := handler(msg)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(produced) != 1 {
		t.Fatalf("expected single outgoing message, got %d", len(produced))
	}
	if produced[0].Metadata["processed_by"] != "typed_handler" {
		t.Fatalf("missing propagated metadata")
	}
}

func TestBuildProtoHandlerHonoursCustomMetadata(t *testing.T) {
	prototype := &structpb.Struct{}
	handler, err := buildProtoHandler(prototype, func(ctx context.Context, evt ProtoMessageContext[*structpb.Struct]) ([]ProtoMessageOutput, error) {
		md := evt.CloneMetadata()
		md["origin"] = "cloned"
		return []ProtoMessageOutput{{
			Message:  &structpb.Struct{},
			Metadata: md,
		}}, nil
	}, nil)
	if err != nil {
		t.Fatalf("unexpected error building handler: %v", err)
	}

	msg := message.NewMessage(CreateULID(), []byte(`{}`))
	produced, err := handler(msg)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if produced[0].Metadata["origin"] != "cloned" {
		t.Fatalf("custom metadata not applied")
	}
}

func TestRegisterProtoHandlerValidations(t *testing.T) {
	svc := newTestService(t)
	err := RegisterProtoHandler(nil, ProtoHandlerRegistration[*structpb.Struct]{
		Handler: func(context.Context, ProtoMessageContext[*structpb.Struct]) ([]ProtoMessageOutput, error) {
			return nil, nil
		},
	})
	if err == nil {
		t.Fatalf("expected error when service nil")
	}

	err = RegisterProtoHandler(svc, ProtoHandlerRegistration[*structpb.Struct]{
		ConsumeQueue: "queue",
		Handler:      nil,
	})
	if err == nil {
		t.Fatalf("expected error when handler nil")
	}

	if err := RegisterProtoHandler(svc, ProtoHandlerRegistration[*structpb.Struct]{
		ConsumeQueue: "queue",
		PublishQueue: "out",
		Handler: func(context.Context, ProtoMessageContext[*structpb.Struct]) ([]ProtoMessageOutput, error) {
			return nil, nil
		},
	}); err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}
	if _, ok := svc.router.Handlers()[fmt.Sprintf("%T-Handler", &structpb.Struct{})]; !ok {
		t.Fatalf("typed handler not registered")
	}
	if err := RegisterProtoHandler(svc, ProtoHandlerRegistration[*structpb.Struct]{
		Name:         "typed_inferred",
		ConsumeQueue: "queue",
		PublishQueue: "out",
		Handler: func(context.Context, ProtoMessageContext[*structpb.Struct]) ([]ProtoMessageOutput, error) {
			return nil, nil
		},
	}); err != nil {
		t.Fatalf("expected handler to infer consume type: %v", err)
	}
	if _, ok := svc.router.Handlers()["typed_inferred"]; !ok {
		t.Fatalf("typed handler (inferred) not registered")
	}
}

func TestRegisterProtoHandlerRegistersPublishTypes(t *testing.T) {
	svc := newTestService(t)
	primary := &structpb.Struct{}
	extra := &structpb.ListValue{}

	if err := RegisterProtoHandler(svc, ProtoHandlerRegistration[*structpb.Struct]{
		Name:         "typed",
		ConsumeQueue: "queue",
		PublishQueue: "out",
		Options: []ProtoHandlerOption{
			WithPublishMessageTypes(primary, extra),
		},
		Handler: func(context.Context, ProtoMessageContext[*structpb.Struct]) ([]ProtoMessageOutput, error) {
			return nil, nil
		},
	}); err != nil {
		t.Fatalf("unexpected error registering handler: %v", err)
	}

	if _, ok := svc.protoRegistry[fmt.Sprintf("%T", primary)]; !ok {
		t.Fatalf("primary publish type not registered")
	}
	if _, ok := svc.protoRegistry[fmt.Sprintf("%T", extra)]; !ok {
		t.Fatalf("option publish type not registered")
	}
}

func exampleStruct(customerID string) *structpb.Struct {
	fields := map[string]*structpb.Value{}
	if customerID != "" {
		fields["customer_id"] = structpb.NewStringValue(customerID)
	}
	return &structpb.Struct{Fields: fields}
}
