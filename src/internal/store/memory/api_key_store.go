package store

// F-002 (Sprint 4 security review): in-memory implementation of
// APIKeyStore. Production-grade persistence (Postgres-backed) is deferred to
// a follow-up sprint task — see the TODO at the bottom of this file.
//
// This file lives under `store/memory/` to follow the same layout as the
// `store/postgres/` subpackage. It is intentionally self-contained (its
// own mutex, its own map) so it can be wired into the auth service
// without taking a dependency on the main `memoryStore` struct or
// requiring a new accessor on the top-level Store interface.

import (
	"context"
	"sync"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
)

// memoryAPIKeyStore is the in-memory implementation of APIKeyStore.
//
// Concurrency: protected by mu (RWMutex). Reads use RLock; writes use Lock.
type memoryAPIKeyStore struct {
	mu   sync.RWMutex
	keys map[string]*model.APIKey
}

// NewMemoryAPIKeyStore builds an in-memory APIKeyStore pre-populated with
// the provided seed. The seed is copied (defensive — the caller may reuse
// the slice) and each entry's hash is normalised to lowercase so lookup is
// case-insensitive.
//
// Returns a value satisfying the APIKeyStore interface, not the concrete
// type, so the auth service can be wired without importing this file.
func NewMemoryAPIKeyStore(seed []model.APIKey) APIKeyStore {
	s := &memoryAPIKeyStore{
		keys: make(map[string]*model.APIKey, len(seed)),
	}
	for i := range seed {
		entry := seed[i] // take a copy so the caller's slice stays untouched
		entry.KeyHash = normaliseHash(entry.KeyHash)
		// Copy pointer-typed fields too so a later mutation of the
		// caller's entry does not bleed into the store.
		if entry.ExpiresAt != nil {
			t := *entry.ExpiresAt
			entry.ExpiresAt = &t
		}
		if entry.RevokedAt != nil {
			t := *entry.RevokedAt
			entry.RevokedAt = &t
		}
		if entry.LastUsedAt != nil {
			t := *entry.LastUsedAt
			entry.LastUsedAt = &t
		}
		s.keys[entry.KeyHash] = &entry
	}
	return s
}

// GetByHash returns the APIKey whose KeyHash matches. The hash is
// normalised to lowercase before lookup so callers can pass either case.
// Returns ErrNotFound when no such key exists.
func (s *memoryAPIKeyStore) GetByHash(_ context.Context, hash string) (*model.APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key, ok := s.keys[normaliseHash(hash)]
	if !ok {
		return nil, ErrNotFound
	}
	// Return a shallow copy so the caller cannot mutate the stored
	// entry. Pointer fields are intentionally shared — the auth service
	// only reads them.
	out := *key
	return &out, nil
}

// Revoke marks the key as revoked at time.Now(). Idempotent: revoking an
// already-revoked key leaves the existing RevokedAt unchanged and returns
// nil. Returns ErrNotFound when no key matches.
func (s *memoryAPIKeyStore) Revoke(_ context.Context, hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	key, ok := s.keys[normaliseHash(hash)]
	if !ok {
		return ErrNotFound
	}
	if key.RevokedAt != nil {
		return nil // idempotent
	}
	now := time.Now()
	key.RevokedAt = &now
	return nil
}

// normaliseHash lower-cases the hash so lookups are case-insensitive.
// SHA-256 hex output is canonically lowercase, but a caller passing
// uppercase should still find the entry.
func normaliseHash(h string) string {
	// Avoid pulling in strings just for this — ascii-only fast path.
	out := make([]byte, len(h))
	for i := 0; i < len(h); i++ {
		c := h[i]
		if c >= 'A' && c <= 'F' {
			c += 32
		}
		out[i] = c
	}
	return string(out)
}

// TODO: Postgres-backed APIKeyStore. Required for sprint follow-up.
//
//	Schema (024_create_api_keys.sql):
//	  id            UUID PRIMARY KEY DEFAULT gen_random_uuid()
//	  key_hash      VARCHAR(64) NOT NULL  -- sha256 hex, lowercase
//	  user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE
//	  role          VARCHAR(50) NOT NULL
//	  name          VARCHAR(255) NOT NULL DEFAULT ''
//	  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
//	  expires_at    TIMESTAMPTZ
//	  revoked_at    TIMESTAMPTZ
//	  last_used_at  TIMESTAMPTZ
//	  UNIQUE (key_hash)
//	  INDEX idx_api_keys_user_id (user_id)
//
//	Constraints worth surfacing:
//	  * Partial unique index on key_hash WHERE revoked_at IS NULL is
//	    not strictly needed (key_hash is globally unique), but a CHECK
//	    (key_hash = lower(key_hash)) guards against case-mismatched writes.
//	  * Foreign key on user_id needs ON DELETE CASCADE so revoking a
//	    user cleans up their keys.
//	  * Add Create method to the APIKeyStore interface; cmd/api/main.go
//	    will use it instead of the seed slice once the migration lands.
//
//	See docs/sprint4/infra-validation.md "Outstanding / deferred" for
//	the full task description.
