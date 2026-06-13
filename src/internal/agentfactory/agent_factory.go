// Package agentfactory wraps the Aion runtime with Aion-specific
// defaults (TokenRouter / MiniMax-M3 / YOLO), tracks every spawned
// agent for observability, and SIGTERMs all tracked PIDs on
// Shutdown.
//
// The factory is independent of ExecutionService; it is the lower
// layer that future call sites (POST /v1/tasks/:id/execute, recovery
// flows) use to spawn Aion agents. ExecutionService integration is
// tracked as a follow-up — see docs/sprint5/agent-creation-management-design.md
// section 1.
package agentfactory

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/aion"
	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// DefaultAionModel is the default Aion model when agent.Runtime is empty.
const DefaultAionModel = "MiniMax-M3"

// DefaultAionProvider is the default Aion provider when agent.Runtime is empty.
const DefaultAionProvider = "TokenRouter"

// DefaultAionPermissionMode is the default Aion permission mode when
// agent.Runtime is empty.
const DefaultAionPermissionMode = "YOLO"

// DefaultShutdownGracePeriod is the default time given to tracked
// agents to exit gracefully after SIGTERM.
const DefaultShutdownGracePeriod = 5 * time.Second

// Sentinel errors. Callers use errors.Is() to match.
var (
	// ErrAlreadyShutdown is returned by SpawnAgent when the factory
	// has been shut down.
	ErrAlreadyShutdown = errors.New("agent factory already shut down")
	// ErrAgentNotTracked is returned by lookup helpers when the
	// agent ID is not in the tracking map.
	ErrAgentNotTracked = errors.New("agent not tracked")
	// ErrNilAgent is returned by SpawnAgent when the caller passes
	// a nil agent.
	ErrNilAgent = errors.New("agent is nil")
	// ErrNilRuntime is returned by New when the caller passes a nil
	// aion.Runtime.
	ErrNilRuntime = errors.New("runtime is nil")
)

// processHandlePrefix is the prefix the ProcessRuntime writes on every
// handle (see src/internal/aion/process.go).
const processHandlePrefix = "proc-"

// Config configures an AgentFactory. Zero values use the Aion defaults
// (MiniMax-M3, TokenRouter, YOLO, 5s shutdown grace, zap.NewNop
// logger).
type Config struct {
	// DefaultModel is the Aion model used when agent.Runtime does not
	// specify one. Defaults to DefaultAionModel.
	DefaultModel string
	// DefaultProvider is the Aion provider used when agent.Runtime
	// does not specify one. Defaults to DefaultAionProvider.
	DefaultProvider string
	// DefaultPermissionMode is the Aion permission mode used when
	// agent.Runtime does not specify one. Defaults to
	// DefaultAionPermissionMode.
	DefaultPermissionMode string
	// ShutdownGracePeriod is the advisory time given to tracked
	// agents to exit after SIGTERM. Currently advisory only; the
	// runtime's Close() will SIGKILL the subprocess group as a
	// backstop. Defaults to DefaultShutdownGracePeriod.
	ShutdownGracePeriod time.Duration
	// Logger is the structured logger. Defaults to zap.NewNop().
	Logger *zap.Logger
}

// runtimeOverrides mirrors the JSON shape of model.Agent.Runtime when
// it carries Aion-specific overrides.
type runtimeOverrides struct {
	Model          string `json:"model"`
	Provider       string `json:"provider"`
	PermissionMode string `json:"permission_mode"`
}

// AgentHandle is a tracked reference to a spawned Aion agent process.
// It is a snapshot of the factory's internal state at SpawnAgent
// time; the caller does not mutate it.
type AgentHandle struct {
	// AgentID is the model.Agent.ID.
	AgentID uuid.UUID
	// ExecutionID is the model.Execution.ID this agent is working on.
	ExecutionID uuid.UUID
	// ProjectID is the model.Agent.ProjectID.
	ProjectID uuid.UUID
	// Role is the agent's role (developer, reviewer, qa, devops, etc.).
	// Free-text per model.Agent conventions.
	Role string
	// Model is the resolved Aion model.
	Model string
	// Provider is the resolved Aion provider.
	Provider string
	// PermissionMode is the resolved Aion permission mode.
	PermissionMode string
	// WorkerHandle is the underlying aion.WorkerHandle. Persist this
	// in model.Worker.Handle for Wait/Cancel coordination.
	WorkerHandle aion.WorkerHandle
	// PID is the OS process ID for subprocess runtimes, or 0 for
	// mock runtimes. SIGTERM is sent to this PID on Shutdown().
	PID int
	// StartedAt is when the agent process was spawned (UTC).
	StartedAt time.Time
}

