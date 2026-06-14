//go:build !linux

package agentfactory

import "errors"

// signalTerm is the non-Linux stub for sending a shutdown signal to a
// tracked agent PID.
//
// Windows does not implement POSIX signals; SIGTERM is undefined and
// syscall.Kill is not available. The agent subprocess management
// strategy on Windows is delegated to the Aion runtime's own
// subprocess handle (os.Process.Signal) — see signals_windows.go
// for the concrete implementation in a follow-up. For now, callers
// see a non-fatal error which is logged and Shutdown continues for
// the remaining tracked agents.
func signalTerm(pid int) error {
	return errors.New("agentfactory: signalTerm not implemented on this platform; use os.FindProcess + os.Process.Signal instead")
}
