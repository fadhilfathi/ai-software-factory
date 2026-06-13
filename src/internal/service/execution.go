package service

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ExecutionService is the Sprint 4 (TASK-405) implementation. It
// owns the execution lifecycle:
//
//   * CreateExecution     — validate task/agent, insert row,
//                            start the mock background goroutine
//   * GetExecution        — read-by-id
//   * ListExecutions      — keyset-paginated list
//   * UpdateExecutionStatus — state-transition-guarded PATCH
//
// The mock goroutine simulates a real agent run: it sleeps 2-3s
// (configurable), then transitions the row to 'completed' or
// 'failed' depending on a configurable failure rate. The default
// failure rate is read from the EXECUTION_MOCK_FAILURE_RATE env
// var (default 0.0 = never fail). Tests override the rate and the
// sleep function via ExecutionServiceConfig.
//
// Shutdown model: we use a service-level context (stop) that is
// cancelled by Shutdown(). Each in-flight mock goroutine selects
// on time.After(...) and stop.Done(). The WaitGroup is used by
// Shutdown() to wait for in-flight goroutines to drain (with the
// caller's ctx as a timeout). This is documented in the
// Sprint 4 design note.
type ExecutionService struct {
	store store.Store
	log   *zap.Logger
	cfg   *ExecutionServiceConfig

	// stop is the service-level context. It is cancelled by
	// Shutdown(). All in-flight mock goroutines derive their
	// lifecycle from this context.
	stop     context.Context
	stopOnce sync.Once
	cancel   context.CancelFunc

	// wg tracks in-flight mock goroutines so Shutdown() can
	// drain them gracefully. We pick WaitGroup over per-call
	// ctx propagation because the goroutines are short-lived
	// and don't take a caller-supplied ctx; WaitGroup is the
	// simplest primitive for "wait for all in-flight to
	// finish".
	wg sync.WaitGroup

	// randMu guards the package-level math/rand source. We use
	// math/rand (not crypto/rand) because the failure-rate
	// decision is not security-sensitive; math/rand is faster
	// and the test can seed it deterministically.
	randMu sync.Mutex
	rand   *rand.Rand
}

// ExecutionServiceConfig is the injectable configuration for
// ExecutionService. Production code passes nil to NewExecutionService
// and gets sensible defaults. Tests pass a custom config to make the
// mock goroutine deterministic and fast.
//
// MockSleep returns the duration the mock goroutine should sleep
// before transitioning the row. The default is 2-3s. Tests should
// pass a function that returns a short, deterministic duration.
//
// MockFailureRate is a probability in [0.0, 1.0]. The default is
// read from EXECUTION_MOCK_FAILURE_RATE (env var) and falls back
// to 0.0 (never fail) when unset. Tests can override directly.
type ExecutionServiceConfig struct {
	MockSleep       func() time.Duration
	MockFailureRate float64
}

// DefaultExecutionServiceConfig returns the production default
// config. The failure rate is read from EXECUTION_MOCK_FAILURE_RATE
// (default 0.0). The sleep function returns 2-3s with a uniform
// random draw.
func DefaultExecutionServiceConfig() *ExecutionServiceConfig {
	return &ExecutionServiceConfig{
		MockSleep:       defaultMockSleep,
		MockFailureRate: envFloat("EXECUTION_MOCK_FAILURE_RATE", 0.0),
	}
}

func defaultMockSleep() time.Duration {
	const lo, hi = 2 * time.Second, 3 * time.Second
	return lo + time.Duration(rand.Int63n(int64(hi-lo)))
}

