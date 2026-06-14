//go:build linux

package agentfactory

import "syscall"

// signalTerm sends SIGTERM to the process with the given PID.
//
// On Linux this is the canonical "please shut down cleanly" signal and
// matches the Aion subprocess shutdown contract (AION-SHUTDOWN-001).
//
// This file is gated to Linux; the Windows build is handled by
// signals_other.go, which returns a non-fatal error instead of crashing
// the build. See fix/A002-handbacks for the rationale.
func signalTerm(pid int) error {
	return syscall.Kill(pid, syscall.SIGTERM)
}
