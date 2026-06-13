// This file is a temporary verification artifact for TASK-431.
// It deliberately contains a failing test to prove the Sprint Quality Gate
// is currently masking unit test failures. Delete this file after the
// TASK-431 audit (docs/sprint5/quality-gate-audit.md) is written.
//
// Run on branch: test/sqg-verify-broken-baseline (and verify/sprint-5-quality-gate-strict-catches)
// Expected behavior:
//   - With masking in place (current main): go test ./internal/sqgverify/... exits non-zero,
//     but the Sprint Quality Gate job goes GREEN because step 8 has
//     `continue-on-error: true` and `|| { ...; exit 0; }`.
//   - With masking removed (fix branch + this test): go test exits non-zero,
//     and the Sprint Quality Gate job goes RED at step 8.
package sqgverify

import "testing"

// TestSQGWillCatchThisFail is the sentinel. If the Sprint Quality Gate is
// doing its job, this test's failure must propagate to the gate's exit code.
// If the gate is still masking, the test will fail locally and on the runner,
// but the job will report GREEN — which is the bug TASK-431 fixes.
func TestSQGWillCatchThisFail(t *testing.T) {
	t.Fatal("intentional SQG verification failure (TASK-431 baseline; safe to delete)")
}
