package config

import "github.com/spf13/viper"

type AuthConfig struct {
	Password string
}

func loadAuthConfig() AuthConfig {
	return AuthConfig{
		Password: viper.GetString("AUTH_PASSWORD"),
	}
}
