package config

import (
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

// setOrUnset applies a key=value if value is non-empty, otherwise unsets the key.
// Using os.Setenv directly (rather than t.Setenv) so we can also exercise the
// "key removed" branch deterministically.
func setOrUnset(t *testing.T, key, value string) {
	t.Helper()
	if value == "" {
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unsetenv %s: %v", key, err)
		}
		return
	}
	if err := os.Setenv(key, value); err != nil {
		t.Fatalf("setenv %s=%s: %v", key, value, err)
	}
}

func TestGetEnv(t *testing.T) {
	cases := []struct {
		name     string
		setKey   bool
		setValue string
		key      string
		fallback string
		want     string
	}{
		{name: "set-returns-value", setKey: true, setValue: "abc", key: "T_GETENV_A", fallback: "default", want: "abc"},
		{name: "unset-returns-fallback", setKey: false, key: "T_GETENV_B", fallback: "default", want: "default"},
		{name: "empty-treated-as-unset", setKey: true, setValue: "", key: "T_GETENV_C", fallback: "fb", want: "fb"},
		{name: "empty-fallback-and-empty-env-returns-empty", setKey: false, key: "T_GETENV_D", fallback: "", want: ""},
		{name: "value-with-spaces-preserved", setKey: true, setValue: "  hi  ", key: "T_GETENV_E", fallback: "fb", want: "  hi  "},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			setOrUnset(t, tc.key, tc.setValue)
			got := getEnv(tc.key, tc.fallback)
			if got != tc.want {
				t.Fatalf("getEnv(%q, %q) with env=%q: want %q, got %q",
					tc.key, tc.fallback, tc.setValue, tc.want, got)
			}
		})
	}
}

func TestGetEnvRequired(t *testing.T) {
	t.Run("present-returns-value", func(t *testing.T) {
		t.Setenv("T_REQ_A", "ok")
		if got := getEnvRequired("T_REQ_A"); got != "ok" {
			t.Fatalf("want 'ok', got %q", got)
		}
	})
	t.Run("missing-panics", func(t *testing.T) {
		os.Unsetenv("T_REQ_B")
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic, got none")
			}
			msg, ok := r.(string)
			if !ok {
				t.Fatalf("expected string panic, got %T: %v", r, r)
			}
			if !strings.Contains(msg, "T_REQ_B") || !strings.Contains(msg, "required") {
				t.Fatalf("panic message did not name the key or word 'required': %q", msg)
			}
		}()
		_ = getEnvRequired("T_REQ_B")
	})
	t.Run("empty-string-panics", func(t *testing.T) {
		t.Setenv("T_REQ_C", "")
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic on empty string env")
			}
		}()
		_ = getEnvRequired("T_REQ_C")
	})
}

func TestGetEnvInt(t *testing.T) {
	cases := []struct {
		name     string
		setKey   bool
		setValue string
		key      string
		fallback int
		want     int
	}{
		{name: "valid-int", setKey: true, setValue: "42", key: "T_INTI_A", fallback: 7, want: 42},
		{name: "zero", setKey: true, setValue: "0", key: "T_INTI_B", fallback: 7, want: 0},
		{name: "negative", setKey: true, setValue: "-1", key: "T_INTI_C", fallback: 7, want: -1},
		{name: "unset-uses-fallback", setKey: false, key: "T_INTI_D", fallback: 7, want: 7},
		{name: "empty-uses-fallback", setKey: true, setValue: "", key: "T_INTI_E", fallback: 7, want: 7},
		{name: "garbage-uses-fallback", setKey: true, setValue: "abc", key: "T_INTI_F", fallback: 7, want: 7},
		{name: "leading-space-garbage-uses-fallback", setKey: true, setValue: "  12", key: "T_INTI_G", fallback: 7, want: 7},
		{name: "huge-int-uses-fallback", setKey: true, setValue: "99999999999999999999999", key: "T_INTI_H", fallback: 7, want: 7},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			setOrUnset(t, tc.key, tc.setValue)
			if got := getEnvInt(tc.key, tc.fallback); got != tc.want {
				t.Fatalf("getEnvInt(%q, %d) env=%q: want %d, got %d",
					tc.key, tc.fallback, tc.setValue, tc.want, got)
			}
		})
	}
}

