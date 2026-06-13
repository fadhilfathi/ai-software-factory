package store

// F-002 (Sprint 4 security review): API-key lookup seam.
//
// The auth service depends on this interface (not on a concrete store) so
// that the in-memory implementation used in this patch can be swapped for a
// Postgres-backed implementation in a follow-up sprint task without
// touching the auth service or the middleware. See api_key_store.go in the
// `memory` subpackage for the in-memory implementation.
//
// All implementations MUST key the lookup by the SHA-256 hex of the
// post-`ak_` part of the token. The raw token is never stored or indexed.

import (
	"context"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
)

// APIKeyStore is the persistence seam for `ak_*` API keys.
//
// The interface is intentionally narrow: just enough for the middleware's
// hot path (GetByHash) and the future admin revoke flow (Revoke). The
// Postgres follow-up will add a Create method; it is intentionally absent
// here because this patch is about closing the bypass, not building a key
// management UI.
type APIKeyStore interface {
	// GetByHash returns the APIKey whose KeyHash matches the argument
	// (case-insensitive on hex). Returns ErrNotFound when no such key
	// exists. The caller (auth.ValidateAPIKey) is responsible for the
	// revoked/expired checks; this method only does the lookup.
	GetByHash(ctx context.Context, hash string) (*model.APIKey, error)

	// Revoke marks the key as revoked. Idempotent: revoking an already-
	// revoked key returns nil. Returns ErrNotFound when no key matches.
	Revoke(ctx context.Context, hash string) error
}
