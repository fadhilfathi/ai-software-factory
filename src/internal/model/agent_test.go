package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAgentStatusConstants asserts that the AgentStatus enum values are
// equal to their canonical string forms. Covers only the 6 production
// lifecycle states that are in AllAgentStatuses and validated by the
// DB CHECK constraint on agents.status. The additional test-visible
// status constants (AgentSpawning, AgentWorking, AgentCompleted,
// AgentFailed) declared in agent.go are intentionally NOT asserted
// here — they are non-canonical helpers, kept for call-site readability
// but not part of the lifecycle state set. See the comment on those
// constants in agent.go for why they are kept out of AllAgentStatuses.
func TestAgentStatusConstants(t *testing.T) {
	assert.Equal(t, AgentStatus("initializing"), AgentInitializing)
	assert.Equal(t, AgentStatus("idle"), AgentIdle)
	assert.Equal(t, AgentStatus("busy"), AgentBusy)
	assert.Equal(t, AgentStatus("paused"), AgentPaused)
	assert.Equal(t, AgentStatus("error"), AgentError)
	assert.Equal(t, AgentStatus("retired"), AgentRetired)
}

// TestAgentStructFields exercises the real Agent struct fields. The
// real struct uses Role (free-form string) rather than the parallel
// "Type" enum on the test side, and stores Capability as a []string
// of catalog names rather than as a typed enum slice. ProjectID is a
// uuid.UUID — see migrations 008 and 016 for the column type.
func TestAgentStructFields(t *testing.T) {
	projectID := uuid.New()
	now := time.Now()
	agent := Agent{
		ID:          uuid.New(),
		ProjectID:   projectID,
		Name:        "Agent Smith",
		Role:        "developer",
		Status:      AgentIdle,
		Capabilities: []string{"code_implementation"},
		Metadata:    json.RawMessage(`{"team": "core"}`),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	assert.NotEqual(t, uuid.Nil, agent.ID)
	assert.Equal(t, projectID, agent.ProjectID)
	assert.Equal(t, "Agent Smith", agent.Name)
	assert.Equal(t, "developer", agent.Role)
	assert.Equal(t, AgentIdle, agent.Status)
	assert.Equal(t, []string{"code_implementation"}, agent.Capabilities)
	assert.Equal(t, json.RawMessage(`{"team": "core"}`), agent.Metadata)
}

// TestAgentRunStatusConstants asserts the AgentRunStatus enum values
// match their canonical string forms. The real Execution model uses
// ExecutionStatus (see execution.go), not AgentRunStatus; the latter
// is a parallel enum kept for run-level lifecycle reporting in the
// agent activity dashboard.
func TestAgentRunStatusConstants(t *testing.T) {
	assert.Equal(t, AgentRunStatus("pending"), RunPending)
	assert.Equal(t, AgentRunStatus("running"), RunRunning)
	assert.Equal(t, AgentRunStatus("completed"), RunCompleted)
	assert.Equal(t, AgentRunStatus("failed"), RunFailed)
	assert.Equal(t, AgentRunStatus("cancelled"), RunCancelled)
}

// TestExecutionStructFields exercises the real Execution struct
// (the model is `Execution`, not `AgentRun`; the original test
// referenced `AgentRun` as part of the parallel design that was
// never reconciled with the implementation). The Execution struct
// has no `Input`/`Output` fields — those are exposed only via the
// Deliverable table (see model/deliverable.go and TASK-406).
func TestExecutionStructFields(t *testing.T) {
	startedAt := time.Now()
	errMsg := "agent timeout"
	exec := Execution{
		ExecutionID:  uuid.New(),
		TaskID:       uuid.New(),
		AgentID:      uuid.New(),
		Status:       ExecutionStatusRunning,
		StartedAt:    startedAt,
		ErrorMessage: &errMsg,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	assert.NotEqual(t, uuid.Nil, exec.ExecutionID)
	assert.Equal(t, ExecutionStatusRunning, exec.Status)
	assert.NotNil(t, exec.ErrorMessage)
	assert.Equal(t, "agent timeout", *exec.ErrorMessage)
}

// TestAssignmentStructFields exercises the real Assignment struct.
// The real struct uses AssignmentStatus (an enum) rather than a
// free-form string, and has no `EstimatedCompletion` field — see
// assignments-doc-vs-code-2026-06-12.md for the doc/code drift
// analysis and the Sprint 5 follow-up plan.
func TestAssignmentStructFields(t *testing.T) {
	taskID := uuid.New()
	agentID := uuid.New()
	assignedAt := time.Now()
	assign := Assignment{
		ID:         uuid.New(),
		TaskID:     taskID,
		AgentID:    agentID,
		Status:     AssignmentStatusActive,
		AssignedAt: assignedAt,
	}
	assert.NotEqual(t, uuid.Nil, assign.ID)
	assert.Equal(t, taskID, assign.TaskID)
	assert.Equal(t, agentID, assign.AgentID)
	assert.Equal(t, "active", string(assign.Status))
}

// TestAgentTypeConstants asserts the AgentType enum values match
// their canonical string forms. The six values are the Sprint 4
// design set declared in agent_type.go.
func TestAgentTypeConstants(t *testing.T) {
	assert.Equal(t, AgentPM, AgentType("pm"))
	assert.Equal(t, AgentArch, AgentType("architect"))
	assert.Equal(t, AgentDev, AgentType("developer"))
	assert.Equal(t, AgentReviewer, AgentType("reviewer"))
	assert.Equal(t, AgentQA, AgentType("qa"))
	assert.Equal(t, AgentDevOps, AgentType("devops"))
}

// TestIsValidAgentType exercises the IsValidAgentType predicate.
func TestIsValidAgentType(t *testing.T) {
	assert.True(t, IsValidAgentType("pm"))
	assert.True(t, IsValidAgentType("architect"))
	assert.True(t, IsValidAgentType("developer"))
	assert.True(t, IsValidAgentType("reviewer"))
	assert.True(t, IsValidAgentType("qa"))
	assert.True(t, IsValidAgentType("devops"))
	assert.False(t, IsValidAgentType("unknown_type"))
	assert.False(t, IsValidAgentType(""))
}

// TestAllAgentTypes asserts the AllAgentTypes slice contains all six
// known agent types in the order they are declared.
func TestAllAgentTypes(t *testing.T) {
	require.Len(t, AllAgentTypes, 6)
	assert.Contains(t, AllAgentTypes, AgentPM)
	assert.Contains(t, AllAgentTypes, AgentArch)
	assert.Contains(t, AllAgentTypes, AgentDev)
	assert.Contains(t, AllAgentTypes, AgentReviewer)
	assert.Contains(t, AllAgentTypes, AgentQA)
	assert.Contains(t, AllAgentTypes, AgentDevOps)
}

// TestAgentTypeCapabilities asserts the per-type default capability
// set matches the design. The map is the canonical "what skills does
// each agent type embody" answer for routing and reporting.
func TestAgentTypeCapabilities(t *testing.T) {
	assert.Equal(t, []AgentCapability{CapRequirementAnalysis, CapTaskDecomposition}, AgentTypeCapabilities[AgentPM])
	assert.Equal(t, []AgentCapability{CapSystemDesign, CapAPIDesign}, AgentTypeCapabilities[AgentArch])
	assert.Equal(t, []AgentCapability{CapCodeImplementation}, AgentTypeCapabilities[AgentDev])
	assert.Equal(t, []AgentCapability{CapCodeReview, CapSecurityScan}, AgentTypeCapabilities[AgentReviewer])
	assert.Equal(t, []AgentCapability{CapTestPlanning, CapTestExecution}, AgentTypeCapabilities[AgentQA])
	assert.Equal(t, []AgentCapability{CapCICD, CapDeployment, CapInfrastructure}, AgentTypeCapabilities[AgentDevOps])
}

// TestDefaultCapabilitiesForType asserts the []string projection of
// the AgentTypeCapabilities map. Returns nil for unknown types so
// callers can use the result directly without a separate check.
func TestDefaultCapabilitiesForType(t *testing.T) {
	assert.Nil(t, DefaultCapabilitiesForType("unknown_type"))
	assert.Equal(t, []string{"code_implementation"}, DefaultCapabilitiesForType("developer"))
	assert.Equal(t, []string{"code_review", "security_scan"}, DefaultCapabilitiesForType("reviewer"))
	assert.Equal(t, []string{"test_planning", "test_execution"}, DefaultCapabilitiesForType("qa"))
}
