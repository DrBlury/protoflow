// Package aws provides an AWS SNS/SQS transport for protoflow.
package aws

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
	smithyendpoints "github.com/aws/smithy-go/endpoints"

	"github.com/drblury/protoflow/transport"
)

// TransportName is the name used to register this transport.
const TransportName = "aws"

const (
	localstackAccountID = "000000000000"
	awsAccountIDLength  = 12
)

// DefaultConfigLoader allows overriding the AWS config loader for testing.
var DefaultConfigLoader = awsconfig.LoadDefaultConfig

// TopicResolverFactory allows overriding the topic resolver creation for testing.
var TopicResolverFactory = sns.NewGenerateArnTopicResolver

// PublisherFactory allows overriding the publisher creation for testing.
var PublisherFactory = func(cfg sns.PublisherConfig, logger watermill.LoggerAdapter) (message.Publisher, error) {
	return sns.NewPublisher(cfg, logger)
}

// SubscriberFactory allows overriding the subscriber creation for testing.
var SubscriberFactory = func(cfg sns.SubscriberConfig, sqsCfg sqs.SubscriberConfig, logger watermill.LoggerAdapter) (message.Subscriber, error) {
	return sns.NewSubscriber(cfg, sqsCfg, logger)
}

func init() {
	transport.RegisterWithCapabilities(TransportName, Build, transport.AWSCapabilities)
}

// Build creates a new AWS SNS/SQS transport.
func Build(ctx context.Context, cfg transport.Config, logger watermill.LoggerAdapter) (transport.Transport, error) {
	awsCfg, err := createAWSConfig(ctx, cfg, logger)
	if err != nil {
		return transport.Transport{}, err
	}
	logger.Info("Created AWS config", watermill.LogFields{
		"region":          safeAWSRegion(awsCfg),
		"custom_endpoint": hasCustomEndpoint(awsCfg),
	})

	publisher, err := createPublisher(cfg, logger, awsCfg)
	if err != nil {
		return transport.Transport{}, err
	}

	subscriber, err := createSubscriber(cfg, logger, awsCfg)
	if err != nil {
		return transport.Transport{}, err
	}

	return transport.Transport{
		Publisher:  publisher,
		Subscriber: subscriber,
	}, nil
}

// Capabilities returns the capabilities of this transport.
func Capabilities() transport.Capabilities {
	return transport.AWSCapabilities
}

func createAWSConfig(ctx context.Context, cfg transport.Config, logger watermill.LoggerAdapter) (*aws.Config, error) {
	var opts []func(*awsconfig.LoadOptions) error

	if cfg != nil {
		region := cfg.GetAWSRegion()
		accessKey := cfg.GetAWSAccessKeyID()
		secretKey := cfg.GetAWSSecretAccessKey()

		if region != "" {
			logger.Info("Setting AWS region from config", watermill.LogFields{"region": region})
			opts = append(opts, awsconfig.WithRegion(region))
		}
		if accessKey != "" && secretKey != "" {
			logger.Info("Using static AWS credentials from config", watermill.LogFields{})
			opts = append(opts, awsconfig.WithCredentialsProvider(staticCredentialsProvider(accessKey, secretKey)))
		}
	}

	awsCfg, err := DefaultConfigLoader(ctx, opts...)
	if err != nil {
		fields := watermill.LogFields{}
		if cfg != nil && cfg.GetAWSRegion() != "" {
			fields["requested_region"] = cfg.GetAWSRegion()
		}
		logger.Error("Failed to load AWS default config", err, fields)
		return nil, err
	}

	// Ensure region is set even if the loader ignores options
	if cfg != nil && cfg.GetAWSRegion() != "" {
		awsCfg.Region = cfg.GetAWSRegion()
	}

	return &awsCfg, nil
}

func createPublisher(cfg transport.Config, logger watermill.LoggerAdapter, awsCfg *aws.Config) (message.Publisher, error) {
	accountID, region := resolveAccountAndRegion(cfg, logger, safeAWSRegion(awsCfg))
	logger.Info("Create AWS Publisher", watermill.LogFields{
		"accountID": accountID,
		"region":    region,
	})

	topicResolver, err := createTopicResolver(accountID, region, logger)
	if err != nil {
		return nil, err
	}

	publisherConfig, err := buildPublisherConfig(cfg, awsCfg, topicResolver, logger)
	if err != nil {
		return nil, err
	}

	return PublisherFactory(publisherConfig, logger)
}

func makeSqsQueueNameGenerator() func(context.Context, sns.TopicArn) (string, error) {
	return func(ctx context.Context, snsTopic sns.TopicArn) (string, error) {
		topic, err := sns.ExtractTopicNameFromTopicArn(snsTopic)
		if err != nil {
			return "", err
		}
		return string(topic), nil
	}
}

