package handler

import (
	"net/http"

	"github.com/fadhilfathi/AI-Software-Factory/internal/service"
	"github.com/gin-gonic/gin"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	svc          service.AuthService
	cookieSecure bool
}

// NewAuthHandler builds an AuthHandler. cookieSecure is the value to use
// for the `Secure` flag on the refresh-token cookie. It should come from
// cfg.Auth.CookieSecure (see src/internal/config). Surfaced by D-002
// sign-off finding F-D002-003 so local HTTP dev (secure=false) and prod
// (secure=true) can both work without code changes.
func NewAuthHandler(svc service.AuthService, cookieSecure bool) *AuthHandler {
	return &AuthHandler{svc: svc, cookieSecure: cookieSecure}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login handles POST /auth/login.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, "INVALID_JSON", "Malformed request body")
		return
	}

	result, svcErr := h.svc.Login(service.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	// Set Refresh Token in an HttpOnly, Secure, SameSite=Strict cookie
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("refresh_token", result.RefreshToken, 7*24*3600, "/", "", h.cookieSecure, true)

	// Return Access Token in JSON
	writeJSON(c, http.StatusOK, gin.H{
		"access_token": result.AccessToken,
		"expires_in":   result.ExpiresIn,
	})
}

// Refresh handles POST /auth/refresh.
func (h *AuthHandler) Refresh(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil || refreshToken == "" {
		writeError(c, http.StatusUnauthorized, "UNAUTHORIZED", "Missing refresh token")
		return
	}

	result, svcErr := h.svc.Refresh(refreshToken)
	if svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	// Set new Refresh Token in an HttpOnly, Secure, SameSite=Strict cookie
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie("refresh_token", result.RefreshToken, 7*24*3600, "/", "", h.cookieSecure, true)

	// Return Access Token in JSON
	writeJSON(c, http.StatusOK, gin.H{
		"access_token": result.AccessToken,
		"expires_in":   result.ExpiresIn,
	})
}

// Logout handles POST /auth/logout.
func (h *AuthHandler) Logout(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil || refreshToken == "" {
		writeJSON(c, http.StatusOK, gin.H{"message": "Already logged out"})
		return
	}

	if svcErr := h.svc.Logout(refreshToken); svcErr != nil {
		writeServiceError(c, svcErr)
		return
	}

	// Clear the refresh token cookie
	c.SetCookie("refresh_token", "", -1, "/", "", h.cookieSecure, true)

	writeJSON(c, http.StatusOK, gin.H{"message": "Logged out successfully"})
}
