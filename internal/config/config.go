package config

import "github.com/spf13/viper"

type Config struct {
	Postgres  PostgresConfig
	Anthropic AnthropicConfig
}

func Load() (*Config, error) {
	viper.SetConfigFile(".env")
	viper.SetConfigType("env")
	viper.AutomaticEnv()

	// .env file is optional — environment variables take precedence
	_ = viper.ReadInConfig()

	cfg := &Config{
		Postgres:  loadPostgresConfig(),
		Anthropic: loadAnthropicConfig(),
	}

	return cfg, nil
}
