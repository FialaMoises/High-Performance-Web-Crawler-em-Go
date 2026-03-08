package config

import (
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	url := "https://example.com"
	cfg := DefaultConfig(url)

	if cfg.StartURL != url {
		t.Errorf("Expected StartURL to be %s, got %s", url, cfg.StartURL)
	}

	if cfg.MaxDepth != 3 {
		t.Errorf("Expected MaxDepth to be 3, got %d", cfg.MaxDepth)
	}

	if cfg.NumWorkers != 10 {
		t.Errorf("Expected NumWorkers to be 10, got %d", cfg.NumWorkers)
	}

	if !cfg.RespectRobotsTxt {
		t.Error("Expected RespectRobotsTxt to be true")
	}

	if !cfg.SameDomainOnly {
		t.Error("Expected SameDomainOnly to be true")
	}
}

func TestConfig_Validate_ValidConfig(t *testing.T) {
	cfg := &Config{
		StartURL:       "https://example.com",
		NumWorkers:     5,
		RequestTimeout: 10 * time.Second,
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("Valid config should not return error, got: %v", err)
	}
}

func TestConfig_Validate_EmptyURL(t *testing.T) {
	cfg := &Config{
		StartURL:       "",
		NumWorkers:     5,
		RequestTimeout: 10 * time.Second,
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error for empty StartURL")
	}

	if _, ok := err.(*ConfigError); !ok {
		t.Errorf("Expected ConfigError, got %T", err)
	}
}

func TestConfig_Validate_InvalidWorkers(t *testing.T) {
	cfg := &Config{
		StartURL:       "https://example.com",
		NumWorkers:     0,
		RequestTimeout: 10 * time.Second,
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error for NumWorkers < 1")
	}
}

func TestConfig_Validate_InvalidTimeout(t *testing.T) {
	cfg := &Config{
		StartURL:       "https://example.com",
		NumWorkers:     5,
		RequestTimeout: 500 * time.Millisecond,
	}

	err := cfg.Validate()
	if err == nil {
		t.Error("Expected error for RequestTimeout < 1 second")
	}
}

func TestConfigError_Error(t *testing.T) {
	err := &ConfigError{
		Field:   "StartURL",
		Message: "cannot be empty",
	}

	expected := "config error: StartURL cannot be empty"
	if err.Error() != expected {
		t.Errorf("Expected error message %q, got %q", expected, err.Error())
	}
}
