# Code Service & Git Integration Logic (TASK-301)

> **Document Version**: 1.1  
> **Status**: Finalized  
> **Owner**: TechLead

---

## 1. Handler Specifications

The `CodeHandler` will implement the following REST endpoints to provide agents and the frontend with full control over the project codebase.

### 1.1 `POST /v1/code/generate`
- **Purpose**: Queues a code generation request.
- **Request Body**:
  ```json
  {
    "project_id": "uuid",
    "task_id": "uuid",
    "specification": "string",
    "files": ["string"]
  }
  ```
- **Internal Logic**:
  1. Validate project and task exist.
  2. Create a `CodeGenRequest` in `queued` status.
  3. Emit a `code.gen.requested` event to the message bus (NATS/Redis).
- **Response**: `202 Accepted` with `CodeGenRequest` summary.

### 1.2 `GET /v1/code/:projectId/files`
- **Purpose**: Lists all files in the project's current working tree.
- **Query Params**: `path` (prefix search), `ext` (extension filter).
- **Internal Logic**:
  1. Fetch file metadata from `project_files` table.
- **Response**: `200 OK` with `[]ProjectFileMetadata`.

### 1.3 `GET /v1/code/:projectId/files/*path`
- **Purpose**: Retrieves content and metadata for a specific file.
- **Internal Logic**:
  1. Fetch from `project_files` where `project_id` and `path` match.
- **Response**: `200 OK` with `ProjectFile` (including `content`).

### 1.4 `POST /v1/code/:projectId/commits`
- **Purpose**: Persists a set of changes as an immutable commit.
- **Request Body**:
  ```json
  {
    "branch": "string",
    "message": "string",
    "files": [{ "path": "string", "content": "string" }]
  }
  ```
- **Internal Logic**:
  1. Generate short SHA.
  2. Insert into `commits` table.
  3. Upsert into `project_files` table (updates working tree).
- **Response**: `201 Created` with `Commit` details.

### 1.5 `GET /v1/code/:projectId/diff`
- **Purpose**: Computes a diff between two states.
- **Query Params**: `from` (SHA), `to` (SHA or "working").
- **Internal Logic**:
  1. If `to == "working"`, compare `from` commit files vs `project_files` table.
  2. Else, compare files between two entries in `commits` table.
- **Response**: `200 OK` with `UnifiedDiff` string.

---

## 2. Git Integration Logic

The platform uses a **Virtual Git Layer** for the MVP, with plans to transition to a real Git backend (Gitea/GitHub) in Sprint 5.

### 2.1 Virtual Git Strategy
- **Commits**: Each commit is a snapshot of changed files stored in the `commits` table.
- **Working Tree**: The `project_files` table represents the "HEAD" of the main branch.
- **Branching**: Simulating branches by filtering `project_files` or `commits` by a `branch` column.

### 2.2 Filesystem Sync
To support the **Execution Sandbox**, the Code Service must be able to "checkout" the working tree to a physical directory:
1. Create temp directory `/tmp/workspaces/:projectId`.
2. Iterate `project_files` and write content to disk.
3. Sandbox mounts this directory as a read-only or read-write volume.
4. After sandbox execution, changed files are read back and synced to the database.

### 2.3 Language Detection
The service automatically detects the language based on file extension to assist in syntax highlighting and sandbox environment selection:
- `.go` -> `go`
- `.ts/.tsx` -> `typescript`
- `.py` -> `python`
- `.rs` -> `rust`

---

## 3. Implementation Plan

1. **Phase 1**: Update `CodeService` and `CodeHandler` to support `ListFiles` and `GetDiff`.
2. **Phase 2**: Implement the Filesystem Sync logic for Sandbox integration.
3. **Phase 3**: Migrate `project_files` and `commits` to PostgreSQL (DataEngineer task).
4. **Phase 4**: Add `analysis` endpoint for basic linting/complexity checks.
