package protoflow

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
)

func TestCreateAWSConfigSetsRegion(t *testing.T) {

	origLoader := awsDefaultConfigLoader
	t.Cleanup(func() { awsDefaultConfigLoader = origLoader })

	awsDefaultConfigLoader = func(ctx context.Context, optFns ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
		return aws.Config{Region: "us-east-1"}, nil
	}

	svc := &Service{
		Conf:   &Config{AWSRegion: "ap-southeast-2"},
		Logger: watermill.NewSlogLoggerWithLevelMapping(newTestLogger(), logLevelMapping),
	}

	cfg := svc.createAWSConfig(context.Background())
	if cfg.Region != "ap-southeast-2" {
		t.Fatalf("expected region override, got %s", cfg.Region)
	}
}

func TestCreateAWSConfigPanicsOnError(t *testing.T) {

	origLoader := awsDefaultConfigLoader
	t.Cleanup(func() { awsDefaultConfigLoader = origLoader })

	awsDefaultConfigLoader = func(ctx context.Context, optFns ...func(*awsconfig.LoadOptions) error) (aws.Config, error) {
		return aws.Config{}, errors.New("boom")
	}

	svc := &Service{Logger: watermill.NewSlogLoggerWithLevelMapping(newTestLogger(), logLevelMapping)}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when config loader fails")
		}
	}()

	svc.createAWSConfig(context.Background())
}

func TestCreateAwsPublisherAccountFallbacks(t *testing.T) {

	origTopic := snsTopicResolverFactory
	origPub := snsPublisherFactory
	t.Cleanup(func() {
		snsTopicResolverFactory = origTopic
		snsPublisherFactory = origPub
	})

	recordedAccountIDs := make([]string, 0, 2)
	snsTopicResolverFactory = func(accountID, region string) (*sns.GenerateArnTopicResolver, error) {
		recordedAccountIDs = append(recordedAccountIDs, accountID)
		return origTopic(accountID, region)
	}

	pub := &testPublisher{}
	snsPublisherFactory = func(cfg sns.PublisherConfig, _ watermill.LoggerAdapter) (message.Publisher, error) {
		for _, opt := range cfg.OptFns {
			opt(&amazonsns.Options{})
		}
		return pub, nil
	}

	svc := &Service{
		Conf:   &Config{AWSAccountID: " '123456789012' ", AWSRegion: "eu-central-1"},
		Logger: watermill.NewSlogLoggerWithLevelMapping(newTestLogger(), logLevelMapping),
	}
	cfg := &aws.Config{}
	svc.createAwsPublisher(svc.Logger, cfg)

	svc.Conf.AWSEndpoint = "http://localhost:4566"
	svc.Conf.AWSAccountID = "bad"
	svc.createAwsPublisher(svc.Logger, cfg)

	if len(recordedAccountIDs) != 2 {
		t.Fatalf("expected two resolver invocations, got %d", len(recordedAccountIDs))
	}
	if recordedAccountIDs[0] != "123456789012" {
		t.Fatalf("expected trimmed account id, got %s", recordedAccountIDs[0])
	}
	if recordedAccountIDs[1] != "000000000000" {
		t.Fatalf("expected fallback account id, got %s", recordedAccountIDs[1])
	}
	if svc.publisher != pub {
		t.Fatal("publisher not assigned")
	}
}

func TestCreateAwsPublisherPanicsOnEndpointParse(t *testing.T) {

	origTopic := snsTopicResolverFactory
	origPub := snsPublisherFactory
	t.Cleanup(func() {
		snsTopicResolverFactory = origTopic
		snsPublisherFactory = origPub
	})

	snsPublisherFactory = func(cfg sns.PublisherConfig, _ watermill.LoggerAdapter) (message.Publisher, error) {
		for _, opt := range cfg.OptFns {
			opt(&amazonsns.Options{})
		}
		return &testPublisher{}, nil
	}

	svc := &Service{
		Conf:   &Config{AWSAccountID: "123456789012", AWSEndpoint: "://bad"},
		Logger: watermill.NewSlogLoggerWithLevelMapping(newTestLogger(), logLevelMapping),
	}
	cfg := &aws.Config{}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for invalid endpoint")
		}
	}()

	svc.createAwsPublisher(svc.Logger, cfg)
}

