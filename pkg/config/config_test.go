package config

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Test Default configuration
	cfg := LoadConfig()
	if cfg.DatabaseType != "memory" {
		t.Errorf("expected default database type to be memory, got %s", cfg.DatabaseType)
	}

	// Test Override configuration
	os.Setenv("DATABASE_TYPE", "postgres")
	os.Setenv("PORT_INGEST", "9000")
	defer func() {
		os.Unsetenv("DATABASE_TYPE")
		os.Unsetenv("PORT_INGEST")
	}()

	cfgOverride := LoadConfig()
	if cfgOverride.DatabaseType != "postgres" {
		t.Errorf("expected overridden database type to be postgres, got %s", cfgOverride.DatabaseType)
	}
	if cfgOverride.PortIngest != "9000" {
		t.Errorf("expected overridden port ingest to be 9000, got %s", cfgOverride.PortIngest)
	}
}
