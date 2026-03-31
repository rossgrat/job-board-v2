package config

import "github.com/spf13/viper"

type TelemetryConfig struct {
	OTLPEndpoint string
}

func loadTelemetryConfig() TelemetryConfig {
	viper.SetDefault("OTEL_EXPORTER_OTLP_ENDPOINT", "localhost:4317")
	return TelemetryConfig{
		OTLPEndpoint: viper.GetString("OTEL_EXPORTER_OTLP_ENDPOINT"),
	}
}
