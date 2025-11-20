package main

import (
	"context"
	"errors"
	"log/slog"
	"os"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/drblury/protoflow"
)

// Simple example showing how to register an untyped handler.
func main() {
	ctx := context.Background()

	baseLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	logger := protoflow.NewSlogServiceLogger(baseLogger)

	cfg := &protoflow.Config{
		PubSubSystem:       "kafka",
		KafkaBrokers:       []string{"localhost:9092"},
		KafkaConsumerGroup: "simple-example",
		PoisonQueue:        "simple.poison",
	}

	svc := protoflow.NewService(cfg, logger, ctx, protoflow.ServiceDependencies{})

	err := protoflow.RegisterMessageHandler(svc, protoflow.MessageHandlerRegistration{
		Name:         "raw-logger",
		ConsumeQueue: "simple.incoming",
		PublishQueue: "simple.outgoing",
		Handler: func(msg *message.Message) ([]*message.Message, error) {
			logger.Info("received raw event", protoflow.LogFields{
				"payload":  string(msg.Payload),
				"metadata": msg.Metadata,
			})

			ack := message.NewMessage(protoflow.CreateULID(), []byte("acknowledged"))
			ack.Metadata = message.Metadata{
				"event_source":         "simple_example",
				"event_message_schema": "example.AckV1",
			}
			return []*message.Message{ack}, nil
		},
	})
	if err != nil {
		panic(err)
	}

	if err := svc.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("router stopped", err, protoflow.LogFields{"handler": "raw-logger"})
	}
}
