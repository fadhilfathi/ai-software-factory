package service

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/fadhilfathi/AI-Software-Factory/internal/validation"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// AuthService handles authentication and token generation.
type authService struct {
	store     store.Store
	log       *zap.Logger
	jwtSecret []byte
}

// AuthService defines authentication and token generation operations.
type AuthService interface {
	Login(req LoginRequest) (*LoginResult, *Error)
	Refresh(refreshToken string) (*LoginResult, *Error)
	Logout(refreshToken string) *Error
	ValidateToken(tokenString string) (*Claims, error)
	ValidateRefreshToken(refreshToken string) (uuid.UUID, error)
}

func NewAuthService(s store.Store, log *zap.Logger, jwtSecret string) AuthService {
	if len(jwtSecret) < 32 {
		log.Fatal("JWT secret must be at least 32 characters")
	}
	return &authService{store: s, log: log, jwtSecret: []byte(jwtSecret)}
}

// LoginRequest carries credentials from the login endpoint.
type LoginRequest struct {
	Email    string
	Password string
}

// LoginResult carries the response data from a successful login.
type LoginResult struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// Login authenticates a user by email and password.
func (s *authService) Login(req LoginRequest) (*LoginResult, *Error) {
	var errs validation.Errors
	validation.NotEmpty(req.Email, "email", "Email", &errs)
	validation.NotEmpty(req.Password, "password", "Password", &errs)
	validation.Email(req.Email, "email", &errs)
	if errs.HasErrors() {
		return nil, validationError(errs)
	}

	user, err := s.store.Users().GetByEmail(req.Email)
	if err != nil {
		s.log.Warn("login attempt for unknown email", zap.String("email_hash", hashForLog(req.Email)))
		return nil, unauthorized("Invalid email or password")
	}

	// Verify password with bcrypt (constant-time comparison)
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		s.log.Warn("login attempt with wrong password", zap.String("email_hash", hashForLog(req.Email)))
		return nil, unauthorized("Invalid email or password")
	}

	// Generate JWT access token
	accessToken, err := s.generateJWT(user.ID, user.Email, 15*time.Minute)
	if err != nil {
		s.log.Error("failed to generate access token", zap.Error(err))
		return nil, internalError("Failed to generate token")
	}

	// Generate refresh token (longer expiry, stored for validation)
	refreshToken, err := s.generateRefreshToken(user.ID)
	if err != nil {
		s.log.Error("failed to generate refresh token", zap.Error(err))
		return nil, internalError("Failed to generate token")
	}

	// Store refresh token hash for revocation support (TTL: 7 days)
	tokenKey := "auth:refresh_token:" + hashToken(refreshToken)
	if err := s.store.Tokens().Set(tokenKey, user.ID, 7*24*3600); err != nil {
		s.log.Error("failed to store refresh token hash", zap.Error(err))
		// Continue even if storage fails; persistence is best-effort for now
	}

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    900, // 15 minutes in seconds
	}, nil
}

// Refresh authenticates a user by refresh token and returns new tokens.
func (s *authService) Refresh(refreshToken string) (*LoginResult, *Error) {
	userID, err := s.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, unauthorized("Invalid or expired refresh token")
	}

	user, dbErr := s.store.Users().GetByID(userID)
	if dbErr != nil {
		return nil, unauthorized("User not found")
	}

	// Generate JWT access token
	accessToken, err := s.generateJWT(user.ID, user.Email, 15*time.Minute)
	if err != nil {
		s.log.Error("failed to generate access token", zap.Error(err))
		return nil, internalError("Failed to generate token")
	}

	// Generate new refresh token
	newRefreshToken, err := s.generateRefreshToken(user.ID)
	if err != nil {
		s.log.Error("failed to generate refresh token", zap.Error(err))
		return nil, internalError("Failed to generate token")
	}

	// Store new refresh token hash
	tokenKey := "auth:refresh_token:" + hashToken(newRefreshToken)
	if err := s.store.Tokens().Set(tokenKey, user.ID, 7*24*3600); err != nil {
		s.log.Error("failed to store new refresh token hash", zap.Error(err))
	}

	// Revoke old refresh token hash
	oldTokenKey := "auth:refresh_token:" + hashToken(refreshToken)
	_ = s.store.Tokens().Delete(oldTokenKey)

	return &LoginResult{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    900,
	}, nil
}

