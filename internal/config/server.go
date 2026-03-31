package config

import "github.com/spf13/viper"

type ServerConfig struct {
	Port string
}

func loadServerConfig() ServerConfig {
	viper.SetDefault("SERVER_PORT", "8080")
	return ServerConfig{
		Port: viper.GetString("SERVER_PORT"),
	}
}