func createSubscriber(cfg transport.Config, logger watermill.LoggerAdapter, awsCfg *aws.Config) (message.Subscriber, error) {
	accountID, region := resolveAccountAndRegion(cfg, logger, safeAWSRegion(awsCfg))
	topicResolver, err := createTopicResolver(accountID, region, logger)
	if err != nil {
		return nil, err
	}

	var snsOpts []func(*amazonsns.Options)
	var sqsOpts []func(*amazonsqs.Options)
	snsOpts, sqsOpts, err = addEndpointResolver(cfg, awsCfg, snsOpts, sqsOpts)
	if err != nil {
		return nil, err
	}

	subscriberConfig := sns.SubscriberConfig{
		AWSConfig:            *awsCfg,
		OptFns:               snsOpts,
		TopicResolver:        topicResolver,
		GenerateSqsQueueName: makeSqsQueueNameGenerator(),
	}

	return SubscriberFactory(
		subscriberConfig,
		sqs.SubscriberConfig{
			AWSConfig: *awsCfg,
			OptFns:    sqsOpts,
		},
		logger,
	)
}

func addEndpointResolver(cfg transport.Config, awsCfg *aws.Config, snsOpts []func(*amazonsns.Options), sqsOpts []func(*amazonsqs.Options)) ([]func(*amazonsns.Options), []func(*amazonsqs.Options), error) {
	if hasCustomEndpoint(awsCfg) {
		endpoint, err := awsEndpointURL(cfg)
		if err != nil {
			return nil, nil, err
		}
		parsedURL, err := url.Parse(*awsCfg.BaseEndpoint)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse BaseEndpoint: %w", err)
		}
		_ = endpoint // For consistency with the original code
		snsOpts = []func(*amazonsns.Options){
			amazonsns.WithEndpointResolverV2(sns.OverrideEndpointResolver{
				Endpoint: smithyendpoints.Endpoint{
					URI: *parsedURL,
				},
			}),
		}
		sqsOpts = []func(*amazonsqs.Options){
			amazonsqs.WithEndpointResolverV2(sqs.OverrideEndpointResolver{
				Endpoint: smithyendpoints.Endpoint{
					URI: *parsedURL,
				},
			}),
		}
	}
	return snsOpts, sqsOpts, nil
}

func resolveAccountAndRegion(cfg transport.Config, logger watermill.LoggerAdapter, fallbackRegion string) (string, string) {
	if cfg == nil {
		return "", fallbackRegion
	}

	accountID := strings.Trim(cfg.GetAWSAccountID(), "\"' ")
	region := cfg.GetAWSRegion()
	if region == "" {
		region = fallbackRegion
	}

	if accountID == "" && useLocalstackEndpoint(cfg) {
		accountID = localstackAccountID
		logger.Info("AWS account ID empty; using LocalStack default", watermill.LogFields{"accountID": accountID})
		return accountID, region
	}

	if accountID != "" && len(accountID) != awsAccountIDLength && useLocalstackEndpoint(cfg) {
		logger.Info("Invalid AWS account ID; falling back to LocalStack default", watermill.LogFields{"accountID": accountID})
		accountID = localstackAccountID
	}

	return accountID, region
}

func useLocalstackEndpoint(cfg transport.Config) bool {
	return cfg != nil && cfg.GetAWSEndpoint() != ""
}

func createTopicResolver(accountID, region string, logger watermill.LoggerAdapter) (sns.TopicResolver, error) {
	topicResolver, err := TopicResolverFactory(accountID, region)
	if err != nil {
		logger.Error("Failed to create SNS topic resolver", err, watermill.LogFields{
			"accountID": accountID,
			"region":    region,
		})
		return nil, err
	}
	return topicResolver, nil
}

func buildPublisherConfig(cfg transport.Config, awsCfg *aws.Config, topicResolver sns.TopicResolver, logger watermill.LoggerAdapter) (sns.PublisherConfig, error) {
	endpoint, err := awsEndpointURL(cfg)
	if err != nil {
		logger.Error("Failed to parse AWS endpoint", err, watermill.LogFields{"endpoint": cfg.GetAWSEndpoint()})
		return sns.PublisherConfig{}, err
	}

	publisherConfig := sns.PublisherConfig{
		TopicResolver: topicResolver,
		AWSConfig:     *awsCfg,
		Marshaler:     sns.DefaultMarshalerUnmarshaler{},
	}

	if endpoint != nil {
		endpointStr := endpoint.String()
		publisherConfig.OptFns = []func(*amazonsns.Options){
			func(o *amazonsns.Options) {
				o.BaseEndpoint = aws.String(endpointStr)
			},
		}
	}

	return publisherConfig, nil
}

func awsEndpointURL(cfg transport.Config) (*url.URL, error) {
	if cfg == nil || cfg.GetAWSEndpoint() == "" {
		return nil, nil
	}

	parsedURL, err := url.Parse(cfg.GetAWSEndpoint())
	if err != nil {
		return nil, fmt.Errorf("failed to parse AWS endpoint: %w", err)
	}
	return parsedURL, nil
}

func safeAWSRegion(cfg *aws.Config) string {
	if cfg == nil {
		return ""
	}
	return cfg.Region
}

func hasCustomEndpoint(cfg *aws.Config) bool {
	return cfg != nil && cfg.BaseEndpoint != nil && *cfg.BaseEndpoint != ""
}

func staticCredentialsProvider(accessKeyID, secretAccessKey string) aws.CredentialsProvider {
	return aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
		return aws.Credentials{
			AccessKeyID:     accessKeyID,
			SecretAccessKey: secretAccessKey,
		}, nil
	})
}