func envFloat(name string, fallback float64) float64 {
	raw, ok := os.LookupEnv(name)
	if !ok {
		return fallback
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return fallback
	}
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// NewExecutionService constructs an ExecutionService. Passing a
// nil cfg uses DefaultExecutionServiceConfig(). The returned
// service owns a stop context; callers must call Shutdown() to
// release it.
func NewExecutionService(s store.Store, log *zap.Logger, cfg *ExecutionServiceConfig) *ExecutionService {
	if cfg == nil {
		cfg = DefaultExecutionServiceConfig()
	}
	if cfg.MockSleep == nil {
		cfg.MockSleep = defaultMockSleep
	}
	ctx, cancel := context.WithCancel(context.Background())
	// Seed math/rand once for the default mock sleep. Tests
	// that need determinism should pass a custom MockSleep.
	return &ExecutionService{
		store:  s,
		log:    log,
		cfg:    cfg,
		stop:   ctx,
		cancel: cancel,
		rand:   rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// Shutdown cancels the service-level stop context (causing any
// in-flight mock goroutines to exit on their next select tick)
// and waits for them to drain. The caller's ctx bounds the wait:
// if the ctx is cancelled before drain completes, Shutdown
// returns ctx.Err() and leaves any still-running goroutines to
// finish on their own.
func (s *ExecutionService) Shutdown(ctx context.Context) error {
	s.stopOnce.Do(func() { s.cancel() })
	done := make(chan struct{})
	go func() { s.wg.Wait(); close(done) }()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// ErrExecutionNotFound is the typed sentinel for a missing row.
// The handler maps this to 404 EXECUTION_NOT_FOUND.
var ErrExecutionNotFound = errors.New("execution not found")

// ErrInvalidStateTransition is the typed sentinel for a PATCH that
// tries to move the row through an edge the state machine
// disallows. The handler maps this to 409 INVALID_STATE_TRANSITION.
var ErrInvalidStateTransition = errors.New("invalid execution state transition")

// ErrTaskNotFound is returned by CreateExecution when the task_id
// does not resolve to an existing task. The handler maps this to
// 404 TASK_NOT_FOUND (matching the existing task handler error).
var ErrTaskNotFound = errors.New("task not found")

// ErrAgentNotFound is returned by CreateExecution when the agent_id
// does not resolve to an existing agent. The handler maps this to
// 404 AGENT_NOT_FOUND.
var ErrAgentNotFound = errors.New("agent not found")

// ErrCrossTenantBlocked is the typed sentinel for a cross-tenant
// access attempt. The handler maps this to 404 CROSS_TENANT_BLOCKED
// (404 rather than 403 to avoid leaking the existence of resources
// in other projects). Returned when the caller's project (resolved
// from the X-Project-ID header) does not match the resource's
// project. TASK-422 (F-016 cross-tenant execution).
var ErrCrossTenantBlocked = errors.New("cross-tenant access blocked")

// validExecutionTransitions encodes the state machine. Terminal
// states (completed/failed) have no outgoing edges. Same-status
// transitions are not modelled as a separate edge — see
// isValidExecutionTransition for the no-op handling.
var validExecutionTransitions = map[model.ExecutionStatus]map[model.ExecutionStatus]struct{}{
	model.ExecutionStatusPending: {
		model.ExecutionStatusRunning:   {},
		model.ExecutionStatusCompleted: {},
		model.ExecutionStatusFailed:    {},
	},
	model.ExecutionStatusRunning: {
		model.ExecutionStatusCompleted: {},
		model.ExecutionStatusFailed:    {},
	},
	model.ExecutionStatusCompleted: {}, // terminal
	model.ExecutionStatusFailed:    {}, // terminal
}

// isValidExecutionTransition returns true if from → to is a legal
// edge. from == to is treated as a legal no-op (idempotent PATCH
// that didn't change anything).
func isValidExecutionTransition(from, to model.ExecutionStatus) bool {
	if from == to {
		return true
	}
	allowed, ok := validExecutionTransitions[from]
	if !ok {
		return false
	}
	_, ok = allowed[to]
	return ok
}

// CreateExecution validates the task and agent exist (404 on
// miss), inserts a new 'pending' row, and starts a background
// mock goroutine that will transition the row to 'completed' or
// 'failed' after a configurable sleep. The returned *Execution
// is the just-created row (status=pending); the caller can poll
// GetExecution to observe the goroutine's transition.
//
// TASK-422 (F-016): callerProjectID is the project the caller is
// authenticated for (resolved from the X-Project-ID header by the
// handler). It must match BOTH the task's project and the agent's
// project; otherwise we return ErrCrossTenantBlocked. We also
// assert task.ProjectID == agent.ProjectID as a defensive
// triple-check — it should never be violated because assignments
// are project-scoped upstream, but if it is, we fail closed.
func (s *ExecutionService) CreateExecution(ctx context.Context, taskID, agentID, callerProjectID uuid.UUID) (*model.Execution, error) {
	// Validate task exists. We do this BEFORE creating the row
	// so we never leave an orphan execution behind for a task
	// that doesn't exist. The store.ErrNotFound path is
	// returned to the caller; the handler maps it to 404.
	task, err := s.store.Tasks().GetByID(taskID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, ErrTaskNotFound
		}
		return nil, fmt.Errorf("lookup task: %w", err)
	}

	// Cross-tenant check: task must be in the caller's project.
	if task.ProjectID != callerProjectID {
		return nil, fmt.Errorf("create execution: task %w", ErrCrossTenantBlocked)
	}

	// Validate agent exists.
	agent, err := s.store.Agents().GetByID(ctx, agentID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, ErrAgentNotFound
		}
		return nil, fmt.Errorf("lookup agent: %w", err)
	}

	// Cross-tenant check: agent must be in the caller's project.
	if agent.ProjectID != callerProjectID {
		return nil, fmt.Errorf("create execution: agent %w", ErrCrossTenantBlocked)
	}

	// Defensive triple-check: task and agent must be in the same
	// project. Assignments are project-scoped upstream so this
	// should never fail in practice, but if it does we fail closed.
	if task.ProjectID != agent.ProjectID {
		return nil, fmt.Errorf("create execution: task and agent project mismatch %w", ErrCrossTenantBlocked)
	}

	now := time.Now().UTC()
	exec := &model.Execution{
		ExecutionID: uuid.New(),
		TaskID:      taskID,
		AgentID:     agentID,
		Status:      model.ExecutionStatusPending,
		StartedAt:   now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.store.Executions().Create(ctx, exec); err != nil {
		return nil, fmt.Errorf("create execution: %w", err)
	}

	// Start the mock goroutine. We use the service-level
	// stop context (not the caller's request ctx) so a
	// cancelled HTTP request doesn't abort the mock
	// simulation — the simulation is a server-side job,
	// not a request-scoped operation.
	s.wg.Add(1)
	go s.mockExecution(exec.ExecutionID)

	return exec, nil
}

// mockExecution is the background goroutine body. It sleeps for
// cfg.MockSleep() (or until s.stop is done), then transitions
// the row to 'completed' or 'failed' depending on cfg.MockFailureRate.
// It logs (does not propagate) any store error from the
// transition write — the caller is long gone and the row is
// observable via GetExecution.
func (s *ExecutionService) mockExecution(executionID uuid.UUID) {
	defer s.wg.Done()

	sleepDur := s.cfg.MockSleep()

	timer := time.NewTimer(sleepDur)
	defer timer.Stop()
	select {
	case <-timer.C:
		// normal sleep completion
	case <-s.stop.Done():
		// shutdown: exit before transitioning. The row
		// stays in 'pending' until the operator PATCHes
		// it (or until the next process restart re-runs
		// the mock).
		return
	}

	// Decide success vs failure.
	s.randMu.Lock()
	roll := s.rand.Float64()
	s.randMu.Unlock()

	if roll < s.cfg.MockFailureRate {
		errMsg := fmt.Sprintf("mock failure (rate=%.2f, roll=%.4f)", s.cfg.MockFailureRate, roll)
		s.transitionFromMock(executionID, model.ExecutionStatusFailed, &errMsg)
		return
	}
	s.transitionFromMock(executionID, model.ExecutionStatusCompleted, nil)
}

// transitionFromMock is the mock goroutine's UpdateStatus path.
// It uses a fresh background context with a short timeout so the
// request ctx cancellation (if any) doesn't abort the write,
// and so shutdown doesn't immediately abort the write either —
// the WaitGroup drain in Shutdown() still waits for these to
// finish, but the write itself should succeed before then.
func (s *ExecutionService) transitionFromMock(executionID uuid.UUID, status model.ExecutionStatus, errMsg *string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := s.store.Executions().UpdateStatus(ctx, executionID, status, errMsg); err != nil {
		s.log.Warn("mock execution update failed",
			zap.String("execution_id", executionID.String()),
			zap.String("target_status", string(status)),
			zap.Error(err),
		)
	}
}

// GetExecution reads a single execution by id. Returns
// ErrExecutionNotFound (mapped to 404 by the handler) on miss.
//
// TASK-422 (F-016): model.Execution has no ProjectID — the
// project is implicit via the parent task. We look up the parent
// task and compare its ProjectID to callerProjectID. On mismatch
// we return ErrCrossTenantBlocked (mapped to 404, not 403, to
// avoid leaking existence).
func (s *ExecutionService) GetExecution(ctx context.Context, id, callerProjectID uuid.UUID) (*model.Execution, error) {
	e, err := s.store.Executions().GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, ErrExecutionNotFound
		}
		return nil, fmt.Errorf("get execution: %w", err)
	}
	task, terr := s.store.Tasks().GetByID(e.TaskID)
	if terr != nil {
		// Parent task disappeared — treat as cross-tenant blocked
		// rather than 500; the row is unreachable to the caller.
		return nil, fmt.Errorf("get execution: %w", ErrCrossTenantBlocked)
	}
	if task.ProjectID != callerProjectID {
		return nil, fmt.Errorf("get execution: %w", ErrCrossTenantBlocked)
	}
	return e, nil
}

// ListExecutions returns a keyset-paginated page of executions
// matching the filter. The store handles cursor normalisation
// (default page size 50, max 200).
//
// TASK-422 (F-016): when a task_id or agent_id filter is set,
// we verify that the referenced task/agent belongs to the caller's
// project. Per-filter AND semantics: a caller asking for
// task_id=T AND agent_id=A must pass the check for BOTH T and A.
// An empty filter (no task/agent) returns rows visible to the
// caller across the project — the store is responsible for the
// actual project-scoping; here we only check filter arguments.
func (s *ExecutionService) ListExecutions(ctx context.Context, filter model.ExecutionFilter, callerProjectID uuid.UUID) (*model.ExecutionListResult, error) {
	if filter.TaskID != uuid.Nil {
		task, terr := s.store.Tasks().GetByID(filter.TaskID)
		if terr != nil {
			return nil, fmt.Errorf("list executions: task filter %w", ErrCrossTenantBlocked)
		}
		if task.ProjectID != callerProjectID {
			return nil, fmt.Errorf("list executions: task filter %w", ErrCrossTenantBlocked)
		}
	}
	if filter.AgentID != uuid.Nil {
		agent, aerr := s.store.Agents().GetByID(ctx, filter.AgentID)
		if aerr != nil {
			return nil, fmt.Errorf("list executions: agent filter %w", ErrCrossTenantBlocked)
		}
		if agent.ProjectID != callerProjectID {
			return nil, fmt.Errorf("list executions: agent filter %w", ErrCrossTenantBlocked)
		}
	}
	return s.store.Executions().List(ctx, filter)
}

// UpdateExecutionStatus is the PATCH path. It validates the
// state transition (returns ErrInvalidStateTransition on a
// disallowed edge), then calls the store's UpdateStatus which
// sets completed_at and updated_at as appropriate. A same-status
// PATCH is treated as a no-op (returns the current row without
// writing).
//
// TASK-422 (F-016): callerProjectID is propagated to the
// internal GetExecution call, which performs the parent-task
// project check. We DO NOT short-circuit on the cross-tenant
// error before the state-transition check because a same-status
// PATCH is a legal no-op and we want it to return 200 with the
// current row (this matches the previous behaviour for the
// same-project case). A cross-tenant caller gets 404 instead.
func (s *ExecutionService) UpdateExecutionStatus(ctx context.Context, id uuid.UUID, newStatus model.ExecutionStatus, errorMessage *string, callerProjectID uuid.UUID) (*model.Execution, error) {
	current, err := s.GetExecution(ctx, id, callerProjectID)
	if err != nil {
		return nil, err
	}
	if !isValidExecutionTransition(current.Status, newStatus) {
		return nil, fmt.Errorf("%w: %s → %s", ErrInvalidStateTransition, current.Status, newStatus)
	}
	if current.Status == newStatus {
		// Idempotent no-op. Skip the write.
		return current, nil
	}
	updated, err := s.store.Executions().UpdateStatus(ctx, id, newStatus, errorMessage)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, ErrExecutionNotFound
		}
		return nil, fmt.Errorf("update execution status: %w", err)
	}
	return updated, nil
}

// IsValidExecutionStatus is a thin wrapper exposing the model-level
// validator. The handler uses it to 400 on a bad status query param.
func (s *ExecutionService) IsValidExecutionStatus(st model.ExecutionStatus) bool {
	return model.IsValidExecutionStatus(st)
}