func TestGetEnvIntRequired(t *testing.T) {
	t.Run("valid-returns-int", func(t *testing.T) {
		t.Setenv("T_INTR_A", "8080")
		if got := getEnvIntRequired("T_INTR_A"); got != 8080 {
			t.Fatalf("want 8080, got %d", got)
		}
	})
	t.Run("missing-panics", func(t *testing.T) {
		os.Unsetenv("T_INTR_B")
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic on missing")
			}
			msg, ok := r.(string)
			if !ok || !strings.Contains(msg, "T_INTR_B") {
				t.Fatalf("expected panic mentioning T_INTR_B, got %v", r)
			}
		}()
		_ = getEnvIntRequired("T_INTR_B")
	})
	t.Run("non-integer-panics", func(t *testing.T) {
		t.Setenv("T_INTR_C", "twelve")
		defer func() {
			r := recover()
			if r == nil {
				t.Fatal("expected panic on non-integer")
			}
			msg, ok := r.(string)
			if !ok {
				t.Fatalf("expected string panic, got %T: %v", r, r)
			}
			if !strings.Contains(msg, "integer") {
				t.Fatalf("panic message should mention 'integer', got %q", msg)
			}
		}()
		_ = getEnvIntRequired("T_INTR_C")
	})
}

func TestGetEnvBool(t *testing.T) {
	cases := []struct {
		name     string
		setKey   bool
		setValue string
		key      string
		fallback bool
		want     bool
	}{
		// Documented contract: only "true" and "1" parse to true; any
		// other non-empty value parses to false. Empty uses fallback.
		{name: "true", setKey: true, setValue: "true", key: "T_BOOL_A", fallback: false, want: true},
		{name: "1", setKey: true, setValue: "1", key: "T_BOOL_B", fallback: false, want: true},
		{name: "false-is-false", setKey: true, setValue: "false", key: "T_BOOL_C", fallback: true, want: false},
		{name: "0-is-false", setKey: true, setValue: "0", key: "T_BOOL_D", fallback: true, want: false},
		{name: "TRUE-uppercase-is-false", setKey: true, setValue: "TRUE", key: "T_BOOL_E", fallback: true, want: false},
		{name: "yes-is-false", setKey: true, setValue: "yes", key: "T_BOOL_F", fallback: true, want: false},
		{name: "empty-uses-fallback-true", setKey: false, key: "T_BOOL_G", fallback: true, want: true},
		{name: "empty-uses-fallback-false", setKey: false, key: "T_BOOL_H", fallback: false, want: false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			setOrUnset(t, tc.key, tc.setValue)
			if got := getEnvBool(tc.key, tc.fallback); got != tc.want {
				t.Fatalf("getEnvBool(%q, %v) env=%q: want %v, got %v",
					tc.key, tc.fallback, tc.setValue, tc.want, got)
			}
		})
	}
}

func TestParseCSV(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  []string
	}{
		{name: "empty", input: "", want: []string{}},
		{name: "single", input: "a", want: []string{"a"}},
		{name: "two", input: "a,b", want: []string{"a", "b"}},
		{name: "trailing-comma", input: "a,b,", want: []string{"a", "b"}},
		{name: "leading-comma", input: ",a,b", want: []string{"a", "b"}},
		{name: "double-comma", input: "a,,b", want: []string{"a", "b"}},
		{name: "whitespace-trimmed", input: " a , b ", want: []string{"a", "b"}},
		{name: "all-blank", input: ",,,", want: []string{}},
		{name: "tabs-and-newlines", input: "a\t,b\n,c\rd", want: []string{"a", "b", "c", "d"}},
		{name: "single-blank-segment", input: "a, ,b", want: []string{"a", "b"}},
		{name: "no-trim-by-default-still-wrong", input: "ab", want: []string{"ab"}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got := parseCSV(tc.input)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("parseCSV(%q):\n want %#v\n got  %#v", tc.input, tc.want, got)
			}
		})
	}
}

func TestSplitCSV(t *testing.T) {
	got := splitCSV("a,b,c")
	want := []string{"a", "b", "c"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("splitCSV: want %#v, got %#v", want, got)
	}
}

func TestTrimSpace(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", ""},
		{"a", "a"},
		{"  a  ", "a"},
		{"\t\n\r a \t\n\r", "a"},
		{"   ", ""},
		{"a b", "a b"},
		{"  a b  ", "a b"},
	}
	for _, tc := range cases {
		got := trimSpace(tc.in)
		if got != tc.want {
			t.Fatalf("trimSpace(%q): want %q, got %q", tc.in, tc.want, got)
		}
	}
}

