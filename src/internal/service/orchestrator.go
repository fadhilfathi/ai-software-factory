package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/fadhilfathi/AI-Software-Factory/internal/config"
	"github.com/fadhilfathi/AI-Software-Factory/internal/model"
	"github.com/fadhilfathi/AI-Software-Factory/internal/store"
	"go.uber.org/zap"
)

// AgentOrchestrator manages the lifecycle and task execution of AI agents.
type AgentOrchestrator interface {
	StartMonitoring(ctx context.Context)
	HandleAgentFailure(agentID string) error
	SpawnAgentProcess(ctx context.Context, agent *model.Agent) error
}

type agentOrchestrator struct {
	store      store.Store
	config     *config.Config
	log        *zap.Logger
	mu         sync.Mutex
	activePods map[string]context.CancelFunc
	dockerCli  *client.Client
}

func NewAgentOrchestrator(s store.Store, cfg *config.Config, log *zap.Logger) (AgentOrchestrator, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	return &agentOrchestrator{
		store:      s,
		config:     cfg,
		log:        log,
		activePods: make(map[string]context.CancelFunc),
		dockerCli:  cli,
	}, nil
}

// StartMonitoring begins the periodic check for agent health and status.
func (o *agentOrchestrator) StartMonitoring(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	o.log.Info("Agent Orchestrator monitoring started")

	for {
		select {
		case <-ticker.C:
			o.checkAgentHealth()
		case <-ctx.Done():
			o.log.Info("Agent Orchestrator monitoring stopped")
			return
		}
	}
}

func (o *agentOrchestrator) checkAgentHealth() {
	o.log.Debug("Checking agent health")
}

// HandleAgentFailure attempts to recover or re-assign tasks for a failed agent.
func (o *agentOrchestrator) HandleAgentFailure(agentID string) error {
	o.log.Warn("Handling agent failure", zap.String("agent_id", agentID))
	return fmt.Errorf("not implemented")
}

// SpawnAgentProcess launches an agent in an isolated environment using Docker.
func (o *agentOrchestrator) SpawnAgentProcess(ctx context.Context, agent *model.Agent) error {
	o.log.Info("Spawning agent container", zap.String("agent_id", agent.ID.String()), zap.String("runtime", o.config.Agent.Runtime))

	containerName := fmt.Sprintf("agent-%s", agent.ID)
	
	// Create container with isolation/sandboxing settings
	resp, err := o.dockerCli.ContainerCreate(ctx, &container.Config{
		Image: "ai-software-factory-agent:latest",
		Env: []string{
			fmt.Sprintf("AGENT_ID=%s", agent.ID),
		},
	}, &container.HostConfig{
		Runtime: o.config.Agent.Runtime,
		// Enforce isolation by limiting resources and security
		Resources: container.Resources{
			Memory:   o.config.Agent.MemoryLimit,
			CPUQuota: o.config.Agent.CPULimit,
		},
		AutoRemove:     true, // Clean up container after it exits
		ReadonlyRootfs: true, // Prevent writing to root filesystem
		CapDrop:        []string{"ALL"},
		SecurityOpt:    []string{"no-new-privileges"},
		NetworkMode:    "none", // Disable networking for maximum isolation
	}, nil, nil, containerName)
	
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	// Start the container
	err = o.dockerCli.ContainerStart(ctx, resp.ID, container.StartOptions{})
	if err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	o.log.Info("Agent container started successfully", zap.String("container_id", resp.ID))
	return nil
}
