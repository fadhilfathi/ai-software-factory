package middleware

import (
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

func (m *MockAuthService) ValidateToken(token string) (*service.Claims, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*service.Claims), args.Error(1)
}

func (m *MockAuthService) ValidateRefreshToken(refreshToken string) (uuid.UUID, error) {
	args := m.Called(refreshToken)
	return args.Get(0).(uuid.UUID), args.Error(1)
}

func TestAuthMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	publicPaths := func(c *gin.Context) bool {
		return c.Request.URL.Path == "/public"
	}

	t.Run("Valid Token", func(t *testing.T) {
		mockSvc := new(MockAuthService)
		claims := &service.Claims{
			UserID: "user-123",
			Role:   "admin",
		}
		mockSvc.On("ValidateToken", "valid-token").Return(claims, nil)

		r := gin.New()
		r.Use(Auth(mockSvc, publicPaths))
		r.GET("/protected", func(c *gin.Context) {
			userID, _ := c.Get(UserIDKey)
			role, _ := c.Get(RoleKey)
			c.JSON(http.StatusOK, gin.H{"user_id": userID, "role": role})
		})

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Bearer valid-token")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), `"user_id":"user-123"`)
		assert.Contains(t, w.Body.String(), `"role":"admin"`)
		mockSvc.AssertExpectations(t)
	})

	t.Run("Missing Authorization Header", func(t *testing.T) {
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
		assert.Contains(t, w.Body.String(), "Missing Authorization header")
	})

	t.Run("Invalid Authorization Scheme", func(t *testing.T) {
		mockSvc := new(MockAuthService)
		r := gin.New()
		r.Use(Auth(mockSvc, publicPaths))
		r.GET("/protected", func(c *gin.Context) {
			c.Status(http.StatusOK)
		})

		req := httptest.NewRequest("GET", "/protected", nil)
		req.Header.Set("Authorization", "Basic dXNlcjpwYXNz")
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Invalid authorization scheme")
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
