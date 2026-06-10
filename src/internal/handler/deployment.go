package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/example/project/internal/model"
	"github.com/example/project/internal/service"
)

// DeploymentHandler handles deployment lifecycle endpoints.
type DeploymentHandler struct {
	svc *service.DeploymentService
}

func NewDeploymentHandler(svc *service.DeploymentService) *DeploymentHandler {
	return &DeploymentHandler{svc: svc}
}

type triggerDeploymentRequest struct {
	ProjectID   string `json:"project_id"`
	Environment string `json:"environment"`
	Branch      string `json:"branch"`
}

type deploymentResponse struct {
	ID            string           `json:"id"`
	Status        string           `json:"status"`
	Environment   string           `json:"environment"`
	URL           string           `json:"url,omitempty"`
	EstimatedTime int              `json:"estimated_time,omitempty"`
	StartedAt     string           `json:"started_at,omitempty"`
	CompletedAt   string           `json:"completed_at,omitempty"`
	Steps         []deploymentStep `json:"steps,omitempty"`
	RollbackFrom  string           `json:"rollback_from,omitempty"`
	RollbackTo    string           `json:"rollback_to,omitempty"`
}

type deploymentStep struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Duration int    `json:"duration"`
}

// Trigger handles POST /deployments.
func (h *DeploymentHandler) Trigger(w http.ResponseWriter, r *http.Request) {
	var req triggerDeploymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "INVALID_JSON", "Malformed request body")
		return
	}

	deployment, svcErr := h.svc.TriggerDeployment(service.TriggerDeploymentRequest{
		ProjectID:   req.ProjectID,
		Environment: req.Environment,
		Branch:      req.Branch,
	})
	if svcErr != nil {
		writeServiceError(w, svcErr)
		return
	}

	writeJSON(w, http.StatusAccepted, toDeploymentResponse(deployment))
}

// GetStatus handles GET /deployments/{id}.
func (h *DeploymentHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Deployment ID is required")
		return
	}

	deployment, svcErr := h.svc.GetDeployment(id)
	if svcErr != nil {
		writeServiceError(w, svcErr)
		return
	}

	writeJSON(w, http.StatusOK, toDeploymentResponse(deployment))
}

// Rollback handles POST /deployments/{id}/rollback.
func (h *DeploymentHandler) Rollback(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Deployment ID is required")
		return
	}

	deployment, svcErr := h.svc.RollbackDeployment(id)
	if svcErr != nil {
		writeServiceError(w, svcErr)
		return
	}

	writeJSON(w, http.StatusOK, toDeploymentResponse(deployment))
}

func toDeploymentResponse(d *model.Deployment) deploymentResponse {
	resp := deploymentResponse{
		ID:            d.ID,
		Status:        string(d.Status),
		Environment:   string(d.Environment),
		URL:           d.URL,
		EstimatedTime: d.EstimatedTime,
		RollbackFrom:  d.RollbackFrom,
		RollbackTo:    d.RollbackTo,
	}
	if d.StartedAt != nil {
		resp.StartedAt = d.StartedAt.Format(time.RFC3339)
	}
	if d.CompletedAt != nil {
		resp.CompletedAt = d.CompletedAt.Format(time.RFC3339)
	}
	steps := make([]deploymentStep, len(d.Steps))
	for i, s := range d.Steps {
		steps[i] = deploymentStep{Name: s.Name, Status: s.Status, Duration: s.Duration}
	}
	resp.Steps = steps
	return resp
}
