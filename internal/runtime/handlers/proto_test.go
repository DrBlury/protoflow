package handlers

import (
	"context"
	"errors"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	errspkg "github.com/drblury/protoflow/internal/runtime/errors"
	idspkg "github.com/drblury/protoflow/internal/runtime/ids"
	metadatapkg "github.com/drblury/protoflow/internal/runtime/metadata"
)

func TestBuildProtoHandlerProcessesPayload(t *testing.T) {
	prototype := &structpb.Struct{}
	validated := 0
	capturedMetadata := make([]metadatapkg.Metadata, 0, 1)
	factory := func(msg proto.Message, metadata metadatapkg.Metadata) (*message.Message, error) {
		capturedMetadata = append(capturedMetadata, metadata.Clone())
		return message.NewMessage(idspkg.CreateULID(), []byte("ok")), nil
	}

	handler, err := BuildProtoHandler(prototype, func(ctx context.Context, evt ProtoMessageContext[*structpb.Struct]) ([]ProtoMessageOutput, error) {
		if ctx == nil {
			t.Fatalf("context should not be nil")
		}
		md := evt.CloneMetadata()
		md["seen"] = "true"
		return []ProtoMessageOutput{{Message: evt.Payload, Metadata: md}}, nil
	}, func(msg proto.Message) error {
		validated++
		if msg == nil {
			return errors.New("message missing")
		}
		return nil
	}, factory)
	if err != nil {
		t.Fatalf("unexpected error building handler: %v", err)
	}

	payload := &structpb.Struct{}
	bytes, err := payload.MarshalJSON()
	if err != nil {
		t.Fatalf("failed to marshal proto: %v", err)
	}

	msg := message.NewMessage(idspkg.CreateULID(), bytes)
	msg.Metadata = message.Metadata{"origin": "test"}

	produced, err := handler(msg)
	if err != nil {
		t.Fatalf("handler returned error: %v", err)
	}
	if len(produced) != 1 {
		t.Fatalf("expected single outgoing message, got %d", len(produced))
	}
	if validated != 1 {
		t.Fatalf("expected validator to run once, got %d", validated)
	}
	if capturedMetadata[0]["seen"] != "true" {
		t.Fatalf("expected metadata to be cloned")
	}
}

func TestBuildProtoHandlerValidations(t *testing.T) {
	_, err := BuildProtoHandler[*structpb.Struct](nil, nil, nil, nil)
	if !errors.Is(err, errspkg.ErrHandlerRequired) {
		t.Fatalf("expected handler required error, got %v", err)
	}

	_, err = BuildProtoHandler[*structpb.Struct](nil, func(context.Context, ProtoMessageContext[*structpb.Struct]) ([]ProtoMessageOutput, error) {
		return nil, nil
	}, nil, func(proto.Message, metadatapkg.Metadata) (*message.Message, error) {
		return nil, nil
	})
	if !errors.Is(err, errspkg.ErrConsumeMessageTypeRequired) {
		t.Fatalf("expected consume type required error, got %v", err)
	}

	_, err = BuildProtoHandler(&structpb.Struct{}, func(context.Context, ProtoMessageContext[*structpb.Struct]) ([]ProtoMessageOutput, error) {
		return nil, nil
	}, nil, nil)
	if err == nil {
		t.Fatal("expected error when factory missing")
	}
}

func TestClonePrototypeValidations(t *testing.T) {
	var zero *structpb.Struct
	if _, err := clonePrototype(zero); !errors.Is(err, errspkg.ErrConsumeMessageTypeRequired) {
		t.Fatalf("expected consume type required error, got %v", err)
	}

	prototype := &structpb.Struct{}
	cloned, err := clonePrototype(prototype)
	if err != nil {
		t.Fatalf("unexpected clone error: %v", err)
	}
	if cloned == prototype {
		t.Fatalf("expected clone to return new instance")
	}
}

func TestEnsureProtoPrototype(t *testing.T) {
	var zero *structpb.Struct
	inst, err := EnsureProtoPrototype(zero)
	if err != nil {
		t.Fatalf("unexpected error ensuring prototype: %v", err)
	}
	if inst == nil {
		t.Fatal("expected instance to be created")
	}
}

func TestConvertProtoOutputsUsesFallbackMetadata(t *testing.T) {
	fallback := metadatapkg.Metadata{"origin": "fallback"}
	factory := func(msg proto.Message, metadata metadatapkg.Metadata) (*message.Message, error) {
		if metadata["origin"] != "fallback" {
			t.Fatalf("expected fallback metadata, got %#v", metadata)
		}
		return message.NewMessage(idspkg.CreateULID(), []byte("ok")), nil
	}

	produced, err := convertProtoOutputs([]ProtoMessageOutput{{Message: &structpb.Struct{}, Metadata: nil}}, fallback, factory)
	if err != nil {
		t.Fatalf("unexpected error converting outputs: %v", err)
	}
	if len(produced) != 1 {
		t.Fatalf("expected single message, got %d", len(produced))
	}

	factoryErr := errors.New("factory")
	_, err = convertProtoOutputs([]ProtoMessageOutput{{Message: &structpb.Struct{}}}, fallback, func(proto.Message, metadatapkg.Metadata) (*message.Message, error) {
		return nil, factoryErr
	})
	if !errors.Is(err, factoryErr) {
		t.Fatalf("expected factory error, got %v", err)
	}

	msgs, err := convertProtoOutputs(nil, fallback, factory)
	if msgs != nil || err != nil {
		t.Fatal("expected nil result when no outputs")
	}
}

func TestIsNilProto(t *testing.T) {
	var nilStruct *structpb.Struct
	if !isNilProto(nilStruct) {
		t.Fatal("expected nil pointer to be detected")
	}

	nonNil := &structpb.Struct{}
	if isNilProto(nonNil) {
		t.Fatal("expected non-nil pointer to be detected")
	}
}
