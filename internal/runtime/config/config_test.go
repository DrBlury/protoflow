package config

import (
	"strings"
	"testing"
	"time"
)

func TestConfigStringRedaction(t *testing.T) {
	cfg := Config{
		AWSAccessKeyID:     "my-access-key",
		AWSSecretAccessKey: "my-secret-key",
		AWSRegion:          "us-east-1",
	}

	str := cfg.String()

	if strings.Contains(str, "my-access-key") {
		t.Error("Config.String() should redact AWSAccessKeyID")
	}
	if strings.Contains(str, "my-secret-key") {
		t.Error("Config.String() should redact AWSSecretAccessKey")
	}
	if !strings.Contains(str, "***REDACTED***") {
		t.Error("Config.String() should contain redaction marker")
	}
	if !strings.Contains(str, "us-east-1") {
		t.Error("Config.String() should contain non-sensitive fields")
	}
}

func TestConfigStringRedactsURLCredentials(t *testing.T) {
	cfg := Config{
		RabbitMQURL: "amqp://user:secret-password@localhost:5672/",
		NATSURL:     "nats://admin:nats-secret@localhost:4222",
	}

	str := cfg.String()

	if strings.Contains(str, "secret-password") {
		t.Error("Config.String() should redact RabbitMQ password")
	}
	if strings.Contains(str, "nats-secret") {
		t.Error("Config.String() should redact NATS password")
	}
	// Should still contain username
	if !strings.Contains(str, "user") {
		t.Error("Config.String() should preserve username in RabbitMQ URL")
	}
	if !strings.Contains(str, "admin") {
		t.Error("Config.String() should preserve username in NATS URL")
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name      string
		config    Config
		wantError bool
		errorMsg  string
	}{
		{
			name:      "empty config is valid (channel transport)",
			config:    Config{},
			wantError: false,
		},
		{
			name:      "channel transport valid",
			config:    Config{PubSubSystem: "channel"},
			wantError: false,
		},
		{
			name:      "kafka missing brokers",
			config:    Config{PubSubSystem: "kafka"},
			wantError: true,
			errorMsg:  "kafka: brokers are required",
		},
		{
			name: "kafka valid",
			config: Config{
				PubSubSystem: "kafka",
				KafkaBrokers: []string{"localhost:9092"},
			},
			wantError: false,
		},
		{
			name:      "rabbitmq missing url",
			config:    Config{PubSubSystem: "rabbitmq"},
			wantError: true,
			errorMsg:  "rabbitmq: URL is required",
		},
		{
			name: "rabbitmq valid",
			config: Config{
				PubSubSystem: "rabbitmq",
				RabbitMQURL:  "amqp://localhost:5672",
			},
			wantError: false,
		},
		{
			name:      "nats missing url",
			config:    Config{PubSubSystem: "nats"},
			wantError: true,
			errorMsg:  "nats: URL is required",
		},
		{
			name: "nats valid",
			config: Config{
				PubSubSystem: "nats",
				NATSURL:      "nats://localhost:4222",
			},
			wantError: false,
		},
		{
			name:      "aws missing region",
			config:    Config{PubSubSystem: "aws"},
			wantError: true,
			errorMsg:  "aws: region is required",
		},
		{
			name: "aws valid",
			config: Config{
				PubSubSystem: "aws",
				AWSRegion:    "us-east-1",
			},
			wantError: false,
		},
		{
			name:      "custom transport is allowed",
			config:    Config{PubSubSystem: "custom"},
			wantError: false,
		},
		{
			name:      "negative max retries invalid",
			config:    Config{RetryMaxRetries: -1},
			wantError: true,
			errorMsg:  "retry: max retries cannot be negative",
		},
		{
			name:      "negative initial interval invalid",
			config:    Config{RetryInitialInterval: -1 * time.Second},
			wantError: true,
			errorMsg:  "retry: initial interval cannot be negative",
		},
		{
			name: "initial interval exceeds max invalid",
			config: Config{
				RetryInitialInterval: 10 * time.Second,
				RetryMaxInterval:     5 * time.Second,
			},
			wantError: true,
			errorMsg:  "retry: initial interval cannot exceed max interval",
		},
		{
			name:      "invalid metrics port",
			config:    Config{MetricsPort: 70000},
			wantError: true,
			errorMsg:  "metrics: invalid port",
		},
		{
			name:      "invalid webui port",
			config:    Config{WebUIPort: -1},
			wantError: true,
			errorMsg:  "webui: invalid port",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantError {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errorMsg)
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errorMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidateConfigNil(t *testing.T) {
	err := ValidateConfig(nil)
	if err == nil {
		t.Error("expected error for nil config")
	}
	if !strings.Contains(err.Error(), "nil") {
		t.Errorf("expected error message to mention nil, got %q", err.Error())
	}
}

func TestRedactURLCredentials(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		shouldContain    string
		shouldNotContain string
	}{
		{
			name:          "URL without credentials",
			input:         "amqp://localhost:5672/",
			shouldContain: "localhost:5672",
		},
		{
			name:          "URL with username only",
			input:         "amqp://user@localhost:5672/",
			shouldContain: "user@localhost",
		},
		{
			name:             "URL with credentials",
			input:            "amqp://user:password@localhost:5672/",
			shouldContain:    "REDACTED",
			shouldNotContain: "password",
		},
		{
			name:          "invalid URL",
			input:         "not-a-valid-url://[invalid",
			shouldContain: "REDACTED",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := redactURLCredentials(tt.input)
			if tt.shouldContain != "" && !strings.Contains(result, tt.shouldContain) {
				t.Errorf("expected result to contain %q, got %q", tt.shouldContain, result)
			}
			if tt.shouldNotContain != "" && strings.Contains(result, tt.shouldNotContain) {
				t.Errorf("expected result to NOT contain %q, got %q", tt.shouldNotContain, result)
			}
		})
	}
}
