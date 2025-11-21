package runtime

import (
	"context"
	"errors"
	"testing"

	"github.com/ThreeDotsLabs/watermill/message"
	"google.golang.org/protobuf/types/known/structpb"

	errspkg "github.com/drblury/protoflow/internal/runtime/errors"
	metadatapkg "github.com/drblury/protoflow/internal/runtime/metadata"
)

type publisherTestContextKey struct{}

var testCtxKey = publisherTestContextKey{}

func TestNewMessageFromProtoValidations(t *testing.T) {
	if _, err := NewMessageFromProto(nil, nil); err == nil {
		t.Fatal("expected error when event nil")
	}

	metadata := metadatapkg.Metadata{"origin": "unit"}
	msg, err := NewMessageFromProto(&structpb.Struct{}, metadata)
	if err != nil {
		t.Fatalf("unexpected error creating message: %v", err)
	}
	if msg.Metadata["event_message_schema"] == "" {
		t.Fatal("expected schema metadata to be set")
	}
	if msg.Metadata["origin"] != "unit" {
		t.Fatalf("expected metadata to be preserved, got %#v", msg.Metadata)
	}
}

func TestPublishProtoValidations(t *testing.T) {
	payload := &structpb.Struct{}
	if err := PublishProto(context.Background(), nil, "topic", payload, nil); !errors.Is(err, errspkg.ErrPublisherRequired) {
		t.Fatalf("expected publisher required error, got %v", err)
	}
	if err := PublishProto(context.Background(), &recordingPublisher{}, "", payload, nil); !errors.Is(err, errspkg.ErrTopicRequired) {
		t.Fatalf("expected topic required error, got %v", err)
	}
}

func TestPublishProtoSetsContextAndTopic(t *testing.T) {
	payload := &structpb.Struct{}
	recorder := &recordingPublisher{}
	ctx := context.WithValue(context.Background(), testCtxKey, "ctx")
	metadata := metadatapkg.Metadata{"origin": "test"}

	if err := PublishProto(ctx, recorder, "orders", payload, metadata); err != nil {
		t.Fatalf("unexpected publish error: %v", err)
	}
	if len(recorder.topics) != 1 || recorder.topics[0] != "orders" {
		t.Fatalf("expected topic to be recorded, got %#v", recorder.topics)
	}
	if recorder.messages[0].Context().Value(testCtxKey) != "ctx" {
		t.Fatal("expected context to be attached to message")
	}
}

func TestServicePublishProtoValidatesReceiver(t *testing.T) {
	var svc *Service
	if err := svc.PublishProto(context.Background(), "topic", &structpb.Struct{}, nil); err == nil {
		t.Fatal("expected error when service nil")
	}
}

type recordingPublisher struct {
	topics   []string
	messages []*message.Message
	err      error
}

func (p *recordingPublisher) Publish(topic string, messages ...*message.Message) error {
	if p.err != nil {
		return p.err
	}
	p.topics = append(p.topics, topic)
	p.messages = append(p.messages, messages...)
	return nil
}

func (p *recordingPublisher) Close() error { return nil }
