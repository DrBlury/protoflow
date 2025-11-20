package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/drblury/protoflow"
	"github.com/drblury/protoflow/examples/models"
)

func main() {
	ctx := context.Background()
	logger := protoflow.NewSlogServiceLogger(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	deps := protoflow.ServiceDependencies{
		Validator: sampleValidator{},
		Outbox:    newMemoryOutbox(),
		Middlewares: []protoflow.MiddlewareRegistration{
			metricsMiddleware(),
		},
	}

	cfg := &protoflow.Config{
		PubSubSystem:       "kafka",
		KafkaBrokers:       []string{"localhost:9092"},
		KafkaConsumerGroup: "full-example",
		PoisonQueue:        "orders.poison",
		RetryMaxRetries:    5,
	}

	svc := protoflow.NewService(cfg, logger, ctx, deps)

	must(protoflow.RegisterProtoHandler(svc, protoflow.ProtoHandlerRegistration[*models.OrderCreated]{
		Name:               "proto-created",
		ConsumeQueue:       "orders.created",
		PublishQueue:       "orders.processed",
		ConsumeMessageType: &models.OrderCreated{},
		PublishMessageType: &models.OrderProcessed{},
		ValidateOutgoing:   true,
		Handler: func(ctx context.Context, evt protoflow.ProtoMessageContext[*models.OrderCreated]) ([]protoflow.ProtoMessageOutput, error) {
			out := &models.OrderProcessed{
				OrderId: evt.Payload.GetOrderId(),
				Status:  "processed",
			}
			metadata := evt.CloneMetadata()
			metadata["processed_by"] = "proto-handler"
			return []protoflow.ProtoMessageOutput{{Message: out, Metadata: metadata}}, nil
		},
	}))

	must(protoflow.RegisterJSONHandler(svc, protoflow.JSONHandlerRegistration[*incomingJSON, *outgoingJSON]{
		Name:               "json-ingest",
		ConsumeQueue:       "json.orders",
		PublishQueue:       "json.audit",
		ConsumeMessageType: &incomingJSON{},
		Handler: func(ctx context.Context, evt protoflow.JSONMessageContext[*incomingJSON]) ([]protoflow.JSONMessageOutput[*outgoingJSON], error) {
			metadata := evt.CloneMetadata()
			metadata["json_seen"] = time.Now().UTC().Format(time.RFC3339)
			return []protoflow.JSONMessageOutput[*outgoingJSON]{
				{Message: &outgoingJSON{ID: evt.Payload.ID, Status: "ok"}, Metadata: metadata},
			}, nil
		},
	}))

	must(protoflow.RegisterMessageHandler(svc, protoflow.MessageHandlerRegistration{
		Name:         "raw-audit",
		ConsumeQueue: "raw.audit",
		PublishQueue: "raw.archive",
		Handler: func(msg *message.Message) ([]*message.Message, error) {
			svc.Logger.Info("raw handler", watermill.LogFields{"message_id": msg.UUID})
			return nil, nil
		},
	}))

	must(svc.RegisterMiddleware(protoflow.MiddlewareRegistration{
		Name: "custom_logging",
		Builder: func(s *protoflow.Service) (message.HandlerMiddleware, error) {
			return func(h message.HandlerFunc) message.HandlerFunc {
				return func(msg *message.Message) ([]*message.Message, error) {
					start := time.Now()
					out, err := h(msg)
					s.Logger.Info("handler completed", watermill.LogFields{
						"duration_ms": time.Since(start).Milliseconds(),
						"message_id":  msg.UUID,
					})
					return out, err
				}
			}, nil
		},
	}))

	go func() {
		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				metadata := protoflow.Metadata{"event_source": "full-example"}
				evt := &models.OrderCreated{OrderId: protoflow.CreateULID(), Customer: "demo"}
				_ = svc.PublishProto(context.Background(), "orders.created", evt, metadata)
			}
		}
	}()

	if err := svc.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("router stopped", err, protoflow.LogFields{"example": "full"})
	}
}

type incomingJSON struct {
	ID string `json:"id"`
}

type outgoingJSON struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

type sampleValidator struct{}

func (sampleValidator) Validate(msg any) error {
	if msg == nil {
		return errors.New("message is nil")
	}
	return nil
}

type memoryOutbox struct {
	mu      sync.Mutex
	records []protoflow.Metadata
}

func newMemoryOutbox() *memoryOutbox { return &memoryOutbox{} }

func (o *memoryOutbox) StoreOutgoingMessage(_ context.Context, eventType, uuid, payload string) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.records = append(o.records, protoflow.Metadata{
		"event_type": eventType,
		"uuid":       uuid,
		"payload":    payload,
	})
	return nil
}

func metricsMiddleware() protoflow.MiddlewareRegistration {
	return protoflow.MiddlewareRegistration{
		Name: "metrics",
		Builder: func(s *protoflow.Service) (message.HandlerMiddleware, error) {
			return func(h message.HandlerFunc) message.HandlerFunc {
				return func(msg *message.Message) ([]*message.Message, error) {
					start := time.Now()
					events, err := h(msg)
					s.Logger.Debug("metrics", watermill.LogFields{
						"duration_ms": time.Since(start).Milliseconds(),
						"published":   len(events),
					})
					return events, err
				}
			}, nil
		},
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
