package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Server    ServerConfig
	Database  DatabaseConfig
	Auth      AuthConfig
	CORS      CORSConfig
	RateLimit RateLimitConfig
	Agent     AgentConfig
}

type ServerConfig struct {
	Host string
	Port int
}

type DatabaseConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	Name     string
}

type AuthConfig struct {
	JWTSecret string
	// CookieSecure controls the `Secure` flag on the refresh-token
	// cookie set by handler.AuthHandler (Login / Refresh / Logout).
	// Defaults: true when APP_ENV (or ENV) is "production" or "prod",
	// false otherwise. Override with AUTH_COOKIE_SECURE=true|false.
	// Surfaced by D-002 sign-off finding F-D002-003: a hard-coded
	// `secure=true` broke local HTTP dev. The env var makes the
	// production/dev split explicit at deploy time.
	CookieSecure bool
}

type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

type RateLimitConfig struct {
	RequestsPerMinute int
	Burst             int
}

type AgentConfig struct {
	// Runtime selects between legacy runc sandboxing and the new
	// aion runtime. Values: "runc" (default) or "aion". TASK-501
	// wires "aion" through NewExecutionService; "runc" continues
	// to flow through sandbox.go and orchestrator.go unchanged.
	Runtime     string
	MemoryLimit int64
	CPULimit    int64

	// AionBinary is the absolute path or PATH-name for the aion
	// CLI used by aion.ProcessRuntime. Empty string means the
	// runtime will look up the binary on $PATH via exec.LookPath.
	// Default: "aion". AION_BINARY overrides.
	AionBinary string

	// AionModel is the model identifier passed to the aion
	// worker (e.g. "sonnet", "opus", "haiku"). Default: "sonnet".
	// AION_MODEL overrides.
	AionModel string

	// AionProvider is the upstream provider identifier passed to
	// the aion worker (e.g. "anthropic", "openai"). Default:
	// "anthropic". AION_PROVIDER overrides.
	AionProvider string

	// AionPermissionMode is the permission mode flag passed to
	// the aion worker (e.g. "default", "acceptEdits",
	// "bypassPermissions"). Default: "default".
	// AION_PERMISSION_MODE overrides.
	AionPermissionMode string

	// AionMaxConcurrent caps the number of in-flight aion workers
	// the process will accept. The execution service rejects new
	// CreateExecution calls with a 503-equivalent sentinel when
	// the limit is reached, rather than queueing. Default: 8.
	// AION_MAX_CONCURRENT overrides.
	AionMaxConcurrent int

	// AionWaitTimeoutSeconds is the maximum time a single aion
	// worker will run before the runtime kills the subprocess and
	// the execution service records a timeout failure. Default:
	// 600 (10 minutes). AION_WAIT_TIMEOUT overrides.
	AionWaitTimeoutSeconds int
}

func Load() *Config {
	cfg := &Config{
		Server: ServerConfig{
			Host: getEnv("SERVER_HOST", "localhost"),
			Port: getEnvInt("SERVER_PORT", 8080),
		},
		Database: DatabaseConfig{
			Host:     getEnvRequired("DB_HOST"),
			Port:     getEnvIntRequired("DB_PORT"),
			User:     getEnvRequired("DB_USER"),
			Password: getEnvRequired("DB_PASSWORD"),
			Name:     getEnvRequired("DB_NAME"),
		},
		Auth: AuthConfig{
			JWTSecret:    getEnvRequired("JWT_SECRET"),
			CookieSecure: getEnvBool("AUTH_COOKIE_SECURE", isProductionEnv()),
		},
		CORS: CORSConfig{
			AllowedOrigins:   parseCSV(getEnv("CORS_ALLOWED_ORIGINS", "")),
			AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Request-ID"},
			AllowCredentials: getEnvBool("CORS_ALLOW_CREDENTIALS", false),
			MaxAge:           86400,
		},
		RateLimit: RateLimitConfig{
			RequestsPerMinute: getEnvInt("RATE_LIMIT_RPM", 100),
			Burst:             getEnvInt("RATE_LIMIT_BURST", 20),
		},
		Agent: AgentConfig{
			// Legacy sandbox/orchestrator knobs (unchanged).
			Runtime:     getEnv("AGENT_RUNTIME", "runc"),
			MemoryLimit: int64(getEnvInt("AGENT_MEMORY_MB", 512)) * 1024 * 1024,
			CPULimit:    int64(getEnvInt("AGENT_CPU_LIMIT", 50000)),
			// TASK-501 Aion runtime knobs.
			AionBinary:            getEnv("AION_BINARY", "aion"),
			AionModel:             getEnv("AION_MODEL", "sonnet"),
			AionProvider:          getEnv("AION_PROVIDER", "anthropic"),
			AionPermissionMode:    getEnv("AION_PERMISSION_MODE", "default"),
			AionMaxConcurrent:     getEnvInt("AION_MAX_CONCURRENT", 8),
			AionWaitTimeoutSeconds: getEnvInt("AION_WAIT_TIMEOUT", 600),
		},
	}
	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvRequired(key string) string {
	v := os.Getenv(key)
	if v == "" {
		panic("required environment variable " + key + " is not set")
	}
	return v
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvIntRequired(key string) int {
	v := os.Getenv(key)
	if v == "" {
		panic("required environment variable " + key + " is not set")
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		panic("environment variable " + key + " must be an integer: " + err.Error())
	}
	return i
}

func getEnvBool(key string, fallback bool) bool {
	if v := os.Getenv(key); v != "" {
		return v == "true" || v == "1"
	}
	return fallback
}

// isProductionEnv reports whether the running process should be treated as
// a production deployment for the purposes of secure-cookie defaulting and
// similar dev/prod splits. Checks APP_ENV first, then ENV. Recognised
// production values: "production", "prod" (case-insensitive). Empty /
// unrecognised values return false, i.e. dev-friendly defaults.
func isProductionEnv() bool {
	for _, k := range []string{"APP_ENV", "ENV"} {
		if v := strings.ToLower(strings.TrimSpace(os.Getenv(k))); v != "" {
			return v == "production" || v == "prod"
		}
	}
	return false
}

func parseCSV(s string) []string {
	if s == "" {
		return []string{}
	}
	var result []string
	for _, part := range splitCSV(s) {
		if trimmed := trimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func splitCSV(s string) []string {
	var result []string
	var current string
	for _, r := range s {
		if r == ',' {
			result = append(result, current)
			current = ""
		} else {
			current += string(r)
		}
	}
	result = append(result, current)
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '	' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '	' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}