// Package aion tests.
//
// All tests use MockRuntime (Mode A) since the `aion` CLI is not
// available on the dev host. ProcessRuntime is exercised in CI
// when AION_E2E=1 (see docs/sprint5/integration-test-plan.md §2).
package aion_test

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/aion"
	"github.com/google/uuid"
)

// ----------------------------------------------------------------------------
// Spec validation
// ----------------------------------------------------------------------------

func TestWorkerSpec_Validate_MissingFields(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(s *aion.WorkerSpec)
		wantErr bool
	}{
		{"all-valid", func(s *aion.WorkerSpec) {}, false},
		{"missing-execution-id", func(s *aion.WorkerSpec) { s.ExecutionID = uuid.Nil }, true},
		{"missing-task-id", func(s *aion.WorkerSpec) { s.TaskID = uuid.Nil }, true},
		{"missing-agent-id", func(s *aion.WorkerSpec) { s.AgentID = uuid.Nil }, true},
		{"missing-project-id", func(s *aion.WorkerSpec) { s.ProjectID = uuid.Nil }, true},
		{"missing-model", func(s *aion.WorkerSpec) { s.Model = "" }, true},
		{"missing-provider", func(s *aion.WorkerSpec) { s.Provider = "" }, true},
		{"missing-permission-mode", func(s *aion.WorkerSpec) { s.PermissionMode = "" }, true},
		{"attempt-zero", func(s *aion.WorkerSpec) { s.Attempt = 0 }, true},
		{"attempt-negative", func(s *aion.WorkerSpec) { s.Attempt = -3 }, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := aion.WorkerSpec{
				ExecutionID:    uuid.New(),
				TaskID:         uuid.New(),
				AgentID:        uuid.New(),
				ProjectID:      uuid.New(),
				Model:          "MiniMax-M3",
				Provider:       "TokenRouter",
				PermissionMode: "YOLO",
				Attempt:        1,
			}
			tt.mutate(&s)
			err := s.Validate()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Validate() err = %v, wantErr = %v", err, tt.wantErr)
			}
			if err != nil && !errors.Is(err, aion.ErrInvalidSpec) {
				t.Fatalf("expected ErrInvalidSpec, got %v", err)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// WorkerStatus.IsTerminal
// ----------------------------------------------------------------------------

func TestWorkerStatus_IsTerminal(t *testing.T) {
	cases := []struct {
		s    aion.WorkerStatus
		want bool
	}{
		{aion.WorkerPending, false},
		{aion.WorkerRunning, false},
		{aion.WorkerCompleted, true},
		{aion.WorkerFailed, true},
		{aion.WorkerCancelled, true},
		{"", false},
	}
	for _, c := range cases {
		t.Run(string(c.s), func(t *testing.T) {
			if got := c.s.IsTerminal(); got != c.want {
				t.Errorf("IsTerminal() = %v, want %v", got, c.want)
			}
		})
	}
}

// ----------------------------------------------------------------------------
// MockRuntime: happy path
// ----------------------------------------------------------------------------

func TestMockRuntime_SpawnAndWait_CompletesImmediately(t *testing.T) {
	rt := aion.NewMockRuntime()
	defer rt.Close()

	spec := aion.WorkerSpec{
		ExecutionID:    uuid.New(),
		TaskID:         uuid.New(),
		AgentID:        uuid.New(),
		ProjectID:      uuid.New(),
		Model:          "MiniMax-M3",
		Provider:       "TokenRouter",
		PermissionMode: "YOLO",
		Attempt:        1,
	}

	handle, err := rt.Spawn(context.Background(), spec)
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	if handle == "" {
		t.Fatal("Spawn returned empty handle")
	}

	res, err := rt.Wait(context.Background(), handle)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if res.Status != aion.WorkerCompleted {
		t.Errorf("Status = %v, want %v", res.Status, aion.WorkerCompleted)
	}
	if res.ExecutionID != spec.ExecutionID {
		t.Errorf("ExecutionID = %v, want %v", res.ExecutionID, spec.ExecutionID)
	}
}

// ----------------------------------------------------------------------------
// MockRuntime: script-driven delay
// ----------------------------------------------------------------------------

func TestMockRuntime_SpawnAndWait_WithDelay(t *testing.T) {
	rt := aion.NewMockRuntime()
	defer rt.Close()

	spec := aion.WorkerSpec{
		ExecutionID:    uuid.New(),
		TaskID:         uuid.New(),
		AgentID:        uuid.New(),
		ProjectID:      uuid.New(),
		Model:          "MiniMax-M3",
		Provider:       "TokenRouter",
		PermissionMode: "YOLO",
		Attempt:        1,
	}
	rt.RegisterScript(spec.ExecutionID.String(), aion.FakeScript{
		Delay:   50 * time.Millisecond,
		Outcome: aion.WorkerCompleted,
		Result:  json.RawMessage(`{"ok":true}`),
	})

	start := time.Now()
	handle, err := rt.Spawn(context.Background(), spec)
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	res, err := rt.Wait(context.Background(), handle)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	elapsed := time.Since(start)

	if res.Status != aion.WorkerCompleted {
		t.Errorf("Status = %v, want completed", res.Status)
	}
	if string(res.Result) != `{"ok":true}` {
		t.Errorf("Result = %s, want {\"ok\":true}", res.Result)
	}
	if elapsed < 50*time.Millisecond {
		t.Errorf("Wait returned in %v, expected >= 50ms", elapsed)
	}
}

// ----------------------------------------------------------------------------
// MockRuntime: failed outcome
// ----------------------------------------------------------------------------

func TestMockRuntime_SpawnAndWait_Failed(t *testing.T) {
	rt := aion.NewMockRuntime()
	defer rt.Close()

	spec := aion.WorkerSpec{
		ExecutionID:    uuid.New(),
		TaskID:         uuid.New(),
		AgentID:        uuid.New(),
		ProjectID:      uuid.New(),
		Model:          "MiniMax-M3",
		Provider:       "TokenRouter",
		PermissionMode: "YOLO",
		Attempt:        1,
	}
	rt.RegisterScript(spec.ExecutionID.String(), aion.FakeScript{
		Outcome:      aion.WorkerFailed,
		ErrorMessage: "boom",
	})

	handle, err := rt.Spawn(context.Background(), spec)
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}
	res, err := rt.Wait(context.Background(), handle)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if res.Status != aion.WorkerFailed {
		t.Errorf("Status = %v, want failed", res.Status)
	}
	if res.ErrorMessage != "boom" {
		t.Errorf("ErrorMessage = %q, want %q", res.ErrorMessage, "boom")
	}
}

// ----------------------------------------------------------------------------
// MockRuntime: cancel
// ----------------------------------------------------------------------------

func TestMockRuntime_Cancel_StopsWorker(t *testing.T) {
	rt := aion.NewMockRuntime()
	defer rt.Close()

	spec := aion.WorkerSpec{
		ExecutionID:    uuid.New(),
		TaskID:         uuid.New(),
		AgentID:        uuid.New(),
		ProjectID:      uuid.New(),
		Model:          "MiniMax-M3",
		Provider:       "TokenRouter",
		PermissionMode: "YOLO",
		Attempt:        1,
	}
	rt.RegisterScript(spec.ExecutionID.String(), aion.FakeScript{
		Delay: 5 * time.Second, // long enough that the timer hasn't fired
	})

	handle, err := rt.Spawn(context.Background(), spec)
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	if err := rt.Cancel(context.Background(), handle); err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	res, err := rt.Wait(context.Background(), handle)
	if err != nil {
		t.Fatalf("Wait: %v", err)
	}
	if res.Status != aion.WorkerCancelled {
		t.Errorf("Status = %v, want cancelled", res.Status)
	}
}

// ----------------------------------------------------------------------------
// MockRuntime: unknown handle
// ----------------------------------------------------------------------------

func TestMockRuntime_Wait_UnknownHandle(t *testing.T) {
	rt := aion.NewMockRuntime()
	defer rt.Close()

	_, err := rt.Wait(context.Background(), aion.WorkerHandle("bogus"))
	if !errors.Is(err, aion.ErrWorkerNotFound) {
		t.Errorf("expected ErrWorkerNotFound, got %v", err)
	}
}

func TestMockRuntime_Cancel_UnknownHandle(t *testing.T) {
	rt := aion.NewMockRuntime()
	defer rt.Close()

	err := rt.Cancel(context.Background(), aion.WorkerHandle("bogus"))
	if !errors.Is(err, aion.ErrWorkerNotFound) {
		t.Errorf("expected ErrWorkerNotFound, got %v", err)
	}
}

// ----------------------------------------------------------------------------
// MockRuntime: context cancellation
// ----------------------------------------------------------------------------

func TestMockRuntime_Wait_ContextCancelled(t *testing.T) {
	rt := aion.NewMockRuntime()
	defer rt.Close()

	spec := aion.WorkerSpec{
		ExecutionID:    uuid.New(),
		TaskID:         uuid.New(),
		AgentID:        uuid.New(),
		ProjectID:      uuid.New(),
		Model:          "MiniMax-M3",
		Provider:       "TokenRouter",
		PermissionMode: "YOLO",
		Attempt:        1,
	}
	rt.RegisterScript(spec.ExecutionID.String(), aion.FakeScript{
		Delay: 5 * time.Second,
	})

	handle, err := rt.Spawn(context.Background(), spec)
	if err != nil {
		t.Fatalf("Spawn: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err = rt.Wait(ctx, handle)
	if !errors.Is(err, aion.ErrWorkerTimeout) {
		t.Errorf("expected ErrWorkerTimeout, got %v", err)
	}
}

// ----------------------------------------------------------------------------
// MockRuntime: closed runtime
// ----------------------------------------------------------------------------

func TestMockRuntime_Close_RejectsNewSpawns(t *testing.T) {
	rt := aion.NewMockRuntime()
	if err := rt.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	spec := aion.WorkerSpec{
		ExecutionID:    uuid.New(),
		TaskID:         uuid.New(),
		AgentID:        uuid.New(),
		ProjectID:      uuid.New(),
		Model:          "MiniMax-M3",
		Provider:       "TokenRouter",
		PermissionMode: "YOLO",
		Attempt:        1,
	}
	_, err := rt.Spawn(context.Background(), spec)
	if !errors.Is(err, aion.ErrRuntimeClosed) {
		t.Errorf("expected ErrRuntimeClosed, got %v", err)
	}
}

// ----------------------------------------------------------------------------
// MockRuntime: concurrent spawns
// ----------------------------------------------------------------------------

func TestMockRuntime_ConcurrentSpawns(t *testing.T) {
	rt := aion.NewMockRuntime()
	defer rt.Close()

	const N = 50
	var wg sync.WaitGroup
	wg.Add(N)
	results := make(chan aion.WorkerResult, N)

	for i := 0; i < N; i++ {
		go func() {
			defer wg.Done()
			spec := aion.WorkerSpec{
				ExecutionID:    uuid.New(),
				TaskID:         uuid.New(),
				AgentID:        uuid.New(),
				ProjectID:      uuid.New(),
				Model:          "MiniMax-M3",
				Provider:       "TokenRouter",
				PermissionMode: "YOLO",
				Attempt:        1,
			}
			rt.RegisterScript(spec.ExecutionID.String(), aion.FakeScript{
				Delay:   time.Duration(i%5) * time.Millisecond,
				Outcome: aion.WorkerCompleted,
			})
			handle, err := rt.Spawn(context.Background(), spec)
			if err != nil {
				t.Errorf("Spawn: %v", err)
				return
			}
			res, err := rt.Wait(context.Background(), handle)
			if err != nil {
				t.Errorf("Wait: %v", err)
				return
			}
			results <- res
		}()
	}
	wg.Wait()
	close(results)

	count := 0
	for r := range results {
		count++
		if r.Status != aion.WorkerCompleted {
			t.Errorf("Status = %v, want completed", r.Status)
		}
	}
	if count != N {
		t.Errorf("got %d results, want %d", count, N)
	}
}
