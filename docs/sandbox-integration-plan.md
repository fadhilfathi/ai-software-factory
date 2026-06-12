# Execution Sandbox Integration Plan (gVisor / Firecracker)

## 1. Objective
To provide a secure, isolated environment for executing untrusted agent-generated code and running automated test suites, mitigating the risk of sandbox escapes and host system compromise.

## 2. Technology Evaluation

### gVisor (runsc)
- **Architecture**: A user-space kernel that intercepts and handles syscalls, providing strong isolation with a smaller footprint than a full VM.
- **Integration**: Plugs directly into Docker as a custom runtime (`--runtime=runsc`).
- **Best For**: Containerized workloads that require strong kernel isolation without the overhead of full virtualization.
- **Complexity**: Low-Medium (requires host-side installation and `daemon.json` config).

### Firecracker
- **Architecture**: A MicroVM manager that uses KVM to create lightweight, fast-booting virtual machines.
- **Integration**: Requires specialized orchestrators like `firecracker-containerd` or custom VM management logic.
- **Best For**: High-security multi-tenant workloads or workloads requiring very fast boot times with full hardware-level isolation.
- **Complexity**: High (requires KVM support and more complex orchestration than Docker runtimes).

## 3. Recommended Approach: gVisor (runsc)
Given our existing infrastructure relies heavily on the Docker SDK, gVisor provides the best balance of security and ease of integration.

### Implementation Steps

#### Phase 1: Host Preparation (Manual/DevOps)
1. Install `runsc` on the Docker host.
2. Configure Docker to recognize the `runsc` runtime:
   ```json
   {
     "runtimes": {
       "runsc": {
         "path": "/usr/local/bin/runsc"
       }
     }
   }
   ```
3. Restart Docker.

#### Phase 2: Orchestrator Hardening
Update `src/internal/service/orchestrator.go` to support secure runtime options:

```go
func (o *agentOrchestrator) SpawnAgentProcess(ctx context.Context, agent *model.Agent) error {
    // ...
    hostConfig := &container.HostConfig{
        Runtime: "runsc", // Use gVisor for strong isolation
        Resources: container.Resources{
            Memory: 512 * 1024 * 1024,
            CPUQuota: 50000,
        },
        // Harden the container
        ReadonlyRootfs: true,
        CapDrop: []string{"ALL"}, // Drop all capabilities
        SecurityOpt: []string{"no-new-privileges"},
        AutoRemove: true,
    }
    // ...
}
```

#### Phase 3: Code Execution Service (New)
Implement a specialized service for running code snippets:
- Uses short-lived gVisor containers.
- Mounts code in a read-only volume or via `stdin`.
- Disables networking by default (`NetworkMode: "none"`).
- Captures `stdout`/`stderr` and exit codes.

## 4. Security Controls Matrix

| Control | Implementation | Purpose |
|---------|----------------|---------|
| Runtime Isolation | gVisor (`runsc`) | Prevents kernel exploit sandbox escapes |
| Resource Limits | Cgroups (CPU/Mem) | Prevents DoS (fork bombs, memory exhaustion) |
| Read-only FS | `ReadonlyRootfs: true` | Prevents persistence and tampering with the environment |
| Capability Drop | `CapDrop: ["ALL"]` | Removes root-like privileges within the container |
| No Networking | `NetworkMode: "none"` | Prevents data exfiltration and side-channel attacks |
| Timeouts | Service-level Context | Prevents long-running or hanging executions |

## 5. Next Steps
1. **Verification**: Confirm if the target environment supports gVisor (requires Linux x86_64).
2. **POC**: Implement a simple `SandboxService` that uses `runsc` to run a "Hello World" in Python.
3. **Integration**: Wire the `CodeService` and `QAService` to use the `SandboxService` for their execution needs.
