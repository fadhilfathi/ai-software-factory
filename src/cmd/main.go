package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/config"
	"github.com/fadhilfathi/AI-Software-Factory/internal/events"
	"github.com/fadhilfathi/AI-Software-Factory/internal/logger"
	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/middleware"
	"github.com/fadhilfathi/AI-Software-Factory/internal/router"
	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store/postgres"
	"github.com/fadhilfathi/AI-Software-Factory/db"
	"github.com/google/uuid"
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
		// TASK-512 follow-up: use DefaultConfig() as base so MaxConns/MinConns/timeouts
		// get sensible defaults (Go zero values would have pgxpool reject with
		// "MaxSize must be >= 1"). Override Host last so DB_HOST wins.
		dbCfg := db.DefaultConfig()
		dbCfg.Host = dbHost

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

	// TASK-503 (Sprint 5, minimal): instantiate the in-process event bus
	// and pass it into the Services container. The bus is NOT yet wired
	// into the ExecutionService itself (that refactor is Sprint 6 per
	// Lead's dispatch 2026-06-14); it is stored on Services.Bus so future
	// TASK-501/TASK-505/TASK-506 code can publish and subscribe without
	// having to thread a new dependency through the constructor chain.
	bus := events.NewMemoryBus()

	svc := service.New(st, buildAPIKeyStore(), logger, cfg, bus)
	corsMW := middleware.CORSConfig{
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		AllowedMethods:   cfg.CORS.AllowedMethods,
		AllowedHeaders:   cfg.CORS.AllowedHeaders,
		AllowCredentials: cfg.CORS.AllowCredentials,
		MaxAge:           cfg.CORS.MaxAge,
	}
	// TASK-512 follow-up (5th bug): use DefaultRateLimitConfig() as base so KeyFunc
	// gets a non-nil default (IP-based). Without it, middleware.go:261 panics on every
	// request with 'invalid memory address or nil pointer dereference' (nil func ptr).
	rateMW := middleware.DefaultRateLimitConfig()
	rateMW.RequestsPerMinute = cfg.RateLimit.RequestsPerMinute
	rateMW.Burst = cfg.RateLimit.Burst
	r := router.New(svc, corsMW, rateMW)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: r,
	}

	log.Printf("API server starting on %s", addr)

	if err := runServer(srv, svc); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// runServer starts srv in a goroutine, blocks until SIGINT/SIGTERM arrives
// (or the server fails to start), then runs the graceful shutdown sequence.
// The graceful shutdown drains in-flight HTTP requests first and then the
// Execution service's in-flight goroutines, bounded by the SHUTDOWN_GRACE
// timeout (default 10s).
func runServer(srv *http.Server, svc *service.Services) error {
	serverErr := make(chan error, 1)
	go func() {
		err := srv.ListenAndServe()
		// http.ErrServerClosed is the expected return after Shutdown() — not an error.
		if err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
		close(serverErr)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		return err
	case sig := <-quit:
		log.Printf("received %s; beginning graceful shutdown", sig)
		gracefulShutdown(srv, svc)
		return nil
	}
}

// gracefulShutdown drains the HTTP server (stop accepting + wait for
// in-flight requests) and then the Execution service (cancel the service
// stop context + wait for in-flight mock goroutines). The same SHUTDOWN_GRACE
// budget is shared: HTTP gets the full budget, Execution gets whatever
// remains. If the budget is exhausted, each step returns its ctx.Err()
// and the process continues (no further work to do after that).
func gracefulShutdown(srv *http.Server, svc *service.Services) {
	grace := parseGraceTimeout()
	start := time.Now()
	log.Printf("grace timeout: %s (configurable via SHUTDOWN_GRACE)", grace)

	// 1) HTTP server: stop accepting new connections, drain in-flight.
	httpCtx, httpCancel := context.WithTimeout(context.Background(), grace)
	if err := srv.Shutdown(httpCtx); err != nil {
		log.Printf("HTTP server shutdown error: %v", err)
	}
	httpCancel()
	log.Printf("HTTP server stopped in %s", time.Since(start))

	// 2) Execution service: cancel stop + drain in-flight mock goroutines,
	//    bounded by whatever grace is left.
	remaining := grace - time.Since(start)
	if remaining < 0 {
		remaining = 0
	}
	execStart := time.Now()
	execCtx, execCancel := context.WithTimeout(context.Background(), remaining)
	if err := svc.Execution.Shutdown(execCtx); err != nil {
		log.Printf("Execution service shutdown error: %v", err)
	}
	execCancel()
	log.Printf("Execution service stopped in %s", time.Since(execStart))
}

