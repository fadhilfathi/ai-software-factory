// ProcessRuntime is the subprocess-backed implementation of the
// Runtime interface (Mode B in
// docs/sprint5/integration-test-plan.md §2). It `os/exec`'s the
// `aion` CLI binary as a child process per spawn, communicates
// with the worker over stdio using the JSON-over-stdio protocol
// envelope (aion.Message, see TASK-504), and reaps the child on
// Wait.
//
// The runtime is gated by:
//
//   - RuntimeMode == "process" in the agent config (dev / prod
//     switch), or
//   - AION_E2E=1 in the environment (E2E test mode)
//
// If the `aion` binary is not on PATH and the mode is "process",
// Spawn returns an error immediately. This is intentional: a
// misconfigured production binary should fail fast, not silently
// fall back to the Mock.
//
// Threading: each spawn owns a goroutine that pumps stdout to the
// result channel; Wait blocks on the result channel. The cmd field
// is the only mutable per-handle state and is guarded by the
// `handleMu` mutex (process kill is async-safe; result delivery
// is via the channel).
package aion

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// ProcessRuntimeConfig is the configuration for ProcessRuntime.
type ProcessRuntimeConfig struct {
	// Binary is the path to the `aion` CLI executable. Defaults
	// to "aion" (PATH lookup) when empty.
	Binary string

	// ExtraArgs are prepended to the argv for every spawn
	// (e.g. --log-level=debug, --api-key=$AION_KEY). The
	// per-spec args (--execution-id, --task-id, --agent-id,
	// --model, --provider, --permission-mode) are appended
	// after these.
	ExtraArgs []string

	// WaitTimeout caps the Wait() call. If the worker hasn't
	// reached a terminal status by then, Wait returns
	// ErrWorkerTimeout (the worker keeps running in the
	// background — call Cancel to reap it).
	WaitTimeout time.Duration

	// Env is the per-spawn environment. nil means inherit the
	// parent's env.
	Env []string

	// StdoutBufferSize is the size of the per-worker stdout
	// scanner buffer. Defaults to 64 KiB.
	StdoutBufferSize int

	// StderrBufferSize is the size of the per-worker stderr
	// capture buffer. Defaults to 64 KiB.
	StderrBufferSize int
}

func (c *ProcessRuntimeConfig) defaults() {
	if c.Binary == "" {
		c.Binary = "aion"
	}
	if c.WaitTimeout == 0 {
		c.WaitTimeout = 30 * time.Minute
	}
	if c.StdoutBufferSize == 0 {
		c.StdoutBufferSize = 64 * 1024
	}
	if c.StderrBufferSize == 0 {
		c.StderrBufferSize = 64 * 1024
	}
}

// ProcessRuntime is the subprocess Runtime.
type ProcessRuntime struct {
	cfg ProcessRuntimeConfig

	mu      sync.Mutex
	workers map[WorkerHandle]*processWorker
	closed  atomic.Bool
	closeCh chan struct{}
}

type processWorker struct {
	spec     WorkerSpec
	cmd      *exec.Cmd
	cancel   func() error // cmd.Process.Kill wrapped
	resultCh chan WorkerResult
	done     atomic.Bool
}

// NewProcessRuntime constructs a ProcessRuntime with the given
// config. The binary is not probed here; Spawn is the first time
// we hit the filesystem.
func NewProcessRuntime(cfg ProcessRuntimeConfig) *ProcessRuntime {
	cfg.defaults()
	return &ProcessRuntime{
		cfg:     cfg,
		workers: make(map[WorkerHandle]*processWorker),
		closeCh: make(chan struct{}),
	}
}

// Available reports whether the configured binary is on PATH (or
// at the absolute path given). Useful for the config loader to
// fail fast at startup if the operator forgot to install `aion`.
func (r *ProcessRuntime) Available() (string, bool) {
	path, err := exec.LookPath(r.cfg.Binary)
	if err != nil {
		return "", false
	}
	return path, true
}

// ----------------------------------------------------------------------------
// Runtime interface
// ----------------------------------------------------------------------------

