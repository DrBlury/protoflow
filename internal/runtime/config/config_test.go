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
	if !strings.Contains(str, "user") {
		t.Error("Config.String() should preserve username in RabbitMQ URL")
	}
	if !strings.Contains(str, "admin") {
		t.Error("Config.String() should preserve username in NATS URL")
	}
}

// Transport validation tests
func TestConfigValidate_ChannelTransport(t *testing.T) {
	tests := []struct {
		name   string
		config Config
	}{
		{"empty config defaults to channel", Config{}},
		{"explicit channel", Config{PubSubSystem: "channel"}},
		{"gochannel alias", Config{PubSubSystem: "gochannel"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.config.Validate(); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestConfigValidate_KafkaTransport(t *testing.T) {
	t.Run("missing brokers", func(t *testing.T) {
		cfg := Config{PubSubSystem: "kafka"}
		err := cfg.Validate()
		assertErrorContains(t, err, "kafka: brokers are required")
	})

	t.Run("valid", func(t *testing.T) {
		cfg := Config{PubSubSystem: "kafka", KafkaBrokers: []string{"localhost:9092"}}
		if err := cfg.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestConfigValidate_RabbitMQTransport(t *testing.T) {
	t.Run("missing url", func(t *testing.T) {
		cfg := Config{PubSubSystem: "rabbitmq"}
		err := cfg.Validate()
		assertErrorContains(t, err, "rabbitmq: URL is required")
	})

	t.Run("valid", func(t *testing.T) {
		cfg := Config{PubSubSystem: "rabbitmq", RabbitMQURL: "amqp://localhost:5672"}
		if err := cfg.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestConfigValidate_NATSTransport(t *testing.T) {
	t.Run("missing url", func(t *testing.T) {
		cfg := Config{PubSubSystem: "nats"}
		err := cfg.Validate()
		assertErrorContains(t, err, "nats: URL is required")
	})

	t.Run("valid", func(t *testing.T) {
		cfg := Config{PubSubSystem: "nats", NATSURL: "nats://localhost:4222"}
		if err := cfg.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestConfigValidate_AWSTransport(t *testing.T) {
	t.Run("missing region", func(t *testing.T) {
		cfg := Config{PubSubSystem: "aws"}
		err := cfg.Validate()
		assertErrorContains(t, err, "aws: region is required")
	})

	t.Run("valid", func(t *testing.T) {
		cfg := Config{PubSubSystem: "aws", AWSRegion: "us-east-1"}
		if err := cfg.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestConfigValidate_CustomTransport(t *testing.T) {
	cfg := Config{PubSubSystem: "custom-transport"}
	if err := cfg.Validate(); err != nil {
		t.Errorf("custom transport should be allowed: %v", err)
	}
}

// Retry configuration tests
func TestConfigValidate_RetryConfig(t *testing.T) {
	t.Run("negative max retries", func(t *testing.T) {
		cfg := Config{RetryMaxRetries: -1}
		err := cfg.Validate()
		assertErrorContains(t, err, "retry: max retries cannot be negative")
	})

	t.Run("negative initial interval", func(t *testing.T) {
		cfg := Config{RetryInitialInterval: -1 * time.Second}
		err := cfg.Validate()
		assertErrorContains(t, err, "retry: initial interval cannot be negative")
	})

	t.Run("negative max interval", func(t *testing.T) {
		cfg := Config{RetryMaxInterval: -1 * time.Second}
		err := cfg.Validate()
		assertErrorContains(t, err, "retry: max interval cannot be negative")
	})

	t.Run("initial exceeds max", func(t *testing.T) {
		cfg := Config{
			RetryInitialInterval: 10 * time.Second,
			RetryMaxInterval:     5 * time.Second,
		}
		err := cfg.Validate()
		assertErrorContains(t, err, "retry: initial interval cannot exceed max interval")
	})

	t.Run("valid retry config", func(t *testing.T) {
		cfg := Config{
			RetryMaxRetries:      5,
			RetryInitialInterval: 1 * time.Second,
			RetryMaxInterval:     30 * time.Second,
		}
		if err := cfg.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

// Port configuration tests
func TestConfigValidate_Ports(t *testing.T) {
	t.Run("invalid metrics port high", func(t *testing.T) {
		cfg := Config{MetricsPort: 70000}
		err := cfg.Validate()
		assertErrorContains(t, err, "metrics: invalid port")
	})

	t.Run("invalid webui port negative", func(t *testing.T) {
		cfg := Config{WebUIPort: -1}
		err := cfg.Validate()
		assertErrorContains(t, err, "webui: invalid port")
	})

	t.Run("valid ports", func(t *testing.T) {
		cfg := Config{MetricsPort: 9090, WebUIPort: 8081}
		if err := cfg.Validate(); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
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

// assertErrorContains is a test helper that checks if an error contains a substring.
func assertErrorContains(t *testing.T, err error, want string) {
	t.Helper()
	if err == nil {
		t.Errorf("expected error containing %q, got nil", want)
		return
	}
	if !strings.Contains(err.Error(), want) {
		t.Errorf("expected error containing %q, got %q", want, err.Error())
	}
}
