package service

import (
	"time"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"github.com/fadhilfathi/AI-Software-Factory/internal/validation"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// DeploymentService handles deployment operations.
type DeploymentService struct {
	store store.Store
	log   *zap.Logger
}

func NewDeploymentService(s store.Store, log *zap.Logger) *DeploymentService {
	return &DeploymentService{store: s, log: log}
}

// TriggerDeploymentRequest carries deployment trigger input.
type TriggerDeploymentRequest struct {
	ProjectID   string
	Environment string
	Branch      string
}

// TriggerDeployment creates a new deployment request.
func (s *DeploymentService) TriggerDeployment(req TriggerDeploymentRequest) (*model.Deployment, *Error) {
	var errs validation.Errors
	validation.NotEmpty(req.ProjectID, "project_id", "Project ID", &errs)
	validation.NotEmpty(req.Environment, "environment", "Environment", &errs)
	validation.AllowedStrings(req.Environment, validDeploymentEnvironments, "environment", "Environment", &errs)
	if errs.HasErrors() {
		return nil, validationError(errs)
	}

	// Verify project exists
	if _, err := s.store.Projects().GetByID(uuid.MustParse(req.ProjectID)); err != nil {
		return nil, notFound("Project not found")
	}

	now := time.Now().UTC()
	deployment := &model.Deployment{
		ID:            uuid.New().String(),
		ProjectID:     req.ProjectID,
		Environment:   model.Environment(req.Environment),
		Branch:        req.Branch,
		Status:        model.DeployQueued,
		EstimatedTime: 600,
		Steps: []model.DeploymentStep{
			{Name: "build", Status: "pending", Duration: 0},
			{Name: "test", Status: "pending", Duration: 0},
			{Name: "deploy", Status: "pending", Duration: 0},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err := s.store.Deployments().Create(deployment); err != nil {
		s.log.Error("failed to create deployment", zap.Error(err))
		return nil, internalError("Failed to create deployment")
	}

	return deployment, nil
}

// GetDeployment returns a deployment by ID.
func (s *DeploymentService) GetDeployment(id string) (*model.Deployment, *Error) {
	deployment, err := s.store.Deployments().GetByID(id)
	if err != nil {
		return nil, notFound("Deployment not found")
	}
	return deployment, nil
}

// RollbackDeployment creates a rollback deployment from a previous one.
func (s *DeploymentService) RollbackDeployment(id string) (*model.Deployment, *Error) {
	existing, err := s.store.Deployments().GetByID(id)
	if err != nil {
		return nil, notFound("Deployment not found")
	}

	now := time.Now().UTC()
	rollback := &model.Deployment{
		ID:            uuid.New().String(),
		ProjectID:     existing.ProjectID,
		Environment:   existing.Environment,
		Status:        model.DeployRollingBack,
		RollbackFrom:  id,
		RollbackTo:    existing.ID,
		EstimatedTime: 600,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	if err := s.store.Deployments().Create(rollback); err != nil {
		s.log.Error("failed to create rollback", zap.Error(err))
		return nil, internalError("Failed to create rollback deployment")
	}

	return rollback, nil
}