func TestCreateAwsSubscriberFallbacks(t *testing.T) {

	origTopic := snsTopicResolverFactory
	origSub := snsSubscriberFactory
	t.Cleanup(func() {
		snsTopicResolverFactory = origTopic
		snsSubscriberFactory = origSub
	})

	recordedAccountIDs := make([]string, 0, 2)
	snsTopicResolverFactory = func(accountID, region string) (*sns.GenerateArnTopicResolver, error) {
		recordedAccountIDs = append(recordedAccountIDs, accountID)
		return origTopic(accountID, region)
	}

	sub := &testSubscriber{}
	snsSubscriberFactory = func(cfg sns.SubscriberConfig, sqsCfg sqs.SubscriberConfig, _ watermill.LoggerAdapter) (message.Subscriber, error) {
		for _, opt := range cfg.OptFns {
			opt(&amazonsns.Options{})
		}
		for _, opt := range sqsCfg.OptFns {
			opt(&amazonsqs.Options{})
		}
		return sub, nil
	}

	svc := &Service{
		Conf:   &Config{AWSAccountID: "", AWSRegion: "us-west-2", AWSEndpoint: "http://localhost:4566"},
		Logger: watermill.NewSlogLoggerWithLevelMapping(newTestLogger(), logLevelMapping),
	}
	cfg := &aws.Config{BaseEndpoint: aws.String("http://localhost:4566")}
	svc.createAwsSubscriber(svc.Logger, cfg)
	if recordedAccountIDs[0] != "000000000000" {
		t.Fatalf("expected default account id, got %s", recordedAccountIDs[0])
	}
	if svc.subscriber != sub {
		t.Fatal("subscriber not assigned")
	}
}

func TestCreateAwsSubscriberPanicsOnResolverError(t *testing.T) {

	origTopic := snsTopicResolverFactory
	t.Cleanup(func() { snsTopicResolverFactory = origTopic })

	snsTopicResolverFactory = func(accountID, region string) (*sns.GenerateArnTopicResolver, error) {
		return nil, errors.New("resolve")
	}

	svc := &Service{
		Conf:   &Config{},
		Logger: watermill.NewSlogLoggerWithLevelMapping(newTestLogger(), logLevelMapping),
	}
	cfg := &aws.Config{}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on resolver failure")
		}
	}()

	svc.createAwsSubscriber(svc.Logger, cfg)
}

func TestCreateAwsPublisherPanicsOnResolverError(t *testing.T) {

	origTopic := snsTopicResolverFactory
	t.Cleanup(func() { snsTopicResolverFactory = origTopic })

	snsTopicResolverFactory = func(accountID, region string) (*sns.GenerateArnTopicResolver, error) {
		return nil, errors.New("resolver")
	}

	svc := &Service{
		Conf:   &Config{AWSAccountID: "123456789012"},
		Logger: watermill.NewSlogLoggerWithLevelMapping(newTestLogger(), logLevelMapping),
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic when resolver fails")
		}
	}()

	svc.createAwsPublisher(svc.Logger, &aws.Config{})
}

func TestCreateAwsPublisherPanicsOnPublisherError(t *testing.T) {

	origPub := snsPublisherFactory
	t.Cleanup(func() { snsPublisherFactory = origPub })

	snsPublisherFactory = func(cfg sns.PublisherConfig, _ watermill.LoggerAdapter) (message.Publisher, error) {
		return nil, errors.New("publish")
	}

	svc := &Service{
		Conf:   &Config{AWSAccountID: "123456789012"},
		Logger: watermill.NewSlogLoggerWithLevelMapping(newTestLogger(), logLevelMapping),
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on publisher creation failure")
		}
	}()

	svc.createAwsPublisher(svc.Logger, &aws.Config{})
}

func TestCreateAwsSubscriberPanicsOnSubscriberError(t *testing.T) {

	origSub := snsSubscriberFactory
	t.Cleanup(func() { snsSubscriberFactory = origSub })

	snsSubscriberFactory = func(cfg sns.SubscriberConfig, sqsCfg sqs.SubscriberConfig, _ watermill.LoggerAdapter) (message.Subscriber, error) {
		return nil, errors.New("sub")
	}

	svc := &Service{
		Conf:   &Config{},
		Logger: watermill.NewSlogLoggerWithLevelMapping(newTestLogger(), logLevelMapping),
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on subscriber failure")
		}
	}()

	svc.createAwsSubscriber(svc.Logger, &aws.Config{})
}

func TestAddEndpointResolver(t *testing.T) {

	aCfg := &aws.Config{}
	snsOpts, sqsOpts := addEndpointResolver(aCfg, nil, nil)
	if len(snsOpts) != 0 || len(sqsOpts) != 0 {
		t.Fatalf("expected no options when base endpoint unset")
	}

	endpoint := "http://localhost:4566"
	aCfg.BaseEndpoint = aws.String(endpoint)
	snsOpts, sqsOpts = addEndpointResolver(aCfg, nil, nil)
	if len(snsOpts) != 1 || len(sqsOpts) != 1 {
		t.Fatalf("expected endpoint overrides to be configured")
	}

	aCfg.BaseEndpoint = aws.String("://bad")
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for invalid endpoint")
		}
	}()

	addEndpointResolver(aCfg, nil, nil)
}
