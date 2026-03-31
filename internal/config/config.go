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
}

func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	// .env file is optional — environment variables take precedence
	if err := viper.ReadInConfig(); err != nil {
		slog.Error("Failed to read config")
		os.Exit(1)
	}

	cfg := &Config{
		Postgres:  loadPostgresConfig(),
		Anthropic: loadAnthropicConfig(),
		Server:    loadServerConfig(),
	}

	return cfg, nil
}
