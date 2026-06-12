package service

// F-001 (Sprint 4 security review): verify the JWT mints read the role from
// the user record instead of hard-coding Role: "user".
//
// The fix threads `string(user.Role)` into generateJWT and generateRefreshToken.
// These tests assert the role actually flows through to the JWT claims.

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// testJWTSecret is a stable 32+ character secret used by auth tests.
const testJWTSecret = "unit-test-secret-not-used-in-prod-32-chars"

// newTestAuthService builds a bare authService with only the fields that
// generateJWT / generateRefreshToken depend on. No store or log is exercised.
func newTestAuthService() *authService {
	return &authService{
		log:       zap.NewNop(),
		jwtSecret: []byte(testJWTSecret),
	}
}

// TestGenerateJWT_PreservesRole is the direct regression test for F-001.
// It mints a JWT for each known role and asserts the claim round-trips.
func TestGenerateJWT_PreservesRole(t *testing.T) {
	s := newTestAuthService()
	userID := uuid.New()

	tests := []struct {
		name string
		role string
	}{
		{"admin role is preserved", "admin"},
		{"member role is preserved", "member"},
		{"viewer role is preserved", "viewer"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tokenStr, err := s.generateJWT(userID, tc.role, 15*time.Minute)
			if err != nil {
				t.Fatalf("generateJWT returned error: %v", err)
			}
			if tokenStr == "" {
				t.Fatalf("generateJWT returned empty token")
			}

			claims, err := parseClaims(tokenStr)
			if err != nil {
				t.Fatalf("failed to parse generated token: %v", err)
			}

			if claims.Role != tc.role {
				t.Errorf("expected claims.Role=%q, got %q", tc.role, claims.Role)
			}
			if claims.UserID != userID.String() {
				t.Errorf("expected claims.UserID=%q, got %q", userID.String(), claims.UserID)
			}
		})
	}
}

// TestGenerateRefreshToken_PreservesRole is the matching assertion for
// the refresh-token mint path.
func TestGenerateRefreshToken_PreservesRole(t *testing.T) {
	s := newTestAuthService()
	userID := uuid.New()

	tokenStr, err := s.generateRefreshToken(userID, "admin")
	if err != nil {
		t.Fatalf("generateRefreshToken returned error: %v", err)
	}

	claims, err := parseClaims(tokenStr)
	if err != nil {
		t.Fatalf("failed to parse generated refresh token: %v", err)
	}

	if claims.Role != "admin" {
		t.Errorf("expected refresh claims.Role=%q, got %q", "admin", claims.Role)
	}
}

// TestGenerateJWT_NotHardCodedAsUser is the explicit anti-regression check:
// when the caller passes "admin", the claim must NOT be the legacy "user"
// default. This guards against a future refactor re-introducing F-001.
func TestGenerateJWT_NotHardCodedAsUser(t *testing.T) {
	s := newTestAuthService()
	userID := uuid.New()

	tokenStr, err := s.generateJWT(userID, "admin", 15*time.Minute)
	if err != nil {
		t.Fatalf("generateJWT returned error: %v", err)
	}
	claims, err := parseClaims(tokenStr)
	if err != nil {
		t.Fatalf("failed to parse token: %v", err)
	}
	if claims.Role == "user" {
		t.Errorf("F-001 regression: admin role was overwritten with the legacy hard-coded %q value", "user")
	}
}

// parseClaims is a small helper that verifies a token's signature with the
// test secret and returns the Claims.
func parseClaims(tokenStr string) (*Claims, error) {
	parsed, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		// Enforce HMAC signing method (defensive; matches the producer).
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(testJWTSecret), nil
	})
	if err != nil {
		return nil, err
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
}
