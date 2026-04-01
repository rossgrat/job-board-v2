package config

import (
	"log/slog"
	"os"

	"github.com/spf13/viper"
)

type Config struct {
	Postgres  PostgresConfig
	Anthropic AnthropicConfig
	Server    ServerConfig
	Telemetry TelemetryConfig
	Auth      AuthConfig
}

func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	// .env file is optional — environment variables take precedence
	if err := viper.ReadInConfig(); err != nil {
		slog.Info("no .env file found, using environment variables")
	}

	cfg := &Config{
		Postgres:  loadPostgresConfig(),
		Anthropic: loadAnthropicConfig(),
		Server:    loadServerConfig(),
		Telemetry: loadTelemetryConfig(),
		Auth:      loadAuthConfig(),
	}

	return cfg, nil
}
