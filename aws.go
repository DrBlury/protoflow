package protoflow

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill-aws/sns"
	"github.com/ThreeDotsLabs/watermill-aws/sqs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	amazonsns "github.com/aws/aws-sdk-go-v2/service/sns"
	amazonsqs "github.com/aws/aws-sdk-go-v2/service/sqs"
	transport "github.com/aws/smithy-go/endpoints"
)

var (
	awsDefaultConfigLoader  = awsconfig.LoadDefaultConfig
	snsTopicResolverFactory = sns.NewGenerateArnTopicResolver
	snsPublisherFactory     = func(cfg sns.PublisherConfig, logger watermill.LoggerAdapter) (message.Publisher, error) {
		return sns.NewPublisher(cfg, logger)
	}
	snsSubscriberFactory = func(cfg sns.SubscriberConfig, sqsCfg sqs.SubscriberConfig, logger watermill.LoggerAdapter) (message.Subscriber, error) {
		return sns.NewSubscriber(cfg, sqsCfg, logger)
	}
)

const (
	localstackAccountID = "000000000000"
	awsAccountIDLength  = 12
)

func (s *Service) createAWSConfig(ctx context.Context) *aws.Config {
	cfg, err := awsDefaultConfigLoader(ctx)
	if err != nil {
		s.Logger.Error("Failed to load AWS default config", err, watermill.LogFields{"cfg": cfg})
		panic(err)
	}
	if s.Conf != nil && s.Conf.AWSRegion != "" {
		s.Logger.Info("Setting AWS region from config", watermill.LogFields{"region": s.Conf.AWSRegion})
		cfg.Region = s.Conf.AWSRegion
	}

	return &cfg
}

func (s *Service) createAwsPublisher(logger watermill.LoggerAdapter, cfg *aws.Config) {
	accountID, region := s.resolveAccountAndRegion()

	s.Logger.Info("Create AWS Publisher",
		watermill.LogFields{
			"accountID": accountID,
			"region":    region,
		})

	topicResolver := s.mustCreateTopicResolver(accountID, region)
	publisherConfig := s.buildPublisherConfig(cfg, topicResolver)
	s.publisher = s.mustCreatePublisher(publisherConfig, logger)
}

func (s *Service) createAwsSubscriber(logger watermill.LoggerAdapter, cfg *aws.Config) {
	accountID, region := s.resolveAccountAndRegion()

	topicResolver := s.mustCreateTopicResolver(accountID, region)

	name := "subscriber"

	var snsOpts []func(*amazonsns.Options)
	var sqsOpts []func(*amazonsqs.Options)
	snsOpts, sqsOpts = addEndpointResolver(cfg, snsOpts, sqsOpts)

	subscriberConfig := sns.SubscriberConfig{
		AWSConfig: aws.Config{
			Credentials: aws.AnonymousCredentials{},
		},
		OptFns:        snsOpts,
		TopicResolver: topicResolver,
		GenerateSqsQueueName: func(ctx context.Context, snsTopic sns.TopicArn) (string, error) {
			topic, err := sns.ExtractTopicNameFromTopicArn(snsTopic)
			if err != nil {
				return "", err
			}

			return fmt.Sprintf("%v-%v", topic, name), nil
		},
	}

	subscriber, err := snsSubscriberFactory(
		subscriberConfig,
		sqs.SubscriberConfig{
			AWSConfig: *cfg,
			OptFns:    sqsOpts,
		},
		logger,
	)
	if err != nil {
		panic(err)
	}

	s.subscriber = subscriber
}

func addEndpointResolver(cfg *aws.Config, snsOpts []func(*amazonsns.Options), sqsOpts []func(*amazonsqs.Options)) ([]func(*amazonsns.Options), []func(*amazonsqs.Options)) {
	if cfg != nil && cfg.BaseEndpoint != nil && *cfg.BaseEndpoint != "" {
		parsedURL, err := url.Parse(*cfg.BaseEndpoint)
		if err != nil {
			panic(fmt.Sprintf("Failed to parse BaseEndpoint: %v", err))
		}
		snsOpts = []func(*amazonsns.Options){
			amazonsns.WithEndpointResolverV2(sns.OverrideEndpointResolver{
				Endpoint: transport.Endpoint{
					URI: *parsedURL,
				},
			}),
		}
		sqsOpts = []func(*amazonsqs.Options){
			amazonsqs.WithEndpointResolverV2(sqs.OverrideEndpointResolver{
				Endpoint: transport.Endpoint{
					URI: *parsedURL,
				},
			}),
		}
	}
	return snsOpts, sqsOpts
}

func (s *Service) resolveAccountAndRegion() (string, string) {
	if s.Conf == nil {
		return "", ""
	}

	accountID := strings.Trim(s.Conf.AWSAccountID, "\"' ")
	region := s.Conf.AWSRegion

	if accountID == "" && s.useLocalstackEndpoint() {
		accountID = localstackAccountID
		s.Logger.Info("AWS account ID empty; using LocalStack default", watermill.LogFields{"accountID": accountID})
		return accountID, region
	}

	if accountID != "" && len(accountID) != awsAccountIDLength && s.useLocalstackEndpoint() {
		s.Logger.Info("Invalid AWS account ID; falling back to LocalStack default", watermill.LogFields{"accountID": accountID})
		accountID = localstackAccountID
	}

	return accountID, region
}

func (s *Service) useLocalstackEndpoint() bool {
	return s.Conf != nil && s.Conf.AWSEndpoint != ""
}

func (s *Service) mustCreateTopicResolver(accountID, region string) sns.TopicResolver {
	topicResolver, err := snsTopicResolverFactory(accountID, region)
	if err != nil {
		s.Logger.Error("Failed to create SNS topic resolver", err, watermill.LogFields{
			"accountID": accountID,
			"region":    region,
		})
		panic(err)
	}

	return topicResolver
}

func (s *Service) buildPublisherConfig(cfg *aws.Config, topicResolver sns.TopicResolver) sns.PublisherConfig {
	publisherConfig := sns.PublisherConfig{
		TopicResolver: topicResolver,
		AWSConfig:     *cfg,
		Marshaler:     sns.DefaultMarshalerUnmarshaler{},
	}

	if endpoint := s.awsEndpointURL(); endpoint != nil {
		endpointStr := endpoint.String()
		publisherConfig.OptFns = []func(*amazonsns.Options){
			func(o *amazonsns.Options) {
				o.BaseEndpoint = aws.String(endpointStr)
			},
		}
	}

	return publisherConfig
}

func (s *Service) awsEndpointURL() *url.URL {
	if s.Conf == nil || s.Conf.AWSEndpoint == "" {
		return nil
	}

	parsedURL, err := url.Parse(s.Conf.AWSEndpoint)
	if err != nil {
		s.Logger.Error("Failed to parse AWS endpoint", err, watermill.LogFields{"endpoint": s.Conf.AWSEndpoint})
		panic(err)
	}

	return parsedURL
}

func (s *Service) mustCreatePublisher(cfg sns.PublisherConfig, logger watermill.LoggerAdapter) message.Publisher {
	publisher, err := snsPublisherFactory(cfg, logger)
	if err != nil {
		panic(err)
	}

	return publisher
}
