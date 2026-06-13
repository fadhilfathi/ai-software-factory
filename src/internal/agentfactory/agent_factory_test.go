package agentfactory_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/agentfactory"
	"github.com/fadhilfathi/AI-Software-Factory/internal/aion"
	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/google/uuid"
)

// makeAgent returns a minimal model.Agent for use in SpawnAgent tests.
func makeAgent(projectID uuid.UUID, role string, runtimeJSON string) *model.Agent {
	var runtime json.RawMessage
	if runtimeJSON != "" {
		runtime = json.RawMessage(runtimeJSON)
	}
	return &model.Agent{
		ID:        uuid.New(),
		ProjectID: projectID,
		Role:      role,
		Runtime:   runtime,
	}
}

// TestNew_DefaultsApply verifies that a zero-value Config produces
// the Aion defaults (MiniMax-M3, TokenRouter, YOLO).
func TestNew_DefaultsApply(t *testing.T) {
	t.Parallel()
	rt := aion.NewMockRuntime()
	f, err := agentfactory.New(rt, agentfactory.Config{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if f == nil {
		t.Fatal("New returned nil factory")
	}
}

// TestNew_NilRuntime verifies that passing a nil runtime returns an
// error (the factory requires a runtime to wrap).
func TestNew_NilRuntime(t *testing.T) {
	t.Parallel()
	_, err := agentfactory.New(nil, agentfactory.Config{})
	if err == nil {
		t.Fatal("New(nil, ...) returned nil error; want ErrNilRuntime")
	}
	if !errors.Is(err, agentfactory.ErrNilRuntime) {
		t.Errorf("New(nil, ...) error = %v; want ErrNilRuntime", err)
	}
}

// TestSpawnAgent_NilAgent verifies that passing a nil agent returns
// an error.
func TestSpawnAgent_NilAgent(t *testing.T) {
	t.Parallel()
	rt := aion.NewMockRuntime()
	f, _ := agentfactory.New(rt, agentfactory.Config{})

	_, err := f.SpawnAgent(context.Background(), nil, uuid.New(), "input")
	if err == nil {
		t.Fatal("SpawnAgent(nil, ...) returned nil error; want ErrNilAgent")
	}
	if !errors.Is(err, agentfactory.ErrNilAgent) {
		t.Errorf("SpawnAgent(nil, ...) error = %v; want ErrNilAgent", err)
	}
}

// TestSpawnAgent_DefaultsApplied verifies that an agent with empty
// Runtime gets the factory's Aion defaults on the underlying
// aion.WorkerSpec.
func TestSpawnAgent_DefaultsApplied(t *testing.T) {
	t.Parallel()
	rt := aion.NewMockRuntime()
	f, _ := agentfactory.New(rt, agentfactory.Config{})

	projectID := uuid.New()
	agent := makeAgent(projectID, "developer", "")
	execID := uuid.New()

	h, err := f.SpawnAgent(context.Background(), agent, execID, "build a thing")
	if err != nil {
		t.Fatalf("SpawnAgent: %v", err)
	}
	if h == nil {
		t.Fatal("SpawnAgent returned nil handle")
	}
	if h.AgentID != agent.ID {
		t.Errorf("handle.AgentID = %v; want %v", h.AgentID, agent.ID)
	}
	if h.ExecutionID != execID {
		t.Errorf("handle.ExecutionID = %v; want %v", h.ExecutionID, execID)
	}
	if h.ProjectID != projectID {
		t.Errorf("handle.ProjectID = %v; want %v", h.ProjectID, projectID)
	}
	if h.Role != "developer" {
		t.Errorf("handle.Role = %q; want %q", h.Role, "developer")
	}
	if h.Model != agentfactory.DefaultAionModel {
		t.Errorf("handle.Model = %q; want %q", h.Model, agentfactory.DefaultAionModel)
	}
	if h.Provider != agentfactory.DefaultAionProvider {
		t.Errorf("handle.Provider = %q; want %q", h.Provider, agentfactory.DefaultAionProvider)
	}
	if h.PermissionMode != agentfactory.DefaultAionPermissionMode {
		t.Errorf("handle.PermissionMode = %q; want %q", h.PermissionMode, agentfactory.DefaultAionPermissionMode)
	}
	// Mock handle doesn't have a PID; ParsePIDFromHandle returns 0.
	if h.PID != 0 {
		t.Errorf("handle.PID = %d; want 0 for mock runtime", h.PID)
	}
	if h.WorkerHandle == "" {
		t.Error("handle.WorkerHandle is empty")
	}
	if h.StartedAt.IsZero() {
		t.Error("handle.StartedAt is zero")
	}
}

// TestSpawnAgent_AgentRuntimeOverrides verifies that values from
// agent.Runtime (JSON) override the factory defaults.
func TestSpawnAgent_AgentRuntimeOverrides(t *testing.T) {
	t.Parallel()
	rt := aion.NewMockRuntime()
	f, _ := agentfactory.New(rt, agentfactory.Config{})

	agent := makeAgent(uuid.New(), "reviewer",
		`{"model":"claude-opus-4-7","provider":"Anthropic","permission_mode":"safe"}`)
	execID := uuid.New()

	h, err := f.SpawnAgent(context.Background(), agent, execID, "review this")
	if err != nil {
		t.Fatalf("SpawnAgent: %v", err)
	}
	if h.Model != "claude-opus-4-7" {
		t.Errorf("handle.Model = %q; want %q", h.Model, "claude-opus-4-7")
	}
	if h.Provider != "Anthropic" {
		t.Errorf("handle.Provider = %q; want %q", h.Provider, "Anthropic")
	}
	if h.PermissionMode != "safe" {
		t.Errorf("handle.PermissionMode = %q; want %q", h.PermissionMode, "safe")
	}
}

// TestSpawnAgent_AgentRuntimePartialOverrides verifies that missing
// fields in agent.Runtime fall through to factory defaults.
func TestSpawnAgent_AgentRuntimePartialOverrides(t *testing.T) {
	t.Parallel()
	rt := aion.NewMockRuntime()
	f, _ := agentfactory.New(rt, agentfactory.Config{})

	// Only override model; provider + permission_mode fall through.
	agent := makeAgent(uuid.New(), "qa", `{"model":"claude-opus-4-7"}`)
	execID := uuid.New()

	h, err := f.SpawnAgent(context.Background(), agent, execID, "test this")
	if err != nil {
		t.Fatalf("SpawnAgent: %v", err)
	}
	if h.Model != "claude-opus-4-7" {
		t.Errorf("handle.Model = %q; want %q", h.Model, "claude-opus-4-7")
	}
	if h.Provider != agentfactory.DefaultAionProvider {
		t.Errorf("handle.Provider = %q; want %q", h.Provider, agentfactory.DefaultAionProvider)
	}
	if h.PermissionMode != agentfactory.DefaultAionPermissionMode {
		t.Errorf("handle.PermissionMode = %q; want %q", h.PermissionMode, agentfactory.DefaultAionPermissionMode)
	}
}

// TestSpawnAgent_AgentRuntimeMalformedJSON verifies that malformed
// JSON in agent.Runtime falls through to factory defaults without
// failing the spawn.
func TestSpawnAgent_AgentRuntimeMalformedJSON(t *testing.T) {
	t.Parallel()
	rt := aion.NewMockRuntime()
	f, _ := agentfactory.New(rt, agentfactory.Config{})

	agent := makeAgent(uuid.New(), "devops", `{not valid json`)
	execID := uuid.New()

	h, err := f.SpawnAgent(context.Background(), agent, execID, "deploy this")
	if err != nil {
		t.Fatalf("SpawnAgent: %v", err)
	}
	if h.Model != agentfactory.DefaultAionModel {
		t.Errorf("handle.Model = %q; want %q (fall-through to default)", h.Model, agentfactory.DefaultAionModel)
	}
}

// TestGet_TrackedAfterSpawn verifies that Get returns the spawned
// handle.
func TestGet_TrackedAfterSpawn(t *testing.T) {
	t.Parallel()
	rt := aion.NewMockRuntime()
	f, _ := agentfactory.New(rt, agentfactory.Config{})

	agent := makeAgent(uuid.New(), "developer", "")
	h, err := f.SpawnAgent(context.Background(), agent, uuid.New(), "x")
	if err != nil {
		t.Fatalf("SpawnAgent: %v", err)
	}

	got, ok := f.Get(agent.ID)
	if !ok {
		t.Fatalf("Get(%v) returned ok=false; want true", agent.ID)
	}
	if got.AgentID != h.AgentID {
		t.Errorf("Get returned AgentID=%v; want %v", got.AgentID, h.AgentID)
	}
	if got.WorkerHandle != h.WorkerHandle {
		t.Errorf("Get returned WorkerHandle=%q; want %q", got.WorkerHandle, h.WorkerHandle)
	}
}

// TestGet_NotFound verifies that Get returns false for an unknown
// agent ID.
func TestGet_NotFound(t *testing.T) {
	t.Parallel()
	rt := aion.NewMockRuntime()
	f, _ := agentfactory.New(rt, agentfactory.Config{})

	_, ok := f.Get(uuid.New())
	if ok {
		t.Error("Get on fresh factory returned ok=true; want false")
	}
}

// TestTracked_Empty verifies that Tracked returns an empty slice on
// a fresh factory.
func TestTracked_Empty(t *testing.T) {
	t.Parallel()
	rt := aion.NewMockRuntime()
	f, _ := agentfactory.New(rt, agentfactory.Config{})

	if got := f.Tracked(); len(got) != 0 {
		t.Errorf("Tracked on fresh factory = %d items; want 0", len(got))
	}
	if got := f.TrackedCount(); got != 0 {
		t.Errorf("TrackedCount on fresh factory = %d; want 0", got)
	}
}

// TestTracked_MultipleAgents verifies that multiple spawns are all
// tracked.
func TestTracked_MultipleAgents(t *testing.T) {
	t.Parallel()
	rt := aion.NewMockRuntime()
	f, _ := agentfactory.New(rt, agentfactory.Config{})

	projectID := uuid.New()
	agents := []*model.Agent{
		makeAgent(projectID, "developer", ""),
		makeAgent(projectID, "reviewer", ""),
		makeAgent(projectID, "qa", ""),
	}
	for i, agent := range agents {
		_, err := f.SpawnAgent(context.Background(), agent, uuid.New(), "task")
		if err != nil {
			t.Fatalf("SpawnAgent #%d: %v", i, err)
		}
	}

	if got := f.TrackedCount(); got != 3 {
		t.Errorf("TrackedCount = %d; want 3", got)
	}
	tracked := f.Tracked()
	if len(tracked) != 3 {
		t.Fatalf("Tracked = %d items; want 3", len(tracked))
	}
	// Each handle should be distinct by AgentID.
	seen := make(map[uuid.UUID]bool)
	for _, h := range tracked {
		seen[h.AgentID] = true
	}
	for _, agent := range agents {
		if !seen[agent.ID] {
			t.Errorf("Tracked missing agent %v", agent.ID)
		}
	}
}

// TestShutdown_Idempotent verifies that calling Shutdown twice is
// safe (only the first call has effect).
func TestShutdown_Idempotent(t *testing.T) {
	t.Parallel()
	rt := aion.NewMockRuntime()
	f, _ := agentfactory.New(rt, agentfactory.Config{})

	// Spawn a few agents first.
	for i := 0; i < 3; i++ {
		_, err := f.SpawnAgent(context.Background(), makeAgent(uuid.New(), "developer", ""), uuid.New(), "x")
		if err != nil {
			t.Fatalf("SpawnAgent: %v", err)
		}
	}
	if got := f.TrackedCount(); got != 3 {
		t.Fatalf("TrackedCount before Shutdown = %d; want 3", got)
	}

	if err := f.Shutdown(context.Background()); err != nil {
		t.Errorf("first Shutdown: %v", err)
	}
	if got := f.TrackedCount(); got != 0 {
		t.Errorf("TrackedCount after first Shutdown = %d; want 0", got)
	}
	// Second call should be a no-op.
	if err := f.Shutdown(context.Background()); err != nil {
		t.Errorf("second Shutdown: %v", err)
	}
}

// TestShutdown_AlreadyShutdownOnSpawn verifies that SpawnAgent after
// Shutdown returns ErrAlreadyShutdown.
func TestShutdown_AlreadyShutdownOnSpawn(t *testing.T) {
	t.Parallel()
	rt := aion.NewMockRuntime()
	f, _ := agentfactory.New(rt, agentfactory.Config{})

	if err := f.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	_, err := f.SpawnAgent(context.Background(), makeAgent(uuid.New(), "developer", ""), uuid.New(), "x")
	if !errors.Is(err, agentfactory.ErrAlreadyShutdown) {
		t.Errorf("SpawnAgent after Shutdown: err = %v; want ErrAlreadyShutdown", err)
	}
}

// TestShutdown_ClosesRuntime verifies that Shutdown closes the
// underlying runtime.
func TestShutdown_ClosesRuntime(t *testing.T) {
	t.Parallel()
	rt := aion.NewMockRuntime()
	f, _ := agentfactory.New(rt, agentfactory.Config{})

	_, err := f.SpawnAgent(context.Background(), makeAgent(uuid.New(), "developer", ""), uuid.New(), "x")
	if err != nil {
		t.Fatalf("SpawnAgent: %v", err)
	}

	if err := f.Shutdown(context.Background()); err != nil {
		t.Errorf("Shutdown: %v", err)
	}
	// After Close, the runtime's Spawn should return ErrRuntimeClosed.
	_, err = rt.Spawn(context.Background(), aion.WorkerSpec{
		ExecutionID: uuid.New(),
		AgentID:     uuid.New(),
		ProjectID:   uuid.New(),
		Model:       "x",
		Provider:    "y",
		Input:       "z",
	})
	if !errors.Is(err, aion.ErrRuntimeClosed) {
		t.Errorf("Spawn after Close: err = %v; want ErrRuntimeClosed", err)
	}
}

// TestShutdown_ConcurrentSafety verifies that Shutdown is safe to
// call concurrently with SpawnAgent.
func TestShutdown_ConcurrentSafety(t *testing.T) {
	t.Parallel()
	rt := aion.NewMockRuntime()
	f, _ := agentfactory.New(rt, agentfactory.Config{})

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_ = f.Shutdown(context.Background())
	}()
	go func() {
		defer wg.Done()
		// Spawn may succeed or fail with ErrAlreadyShutdown depending
		// on which goroutine wins. Both outcomes are acceptable.
		_, _ = f.SpawnAgent(context.Background(), makeAgent(uuid.New(), "developer", ""), uuid.New(), "x")
	}()
	wg.Wait()
}

// TestParsePIDFromHandle_Valid verifies that a well-formed process
// handle parses to its PID.
func TestParsePIDFromHandle_Valid(t *testing.T) {
	t.Parallel()
	pid, err := agentfactory.ParsePIDFromHandle(aion.WorkerHandle("proc-12345-abc-def"))
	if err != nil {
		t.Fatalf("ParsePIDFromHandle: %v", err)
	}
	if pid != 12345 {
		t.Errorf("pid = %d; want 12345", pid)
	}
}

// TestParsePIDFromHandle_LargePID verifies that large PIDs (up to
// int max) parse correctly.
func TestParsePIDFromHandle_LargePID(t *testing.T) {
	t.Parallel()
	pid, err := agentfactory.ParsePIDFromHandle(aion.WorkerHandle("proc-2147483647-abc-def"))
	if err != nil {
		t.Fatalf("ParsePIDFromHandle: %v", err)
	}
	if pid != 2147483647 {
		t.Errorf("pid = %d; want 2147483647", pid)
	}
}

// TestParsePIDFromHandle_MockHandle verifies that a mock-runtime
// handle returns an error (not a "proc-" prefix).
func TestParsePIDFromHandle_MockHandle(t *testing.T) {
	t.Parallel()
	_, err := agentfactory.ParsePIDFromHandle(aion.WorkerHandle("mock-1-abc-def"))
	if err == nil {
		t.Error("ParsePIDFromHandle on mock handle returned nil error; want error")
	}
}

// TestParsePIDFromHandle_MalformedPID verifies that a handle with a
// non-numeric PID segment returns an error.
func TestParsePIDFromHandle_MalformedPID(t *testing.T) {
	t.Parallel()
	_, err := agentfactory.ParsePIDFromHandle(aion.WorkerHandle("proc-abc-def"))
	if err == nil {
		t.Error("ParsePIDFromHandle on malformed PID returned nil error; want error")
	}
}

// TestParsePIDFromHandle_NoDashSeparator verifies that a handle
// without a dash separator after the PID returns an error.
func TestParsePIDFromHandle_NoDashSeparator(t *testing.T) {
	t.Parallel()
	_, err := agentfactory.ParsePIDFromHandle(aion.WorkerHandle("proc-12345"))
	if err == nil {
		t.Error("ParsePIDFromHandle on handle without dash separator returned nil error; want error")
	}
}

// TestSpawnAgent_PopulatesStartedAt verifies that SpawnAgent sets a
// non-zero StartedAt close to "now".
func TestSpawnAgent_PopulatesStartedAt(t *testing.T) {
	t.Parallel()
	rt := aion.NewMockRuntime()
	f, _ := agentfactory.New(rt, agentfactory.Config{})

	before := time.Now().UTC()
	h, err := f.SpawnAgent(context.Background(), makeAgent(uuid.New(), "developer", ""), uuid.New(), "x")
	after := time.Now().UTC()
	if err != nil {
		t.Fatalf("SpawnAgent: %v", err)
	}
	if h.StartedAt.Before(before) || h.StartedAt.After(after) {
		t.Errorf("StartedAt = %v; want in [%v, %v]", h.StartedAt, before, after)
	}
}
