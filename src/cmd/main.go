package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/fadhilfathi/AI-Software-Factory/internal/config"
	"github.com/fadhilfathi/AI-Software-Factory/internal/logger"
	"github.com/fadhilfathi/AI-Software-Factory/internal/router"
	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
)

func main() {
	cfg := config.Load()

	// Override port from env if set.
	if p := os.Getenv("PORT"); p != "" {
		if port, err := strconv.Atoi(p); err == nil {
			cfg.Server.Port = port
		}
	}

	logger, err := logger.New(os.Getenv("LOG_LEVEL"))
	if err != nil {
		log.Fatalf("failed to initialize logger: %v", err)
	}

	// Initialize in-memory store and service layer.
	st := store.NewMemoryStore()
	svc := service.New(st, logger, cfg.Auth.JWTSecret)

	r := router.New(svc, cfg.CORS, cfg.RateLimit)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("API server starting on %s", addr)

	if err := r.Run(addr); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
