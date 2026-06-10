package service

import (
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/example/project/internal/store"
	"github.com/example/project/internal/validation"
	"go.uber.org/zap"
)

// AuthService handles authentication and token generation.
type AuthService struct {
	store store.Store
	log   *zap.Logger
}

func NewAuthService(s store.Store, log *zap.Logger) *AuthService {
	return &AuthService{store: s, log: log}
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
func (s *AuthService) Login(req LoginRequest) (*LoginResult, *Error) {
	var errs validation.Errors
	validation.NotEmpty(req.Email, "email", "Email", &errs)
	validation.NotEmpty(req.Password, "password", "Password", &errs)
	validation.Email(req.Email, "email", &errs)
	if errs.HasErrors() {
		return nil, validationError(errs)
	}

	user, err := s.store.Users().GetByEmail(req.Email)
	if err != nil {
		s.log.Warn("login attempt for unknown email", zap.String("email", req.Email))
		return nil, unauthorized("Invalid email or password")
	}

	// For now, plain-text password comparison (TODO: bcrypt)
	if user.Password != req.Password {
		s.log.Warn("login attempt with wrong password", zap.String("email", req.Email))
		return nil, unauthorized("Invalid email or password")
	}

	// Generate a mock token
	token := hashToken(user.ID + ":" + req.Email + ":" + time.Now().String())

	return &LoginResult{
		AccessToken:  token,
		RefreshToken: hashToken(token + ":refresh"),
		ExpiresIn:    86400,
	}, nil
}

// ValidateToken validates a bearer token and returns the user ID.
// This is a placeholder — real JWT validation would go here.
func (s *AuthService) ValidateToken(token string) (string, error) {
	// Accept any non-empty token with the right prefix
	if len(token) < 10 {
		return "", unauthorized("Invalid token")
	}
	return "user_from_jwt", nil
}

func hashToken(input string) string {
	h := sha256.Sum256([]byte(input))
	return hex.EncodeToString(h[:16])
}
