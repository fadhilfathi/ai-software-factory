# Agent Communication Schema (TASK-504)

**Status:** Sprint 5 deliverable
**Owner:** Developer-02
**Date:** 2026-06-14
**Brief:** `docs/sprint5/brief.md` §6.4

This document is the canonical reference for the JSON-over-stdio
protocol between the API server and an Aion worker subprocess. The
protocol is the contract that lets us swap worker implementations
(real subprocess, mock in-process, future SDK) without changing the
API surface. It is intentionally small.

## 1. Overview

The Aion runtime spawns a worker as a child process (see
`internal/aion/process.go`). The worker emits a stream of
JSON-formatted **frames** on **stdout**; one frame per line. The API
reads stdout line-by-line, JSON-decodes each frame into an
`aion.Message`, and dispatches based on `Type`.

The spec identity (which execution, task, agent, project, model,
provider, permission mode, attempt number) is conveyed to the worker
via **argv** at spawn time, not via Message frames. The Message
channel carries **state** (started, progress, result, error,
cancelled) — not the spec.

**Direction in (worker → API):** the Message stream on worker stdout.
This is the only direction the worker speaks.

**Direction out (API → worker):** currently argv only. The worker
**does not read stdin** in Sprint 5. Stdin is reserved for future use
(see §9 below).

The brief is explicit about the body shape:

> agent_messages.body = plain text + structured fields (NOT markdown)

So `body` carries a small JSON object with primitive fields (strings,
numbers, booleans, nested objects of those types) — never a markdown
blob, never HTML, never a binary payload.

## 2. Transport

* **Line-delimited JSON** on the worker's stdout. Each line is a
  single JSON object terminated by `\n`. No embedded newlines inside
  a frame (the writer must serialize compactly with no pretty
  printing).
* **Encoding:** UTF-8. No BOM.
* **Frame ordering:** frames arrive in emit order. The reader does
  not reorder. Concurrent writers must serialize.
* **Backpressure:** none. The API uses a buffered scanner
  (`bufio.Scanner` with a 1 MiB line max). The worker must not emit
  frames larger than 1 MiB; larger outputs go in `body` as a
  reference (e.g., an S3 URI) and the API fetches the full payload
  separately.
* **Termination:** the worker MUST emit exactly one terminal frame
  (`result`, `error`, or `cancelled`) before exit. The API treats an
  abrupt exit with no terminal frame as a crash.

## 3. Envelope reference

```go
// internal/aion/runtime.go
type Message struct {
    Type        string          `json:"type"`         // "started" | "progress" | "result" | "error" | "cancelled"
    ExecutionID uuid.UUID       `json:"execution_id"`
    Body        json.RawMessage `json:"body,omitempty"`
    Error       string          `json:"error,omitempty"`
    At          time.Time       `json:"at"`
}
```

| Field        | Type                | Required | Notes |
|--------------|---------------------|----------|-------|
| `type`       | string              | yes      | One of 5 known values (§4). Unknown values are logged and discarded. |
| `execution_id` | string (UUID)     | yes      | Must match the `--execution-id` argv the worker was spawned with. Mismatches are logged. |
| `body`       | object              | no       | Omitted for `started` / `cancelled`. Required for `result`. Optional for `progress` / `error`. |
| `error`      | string              | no       | Present iff `type == "error"`. Plain text, not structured. |
| `at`         | string (RFC 3339)   | yes      | Wall-clock time the worker emitted the frame. Server-side clock skew is tolerated up to 60s. |

The Go `time.Time` field marshals as RFC 3339 nanoseconds in UTC. The
worker should use `time.Now().UTC()` at emit time.

## 4. Frame type catalogue

### 4.1 `started`

The worker has booted, parsed argv, and is ready to run.

```json
{
  "type": "started",
  "execution_id": "019ec21c-d49b-7510-ba27-a0967f7fb2a4",
  "at": "2026-06-14T01:30:00.123456789Z"
}
```

* `body` MUST be omitted.
* The API logs this frame and does not surface it to the user. It
  serves as a liveness signal: if the worker doesn't emit `started`
  within 30s of spawn, the API times out the execution.

### 4.2 `progress` (Sprint 6+)

Partial output during execution. Reserved for TASK-506 (live
dashboard). Sprint 5 workers MUST NOT emit this type — the API
ignores it.

```json
{
  "type": "progress",
  "execution_id": "019ec21c-d49b-7510-ba27-a0967f7fb2a4",
  "body": {
    "stage": "running_tests",
    "percent": 42,
    "message": "running test 21 of 50"
  },
  "at": "2026-06-14T01:30:05.000000000Z"
}
```

* `body` is required and contains the three optional fields `stage`
  (string), `percent` (integer 0-100), `message` (string).