// parseGraceTimeout reads the SHUTDOWN_GRACE env var (e.g. "10s", "500ms").
// Defaults to 10s. Invalid values fall back to 10s with a warning.
func parseGraceTimeout() time.Duration {
	const fallback = 10 * time.Second
	raw, ok := os.LookupEnv("SHUTDOWN_GRACE")
	if !ok {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		log.Printf("WARN: invalid SHUTDOWN_GRACE=%q; using %s", raw, fallback)
		return fallback
	}
	return d
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// buildAPIKeyStore constructs the in-memory APIKeyStore used by the auth
// service. The seed is read from the `API_KEYS_DEV` env var; format is a
// semicolon-separated list of `<raw_key>,<user_uuid>` entries, e.g.:
//
//	API_KEYS_DEV=ak_devkey_001,11111111-1111-1111-1111-111111111111;ak_devkey_002,22222222-...
//
// The raw key is hashed (sha256 of the post-`ak_` part) before being
// stored; the raw key is never persisted. The role defaults to "api".
//
// If the env var is unset, a single dev key is loaded with a clearly
// placeholder name; a warning is logged in both cases. Production
// deployments MUST set API_KEYS_DEV to real keys (or set it to empty to
// disable API key auth entirely — the bypass is closed in all cases
// because ValidateAPIKey returns ErrUnauthorized on a missing key).
func buildAPIKeyStore() store.APIKeyStore {
	raw := os.Getenv("API_KEYS_DEV")
	var seed []model.APIKey

	if raw == "" {
		// Hardcoded dev fallback. The hash is constant; do not use in
		// production. To disable API key auth in any environment, set
		// API_KEYS_DEV to "" (empty string).
		const devKey = "ak_dev_local_only_change_in_production_2026_06_12"
		const devUUID = "00000000-0000-0000-0000-000000000001"
		raw = devKey + "," + devUUID
		log.Printf("WARN: API_KEYS_DEV not set; using a single dev fallback key. DO NOT use in production.")
	}

	for _, entry := range strings.Split(raw, ";") {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, ",", 2)
		if len(parts) != 2 {
			log.Printf("WARN: ignoring malformed API_KEYS_DEV entry: %q", entry)
			continue
		}
		token := strings.TrimSpace(parts[0])
		uidStr := strings.TrimSpace(parts[1])
		if !strings.HasPrefix(token, "ak_") || len(token) <= len("ak_") {
			log.Printf("WARN: ignoring API_KEYS_DEV entry with bad key prefix: %q", entry)
			continue
		}
		uid, err := uuid.Parse(uidStr)
		if err != nil {
			log.Printf("WARN: ignoring API_KEYS_DEV entry with bad user uuid: %q (%v)", entry, err)
			continue
		}

		body := strings.TrimPrefix(token, "ak_")
		sum := sha256.Sum256([]byte(body))
		hash := hex.EncodeToString(sum[:])

		seed = append(seed, model.APIKey{
			KeyHash:   hash,
			UserID:    uid,
			Role:      "api",
			Name:      "dev-seed",
			CreatedAt: time.Now().UTC(),
		})
	}

	log.Printf("INFO: loaded %d API key dev seed entries", len(seed))
	return store.NewMemoryAPIKeyStore(seed)
}
