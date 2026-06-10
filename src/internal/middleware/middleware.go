package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"
)

type contextKey string

const (
	// UserIDKey is the context key for the authenticated user's ID.
	UserIDKey contextKey = "user_id"
	// RequestIDKey is the context key for the unique request identifier.
	RequestIDKey contextKey = "request_id"
)

// isPublicPath is a function that returns true when a request should skip auth.
type isPublicPath func(r *http.Request) bool

// Auth provides JWT and API Key authentication.
// Pass publicRoutes to exempt specific routes (e.g. login, register, healthz).
// For now it extracts the Authorization header and sets a placeholder user_id.
// TODO: integrate real JWT validation.
func Auth(publicPaths isPublicPath) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if publicPaths(r) {
				next.ServeHTTP(w, r)
				return
			}

			auth := r.Header.Get("Authorization")
			if auth == "" {
				http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"Missing Authorization header"}}`, http.StatusUnauthorized)
				return
			}

			var userID string
			if strings.HasPrefix(auth, "Bearer ") {
				token := strings.TrimPrefix(auth, "Bearer ")
				// Accept API key pattern: ak_...
				if strings.HasPrefix(token, "ak_") {
					userID = "api_user"
				} else {
					// JWT placeholder — extract sub claim.
					userID = "user_from_jwt"
				}
			} else {
				http.Error(w, `{"error":{"code":"UNAUTHORIZED","message":"Invalid authorization scheme"}}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), UserIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequestID attaches a unique ID to every request and sets the X-Request-ID header.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = "req_" + strings.ReplaceAll(time.Now().Format("150405.000000"), ".", "")
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), RequestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// Logger logs each HTTP request.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s %v", r.Method, r.URL.Path, r.RemoteAddr, time.Since(start))
	})
}

// Recovery catches panics and returns 500.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic recovered: %v", err)
				http.Error(w, `{"error":{"code":"INTERNAL_ERROR","message":"Internal server error"}}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// CORS sets permissive CORS headers for development.
func CORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