* The API may emit these to the bus as a `ProgressEvent` (TBD in
  Sprint 6). For Sprint 5, the dashboard uses 3s polling and ignores
  progress frames.

### 4.3 `result` (terminal)

The worker succeeded. The API records `Status = completed` and
surfaces `body` as the deliverable.

```json
{
  "type": "result",
  "execution_id": "019ec21c-d49b-7510-ba27-a0967f7fb2a4",
  "body": {
    "output": "all tests passed",
    "artifacts": [
      { "kind": "log", "uri": "s3://bucket/logs/exec-1234.log" },
      { "kind": "patch", "uri": "s3://bucket/patches/exec-1234.diff" }
    ],
    "stats": { "tokens": 12345, "duration_ms": 8200 }
  },
  "at": "2026-06-14T01:30:10.000000000Z"
}
```

* `body` is required. The shape is opaque to the API — the
  deliverable service (TASK-505) interprets it. Common fields are
  `output` (string), `artifacts` (array of `{kind, uri}`), `stats`
  (object).
* This frame drives the `running → completed` transition in the
  state machine (TASK-503).

### 4.4 `error` (terminal)

The worker failed. The API records `Status = failed` and surfaces
`error` as the execution's `ErrorMessage`.

```json
{
  "type": "error",
  "execution_id": "019ec21c-d49b-7510-ba27-a0967f7fb2a4",
  "error": "capability mismatch: agent does not declare 'python>=3.12'",
  "body": {
    "error_code": "CAPABILITY_MISMATCH",
    "details": { "missing": ["python>=3.12"], "agent_declares": ["python>=3.11"] }
  },
  "at": "2026-06-14T01:30:08.000000000Z"
}
```

* `error` is the human-readable message; goes into
  `executions.error_message`.
* `body.error_code` is a structured code for TASK-508 recovery to
  match on. See §7 for the full list.
* This frame drives the `running → failed` transition (or
  `assigned → failed`, `review → failed`).

### 4.5 `cancelled` (terminal)

The worker received a cancel signal and shut down gracefully. The
API records `Status = failed` with `error_code = "CANCELLED"`.

```json
{
  "type": "cancelled",
  "execution_id": "019ec21c-d49b-7510-ba27-a0967f7fb2a4",
  "at": "2026-06-14T01:30:07.000000000Z"
}
```

