package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/example/project/internal/config"
	"github.com/example/project/internal/logger"
	"github.com/example/project/internal/router"
	"github.com/example/project/internal/service"
	"github.com/example/project/internal/store"
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
	svc := service.New(st, logger)

	handler := router.New(svc)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("API server starting on %s", addr)

	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
	}

	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
}