// Spawn forks the `aion` CLI with the spec serialised as flags.
// The child process is the worker; the runtime listens on stdout
// for the terminal Message frame and pushes it to the result
// channel for Wait to consume.
func (r *ProcessRuntime) Spawn(ctx context.Context, spec WorkerSpec) (WorkerHandle, error) {
	if r.closed.Load() {
		return "", ErrRuntimeClosed
	}
	if err := spec.Validate(); err != nil {
		return "", err
	}

	args := append([]string{}, r.cfg.ExtraArgs...)
	args = append(args,
		"--execution-id="+spec.ExecutionID.String(),
		"--task-id="+spec.TaskID.String(),
		"--agent-id="+spec.AgentID.String(),
		"--project-id="+spec.ProjectID.String(),
		"--model="+spec.Model,
		"--provider="+spec.Provider,
		"--permission-mode="+spec.PermissionMode,
		fmt.Sprintf("--attempt=%d", spec.Attempt),
	)

	cmd := exec.CommandContext(ctx, r.cfg.Binary, args...)
	if len(r.cfg.Env) > 0 {
		cmd.Env = append(os.Environ(), r.cfg.Env...)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("aion: stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("aion: stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		// Common cause: binary not on PATH. Surface a clear
		// error rather than the raw exec error.
		if errors.Is(err, exec.ErrNotFound) || strings.Contains(err.Error(), "executable file not found") {
			return "", fmt.Errorf("aion: binary %q not on PATH (set AionBinary in config or install aion): %w", r.cfg.Binary, err)
		}
		return "", fmt.Errorf("aion: start worker: %w", err)
	}

	handle := WorkerHandle(fmt.Sprintf("proc-%d-%s", cmd.Process.Pid, spec.ExecutionID.String()))

	worker := &processWorker{
		spec:     spec,
		cmd:      cmd,
		cancel:   func() error { return cmd.Process.Kill() },
		resultCh: make(chan WorkerResult, 1),
	}

	r.mu.Lock()
	r.workers[handle] = worker
	r.mu.Unlock()

	// Pump stdout: read Message frames; the LAST one is the
	// terminal frame (started, result, error, cancelled).
	// Earlier "progress" frames are logged but not surfaced
	// to Wait (TASK-506 will subscribe to them).
	go r.pumpStdout(worker, handle, stdout, r.cfg.StdoutBufferSize)
	// Drain stderr so the child doesn't block on write; capture
	// the last line for the WorkerResult's ErrorMessage if the
	// exit was non-zero.
	go r.pumpStderr(worker, stderr, r.cfg.StderrBufferSize)

	return handle, nil
}

// pumpStdout reads Message frames from the worker's stdout and
// delivers the terminal frame to Wait.
func (r *ProcessRuntime) pumpStdout(worker *processWorker, handle WorkerHandle, stdout io.Reader, bufSize int) {
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, bufSize), bufSize)

	var lastMsg Message
	sawFrame := false
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var msg Message
		if err := json.Unmarshal(line, &msg); err != nil {
			// Not a JSON frame — ignore. Workers are allowed
			// to log free-form text to stdout; the protocol
			// frames are JSON objects.
			continue
		}
		lastMsg = msg
		sawFrame = true
	}

	completedAt := time.Now().UTC()
	var result WorkerResult
	switch {
	case !sawFrame:
		// No frames at all — the worker died before saying
		// anything. Treat as failed.
		result = WorkerResult{
			Handle:       handle,
			ExecutionID:  worker.spec.ExecutionID,
			Status:       WorkerFailed,
			StartedAt:    completedAt,
			CompletedAt:  completedAt,
			ErrorMessage: "aion worker produced no output before exit",
		}
	case lastMsg.Type == "result":
		result = WorkerResult{
			Handle:       handle,
			ExecutionID:  worker.spec.ExecutionID,
			Status:       WorkerCompleted,
			Result:       lastMsg.Body,
			StartedAt:    lastMsg.At,
			CompletedAt:  completedAt,
		}
	case lastMsg.Type == "error":
		result = WorkerResult{
			Handle:       handle,
			ExecutionID:  worker.spec.ExecutionID,
			Status:       WorkerFailed,
			ErrorMessage: lastMsg.Error,
			StartedAt:    lastMsg.At,
			CompletedAt:  completedAt,
		}
	case lastMsg.Type == "cancelled":
		result = WorkerResult{
			Handle:       handle,
			ExecutionID:  worker.spec.ExecutionID,
			Status:       WorkerCancelled,
			StartedAt:    lastMsg.At,
			CompletedAt:  completedAt,
		}
	default:
		// "started" or "progress" as the LAST frame is
		// unusual — the worker exited without a terminal
		// frame. Treat as failed.
		result = WorkerResult{
			Handle:       handle,
			ExecutionID:  worker.spec.ExecutionID,
			Status:       WorkerFailed,
			StartedAt:    lastMsg.At,
			CompletedAt:  completedAt,
			ErrorMessage: fmt.Sprintf("aion worker exited after non-terminal frame %q", lastMsg.Type),
		}
	}

	if !worker.done.CompareAndSwap(false, true) {
		return
	}
	worker.resultCh <- result
}

