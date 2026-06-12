// Package integration provides DB-agnostic test fixtures for
// Sprint 4 cross-route integration tests. It exposes a thin
// Store alias and two constructors (NewMemoryStore,
// NewPostgresStore) so test code does not need to import the
// store package directly. Sprint 4 wires the integration tests
// against the in-memory store; Sprint 5 may swap in a
// Postgres-backed implementation without touching the test
// logic.
//
// This file is compiled into the main binary. The overhead is
// a few hundred bytes; the goal is to keep the test bootstrap
// future-proof without rebuilding the whole test package.
package integration

import (
	"errors"

	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
)

// Store is a thin alias over store.Store so test helpers can be
// implemented against either the in-memory or Postgres-backed
// implementation without changing the test logic. Aliasing
// (vs redefining the interface) keeps the two implementations
// structurally compatible — if the underlying store.Store
// evolves, the test fixtures pick it up automatically.
type Store = store.Store

// NewMemoryStore returns the default in-memory store. This is
// what Sprint 4 integration tests use. The in-memory store
// self-seeds the six canonical capabilities (architecture,
// coding, testing, security, devops, leadership) so
// capability-based tests do not need additional setup.
func NewMemoryStore() Store {
	return store.NewMemoryStore()
}

// NewPostgresStore returns a Postgres-backed store. Sprint 4
// stubs this as "not implemented" so test scaffolding is
// future-proofed without breaking Sprint 4 runs. Sprint 5 can
// fill in the real implementation (likely calling
// store/postgres.New(url) and applying the Sprint 4 migrations).
func NewPostgresStore(url string) (Store, error) {
	return nil, errors.New("Postgres-backed store not implemented for Sprint 4; see Sprint 5 plan")
}
