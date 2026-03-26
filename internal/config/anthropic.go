package config

import "github.com/spf13/viper"

type AnthropicConfig struct {
	APIKey string
}

func loadAnthropicConfig() AnthropicConfig {
	return AnthropicConfig{
		APIKey: viper.GetString("ANTHROPIC_API_KEY"),
	}
}
