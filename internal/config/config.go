package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	Port           string
	Env            string
	DatabaseURL    string
	WebhookSecrets map[string]string
	LogLevel       string
	LogFormat      string
}

func Load() (*Config, error) {
	var missing []string

	get := func(key string) string {
		v := os.Getenv(key)
		if v == "" {
			missing = append(missing, key)
		}
		return v
	}

	getWithDefault := func(key, def string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return def
	}

	cfg := &Config{
		Port:        getWithDefault("PORT", "8080"),
		Env:         getWithDefault("ENV", "development"),
		DatabaseURL: get("DATABASE_URL"),
		LogLevel:    getWithDefault("LOG_LEVEL", "info"),
		LogFormat:   getWithDefault("LOG_FORMAT", "pretty"),
		WebhookSecrets: map[string]string{
			"custom": get("WEBHOOK_SECRET_CUSTOM"),
		},
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return cfg, nil
}
