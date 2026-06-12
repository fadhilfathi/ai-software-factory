package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/fadhilfathi/AI-Software-Factory/internal/config"
	"github.com/fadhilfathi/AI-Software-Factory/internal/logger"
	"github.com/fadhilfathi/AI-Software-Factory/internal/router"
	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store/postgres"
	"github.com/fadhilfathi/AI-Software-Factory/db"
)

func main() {
	cfg := config.Load()

	if p := os.Getenv("PORT"); p != "" {
		if port, err := strconv.Atoi(p); err == nil {
			cfg.Server.Port = port
		}
	}

	logger, err := logger.New(os.Getenv("LOG_LEVEL"))
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}

	var st store.Store
	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		dbCfg := db.Config{
			Host:     dbHost,
			Port:     getEnvOrDefault("DB_PORT", "5432"),
			User:     getEnvOrDefault("DB_USER", "postgres"),
			Password: getEnvOrDefault("DB_PASSWORD", "postgres"),
			DBName:   getEnvOrDefault("DB_NAME", "ai_factory"),
			SSLMode:  getEnvOrDefault("DB_SSLMODE", "disable"),
		}

		pool, err := db.Connect(context.Background(), dbCfg)
		if err != nil {
			log.Fatalf("failed to connect to database: %v", err)
		}
		defer pool.Close()

		if err := db.RunMigrations(context.Background(), pool, "db/migrations"); err != nil {
			log.Fatalf("failed to run migrations: %v", err)
		}

		st = postgres.NewStore(pool)
		logger.Info("Using PostgreSQL store")
	} else {
		st = store.NewMemoryStore()
		logger.Info("Using in-memory store (no DB_HOST set)")
	}

	svc := service.New(st, logger, cfg)
	r := router.New(svc, cfg.CORS, cfg.RateLimit)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("API server starting on %s", addr)

	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
