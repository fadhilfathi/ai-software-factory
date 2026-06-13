package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockAuthService is a mock implementation of AuthService.
type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) Login(req service.LoginRequest) (*service.LoginResult, *service.Error) {
	args := m.Called(req)
	return args.Get(0).(*service.LoginResult), args.Get(1).(*service.Error)
}

func (m *MockAuthService) Refresh(refreshToken string) (*service.LoginResult, *service.Error) {
	args := m.Called(refreshToken)
	return args.Get(0).(*service.LoginResult), args.Get(1).(*service.Error)
}

func (m *MockAuthService) Logout(refreshToken string) *service.Error {
	args := m.Called(refreshToken)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).(*service.Error)
}

func (m *MockAuthService) ValidateToken(tokenString string) (*service.Claims, error) {
	args := m.Called(tokenString)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.Claims), args.Error(1)
}

func (m *MockAuthService) ValidateRefreshToken(refreshToken string) (uuid.UUID, error) {
	args := m.Called(refreshToken)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

// ValidateAPIKey mirrors the production behaviour. The mock returns a
// (result, err) tuple where the first arg may be nil on error.
func (m *MockAuthService) ValidateAPIKey(_ context.Context, token string) (*service.ValidateAPIKeyResult, *service.Error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		if e := args.Get(1); e != nil {
			return nil, e.(*service.Error)
		}
		return nil, &service.Error{Code: "UNAUTHORIZED", Message: "Invalid API key"}
	}
	return args.Get(0).(*service.ValidateAPIKeyResult), nil
}

func TestAuthMiddleware(t *testing.T) {
	publicPaths := map[string]bool{
		"/health":    true,
		"/v1/auth/login": true,
		"/v1/auth/refresh": true,
	}

	t.Run("Valid JWT Token", func(t *testing.T) {
		mockSvc := new(MockAuthService)
		uid := uuid.New()
		claims := &service.Claims{UserID: uid.String(), Role: "admin"}
		mockSvc.On("ValidateToken", "valid-token").Return(claims, nil)

		r := gin.New()
		r.Use(Auth(mockSvc, publicPaths))
		r.GET("/protected", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Invalid Token", func(t *testing.T) {
		mockSvc := new(MockAuthService)
		mockSvc.On("ValidateToken", "invalid-token").Return(nil, errors.New("invalid token"))

		r := gin.New()
		r.Use(Auth(mockSvc, publicPaths))
		r.GET("/protected", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid or expired token")
		mockSvc.AssertExpectations(t)
	})

	t.Run("Public Path Skip", func(t *testing.T) {
		mockSvc := new(MockAuthService)
		r := gin.New()
		r.Use(Auth(mockSvc, publicPaths))
		r.GET("/public", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/public", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}

// TestAPIKeyMiddleware covers F-002 (Sprint 4 security review). The four
// subtests map directly to the brief: valid_key, unknown_key, empty token,
// non-ak_ token, tampered token. (The brief lists four; we cover five to
// make sure the empty-token edge case is locked down explicitly.)
func TestAPIKeyMiddleware(t *testing.T) {
	publicPaths := map[string]bool{"/health": true}

	t.Run("valid_key", func(t *testing.T) {
		const token = "ak_validsecret_001"
		uid := uuid.New()
		result := &service.ValidateAPIKeyResult{UserID: uid, Role: "api"}

		mockSvc := new(MockAuthService)
		mockSvc.On("ValidateAPIKey", token).Return(result, nil)

		r := gin.New()
		r.Use(Auth(mockSvc, publicPaths))
		r.GET("/protected", func(c *gin.Context) {
			userID, _ := c.Get(UserIDKey)
			role, _ := c.Get(RoleKey)
			c.JSON(http.StatusOK, gin.H{
				"user_id": userID,
				"role":    role,
			})
		})

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), uid.String())
		assert.Contains(t, w.Body.String(), `"role":"api"`)
		mockSvc.AssertExpectations(t)
	})

	t.Run("unknown_key", func(t *testing.T) {
		const token = "ak_unknownsecret_999"
		mockSvc := new(MockAuthService)
		mockSvc.On("ValidateAPIKey", token).Return(nil, &service.Error{
			Code:    "UNAUTHORIZED",
			Message: "Invalid API key",
		})

		r := gin.New()
		r.Use(Auth(mockSvc, publicPaths))
		r.GET("/protected", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "UNAUTHORIZED")
		mockSvc.AssertExpectations(t)
	})

	t.Run("empty_token", func(t *testing.T) {
		// No Authorization header at all — the middleware should
		// fall through to "missing token" handling. The mock is not
		// expected to be called because the `ak_` prefix check
		// happens before the prefix-stripped call.
		mockSvc := new(MockAuthService)
		r := gin.New()
		r.Use(Auth(mockSvc, publicPaths))
		r.GET("/protected", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		mockSvc.AssertNotCalled(t, "ValidateAPIKey")
	})

	t.Run("non_ak_token", func(t *testing.T) {
		// A bearer token that does not start with `ak_` must NOT
		// be routed to ValidateAPIKey. It should be handled by the
		// JWT path; with no JWT mock set up, it gets 401.
		mockSvc := new(MockAuthService)
		r := gin.New()
		r.Use(Auth(mockSvc, publicPaths))
		r.GET("/protected", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer some-jwt-thing")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		mockSvc.AssertNotCalled(t, "ValidateAPIKey")
	})

	t.Run("tampered_token", func(t *testing.T) {
		// A key that looks well-formed but is not in the store.
		// auth.ValidateAPIKey must be called, the mock returns
		// ErrUnauthorized, the middleware responds 401. The point:
		// the middleware does not pre-filter; it delegates the
		// trust decision to the auth service.
		const token = "ak_tamperedsecret_xxx"
		mockSvc := new(MockAuthService)
		mockSvc.On("ValidateAPIKey", token).Return(nil, &service.Error{
			Code:    "UNAUTHORIZED",
			Message: "Invalid API key",
		})

		r := gin.New()
		r.Use(Auth(mockSvc, publicPaths))
		r.GET("/protected", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer "+token)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		mockSvc.AssertExpectations(t)
	})
}
