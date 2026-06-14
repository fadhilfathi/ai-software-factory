package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// fakeAuthService is a stub for service.AuthService. Only the methods
// exercised by the AuthHandler tests (Login, Refresh, Logout) are
// implemented; the rest panic on call (which is fine — they shouldn't
// be reached by these tests).
type fakeAuthService struct {
	loginResult   *service.LoginResult
	loginErr      *service.Error
	refreshResult *service.LoginResult
	refreshErr    *service.Error
	logoutErr     *service.Error
}

func (f *fakeAuthService) Login(req service.LoginRequest) (*service.LoginResult, *service.Error) {
	return f.loginResult, f.loginErr
}

func (f *fakeAuthService) Refresh(refreshToken string) (*service.LoginResult, *service.Error) {
	return f.refreshResult, f.refreshErr
}

func (f *fakeAuthService) Logout(refreshToken string) *service.Error {
	return f.logoutErr
}

func (f *fakeAuthService) ValidateToken(tokenString string) (*service.Claims, error) {
	panic("ValidateToken not expected in cookie-secure test")
}

func (f *fakeAuthService) ValidateRefreshToken(refreshToken string) (uuid.UUID, error) {
	panic("ValidateRefreshToken not expected in cookie-secure test")
}

func (f *fakeAuthService) ValidateAPIKey(ctx context.Context, token string) (*service.ValidateAPIKeyResult, *service.Error) {
	panic("ValidateAPIKey not expected in cookie-secure test")
}

// TestAuthHandler_SecureCookieFlag parametrizes F-D002-003: verifies that
// the refresh-token cookie's Secure flag is driven by the cookieSecure
// constructor argument. D-002 sign-off condition: a hard-coded
// `secure=true` broke local HTTP dev, so the flag must be env-var driven.
//
// D-002 sign-off (aafad88) requires this to land before v1 GA.
func TestAuthHandler_SecureCookieFlag(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name         string
		cookieSecure bool
		wantSecure   bool
	}{
		{name: "prod: secure=true", cookieSecure: true, wantSecure: true},
		{name: "dev:  secure=false", cookieSecure: false, wantSecure: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &fakeAuthService{
				loginResult: &service.LoginResult{
					AccessToken:  "access-1",
					RefreshToken: "refresh-1",
					ExpiresIn:    3600,
				},
				refreshResult: &service.LoginResult{
					AccessToken:  "access-2",
					RefreshToken: "refresh-2",
					ExpiresIn:    3600,
				},
			}
			h := NewAuthHandler(svc, tc.cookieSecure)

			r := gin.New()
			r.POST("/v1/auth/login", h.Login)
			r.POST("/v1/auth/refresh", h.Refresh)
			r.POST("/v1/auth/logout", h.Logout)

			// --- Login: set cookie (body carries email/password) ---
			w := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodPost, "/v1/auth/login",
				strings.NewReader(`{"email":"a@b","password":"x"}`))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("login: got status %d, want 200; body=%s", w.Code, w.Body.String())
			}
			got := extractSecureFlag(t, w.Header().Get("Set-Cookie"), "refresh_token")
			if got != tc.wantSecure {
				t.Errorf("login: Set-Cookie secure=%v, want %v", got, tc.wantSecure)
			}

			// --- Refresh: set cookie (request carries refresh_token cookie) ---
			w = httptest.NewRecorder()
			req = httptest.NewRequest(http.MethodPost, "/v1/auth/refresh", nil)
			req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "refresh-1"})
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("refresh: got status %d, want 200; body=%s", w.Code, w.Body.String())
			}
			got = extractSecureFlag(t, w.Header().Get("Set-Cookie"), "refresh_token")
			if got != tc.wantSecure {
				t.Errorf("refresh: Set-Cookie secure=%v, want %v", got, tc.wantSecure)
			}

			// --- Logout: clear cookie (request carries refresh_token cookie) ---
			w = httptest.NewRecorder()
			req = httptest.NewRequest(http.MethodPost, "/v1/auth/logout", nil)
			req.AddCookie(&http.Cookie{Name: "refresh_token", Value: "refresh-1"})
			r.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("logout: got status %d, want 200; body=%s", w.Code, w.Body.String())
			}
			got = extractSecureFlag(t, w.Header().Get("Set-Cookie"), "refresh_token")
			if got != tc.wantSecure {
				t.Errorf("logout: Set-Cookie secure=%v, want %v", got, tc.wantSecure)
			}
		})
	}
}

// extractSecureFlag parses a Set-Cookie header and returns the value
// of the `Secure` directive for the named cookie. A cookie without the
// Secure directive is considered not-secure (Secure=false).
func extractSecureFlag(t *testing.T, header, name string) bool {
	t.Helper()
	if header == "" {
		t.Fatalf("Set-Cookie header missing for %q", name)
	}
	// Set-Cookie may include multiple cookies separated by ", <name>=" patterns.
	parts := strings.Split(header, ", ")
	for _, p := range parts {
		if !strings.HasPrefix(p, name+"=") {
			continue
		}
		// The cookie spec uses `; ` to separate attribute pairs.
		attrs := strings.Split(p, "; ")
		for _, a := range attrs {
			if strings.EqualFold(strings.TrimSpace(a), "Secure") {
				return true
			}
		}
		return false
	}
	t.Fatalf("no %q cookie found in Set-Cookie: %q", name, header)
	return false
}
