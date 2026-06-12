package service

import (
	"context"

	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/google/uuid"
)

// GateResult represents the outcome of a single quality gate.
type GateResult struct {
	Success bool
	Score   float64
	Issues  []model.ReviewIssue
	Metrics map[string]interface{}
}

// GateRunner defines the interface for running a quality or security gate.
type GateRunner interface {
	Name() string
	Run(ctx context.Context, projectID uuid.UUID, commitSHA string) (*GateResult, error)
}

// LinterGate runs static analysis for style and common errors.
type LinterGate struct {
	orchestrator AgentOrchestrator
}

func (g *LinterGate) Name() string { return "Linter" }

func (g *LinterGate) Run(ctx context.Context, projectID uuid.UUID, commitSHA string) (*GateResult, error) {
	// TODO: Implement actual linter execution via orchestrator
	return &GateResult{Success: true, Score: 100}, nil
}

// SASTGate runs security vulnerability scanning.
type SASTGate struct {
	orchestrator AgentOrchestrator
}

func (g *SASTGate) Name() string { return "SAST" }

func (g *SASTGate) Run(ctx context.Context, projectID uuid.UUID, commitSHA string) (*GateResult, error) {
	// TODO: Implement actual SAST execution via orchestrator
	return &GateResult{Success: true, Score: 100}, nil
}

// ComplexityGate calculates cyclomatic complexity metrics.
type ComplexityGate struct {
	orchestrator AgentOrchestrator
}

func (g *ComplexityGate) Name() string { return "Complexity" }

func (g *ComplexityGate) Run(ctx context.Context, projectID uuid.UUID, commitSHA string) (*GateResult, error) {
	// TODO: Implement actual complexity calculation
	return &GateResult{Success: true, Score: 100}, nil
}
