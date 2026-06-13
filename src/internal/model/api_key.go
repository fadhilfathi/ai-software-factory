package model

// F-002 (Sprint 4 security review): persistent identity for an `ak_*` API key.
//
// IMPORTANT — the `KeyHash` field stores the SHA-256 hex digest of the
// post-`ak_` part of the key ONLY (e.g. for token "ak_abcdef…", the hash is
// sha256("abcdef…") in lowercase hex). The raw key is never persisted or
// indexed. This is the only acceptable way to wire the `KeyHash` field —
// hashing the full "ak_…" string will break clients because every retry
// re-rolls the random suffix.
//
// RevokedAt and LastUsedAt are nullable: nil means "still active" / "never
// used". ExpiresAt nil means "never expires". All three are intentionally
// pointer-typed so the absence is unambiguous on the wire and in code.

import (
	"time"

	"github.com/google/uuid"
)

// APIKey represents a long-lived bearer credential that lets a non-human
// caller (CI job, internal service, script) authenticate against the API.
// Identity is established by the (KeyHash, UserID, Role) tuple; the raw
// `ak_…` string is known only to the issuing user.
type APIKey struct {
	// KeyHash is the lowercase hex SHA-256 of the bytes AFTER the `ak_`
	// prefix. Lookup is by this field; the raw key is never stored.
	KeyHash string

	// UserID is the human user the key acts on behalf of. The middleware
	// stamps this value into Gin context's UserIDKey.
	UserID uuid.UUID

	// Role is the role claim the key carries. Typically "api" or a
	// service-specific role; the middleware stamps this into RoleKey.
	// Use the model.Role* constants where applicable.
	Role string

	// Name is a human-friendly label for the key (e.g. "ci-deploy-prod").
	// Optional but recommended; helps the user revoke the right key later.
	Name string

	// CreatedAt is the issue timestamp (UTC).
	CreatedAt time.Time

	// ExpiresAt is the soft expiry. nil = never expires. A non-nil value
	// whose timestamp is in the past causes ValidateAPIKey to return
	// ErrUnauthorized.
	ExpiresAt *time.Time

	// RevokedAt, if non-nil, marks the key as explicitly revoked. A
	// revoked key must be rejected by ValidateAPIKey even if its hash
	// still matches. Stamping this field is the canonical revocation path
	// (use APIKeyStore.Revoke rather than deleting the row).
	RevokedAt *time.Time

	// LastUsedAt is the most recent successful ValidateAPIKey call. This
	// is best-effort and updated without error propagation; nil on a fresh
	// key. The middleware writes this opportunistically; failure to write
	// must not block the request.
	LastUsedAt *time.Time
}
