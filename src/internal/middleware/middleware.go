package middleware

import (
	"context"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/gin-gonic/gin"
)

const (
	// UserIDKey is the context key for the authenticated user's ID.
	UserIDKey = "user_id"
	// RoleKey is the context key for the authenticated user's role.
	RoleKey = "role"
	// RequestIDKey is the context key for the unique request identifier.
	RequestIDKey = "request_id"
)

// isPublicPath is a function that returns true when a request should skip auth.
type isPublicPath func(c *gin.Context) bool

// Auth provides JWT and API Key authentication.
// Pass publicRoutes to exempt specific routes (e.g. login, register, healthz).
func Auth(authService *service.AuthService, publicPaths isPublicPath) gin.HandlerFunc {
	return func(c *gin.Context) {
		if publicPaths(c) {
			c.Next()
			return
		}

		auth := c.GetHeader("Authorization")
		if auth == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "Missing Authorization header"}})
			return
		}

		var userID string
		var role string
		if strings.HasPrefix(auth, "Bearer ") {
			token := strings.TrimPrefix(auth, "Bearer ")
			// API key pattern: ak_...
			if strings.HasPrefix(token, "ak_") {
				userID = "api_user"
				role = "api_user"
			} else {
				// Validate JWT using AuthService
				claims, err := authService.ValidateToken(token)
				if err != nil {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "Invalid or expired token"}})
					return
				}
				userID = claims.UserID
				role = claims.Role
			}
		} else {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "Invalid authorization scheme"}})
			return
		}

		c.Set(UserIDKey, userID)
		c.Set(RoleKey, role)
		c.Next()
	}
}

// RequestID attaches a unique ID to every request and sets the X-Request-ID header.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader("X-Request-ID")
		if id == "" {
			id = "req_" + strings.ReplaceAll(time.Now().Format("150405.000000"), ".", "")
		}
		c.Header("X-Request-ID", id)
		c.Set(RequestIDKey, id)
		c.Next()
	}
}

// Logger logs each HTTP request.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.Printf("%s %s %s %v", c.Request.Method, c.Request.URL.Path, c.ClientIP(), time.Since(start))
	}
}

// Recovery catches panics and returns 500.
func Recovery() gin.HandlerFunc {
	return gin.RecoveryWithWriter(gin.DefaultErrorWriter, func(c *gin.Context, err any) {
		log.Printf("panic recovered: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "Internal server error"}})
	})
}

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           int
}

// DefaultCORSConfig returns a secure default CORS configuration
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins:   []string{}, // Empty = no origins allowed by default (must configure)
		AllowedMethods:   []string{"GET", "POST", "PATCH", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "X-Request-ID"},
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	}
}

// CORS sets CORS headers based on configuration.
func CORS(cfg CORSConfig) gin.HandlerFunc {
	// Validate config: wildcard not allowed with credentials
	if cfg.AllowCredentials {
		for _, o := range cfg.AllowedOrigins {
			if o == "*" {
				panic("CORS: wildcard origin (*) not allowed when AllowCredentials is true")
			}
		}
	}

	// Pre-compute header values
	allowMethods := strings.Join(cfg.AllowedMethods, ", ")
	allowHeaders := strings.Join(cfg.AllowedHeaders, ", ")
	maxAge := strconv.Itoa(cfg.MaxAge)

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// Check if origin is allowed
		allowed := false
		for _, o := range cfg.AllowedOrigins {
			if o == origin {
				allowed = true
				break
			}
		}

		if allowed {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin") // Important for caching
			if cfg.AllowCredentials {
				c.Header("Access-Control-Allow-Credentials", "true")
			}
		}

		c.Header("Access-Control-Allow-Methods", allowMethods)
		c.Header("Access-Control-Allow-Headers", allowHeaders)
		c.Header("Access-Control-Max-Age", maxAge)

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	RequestsPerMinute int
	Burst             int
	KeyFunc           func(*gin.Context) string
}

// DefaultRateLimitConfig returns a sensible default rate limit configuration
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerMinute: 100,
		Burst:             20,
		KeyFunc: func(c *gin.Context) string {
			ip := c.ClientIP()
			return "ip:" + ip
		},
	}
}

// RateLimit implements a token bucket rate limiter
func RateLimit(cfg RateLimitConfig) gin.HandlerFunc {
	type bucket struct {
		tokens     float64
		lastRefill time.Time
	}

	var (
		mu      sync.Mutex
		buckets = make(map[string]*bucket)
	)

	refillRate := float64(cfg.RequestsPerMinute) / 60.0
	maxTokens := float64(cfg.RequestsPerMinute + cfg.Burst)

	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			mu.Lock()
			now := time.Now()
			for key, b := range buckets {
				if now.Sub(b.lastRefill) > 30*time.Minute {
					delete(buckets, key)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		key := cfg.KeyFunc(c)

		mu.Lock()
		b, exists := buckets[key]
		now := time.Now()

		if !exists {
			b = &bucket{tokens: maxTokens, lastRefill: now}
			buckets[key] = b
		}

		elapsed := now.Sub(b.lastRefill).Seconds()
		b.tokens = min(maxTokens, b.tokens+elapsed*refillRate)
		b.lastRefill = now

		if b.tokens < 1 {
			mu.Unlock()
			c.Header("X-RateLimit-Limit", strconv.Itoa(cfg.RequestsPerMinute))
			c.Header("X-RateLimit-Remaining", "0")
			c.Header("X-RateLimit-Reset", strconv.FormatInt(now.Add(time.Minute).Unix(), 10))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": gin.H{"code": "RATE_LIMITED", "message": "Rate limit exceeded"}})
			return
		}

		b.tokens--
		remaining := int(b.tokens)
		mu.Unlock()

		c.Header("X-RateLimit-Limit", strconv.Itoa(cfg.RequestsPerMinute))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(now.Add(time.Minute).Unix(), 10))

		c.Next()
	}
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
