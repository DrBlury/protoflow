package config

import (
	"strings"
	"testing"
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