// tracked is the factory's internal record per spawned agent. The
// factory keeps a pointer to AgentHandle so the public Tracked()
// and Get() accessors can return copies.
type tracked struct {
	handle *AgentHandle
}

// AgentFactory spawns Aion agent subprocesses and tracks them for
// graceful shutdown. Construct one per process (typically a singleton
// in main.go) and pass it to anything that spawns agents.
//
// AgentFactory is safe for concurrent use by multiple goroutines.
type AgentFactory struct {
	cfg     Config
	runtime aion.Runtime
	log     *zap.Logger

	mu       sync.Mutex
	tracked  map[uuid.UUID]*tracked
	stopOnce sync.Once
	stopped  bool
}

// New creates a new AgentFactory that wraps the given aion.Runtime.
// The runtime is owned by the factory; the factory will call
// Close() on it during Shutdown.
func New(runtime aion.Runtime, cfg Config) (*AgentFactory, error) {
	if runtime == nil {
		return nil, ErrNilRuntime
	}
	if cfg.DefaultModel == "" {
		cfg.DefaultModel = DefaultAionModel
	}
	if cfg.DefaultProvider == "" {
		cfg.DefaultProvider = DefaultAionProvider
	}
	if cfg.DefaultPermissionMode == "" {
		cfg.DefaultPermissionMode = DefaultAionPermissionMode
	}
	if cfg.ShutdownGracePeriod == 0 {
		cfg.ShutdownGracePeriod = DefaultShutdownGracePeriod
	}
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}
	return &AgentFactory{
		cfg:     cfg,
		runtime: runtime,
		log:     cfg.Logger,
		tracked: make(map[uuid.UUID]*tracked),
	}, nil
}

// SpawnAgent spawns a new Aion agent subprocess for the given agent
// and returns a tracked handle. The agent is registered in the
// factory's tracking map; subsequent calls to Shutdown will SIGTERM
// the underlying OS process (if PID is non-zero).
//
// The runtime's Spawn is called synchronously; the caller is
// responsible for any wait/cancel coordination (typically by
// passing the returned WorkerHandle to runtime.Wait or storing it in
// model.Worker.Handle for a follow-up driveWorker goroutine).
//
// SpawnAgent does NOT do project-scope checks. Callers MUST verify
// that the agent is accessible to the caller (F-016) before calling.
func (f *AgentFactory) SpawnAgent(
	ctx context.Context,
	agent *model.Agent,
	executionID uuid.UUID,
	input string,
) (*AgentHandle, error) {
	if agent == nil {
		return nil, ErrNilAgent
	}

	f.mu.Lock()
	if f.stopped {
		f.mu.Unlock()
		return nil, ErrAlreadyShutdown
	}
	f.mu.Unlock()

	modelName, provider, permissionMode := f.resolveRuntime(agent)

	spec := aion.WorkerSpec{
		ExecutionID:    executionID,
		AgentID:        agent.ID,
		ProjectID:      agent.ProjectID,
		Model:          modelName,
		Provider:       provider,
		PermissionMode: permissionMode,
		Input:          input,
		Attempt:        1,
	}

	handle, err := f.runtime.Spawn(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("spawn aion agent: %w", err)
	}

	pid, _ := ParsePIDFromHandle(handle) // 0 for mock runtime

	h := &AgentHandle{
		AgentID:        agent.ID,
		ExecutionID:    executionID,
		ProjectID:      agent.ProjectID,
		Role:           agent.Role,
		Model:          modelName,
		Provider:       provider,
		PermissionMode: permissionMode,
		WorkerHandle:   handle,
		PID:            pid,
		StartedAt:      time.Now().UTC(),
	}

	f.mu.Lock()
	if f.stopped {
		// Race: shutdown happened between our check and now. Best-effort
		// SIGTERM the just-spawned agent before returning the error.
		f.mu.Unlock()
		if pid != 0 {
			if sigErr := syscall.Kill(pid, syscall.SIGTERM); sigErr != nil {
				f.log.Warn("post-spawn SIGTERM failed (race with Shutdown)",
					zap.String("agent_id", agent.ID.String()),
					zap.Int("pid", pid),
					zap.Error(sigErr),
				)
			}
		}
		return nil, ErrAlreadyShutdown
	}
	f.tracked[agent.ID] = &tracked{handle: h}
	f.mu.Unlock()

	f.log.Info("spawned aion agent",
		zap.String("agent_id", agent.ID.String()),
		zap.String("execution_id", executionID.String()),
		zap.String("role", agent.Role),
		zap.String("model", modelName),
		zap.String("provider", provider),
		zap.String("permission_mode", permissionMode),
		zap.Int("pid", pid),
	)

	return h, nil
}