* `body` and `error` MUST be omitted.
* The runtime does NOT send a cancel Message to the worker. The
  worker receives SIGTERM and emits `cancelled` on graceful exit
  (or the runtime kills the process if it doesn't exit within 5s).
  See §9.

## 5. Lifecycle state diagram

```
        spawn
          │
          ▼
   ┌─────────────┐
   │  starting   │  ←── argv parsed
   └──────┬──────┘
          │  emit: started
          ▼
   ┌─────────────┐
   │  running    │  ←── may emit 0..N progress frames
   └──────┬──────┘
          │  emit: result | error | cancelled
          ▼
   ┌─────────────┐
   │  terminal   │  ←── worker exits
   └─────────────┘
```

The runtime (API side) maps this lifecycle to the state machine
in `internal/service/execution.go` (TASK-503):

* `started` → no transition (logged only)
* `progress` → no transition (Sprint 5; Sprint 6+ may use for live updates)
* `result` → `running → completed` (or `assigned → running → completed` if started late)
* `error` → `running → failed` (or `assigned → failed` / `review → failed`)
* `cancelled` → `assigned|running|review → failed` with `error_code = "CANCELLED"`

## 6. Argv contract

The runtime spawns the worker with the following argv. The worker
MUST parse all of these before emitting `started`.

| Position | Flag                 | Type   | Description |
|----------|----------------------|--------|-------------|
| 1        | `--execution-id`     | UUID   | The execution row's primary key |
| 2        | `--task-id`          | UUID   | Parent task ID |
| 3        | `--agent-id`         | UUID   | Agent (worker type) ID |
| 4        | `--project-id`       | UUID   | Tenant scoping |
| 5        | `--model`            | string | LLM model identifier (e.g. `gpt-4o`) |
| 6        | `--provider`         | string | Provider name (e.g. `openai`, `anthropic`) |
| 7        | `--permission-mode`  | string | `read-only` \| `sandbox-write` \| `full` |
| 8        | `--attempt`          | int    | Retry attempt (1 = first, 2 = first retry, ...) |

Reserved for future use (Sprint 6+): the worker MUST accept and
ignore `--stdin-mode=pipe` (the default will become pipe rather
than null). Today the API spawns the worker with `/dev/null` as
stdin.

## 7. Error codes

The `error` and `cancelled` terminal frames surface a structured
`error_code` (in `body.error_code` for `error` frames; implicit
`CANCELLED` for `cancelled` frames). The full catalogue:

| Code                  | Meaning | Recovery action (TASK-508) |
|-----------------------|---------|-----------------------------|
| `CAPABILITY_MISMATCH` | Agent doesn't have required capabilities | Escalate to a different agent type (no retry on the same agent) |
| `TIMEOUT`             | Worker exceeded its time budget | Retry with longer timeout |
| `RUNTIME_CRASH`       | Worker exited without a terminal frame | Retry (max 1 per TASK-508 §6.8) |
| `CANCELLED`           | User/system cancel | No retry |
| `BUDGET_EXCEEDED`     | Token or cost limit hit | Retry with smaller scope or escalate |
| `AUTH_FAILURE`        | Provider rejected credentials | No retry; surface to user |
| `INTERNAL_ERROR`      | Catch-all | Retry once |

The code is opaque to the worker; the worker picks the most
specific code that fits. If none fit, use `INTERNAL_ERROR` and put
the detail in `error`.

## 8. Validation rules

The API validates each frame as it arrives:

1. `type` is one of the 5 known values. Unknown → log a warning, discard the frame (don't crash the reader).
2. `execution_id` parses as a UUID. Mismatch with the argv's `--execution-id` → log a warning, but still process the frame (the worker may have a stale config).
3. `at` parses as RFC 3339. Failure → log a warning, use server time instead.
4. `body` for `result` MUST be present. Missing → treat as `INTERNAL_ERROR` with a synthesized `error` message.
5. The first non-`started` frame MUST be a terminal frame. If the worker emits `progress` first (against protocol), the API logs and continues (Sprint 6+ behavior).
6. After a terminal frame, the reader MUST close and the worker MUST exit. A second terminal frame is a protocol violation; log and discard.

## 9. Future work (Sprint 6+)

* **Stdin for mid-execution instructions.** Sprint 6+ may add a
  direction-out Message stream on worker stdin for runtime → worker
  commands (pause, resume, redirect). The Message envelope will gain
  a `type == "command"` variant.
* **Progress wire-up to the bus.** When TASK-506 needs real-time
  updates (no polling), `progress` frames will be published to
  `events.Bus` as a new event kind.
* **Heartbeat frames.** A `type == "heartbeat"` frame every 10s
  could replace the 30s `started` timeout with a rolling liveness
  check.
* **Worker-side structured logging.** A `type == "log"` frame could
  carry structured logs from the worker, routed to the API's
  `zap.Logger`. Today the worker logs to stderr which is captured
  separately.
* **Protocol versioning.** A `version` field on the envelope will
  be added the first time we need a v2. Today the protocol is v1 by
  convention; the version is implicit.

## 10. Examples

### 10.1 Happy path

Worker side (stdout):
```json
{"type":"started","execution_id":"019ec21c-d49b-7510-ba27-a0967f7fb2a4","at":"2026-06-14T01:30:00.123Z"}
{"type":"progress","execution_id":"019ec21c-d49b-7510-ba27-a0967f7fb2a4","body":{"stage":"running_tests","percent":42},"at":"2026-06-14T01:30:05.000Z"}
{"type":"result","execution_id":"019ec21c-d49b-7510-ba27-a0967f7fb2a4","body":{"output":"all tests passed","stats":{"tokens":12345,"duration_ms":8200}},"at":"2026-06-14T01:30:10.000Z"}
```

API side state transitions:
* `pending` (initial) → no transition on `started` (logged)
* `pending → running` on first `progress` (Sprint 6+; Sprint 5 does this on `started`)
* `running → completed` on `result`

### 10.2 Capability mismatch

```json
{"type":"error","execution_id":"019ec21c-d49b-7510-ba27-a0967f7fb2a4","error":"capability mismatch: agent does not declare 'python>=3.12'","body":{"error_code":"CAPABILITY_MISMATCH","details":{"missing":["python>=3.12"]}},"at":"2026-06-14T01:30:08.000Z"}
```

API side: `running → failed`, `ErrorMessage` set, `error_code = CAPABILITY_MISMATCH`. TASK-508 recovery sees the code and escalates.

### 10.3 Cancellation

API: sends SIGTERM to worker PID.

Worker (graceful):
```json
{"type":"cancelled","execution_id":"019ec21c-d49b-7510-ba27-a0967f7fb2a4","at":"2026-06-14T01:30:07.000Z"}
```

API side: `running → failed`, `error_code = CANCELLED`. No retry.

### 10.4 Crash (no terminal frame)

Worker dies from panic / OOM / SIGKILL.

API side: `pumpStdout` sees EOF before a terminal frame. The API
maps this to `INTERNAL_ERROR` with the synthesized message
`"worker exited without terminal frame"`. State machine: `running
→ failed`. TASK-508 retries once.