// Logout revokes a refresh token.
func (s *authService) Logout(refreshToken string) *Error {
	tokenKey := "auth:refresh_token:" + hashToken(refreshToken)
	if err := s.store.Tokens().Delete(tokenKey); err != nil {
		s.log.Error("failed to revoke refresh token", zap.Error(err))
		return internalError("Failed to logout")
	}
	return nil
}

// hashForLog returns a truncated hash of email for logging (prevents PII in logs)
func hashForLog(email string) string {
	h := sha256.Sum256([]byte(email))
	return hex.EncodeToString(h[:8])
}

// Claims represents the JWT claims.
type Claims struct {
	UserID string `json:"uid"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// ValidateToken validates a bearer token and returns the claims.
func (s *authService) ValidateToken(tokenString string) (*Claims, error) {
	// Parse and validate JWT
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Ensure signing method is HS256
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.getJWTSecret(), nil
	})

	if err != nil {
		return nil, unauthorized("Invalid token")
	}

	if !token.Valid {
		return nil, unauthorized("Invalid token")
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return nil, unauthorized("Invalid token user ID format")
	}

	// Verify user exists and is active
	_, err = s.store.Users().GetByID(userID)
	if err != nil {
		s.log.Warn("token validation failed: user not found", zap.String("user_id", userID.String()))
		return nil, unauthorized("User not found or inactive")
	}

	return claims, nil
}

// generateJWT creates a signed JWT access token
func (s *authService) generateJWT(userID uuid.UUID, email string, expiry time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID.String(),
		Role:   "user", // Defaulting to user; update if role exists in DB
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "ai-software-factory",
			Audience:  jwt.ClaimStrings{"ai-software-factory-api"},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.getJWTSecret())
}

// generateRefreshToken creates a JWT refresh token
func (s *authService) generateRefreshToken(userID uuid.UUID) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID: userID.String(),
		Role:   "user", // Defaulting to user; update if role exists in DB
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(7 * 24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(now),
			Issuer:    "ai-software-factory",
			Audience:  jwt.ClaimStrings{"ai-software-factory-api-refresh"},
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.getJWTSecret())
}

// ValidateRefreshToken validates a refresh token and returns the user ID
func (s *authService) ValidateRefreshToken(refreshToken string) (uuid.UUID, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(refreshToken, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return s.getJWTSecret(), nil
	})

	if err != nil || !token.Valid {
		return uuid.Nil, unauthorized("Invalid refresh token")
	}

	// Verify audience for refresh token
	validAud := false
	for _, a := range claims.Audience {
		if a == "ai-software-factory-api-refresh" {
			validAud = true
			break
		}
	}
	if !validAud {
		return uuid.Nil, unauthorized("Invalid token audience")
	}

	userID, err := uuid.Parse(claims.UserID)
	if err != nil {
		return uuid.Nil, unauthorized("Invalid token user ID format")
	}

	// Check if token exists in revocation store
	tokenKey := "auth:refresh_token:" + hashToken(refreshToken)
	if _, err := s.store.Tokens().Get(tokenKey); err != nil {
		return uuid.Nil, unauthorized("Token revoked or expired")
	}

	return userID, nil
}

// getJWTSecret returns the JWT signing secret
func (s *authService) getJWTSecret() []byte {
	return s.jwtSecret
}

func hashToken(input string) string {
	h := sha256.Sum256([]byte(input))
	return hex.EncodeToString(h[:16])
}
