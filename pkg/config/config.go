package config

import (
	"os"
)

// Config holds all config variables for the Sentinel-Flow Monolith.
type Config struct {
	DatabaseType string // "memory" or "postgres"
	DatabaseURL  string
	BrokerType   string // "memory" or "redis"
	RedisURL     string
	LogFormat    string // "text" or "json"
	PortIngest   string
	PortMarketing string
	PortAnalytics string
	PortCRM       string
	CRMURL       string
}

// LoadConfig reads configuration parameters from environment variables or returns defaults.
func LoadConfig() *Config {
	return &Config{
		DatabaseType:  getEnv("DATABASE_TYPE", "memory"),
		DatabaseURL:   getEnv("DATABASE_URL", "postgres://sentinel_flow_admin:sentinel_flow_secure_password@localhost:5432/sentinel_flow_production?sslmode=disable"),
		BrokerType:    getEnv("BROKER_TYPE", "memory"),
		RedisURL:      getEnv("REDIS_URL", "redis://localhost:6379"),
		LogFormat:     getEnv("LOG_FORMAT", "text"),
		PortIngest:    getEnv("PORT_INGEST", "8081"),
		PortMarketing: getEnv("PORT_MARKETING", "8082"),
		PortAnalytics: getEnv("PORT_ANALYTICS", "8083"),
		PortCRM:       getEnv("PORT_CRM", "8084"),
		CRMURL:        getEnv("CRM_URL", "http://localhost:8084/crm/sync"),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}
