package service

import (
	"context"
	"errors"
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/fadhilfathi/AI-Software-Factory/internal/validation"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// DeliverableService is the Sprint 4 (TASK-406) implementation
// of the deliverable subsystem. Two-table design (data-model.md
// §6):
//
//   * `deliverables`        — the *current* state of each
//     deliverable (current title, current content, current
//     version, current updated_at). One row per deliverable.
//
//   * `deliverable_versions`— the *immutable history* of every
//     title/content the deliverable ever had, keyed by
//     (deliverable_id, version). One row per version.
//
// The append-only invariant is enforced by the service layer:
// CreateDeliverable and UpdateDeliverable both write to BOTH
// tables in a single transaction via s.store.WithTx. The store
// layer is intentionally narrow (the store does not do both
// writes internally); the service composes the two writes.
//
// Handler layer uses a consumer-side interface in the handler
// package (matches the AssignmentHandler + ExecutionHandler
// patterns approved by the Lead).
type DeliverableService struct {
	store store.Store
	log   *zap.Logger
}

func NewDeliverableService(s store.Store, log *zap.Logger) *DeliverableService {
	return &DeliverableService{store: s, log: log}
}

// CreateDeliverableRequest is the input to CreateDeliverable.
// Title is validated for non-emptiness; the rest is validated
// by the service against the task/agent existence check.
type CreateDeliverableRequest struct {
	TaskID   uuid.UUID
	AgentID  uuid.UUID
	Title    string
	Content  string
	CreatedBy *uuid.UUID // optional; from the JWT user_id (nil for system-driven creates)
}

// UpdateDeliverableRequest is the input to UpdateDeliverable.
// The service computes the next version from the current row
// (no client-supplied version); the updatedBy pointer carries
// the JWT user_id for the new deliverable_versions row's
// created_by.
type UpdateDeliverableRequest struct {
	Title     string
	Content   string
	UpdatedBy *uuid.UUID
}

// CreateDeliverable validates the task and agent exist (404 on
// miss), then in a single transaction writes both the main
// `deliverables` row (version=1) and the matching
// `deliverable_versions` row. Returns the created Deliverable.
func (s *DeliverableService) CreateDeliverable(ctx context.Context, req CreateDeliverableRequest) (*model.Deliverable, *Error) {
	var errs validation.Errors
	validation.NotEmpty(req.Title, "title", "Title", &errs)
	if errs.HasErrors() {
		return nil, validationError(errs)
	}

	// F-023 DoS hardening: cap the markdown content size at
	// 1 MiB. The handler also wraps the request body in
	// http.MaxBytesReader with a small headroom over this
	// value, so this check is the second line of defence (for
	// cases where the body snuck in via a path that bypassed
	// the handler, e.g. a future internal call). It runs
	// before any DB I/O.
	if int64(len(req.Content)) > model.MaxDeliverableContentBytes {
		return nil, payloadTooLarge(
			"Deliverable content exceeds maximum allowed size",
			model.MaxDeliverableContentBytes,
		)
	}

	if _, err := s.store.Tasks().GetByID(req.TaskID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, notFound("Task not found")
		}
		s.log.Error("failed to lookup task for create", zap.Error(err))
		return nil, internalError("Failed to validate task")
	}
	if _, err := s.store.Agents().GetByID(ctx, req.AgentID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, notFound("Agent not found")
		}
		s.log.Error("failed to lookup agent for create", zap.Error(err))
		return nil, internalError("Failed to validate agent")
	}

	now := time.Now().UTC()
	d := &model.Deliverable{
		ID:        uuid.New(),
		TaskID:    req.TaskID,
		AgentID:   req.AgentID,
		Title:     req.Title,
		Content:   req.Content,
		Version:   1,
		CreatedAt: now,
		UpdatedAt: now,
	}
	v := &model.DeliverableVersion{
		ID:           uuid.New(),
		DeliverableID: d.ID,
		Version:      1,
		Title:        req.Title,
		Content:      req.Content,
		CreatedAt:    now,
		CreatedBy:    req.CreatedBy,
	}

	// Atomic write: main row + first version row in one tx.
	err := s.store.WithTx(ctx, func(tx store.Tx) error {
		if err := tx.Deliverables().Create(ctx, d); err != nil {
			return err
		}
		if err := tx.DeliverableVersions().Insert(ctx, v); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, store.ErrAlreadyExists) {
			return nil, conflict("Deliverable already exists (id collision; retry)")
		}
		s.log.Error("failed to create deliverable", zap.Error(err))
		return nil, internalError("Failed to create deliverable")
	}

	return d, nil
}

// GetDeliverable returns the deliverable by id. 404 on miss.
func (s *DeliverableService) GetDeliverable(ctx context.Context, id uuid.UUID) (*model.Deliverable, *Error) {
	d, err := s.store.Deliverables().GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, notFound("Deliverable not found")
		}
		s.log.Error("failed to get deliverable", zap.Error(err))
		return nil, internalError("Failed to get deliverable")
	}
	return d, nil
}

