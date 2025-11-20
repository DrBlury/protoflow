package protoflow

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-amqp/v3/pkg/amqp"
	"github.com/ThreeDotsLabs/watermill-aws/sns"
	"github.com/ThreeDotsLabs/watermill-aws/sqs"
	"github.com/ThreeDotsLabs/watermill-kafka/v3/pkg/kafka"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"
)

func newTestSlogLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func newTestLogger() ServiceLogger {
	return NewSlogServiceLogger(newTestSlogLogger())
}

func TestNewServiceConfiguresKafka(t *testing.T) {

	origPub := kafkaPublisherFactory
	origSub := kafkaSubscriberFactory
	t.Cleanup(func() {
		kafkaPublisherFactory = origPub
		kafkaSubscriberFactory = origSub
	})
	recordedPublishConfigs := 0
	recordedSubscribeConfigs := 0
	pub := &testPublisher{}
	sub := &testSubscriber{}
	kafkaPublisherFactory = func(config kafka.PublisherConfig, _ watermill.LoggerAdapter) (message.Publisher, error) {
		recordedPublishConfigs++
		return pub, nil
	}
	kafkaSubscriberFactory = func(config kafka.SubscriberConfig, _ watermill.LoggerAdapter) (message.Subscriber, error) {
		recordedSubscribeConfigs++
		if config.ConsumerGroup != "group" {
			t.Fatalf("unexpected consumer group: %s", config.ConsumerGroup)
		}
		return sub, nil
	}

	cfg := &Config{
		PubSubSystem:       "kafka",
		KafkaBrokers:       []string{"b1"},
		KafkaConsumerGroup: "group",
		PoisonQueue:        "poison",
	}
	logger := newTestLogger()
	svc := NewService(cfg, logger, context.Background(), ServiceDependencies{})

	if svc.publisher != pub {
		t.Fatalf("expected kafka publisher to be assigned")
	}
	if svc.subscriber != sub {
		t.Fatalf("expected kafka subscriber to be assigned")
	}
	if svc.Conf != cfg {
		t.Fatalf("service config not set")
	}
	if svc.router == nil {
		t.Fatal("router should not be nil")
	}
	if recordedPublishConfigs == 0 || recordedSubscribeConfigs == 0 {
		t.Fatal("factories were not invoked")
	}
}

func TestNewServiceConfiguresRabbitMQ(t *testing.T) {

	origConn := amqpConnectionFactory
	origPub := amqpPublisherFactory
	origSub := amqpSubscriberFactory
	t.Cleanup(func() {
		amqpConnectionFactory = origConn
		amqpPublisherFactory = origPub
		amqpSubscriberFactory = origSub
	})

	connCalls := 0
	amqpConnectionFactory = func(config amqp.ConnectionConfig, _ watermill.LoggerAdapter) (*amqp.ConnectionWrapper, error) {
		connCalls++
		if config.AmqpURI != "amqp://guest:guest@localhost" {
			t.Fatalf("unexpected amqp uri: %s", config.AmqpURI)
		}
		return &amqp.ConnectionWrapper{}, nil
	}

	pub := &testPublisher{}
	sub := &testSubscriber{}
	amqpPublisherFactory = func(cfg amqp.Config, _ watermill.LoggerAdapter, conn *amqp.ConnectionWrapper) (message.Publisher, error) {
		if conn == nil {
			t.Fatal("expected connection to be provided")
		}
		return pub, nil
	}
	amqpSubscriberFactory = func(cfg amqp.Config, _ watermill.LoggerAdapter, conn *amqp.ConnectionWrapper) (message.Subscriber, error) {
		if conn == nil {
			t.Fatal("expected connection to be provided")
		}
		return sub, nil
	}

	cfg := &Config{
		PubSubSystem: "rabbitmq",
		RabbitMQURL:  "amqp://guest:guest@localhost",
		PoisonQueue:  "poison",
	}
	svc := NewService(cfg, newTestLogger(), context.Background(), ServiceDependencies{})

	if svc.publisher != pub {
		t.Fatalf("expected rabbit publisher assignment")
	}
	if svc.subscriber != sub {
		t.Fatalf("expected rabbit subscriber assignment")
	}
	if connCalls != 1 {
		t.Fatalf("expected single connection initialisation, got %d", connCalls)
	}
}

func TestNewServiceConfiguresAWS(t *testing.T) {

	origLoader := awsDefaultConfigLoader
	origTopic := snsTopicResolverFactory
	origPub := snsPublisherFactory
	origSub := snsSubscriberFactory
	t.Cleanup(func() {
		awsDefaultConfigLoader = origLoader
		snsTopicResolverFactory = origTopic
		snsPublisherFactory = origPub
		snsSubscriberFactory = origSub
	})

	awsDefaultConfigLoader = func(ctx context.Context, optFns ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
		return aws.Config{Region: "initial"}, nil
	}

	pub := &testPublisher{}
	sub := &testSubscriber{}
	snsTopicResolverFactory = func(accountID, region string) (*sns.GenerateArnTopicResolver, error) {
		if accountID != "123456789012" {
			t.Fatalf("unexpected account id: %s", accountID)
		}
		return origTopic(accountID, region)
	}
	snsPublisherFactory = func(cfg sns.PublisherConfig, _ watermill.LoggerAdapter) (message.Publisher, error) {
		return pub, nil
	}
	snsSubscriberFactory = func(cfg sns.SubscriberConfig, sqsCfg sqs.SubscriberConfig, _ watermill.LoggerAdapter) (message.Subscriber, error) {
		if sqsCfg.AWSConfig.Region != "eu-west-1" {
			t.Fatalf("unexpected sqs region %s", sqsCfg.AWSConfig.Region)
		}
		return sub, nil
	}

	cfg := &Config{
		PubSubSystem: "aws",
		AWSRegion:    "eu-west-1",
		AWSAccountID: "123456789012",
		AWSEndpoint:  "http://localhost:4566",
		PoisonQueue:  "poison",
	}
	svc := NewService(cfg, newTestLogger(), context.Background(), ServiceDependencies{})

	if svc.publisher != pub {
		t.Fatalf("expected aws publisher assignment")
	}
	if svc.subscriber != sub {
		t.Fatalf("expected aws subscriber assignment")
	}
}

