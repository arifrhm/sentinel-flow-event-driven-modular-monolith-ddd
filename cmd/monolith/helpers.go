package main

import (
	"database/sql"
	"log/slog"
	"os"

	_ "github.com/lib/pq"
	"sentinel-flow/pkg/broker"
	"sentinel-flow/pkg/config"
)

func setupLogger(logFormat string) *slog.Logger {
	var logHandler slog.Handler
	if logFormat == "json" {
		logHandler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	} else {
		logHandler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	}
	return slog.New(logHandler)
}

func initDatabase(cfg *config.Config) *sql.DB {
	if cfg.DatabaseType != "postgres" && cfg.DatabaseType != "postgres-mock" {
		slog.Info("[Cluster] Using In-Memory Database Configurations.")
		return nil
	}

	driver := "postgres"
	if cfg.DatabaseType == "postgres-mock" {
		driver = "mockdb"
	}
	if os.Getenv("BE_CRASHER_DB_OPEN") == "1" {
		driver = "invalid_driver"
	}

	db, err := sql.Open(driver, cfg.DatabaseURL)
	if err != nil {
		slog.Error("Failed to initialize PostgreSQL database connection", "error", err)
		os.Exit(1)
	}

	if err := db.Ping(); err != nil {
		slog.Error("Failed to ping PostgreSQL database", "error", err)
		os.Exit(1)
	}

	// Run migrations
	migrationPath := "pkg/db/migrations/0001_init.sql"
	if _, err := os.Stat(migrationPath); os.IsNotExist(err) {
		migrationPath = "../../pkg/db/migrations/0001_init.sql"
	}
	content, err := os.ReadFile(migrationPath)
	if err != nil {
		slog.Error("Failed to read schema migrations file", "path", migrationPath, "error", err)
		os.Exit(1)
	}

	_, err = db.Exec(string(content))
	if err != nil {
		slog.Error("Failed to apply database DDL migrations", "error", err)
		os.Exit(1)
	}

	slog.Info("[Postgres] Database connection initialized and migrations applied.")
	return db
}

func initBroker(cfg *config.Config) broker.Broker {
	if cfg.BrokerType == "redis" {
		redBroker, err := broker.NewRedisBroker(cfg.RedisURL, 10000)
		if err != nil {
			slog.Error("Failed to initialize Redis event broker", "error", err)
			os.Exit(1)
		}
		return redBroker
	}
	slog.Info("[Cluster] Using In-Memory Channel Event Broker.")
	return broker.NewInMemoryBroker(10000)
}