// resolveRuntime returns the (model, provider, permission_mode)
// triple to use for the given agent. Values from agent.Runtime
// (json.RawMessage) take precedence over the factory defaults when
// present.
func (f *AgentFactory) resolveRuntime(agent *model.Agent) (string, string, string) {
	modelName := f.cfg.DefaultModel
	provider := f.cfg.DefaultProvider
	permissionMode := f.cfg.DefaultPermissionMode

	if len(agent.Runtime) > 0 {
		var rt runtimeOverrides
		if err := json.Unmarshal(agent.Runtime, &rt); err == nil {
			if rt.Model != "" {
				modelName = rt.Model
			}
			if rt.Provider != "" {
				provider = rt.Provider
			}
			if rt.PermissionMode != "" {
				permissionMode = rt.PermissionMode
			}
		}
		// Silently ignore JSON parse errors: an agent with malformed
		// Runtime falls back to factory defaults rather than failing
		// the spawn. The agent's Runtime is informational, not
		// load-bearing.
	}

	return modelName, provider, permissionMode
}

// Tracked returns a snapshot of all currently tracked agent handles.
// The returned slice is a copy; mutating it does not affect the
// factory's state.
func (f *AgentFactory) Tracked() []AgentHandle {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]AgentHandle, 0, len(f.tracked))
	for _, t := range f.tracked {
		out = append(out, *t.handle)
	}
	return out
}

// Get returns the tracked handle for the given agent ID. The
// returned bool is false if the agent is not in the tracking map.
func (f *AgentFactory) Get(agentID uuid.UUID) (AgentHandle, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	t, ok := f.tracked[agentID]
	if !ok {
		return AgentHandle{}, false
	}
	return *t.handle, true
}

// TrackedCount returns the number of currently tracked agents. Cheap
// (no allocation).
func (f *AgentFactory) TrackedCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.tracked)
}

// Shutdown sends SIGTERM to every tracked agent subprocess and then
// closes the underlying runtime. Safe to call multiple times - only
// the first call has effect.
//
// The caller's ctx bounds the SIGTERM loop. Agents that don't exit
// within cfg.ShutdownGracePeriod are abandoned; the runtime's
// Close() will SIGKILL the subprocess group as a backstop.
func (f *AgentFactory) Shutdown(ctx context.Context) error {
	var firstCall bool
	f.stopOnce.Do(func() {
		firstCall = true
		f.mu.Lock()
		f.stopped = true
		tracked := f.tracked
		f.tracked = make(map[uuid.UUID]*tracked)
		f.mu.Unlock()

		for agentID, t := range tracked {
			if t.handle.PID == 0 {
				// Mock runtime or unparseable handle - nothing to SIGTERM.
				f.log.Debug("skipping SIGTERM (no PID)",
					zap.String("agent_id", agentID.String()),
				)
				continue
			}
			if err := syscall.Kill(t.handle.PID, syscall.SIGTERM); err != nil {
				f.log.Warn("SIGTERM failed for tracked agent",
					zap.String("agent_id", agentID.String()),
					zap.Int("pid", t.handle.PID),
					zap.Error(err),
				)
				continue
			}
			f.log.Info("SIGTERM sent to tracked agent",
				zap.String("agent_id", agentID.String()),
				zap.Int("pid", t.handle.PID),
			)
		}
	})

	if !firstCall {
		return nil
	}

	// Close the runtime. The runtime is responsible for SIGKILLing
	// any still-running subprocesses.
	if err := f.runtime.Close(); err != nil {
		f.log.Warn("close runtime during shutdown", zap.Error(err))
		return err
	}
	return nil
}

// ParsePIDFromHandle extracts the PID from a process-runtime handle.
// The ProcessRuntime writes handles in the format "proc-<pid>-<uuid>"
// (see src/internal/aion/process.go). Returns 0 and a typed error if
// the handle is not a process handle (e.g., the mock runtime uses
// "fake-<uuid>" / "mock-<n>-<uuid>").
//
// Callers that want to ignore parse errors (e.g., to default PID to 0
// for mock runtimes) can use the result without checking the error.
func ParsePIDFromHandle(h aion.WorkerHandle) (int, error) {
	s := string(h)
	if !strings.HasPrefix(s, processHandlePrefix) {
		return 0, fmt.Errorf("not a process handle: %q", s)
	}
	rest := s[len(processHandlePrefix):]
	dashIdx := strings.Index(rest, "-")
	if dashIdx < 0 {
		return 0, fmt.Errorf("malformed process handle: %q", s)
	}
	pid, err := strconv.Atoi(rest[:dashIdx])
	if err != nil {
		return 0, fmt.Errorf("malformed PID in handle %q: %w", s, err)
	}
	return pid, nil
}
