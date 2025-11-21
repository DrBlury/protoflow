package transport

import (
	"context"
	"errors"
	"testing"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-aws/sns"
	"github.com/ThreeDotsLabs/watermill-aws/sqs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	amazonsns "github.com/aws/aws-sdk-go-v2/service/sns"
	amazonsqs "github.com/aws/aws-sdk-go-v2/service/sqs"

	"github.com/drblury/protoflow/internal/runtime/config"
)

func TestCreateAWSConfigSetsRegion(t *testing.T) {
	origLoader := AWSDefaultConfigLoader
	t.Cleanup(func() { AWSDefaultConfigLoader = origLoader })

	AWSDefaultConfigLoader = func(ctx context.Context, optFns ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
		return aws.Config{Region: "us-east-1"}, nil
	}

	conf := &config.Config{AWSRegion: "ap-southeast-2"}
	cfg, err := createAWSConfig(context.Background(), conf, watermill.NopLogger{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Region != "ap-southeast-2" {
		t.Fatalf("expected region override, got %s", cfg.Region)
	}
}

func TestCreateAWSConfigReturnsError(t *testing.T) {
	origLoader := AWSDefaultConfigLoader
	t.Cleanup(func() { AWSDefaultConfigLoader = origLoader })

	AWSDefaultConfigLoader = func(ctx context.Context, optFns ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
		return aws.Config{}, errors.New("boom")
	}

	if _, err := createAWSConfig(context.Background(), &config.Config{}, watermill.NopLogger{}); err == nil {
		t.Fatal("expected error when config loader fails")
	}
}

func TestCreateAwsPublisherAccountFallbacks(t *testing.T) {
	origTopic := SNSTopicResolverFactory
	origPub := SNSPublisherFactory
	t.Cleanup(func() {
		SNSTopicResolverFactory = origTopic
		SNSPublisherFactory = origPub
	})

	recordedAccountIDs := make([]string, 0, 2)
	SNSTopicResolverFactory = func(accountID, region string) (*sns.GenerateArnTopicResolver, error) {
		recordedAccountIDs = append(recordedAccountIDs, accountID)
		return origTopic(accountID, region)
	}

	pub := &testPublisher{}
	SNSPublisherFactory = func(cfg sns.PublisherConfig, _ watermill.LoggerAdapter) (message.Publisher, error) {
		for _, opt := range cfg.OptFns {
			opt(&amazonsns.Options{})
		}
		return pub, nil
	}

	conf := &config.Config{AWSAccountID: " '123456789012' ", AWSRegion: "eu-central-1"}
	logger := watermill.NopLogger{}
	cfg := &aws.Config{}

	publisher, err := createAwsPublisher(conf, logger, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if publisher != pub {
		t.Fatal("publisher not returned")
	}

	conf.AWSEndpoint = "http://localhost:4566"
	conf.AWSAccountID = "bad"
	if _, err := createAwsPublisher(conf, logger, cfg); err != nil {
		t.Fatalf("unexpected error when falling back: %v", err)
	}

	if len(recordedAccountIDs) != 2 {
		t.Fatalf("expected two resolver invocations, got %d", len(recordedAccountIDs))
	}
	if recordedAccountIDs[0] != "123456789012" {
		t.Fatalf("expected trimmed account id, got %s", recordedAccountIDs[0])
	}
	if recordedAccountIDs[1] != "000000000000" {
		t.Fatalf("expected fallback account id, got %s", recordedAccountIDs[1])
	}
}

func TestCreateAwsPublisherInvalidEndpoint(t *testing.T) {
	conf := &config.Config{AWSAccountID: "123456789012", AWSEndpoint: "://bad"}
	if _, err := createAwsPublisher(conf, watermill.NopLogger{}, &aws.Config{}); err == nil {
		t.Fatal("expected error for invalid endpoint")
	}
}

func TestCreateAwsSubscriberFallbacks(t *testing.T) {
	origTopic := SNSTopicResolverFactory
	origSub := SNSSubscriberFactory
	t.Cleanup(func() {
		SNSTopicResolverFactory = origTopic
		SNSSubscriberFactory = origSub
	})

	recordedAccountIDs := make([]string, 0, 2)
	SNSTopicResolverFactory = func(accountID, region string) (*sns.GenerateArnTopicResolver, error) {
		recordedAccountIDs = append(recordedAccountIDs, accountID)
		return origTopic(accountID, region)
	}

	sub := &testSubscriber{}
	SNSSubscriberFactory = func(cfg sns.SubscriberConfig, sqsCfg sqs.SubscriberConfig, _ watermill.LoggerAdapter) (message.Subscriber, error) {
		for _, opt := range cfg.OptFns {
			opt(&amazonsns.Options{})
		}
		for _, opt := range sqsCfg.OptFns {
			opt(&amazonsqs.Options{})
		}
		return sub, nil
	}

	conf := &config.Config{AWSAccountID: "", AWSRegion: "us-west-2", AWSEndpoint: "http://localhost:4566"}
	cfg := &aws.Config{BaseEndpoint: aws.String("http://localhost:4566")}
	result, err := createAwsSubscriber(conf, watermill.NopLogger{}, cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != sub {
		t.Fatal("subscriber not returned")
	}
	if recordedAccountIDs[0] != "000000000000" {
		t.Fatalf("expected default account id, got %s", recordedAccountIDs[0])
	}
}

func TestCreateAwsSubscriberResolverError(t *testing.T) {
	origTopic := SNSTopicResolverFactory
	t.Cleanup(func() { SNSTopicResolverFactory = origTopic })

	SNSTopicResolverFactory = func(accountID, region string) (*sns.GenerateArnTopicResolver, error) {
		return nil, errors.New("resolve")
	}

	if _, err := createAwsSubscriber(&config.Config{}, watermill.NopLogger{}, &aws.Config{}); err == nil {
		t.Fatal("expected error on resolver failure")
	}
}

func TestCreateAwsPublisherResolverError(t *testing.T) {
	origTopic := SNSTopicResolverFactory
	t.Cleanup(func() { SNSTopicResolverFactory = origTopic })

	SNSTopicResolverFactory = func(accountID, region string) (*sns.GenerateArnTopicResolver, error) {
		return nil, errors.New("resolver")
	}

	if _, err := createAwsPublisher(&config.Config{AWSAccountID: "123456789012"}, watermill.NopLogger{}, &aws.Config{}); err == nil {
		t.Fatal("expected error when resolver fails")
	}
}

func TestCreateAwsPublisherFactoryError(t *testing.T) {
	origPub := SNSPublisherFactory
	t.Cleanup(func() { SNSPublisherFactory = origPub })

	SNSPublisherFactory = func(cfg sns.PublisherConfig, _ watermill.LoggerAdapter) (message.Publisher, error) {
		return nil, errors.New("publish")
	}

	if _, err := createAwsPublisher(&config.Config{AWSAccountID: "123456789012"}, watermill.NopLogger{}, &aws.Config{}); err == nil {
		t.Fatal("expected error when publisher creation fails")
	}
}

func TestCreateAwsSubscriberFactoryError(t *testing.T) {
	origSub := SNSSubscriberFactory
	t.Cleanup(func() { SNSSubscriberFactory = origSub })

	SNSSubscriberFactory = func(cfg sns.SubscriberConfig, sqsCfg sqs.SubscriberConfig, _ watermill.LoggerAdapter) (message.Subscriber, error) {
		return nil, errors.New("sub")
	}

	if _, err := createAwsSubscriber(&config.Config{}, watermill.NopLogger{}, &aws.Config{}); err == nil {
		t.Fatal("expected error when subscriber creation fails")
	}
}

func TestResolveAccountAndRegionFallsBackToAWSConfigRegion(t *testing.T) {
	conf := &config.Config{}
	account, region := resolveAccountAndRegion(conf, watermill.NopLogger{}, "env-region")
	if account != "" {
		t.Fatalf("expected empty account, got %s", account)
	}
	if region != "env-region" {
		t.Fatalf("expected fallback region to be used, got %s", region)
	}

	conf.AWSRegion = "explicit"
	_, region = resolveAccountAndRegion(conf, watermill.NopLogger{}, "env-region")
	if region != "explicit" {
		t.Fatalf("expected explicit config region, got %s", region)
	}
}

func TestAddEndpointResolver(t *testing.T) {
	aCfg := &aws.Config{}
	snsOpts, sqsOpts, err := addEndpointResolver(aCfg, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(snsOpts) != 0 || len(sqsOpts) != 0 {
		t.Fatalf("expected no options when base endpoint unset")
	}

	endpoint := "http://localhost:4566"
	aCfg.BaseEndpoint = aws.String(endpoint)
	snsOpts, sqsOpts, err = addEndpointResolver(aCfg, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(snsOpts) != 1 || len(sqsOpts) != 1 {
		t.Fatalf("expected endpoint overrides to be configured")
	}

	aCfg.BaseEndpoint = aws.String("://bad")
	if _, _, err = addEndpointResolver(aCfg, nil, nil); err == nil {
		t.Fatal("expected error for invalid endpoint")
	}
}

type testPublisher struct{}

func (p *testPublisher) Publish(topic string, messages ...*message.Message) error { return nil }

func (p *testPublisher) Close() error { return nil }

type testSubscriber struct{}

func (s *testSubscriber) Subscribe(ctx context.Context, topic string) (<-chan *message.Message, error) {
	ch := make(chan *message.Message)
	close(ch)
	return ch, nil
}

func (s *testSubscriber) Close() error { return nil }
