package service

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/fadhilfathi/AI-Software-Factory/internal/config"
	"go.uber.org/zap"
)

// SandboxRequest defines the parameters for a sandbox execution.
type SandboxRequest struct {
	Image   string
	Command []string
	Env     []string
	Timeout time.Duration
}

// SandboxResult contains the outcome of a sandbox execution.
type SandboxResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	TimedOut bool
}

// SandboxService provides isolated execution environments.
type SandboxService struct {
	config    *config.Config
	log       *zap.Logger
	dockerCli *client.Client
}

func NewSandboxService(cfg *config.Config, log *zap.Logger) (*SandboxService, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create docker client: %w", err)
	}

	return &SandboxService{
		config:    cfg,
		log:       log,
		dockerCli: cli,
	}, nil
}

// Run executes a command in a hardened sandbox container.
func (s *SandboxService) Run(ctx context.Context, req SandboxRequest) (*SandboxResult, error) {
	s.log.Info("Running sandbox execution", zap.Strings("command", req.Command), zap.String("runtime", s.config.Agent.Runtime))

	if req.Timeout == 0 {
		req.Timeout = 5 * time.Minute
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, req.Timeout)
	defer cancel()

	// Create container with isolation/sandboxing settings
	resp, err := s.dockerCli.ContainerCreate(timeoutCtx, &container.Config{
		Image: req.Image,
		Cmd:   req.Command,
		Env:   req.Env,
		Tty:   false,
	}, &container.HostConfig{
		Runtime: s.config.Agent.Runtime,
		// Enforce isolation by limiting resources and security
		Resources: container.Resources{
			Memory:   s.config.Agent.MemoryLimit,
			CPUQuota: s.config.Agent.CPULimit,
		},
		AutoRemove:     true, // Clean up container after it exits
		ReadonlyRootfs: true, // Prevent writing to root filesystem
		CapDrop:        []string{"ALL"},
		SecurityOpt:    []string{"no-new-privileges"},
		NetworkMode:    "none", // Disable networking for maximum isolation
	}, nil, nil, "")

	if err != nil {
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Start the container
	err = s.dockerCli.ContainerStart(timeoutCtx, resp.ID, container.StartOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// Capture logs
	out, err := s.dockerCli.ContainerLogs(timeoutCtx, resp.ID, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get container logs: %w", err)
	}
	defer out.Close()

	var stdout, stderr bytes.Buffer
	// Docker multiplexes stdout and stderr in the stream returned by ContainerLogs
	// when Tty is false. We should use stdcopy, but for now, we'll just read it all.
	// TODO: Use stdcopy.StdCopy to properly separate stdout and stderr.
	_, _ = io.Copy(&stdout, out)

	// Wait for container to finish
	statusCh, errCh := s.dockerCli.ContainerWait(timeoutCtx, resp.ID, container.WaitConditionNotRunning)
	
	select {
	case err := <-errCh:
		if err != nil {
			return nil, fmt.Errorf("error waiting for container: %w", err)
		}
	case status := <-statusCh:
		return &SandboxResult{
			Stdout:   stdout.String(),
			ExitCode: int(status.StatusCode),
			TimedOut: false,
		}, nil
	case <-timeoutCtx.Done():
		if timeoutCtx.Err() == context.DeadlineExceeded {
			// Attempt to stop the container if it's still running
			stopTimeout := 2 * time.Second
			_ = s.dockerCli.ContainerStop(context.Background(), resp.ID, container.StopOptions{Timeout: &[]int{int(stopTimeout.Seconds())}[0]})
			return &SandboxResult{
				Stdout:   stdout.String(),
				TimedOut: true,
				ExitCode: -1,
			}, nil
		}
		return nil, timeoutCtx.Err()
	}

	return nil, fmt.Errorf("unexpected end of Run")
}