// ListDeliverables returns a keyset-paginated page of
// deliverables matching the filter. The brief requires at
// least one of TaskID or AgentID to be set; if both are uuid.Nil
// we 400 with a validation error. The store handles the
// pagination mechanics (default 50, max 200).
func (s *DeliverableService) ListDeliverables(ctx context.Context, filter model.DeliverableFilter) (*model.DeliverableListResult, *Error) {
	if filter.TaskID == uuid.Nil && filter.AgentID == uuid.Nil {
		return nil, validationSingle("filter", "Provide task_id or agent_id to filter")
	}
	result, err := s.store.Deliverables().List(ctx, filter)
	if err != nil {
		s.log.Error("failed to list deliverables", zap.Error(err))
		return nil, internalError("Failed to list deliverables")
	}
	return result, nil
}

// UpdateDeliverable is the APPEND-ONLY path. In a single
// transaction:
//   1. Read the current `deliverables` row (404 if missing).
//   2. Compute next version = current.Version + 1.
//   3. Insert a new `deliverable_versions` row.
//   4. Update the main `deliverables` row (title, content,
//      version, updated_at).
// Returns the updated Deliverable. A duplicate
// (deliverable_id, version) on the version insert is mapped
// to 409 (the in-memory store returns ErrAlreadyExists; the
// postgres store maps pg 23505 to ErrAlreadyExists).
func (s *DeliverableService) UpdateDeliverable(ctx context.Context, id uuid.UUID, req UpdateDeliverableRequest) (*model.Deliverable, *Error) {
	var errs validation.Errors
	validation.NotEmpty(req.Title, "title", "Title", &errs)
	if errs.HasErrors() {
		return nil, validationError(errs)
	}

	// F-023 DoS hardening: same cap as CreateDeliverable.
	// Runs before the deliverable lookup so a 10 MiB body
	// does not even reach the DB read.
	if int64(len(req.Content)) > model.MaxDeliverableContentBytes {
		return nil, payloadTooLarge(
			"Deliverable content exceeds maximum allowed size",
			model.MaxDeliverableContentBytes,
		)
	}

	current, err := s.store.Deliverables().GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, notFound("Deliverable not found")
		}
		s.log.Error("failed to read deliverable for update", zap.Error(err))
		return nil, internalError("Failed to read deliverable")
	}

	now := time.Now().UTC()
	nextVersion := current.Version + 1

	// Updated main-row state. We start from the current row
	// and overwrite title/content/version/updated_at; the
	// service is the source of truth for what changes.
	updated := &model.Deliverable{
		ID:        current.ID,
		TaskID:    current.TaskID,
		AgentID:   current.AgentID,
		Title:     req.Title,
		Content:   req.Content,
		Version:   nextVersion,
		CreatedAt: current.CreatedAt,
		UpdatedAt: now,
	}
	// New history row.
	v := &model.DeliverableVersion{
		ID:           uuid.New(),
		DeliverableID: current.ID,
		Version:      nextVersion,
		Title:        req.Title,
		Content:      req.Content,
		CreatedAt:    now,
		CreatedBy:    req.UpdatedBy,
	}

	err = s.store.WithTx(ctx, func(tx store.Tx) error {
		if err := tx.DeliverableVersions().Insert(ctx, v); err != nil {
			return err
		}
		if err := tx.Deliverables().Update(ctx, updated); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, store.ErrAlreadyExists) {
			// Duplicate (deliverable_id, version) — concurrent
			// PUTs tried to write the same version. 409.
			return nil, conflict("Duplicate version; another update is in progress")
		}
		if errors.Is(err, store.ErrNotFound) {
			// Main row disappeared between the read and the
			// update. Race; treat as 404.
			return nil, notFound("Deliverable not found")
		}
		s.log.Error("failed to update deliverable", zap.Error(err))
		return nil, internalError("Failed to update deliverable")
	}

	return updated, nil
}

// ListDeliverableVersions returns the immutable history of a
// deliverable, ordered by version DESC. 404 if the deliverable
// itself doesn't exist. The store does not enforce the
// deliverable existence — the service does the existence
// check (cheaper than a join).
func (s *DeliverableService) ListDeliverableVersions(ctx context.Context, deliverableID uuid.UUID) ([]*model.DeliverableVersion, *Error) {
	if _, err := s.store.Deliverables().GetByID(ctx, deliverableID); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, notFound("Deliverable not found")
		}
		s.log.Error("failed to lookup deliverable for list versions", zap.Error(err))
		return nil, internalError("Failed to list versions")
	}
	versions, err := s.store.DeliverableVersions().ListVersions(ctx, deliverableID)
	if err != nil {
		s.log.Error("failed to list deliverable versions", zap.Error(err))
		return nil, internalError("Failed to list versions")
	}
	return versions, nil
}