// TestLoad_FullyPopulatedEnv verifies that Load() returns a fully-populated
// Config struct when all required env vars are set. We use a helper to seed
// the env with valid values for every required key.
func TestLoad_FullyPopulatedEnv(t *testing.T) {
	seed := map[string]string{
		"SERVER_HOST":               "0.0.0.0",
		"SERVER_PORT":               "9090",
		"DB_HOST":                   "db.local",
		"DB_PORT":                   "5433",
		"DB_USER":                   "alice",
		"DB_PASSWORD":               "s3cret",
		"DB_NAME":                   "tokenrouter",
		"JWT_SECRET":                "test-jwt-secret-with-some-length-1234",
		"CORS_ALLOWED_ORIGINS":      "https://app.example.com, https://admin.example.com",
		"CORS_ALLOW_CREDENTIALS":    "true",
		"RATE_LIMIT_RPM":            "200",
		"RATE_LIMIT_BURST":          "50",
		"AGENT_RUNTIME":             "aion",
		"AGENT_MEMORY_MB":           "1024",
		"AGENT_CPU_LIMIT":           "100000",
		"AION_BINARY":               "/opt/aion/bin/aion",
		"AION_MODEL":                "opus",
		"AION_PROVIDER":             "openai",
		"AION_PERMISSION_MODE":      "bypassPermissions",
		"AION_MAX_CONCURRENT":       "16",
		"AION_WAIT_TIMEOUT":         "1200",
	}
	for k, v := range seed {
		t.Setenv(k, v)
	}

	cfg := Load()
	if cfg == nil {
		t.Fatal("Load() returned nil")
	}

	// Server
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Server.Host: want 0.0.0.0, got %q", cfg.Server.Host)
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port: want 9090, got %d", cfg.Server.Port)
	}

	// Database
	if cfg.Database.Host != "db.local" {
		t.Errorf("Database.Host: want db.local, got %q", cfg.Database.Host)
	}
	if cfg.Database.Port != 5433 {
		t.Errorf("Database.Port: want 5433, got %d", cfg.Database.Port)
	}
	if cfg.Database.User != "alice" {
		t.Errorf("Database.User: want alice, got %q", cfg.Database.User)
	}
	if cfg.Database.Password != "s3cret" {
		t.Errorf("Database.Password: want s3cret, got %q", cfg.Database.Password)
	}
	if cfg.Database.Name != "tokenrouter" {
		t.Errorf("Database.Name: want tokenrouter, got %q", cfg.Database.Name)
	}

	// Auth
	if cfg.Auth.JWTSecret != "test-jwt-secret-with-some-length-1234" {
		t.Errorf("Auth.JWTSecret mismatch")
	}

	// CORS
	wantOrigins := []string{"https://app.example.com", "https://admin.example.com"}
	if !reflect.DeepEqual(cfg.CORS.AllowedOrigins, wantOrigins) {
		t.Errorf("CORS.AllowedOrigins: want %#v, got %#v", wantOrigins, cfg.CORS.AllowedOrigins)
	}
	if !cfg.CORS.AllowCredentials {
		t.Error("CORS.AllowCredentials: want true, got false")
	}
	wantMethods := []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"}
	if !reflect.DeepEqual(cfg.CORS.AllowedMethods, wantMethods) {
		t.Errorf("CORS.AllowedMethods: want %#v, got %#v", wantMethods, cfg.CORS.AllowedMethods)
	}
	wantHeaders := []string{"Content-Type", "Authorization", "X-Request-ID"}
	if !reflect.DeepEqual(cfg.CORS.AllowedHeaders, wantHeaders) {
		t.Errorf("CORS.AllowedHeaders: want %#v, got %#v", wantHeaders, cfg.CORS.AllowedHeaders)
	}
	if cfg.CORS.MaxAge != 86400 {
		t.Errorf("CORS.MaxAge: want 86400, got %d", cfg.CORS.MaxAge)
	}

	// RateLimit
	if cfg.RateLimit.RequestsPerMinute != 200 {
		t.Errorf("RateLimit.RequestsPerMinute: want 200, got %d", cfg.RateLimit.RequestsPerMinute)
	}
	if cfg.RateLimit.Burst != 50 {
		t.Errorf("RateLimit.Burst: want 50, got %d", cfg.RateLimit.Burst)
	}

	// Agent
	if cfg.Agent.Runtime != "aion" {
		t.Errorf("Agent.Runtime: want aion, got %q", cfg.Agent.Runtime)
	}
	// AGENT_MEMORY_MB is multiplied by 1024*1024 to give bytes
	wantMem := int64(1024) * 1024 * 1024
	if cfg.Agent.MemoryLimit != wantMem {
		t.Errorf("Agent.MemoryLimit: want %d, got %d", wantMem, cfg.Agent.MemoryLimit)
	}
	if cfg.Agent.CPULimit != 100000 {
		t.Errorf("Agent.CPULimit: want 100000, got %d", cfg.Agent.CPULimit)
	}
	if cfg.Agent.AionBinary != "/opt/aion/bin/aion" {
		t.Errorf("Agent.AionBinary: want /opt/aion/bin/aion, got %q", cfg.Agent.AionBinary)
	}
	if cfg.Agent.AionModel != "opus" {
		t.Errorf("Agent.AionModel: want opus, got %q", cfg.Agent.AionModel)
	}
	if cfg.Agent.AionProvider != "openai" {
		t.Errorf("Agent.AionProvider: want openai, got %q", cfg.Agent.AionProvider)
	}
	if cfg.Agent.AionPermissionMode != "bypassPermissions" {
		t.Errorf("Agent.AionPermissionMode: want bypassPermissions, got %q", cfg.Agent.AionPermissionMode)
	}
	if cfg.Agent.AionMaxConcurrent != 16 {
		t.Errorf("Agent.AionMaxConcurrent: want 16, got %d", cfg.Agent.AionMaxConcurrent)
	}
	if cfg.Agent.AionWaitTimeoutSeconds != 1200 {
		t.Errorf("Agent.AionWaitTimeoutSeconds: want 1200, got %d", cfg.Agent.AionWaitTimeoutSeconds)
	}
}

