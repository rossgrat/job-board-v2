package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type PostgresConfig struct {
	User     string
	Password string
	Host     string
	DB       string
}

func (p PostgresConfig) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:5432/%s?sslmode=disable",
		p.User, p.Password, p.Host, p.DB,
	)
}

func loadPostgresConfig() PostgresConfig {
	return PostgresConfig{
		User:     viper.GetString("POSTGRES_USER"),
		Password: viper.GetString("POSTGRES_PASSWORD"),
		Host:     viper.GetString("POSTGRES_HOST"),
		DB:       viper.GetString("POSTGRES_DB"),
	}
}