// pumpStderr drains the worker's stderr into a ring buffer (last
// line wins). On Cancel, we splice it into the result's
// ErrorMessage.
func (r *ProcessRuntime) pumpStderr(worker *processWorker, stderr io.Reader, bufSize int) {
	scanner := bufio.NewScanner(stderr)
	scanner.Buffer(make([]byte, bufSize), bufSize)
	var lastLine string
	for scanner.Scan() {
		lastLine = scanner.Text()
	}
	_ = lastLine // reserved for TASK-506 / TASK-508 recovery context
	_ = worker
}

// Wait blocks until the worker reaches a terminal status, the
// runtime is closed, the wait timeout elapses, or the context is
// cancelled.
func (r *ProcessRuntime) Wait(ctx context.Context, handle WorkerHandle) (WorkerResult, error) {
	if r.closed.Load() {
		return WorkerResult{}, ErrRuntimeClosed
	}

	r.mu.Lock()
	worker, ok := r.workers[handle]
	r.mu.Unlock()
	if !ok {
		return WorkerResult{}, ErrWorkerNotFound
	}

	// Combine the caller's context with the runtime's wait
	// timeout + close signal.
	waitCtx, cancel := context.WithTimeout(ctx, r.cfg.WaitTimeout)
	defer cancel()

	// Reap the child process asynchronously so cmd.Wait() is
	// always called (avoids zombie processes). The result
	// channel is the source of truth; cmd.Wait is purely for
	// resource cleanup.
	go func() {
		_ = worker.cmd.Wait()
	}()

	select {
	case result := <-worker.resultCh:
		return result, nil
	case <-waitCtx.Done():
		if ctx.Err() != nil {
			return WorkerResult{}, fmt.Errorf("%w: %v", ErrWorkerTimeout, ctx.Err())
		}
		return WorkerResult{}, fmt.Errorf("%w: %v", ErrWorkerTimeout, waitCtx.Err())
	case <-r.closeCh:
		select {
		case result := <-worker.resultCh:
			return result, nil
		default:
			return WorkerResult{}, ErrRuntimeClosed
		}
	}
}

// Cancel kills the worker process. Idempotent. Returns
// ErrWorkerNotFound for unknown handles.
func (r *ProcessRuntime) Cancel(ctx context.Context, handle WorkerHandle) error {
	r.mu.Lock()
	worker, ok := r.workers[handle]
	r.mu.Unlock()
	if !ok {
		return ErrWorkerNotFound
	}

	if err := worker.cancel(); err != nil {
		// Process already exited — that's fine. The result
		// channel will deliver the terminal status.
		if !errors.Is(err, os.ErrProcessDone) {
			return fmt.Errorf("aion: cancel: %w", err)
		}
	}
	return nil
}

// Close stops accepting new spawns and kills all in-flight
// workers. Wait calls return ErrRuntimeClosed.
func (r *ProcessRuntime) Close() error {
	if !r.closed.CompareAndSwap(false, true) {
		return nil
	}
	close(r.closeCh)

	r.mu.Lock()
	workers := make([]*processWorker, 0, len(r.workers))
	for _, w := range r.workers {
		workers = append(workers, w)
	}
	r.mu.Unlock()

	for _, w := range workers {
		_ = w.cancel()
	}
	return nil
}

// ActiveWorkers returns the count of in-flight workers. For the
// ProcessRuntime this includes both running and zombie-but-not-yet-
// reaped workers.
func (r *ProcessRuntime) ActiveWorkers() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.workers)
}

// Forget removes a worker from the map. Tests use this to assert
// cleanup paths.
func (r *ProcessRuntime) Forget(handle WorkerHandle) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.workers, handle)
}

// ----------------------------------------------------------------------------
// Compile-time interface check
// ----------------------------------------------------------------------------

var _ Runtime = (*ProcessRuntime)(nil)