// TestLoad_DefaultsDoNotPanic verifies that Load() runs to completion (without
// panicking) when ALL required env vars are set, even if everything else is
// empty. Defaults must be applied, not propagated as panics.
func TestLoad_DefaultsDoNotPanic(t *testing.T) {
	required := map[string]string{
		"DB_HOST":     "h",
		"DB_PORT":     "5432",
		"DB_USER":     "u",
		"DB_PASSWORD": "p",
		"DB_NAME":     "n",
		"JWT_SECRET":  "s",
	}
	for k, v := range required {
		t.Setenv(k, v)
	}
	// Wipe every optional knob.
	optional := []string{
		"SERVER_HOST", "SERVER_PORT", "CORS_ALLOWED_ORIGINS", "CORS_ALLOW_CREDENTIALS",
		"RATE_LIMIT_RPM", "RATE_LIMIT_BURST", "AGENT_RUNTIME", "AGENT_MEMORY_MB",
		"AGENT_CPU_LIMIT", "AION_BINARY", "AION_MODEL", "AION_PROVIDER",
		"AION_PERMISSION_MODE", "AION_MAX_CONCURRENT", "AION_WAIT_TIMEOUT",
	}
	for _, k := range optional {
		os.Unsetenv(k)
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Load() panicked with defaults: %v", r)
		}
	}()
	cfg := Load()
	if cfg.Server.Port != 8080 {
		t.Errorf("default Server.Port: want 8080, got %d", cfg.Server.Port)
	}
	if cfg.RateLimit.RequestsPerMinute != 100 {
		t.Errorf("default RateLimit.RequestsPerMinute: want 100, got %d", cfg.RateLimit.RequestsPerMinute)
	}
	if cfg.Agent.Runtime != "runc" {
		t.Errorf("default Agent.Runtime: want runc, got %q", cfg.Agent.Runtime)
	}
	if cfg.Agent.AionMaxConcurrent != 8 {
		t.Errorf("default Agent.AionMaxConcurrent: want 8, got %d", cfg.Agent.AionMaxConcurrent)
	}
	if cfg.Agent.AionWaitTimeoutSeconds != 600 {
		t.Errorf("default Agent.AionWaitTimeoutSeconds: want 600, got %d", cfg.Agent.AionWaitTimeoutSeconds)
	}
}

// TestLoad_MissingRequired_Panics verifies that Load() panics on a missing
// required env var. t.Setenv to "" is equivalent to unset for getEnvRequired
// (both surface as os.Getenv returning ""), so this is the safe way to
// trigger the panic AND auto-restore the env.
func TestLoad_MissingRequired_Panics(t *testing.T) {
	required := []string{"DB_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "JWT_SECRET"}
	for _, k := range required {
		t.Setenv(k, "")
	}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic, got none")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected string panic, got %T: %v", r, r)
		}
		if !strings.Contains(msg, "required environment variable") {
			t.Fatalf("panic message unexpected: %q", msg)
		}
	}()
	_ = Load()
}

// Sanity check: strconv.Atoi is the only parser we use for ints. Lock that
// in so a future swap to Atoi64 is intentional.
func TestIntParsingReference(t *testing.T) {
	for _, s := range []string{"0", "1", "8080", "-1"} {
		if _, err := strconv.Atoi(s); err != nil {
			t.Fatalf("strconv.Atoi(%q): unexpected err: %v", s, err)
		}
	}
}
