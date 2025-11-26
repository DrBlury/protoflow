package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Config groups the Pub/Sub settings required to initialise the Service. Each
// transport only uses the keys that are relevant to it.
type Config struct {
	// PubSubSystem selects the backing message infrastructure. Supported values:
	// "kafka", "rabbitmq", or "aws" (SNS/SQS).
	PubSubSystem string

	// Kafka configuration.
	KafkaBrokers       []string
	KafkaClientID      string
	KafkaConsumerGroup string

	// RabbitMQ configuration.
	RabbitMQURL string

	// NATS configuration.
	NATSURL string

	// HTTP configuration.
	HTTPServerAddress string
	// HTTPPublisherURL is the base URL where messages will be sent.
	HTTPPublisherURL string

	// I/O configuration.
	// IOFile is the path to the file used for persistence.
	IOFile string

	// SQLite configuration.
	// SQLiteFile is the path to the SQLite database file.
	// Use ":memory:" for an in-memory database (useful for testing).
	SQLiteFile string

	// PostgreSQL configuration.
	// PostgresURL is the PostgreSQL connection string.
	// Example: "postgres://user:password@localhost:5432/dbname?sslmode=disable"
	PostgresURL string

	// PoisonQueue receives messages that cannot be processed even after retries.
	PoisonQueue string

	// AWS (SNS/SQS) configuration.
	AWSRegion          string
	AWSAccountID       string
	AWSAccessKeyID     string
	AWSSecretAccessKey string
	// AWSEndpoint optionally points to a custom endpoint (for example, LocalStack
	// in local development).
	AWSEndpoint string

	// RetryMiddleware tuning. Zero values fall back to library defaults.
	RetryMaxRetries      int
	RetryInitialInterval time.Duration
	RetryMaxInterval     time.Duration

	// Metrics configuration.
	MetricsEnabled bool
	// MetricsPort is the port where Prometheus metrics will be exposed.
	MetricsPort int

	// WebUI configuration.
	WebUIEnabled bool
	// WebUIPort is the port where the WebUI API will be exposed. Defaults to 8081.
	WebUIPort int
	// WebUICORSAllowedOrigins specifies allowed origins for CORS. Use "*" for development
	// or specific origins like "https://example.com" for production. Empty disables CORS headers.
	WebUICORSAllowedOrigins []string
}

// Getter methods to implement transport.Config interface.
func (c *Config) GetPubSubSystem() string       { return c.PubSubSystem }
func (c *Config) GetKafkaBrokers() []string     { return c.KafkaBrokers }
func (c *Config) GetKafkaConsumerGroup() string { return c.KafkaConsumerGroup }
func (c *Config) GetRabbitMQURL() string        { return c.RabbitMQURL }
func (c *Config) GetNATSURL() string            { return c.NATSURL }
func (c *Config) GetHTTPServerAddress() string  { return c.HTTPServerAddress }
func (c *Config) GetHTTPPublisherURL() string   { return c.HTTPPublisherURL }
func (c *Config) GetIOFile() string             { return c.IOFile }
func (c *Config) GetSQLiteFile() string         { return c.SQLiteFile }
func (c *Config) GetPostgresURL() string        { return c.PostgresURL }
func (c *Config) GetAWSRegion() string          { return c.AWSRegion }
func (c *Config) GetAWSAccountID() string       { return c.AWSAccountID }
func (c *Config) GetAWSAccessKeyID() string     { return c.AWSAccessKeyID }
func (c *Config) GetAWSSecretAccessKey() string { return c.AWSSecretAccessKey }
func (c *Config) GetAWSEndpoint() string        { return c.AWSEndpoint }

func (c Config) String() string {
	// Create a copy to avoid modifying the original
	copy := c
	if copy.AWSSecretAccessKey != "" {
		copy.AWSSecretAccessKey = "***REDACTED***"
	}
	if copy.AWSAccessKeyID != "" {
		copy.AWSAccessKeyID = "***REDACTED***"
	}
	// Redact credentials that may be embedded in connection URLs
	if copy.RabbitMQURL != "" {
		copy.RabbitMQURL = redactURLCredentials(copy.RabbitMQURL)
	}
	if copy.NATSURL != "" {
		copy.NATSURL = redactURLCredentials(copy.NATSURL)
	}
	if copy.PostgresURL != "" {
		copy.PostgresURL = redactURLCredentials(copy.PostgresURL)
	}
	// Use a type alias to avoid infinite recursion when printing
	type configAlias Config
	return fmt.Sprintf("%+v", configAlias(copy))
}

// redactURLCredentials masks password in URLs like amqp://user:pass@host
func redactURLCredentials(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		// If parsing fails, redact the whole thing to be safe
		return "***REDACTED_URL***"
	}
	if parsed.User != nil {
		if _, hasPassword := parsed.User.Password(); hasPassword {
			parsed.User = url.UserPassword(parsed.User.Username(), "***REDACTED***")
		}
	}
	return parsed.String()
}

// Validate checks that the configuration has all required fields for the selected transport.
// Returns an error describing any missing or invalid configuration.
// Note: validation of pubsub system values is lenient to allow custom transport factories.
func (c *Config) Validate() error {
	var errs []error

	errs = append(errs, c.validateTransport()...)
	errs = append(errs, c.validateRetry()...)
	errs = append(errs, c.validatePorts()...)

	return errors.Join(errs...)
}

// validateTransport checks transport-specific required fields.
func (c *Config) validateTransport() []error {
	switch strings.ToLower(c.PubSubSystem) {
	case "kafka":
		if len(c.KafkaBrokers) == 0 {
			return []error{errors.New("kafka: brokers are required")}
		}
	case "rabbitmq":
		if c.RabbitMQURL == "" {
			return []error{errors.New("rabbitmq: URL is required")}
		}
	case "nats":
		if c.NATSURL == "" {
			return []error{errors.New("nats: URL is required")}
		}
	case "aws":
		if c.AWSRegion == "" {
			return []error{errors.New("aws: region is required")}
		}
	}
	// http, io, channel, gochannel, "", and custom transports have no required config
	return nil
}

// validateRetry checks retry configuration values.
func (c *Config) validateRetry() []error {
	var errs []error
	if c.RetryMaxRetries < 0 {
		errs = append(errs, errors.New("retry: max retries cannot be negative"))
	}
	if c.RetryInitialInterval < 0 {
		errs = append(errs, errors.New("retry: initial interval cannot be negative"))
	}
	if c.RetryMaxInterval < 0 {
		errs = append(errs, errors.New("retry: max interval cannot be negative"))
	}
	if c.RetryMaxInterval > 0 && c.RetryInitialInterval > 0 && c.RetryInitialInterval > c.RetryMaxInterval {
		errs = append(errs, errors.New("retry: initial interval cannot exceed max interval"))
	}
	return errs
}

// validatePorts checks port configuration values.
func (c *Config) validatePorts() []error {
	var errs []error
	if c.MetricsPort < 0 || c.MetricsPort > 65535 {
		errs = append(errs, fmt.Errorf("metrics: invalid port %d", c.MetricsPort))
	}
	if c.WebUIPort < 0 || c.WebUIPort > 65535 {
		errs = append(errs, fmt.Errorf("webui: invalid port %d", c.WebUIPort))
	}
	return errs
}

// ValidateConfig is a convenience function to validate a config pointer.
// Returns nil if the config is valid.
func ValidateConfig(c *Config) error {
	if c == nil {
		return errors.New("config is nil")
	}
	return c.Validate()
}