func TestSetupPubSubUnsupportedPanics(t *testing.T) {

	svc := &Service{Conf: &Config{PubSubSystem: "gcp"}}
	logger := newWatermillLogger(newTestLogger())

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for unsupported pubsub system")
		}
	}()

	setupPubSub(svc, svc.Conf, logger, context.Background())
}

func TestServiceStartReturnsWhenContextCancelled(t *testing.T) {

	origRun := routerRun
	defer func() { routerRun = origRun }()
	called := make(chan struct{}, 1)
	routerRun = func(_ *message.Router, runCtx context.Context) error {
		called <- struct{}{}
		<-runCtx.Done()
		return runCtx.Err()
	}
	svc := &Service{router: nil}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		done <- svc.Start(ctx)
	}()
	select {
	case <-called:
	case <-time.After(time.Second):
		t.Fatal("routerRun override not invoked")
	}
	cancel()
	select {
	case err := <-done:
		if err != nil && !errors.Is(err, context.Canceled) {
			t.Fatalf("unexpected error: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("service start did not return after context cancellation")
	}
}

func TestRegisterHandlerValidations(t *testing.T) {

	t.Run("missing handler", testRegisterHandlerValidationsMissingHandler)
	t.Run("missing queue", testRegisterHandlerValidationsMissingQueue)
	t.Run("missing name", testRegisterHandlerValidationsMissingName)
	t.Run("autoname from proto", testRegisterHandlerValidationsAutonameFromProto)
	t.Run("explicit name", testRegisterHandlerValidationsExplicitName)
}

func testRegisterHandlerValidationsMissingHandler(t *testing.T) {
	t.Helper()
	svc := newTestService(t)
	if err := svc.registerHandler(handlerRegistration{ConsumeQueue: "queue"}); err == nil {
		t.Fatal("expected error when handler nil")
	}
}

func testRegisterHandlerValidationsMissingQueue(t *testing.T) {
	t.Helper()
	svc := newTestService(t)
	err := svc.registerHandler(handlerRegistration{Handler: func(msg *message.Message) ([]*message.Message, error) {
		return nil, nil
	}})
	if err == nil {
		t.Fatal("expected error when queue missing")
	}
}

func testRegisterHandlerValidationsMissingName(t *testing.T) {
	t.Helper()
	svc := newTestService(t)
	if err := svc.registerHandler(handlerRegistration{
		ConsumeQueue: "queue",
		Handler: func(msg *message.Message) ([]*message.Message, error) {
			return nil, nil
		},
	}); err == nil {
		t.Fatal("expected error when name missing")
	}
}

func testRegisterHandlerValidationsAutonameFromProto(t *testing.T) {
	t.Helper()
	svc := newTestService(t)
	msg := &structpb.Struct{}
	if err := svc.registerHandler(handlerRegistration{
		ConsumeQueue:       "queue",
		Handler:            func(msg *message.Message) ([]*message.Message, error) { return nil, nil },
		consumeMessageType: msg,
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := svc.protoRegistry["*structpb.Struct"]; !ok {
		t.Fatalf("message prototype not registered")
	}
	handlers := svc.router.Handlers()
	if _, ok := handlers["*structpb.Struct-Handler"]; !ok {
		t.Fatalf("handler not registered with generated name")
	}
}

func testRegisterHandlerValidationsExplicitName(t *testing.T) {
	t.Helper()
	svc := newTestService(t)
	if err := svc.registerHandler(handlerRegistration{
		Name:         "custom",
		ConsumeQueue: "queue",
		Handler:      func(msg *message.Message) ([]*message.Message, error) { return nil, nil },
	}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := svc.router.Handlers()["custom"]; !ok {
		t.Fatalf("handler not registered with explicit name")
	}
}

func TestRegisterProtoMessageAndCloning(t *testing.T) {

	svc := &Service{protoRegistry: make(map[string]func() proto.Message)}
	m := &structpb.Struct{}
	svc.RegisterProtoMessage(m)
	factory, ok := svc.protoRegistry["*structpb.Struct"]
	if !ok {
		t.Fatalf("prototype not stored")
	}
	first := factory()
	second := factory()
	if first == second {
		t.Fatalf("expected distinct clones")
	}
}

func TestUnprocessableEventError(t *testing.T) {

	err := &UnprocessableEventError{eventMessage: "payload", err: errors.New("invalid")}
	if got := err.Error(); got != "unprocessable event: payload error: invalid" {
		t.Fatalf("unexpected error string: %s", got)
	}
}
