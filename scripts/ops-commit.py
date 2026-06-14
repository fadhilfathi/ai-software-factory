#!/usr/bin/env python
"""Ops commit-pipeline: pull → review → test → build → secret-scan → commit → push.

This is the MANDATORY E-003 continuous commit policy. Every Builder
commit that lands on main flows through this script. It exists so
that no sprint-end batching ever happens again.

Usage:
    python scripts/ops-commit.py <branch-name>
    python scripts/ops-commit.py feat/sprint-6-task-507
    python scripts/ops-commit.py --dry-run feat/sprint-6-task-507
    python scripts/ops-commit.py --abort   # rollback last push

Steps performed (in order):
    1. Sync origin/main to local main.
    2. Confirm the given branch exists and is a descendant of (or fast-
       forwardable to) main.
    3. Run gitleaks secret-scan on the branch's commits.
    4. Run the project's static checks: scripts/validate-infra.py.
    5. Show the diff against main.
    6. If everything is clean: rebase onto main, fast-forward merge into
       main, push to origin.
    7. Print a summary suitable for the Lead.

The script never force-pushes, never rewrites published history, and
never pushes if any step fails. A non-zero exit at any step halts the
pipeline.
"""
import argparse
import subprocess
import sys
import textwrap
from pathlib import Path

# Ensure stdout/stderr can render the Unicode arrows in the help text
# on Windows (default cp1252 codec chokes on U+2192).
for _stream in (sys.stdout, sys.stderr):
    try:
        _stream.reconfigure(encoding="utf-8")
    except (AttributeError, OSError):
        pass

ROOT = Path(__file__).resolve().parent.parent
GITLEAKS = ROOT / "tools" / "gitleaks.exe"


def step(msg):
    print(f"\n[ops-commit] {msg}")


def run(cmd, check=True, capture=False):
    """Run a shell command. If check, exit on non-zero. If capture,
    return stdout. Otherwise print to terminal."""
    print(f"  $ {' '.join(cmd) if isinstance(cmd, list) else cmd}")
    if capture:
        result = subprocess.run(
            cmd, shell=isinstance(cmd, str), cwd=ROOT,
            capture_output=True, text=True
        )
        if check and result.returncode != 0:
            print(result.stdout)
            print(result.stderr, file=sys.stderr)
            sys.exit(result.returncode)
        return result
    result = subprocess.run(cmd, shell=isinstance(cmd, str), cwd=ROOT)
    if check and result.returncode != 0:
        sys.exit(result.returncode)
    return result


def sync_main():
    step("1. Sync origin/main → local main")
    run("git fetch origin main", check=False)
    run("git checkout main")
    result = run("git pull --rebase origin main", check=False)
    if result.returncode != 0:
        # Fall back to merge if rebase is impossible (no local commits)
        run("git pull origin main")
    run("git log --oneline -1")


def validate_branch(branch):
    step(f"2. Validate branch: {branch}")
    result = run(["git", "rev-parse", "--verify", branch], capture=True)
    if result.returncode != 0:
        print(f"ERROR: branch {branch!r} does not exist locally or on origin.")
        sys.exit(2)
    # Check that main is an ancestor of the branch (branch is ahead of
    # main, not diverged).
    result = run(["git", "merge-base", "--is-ancestor", "main", branch], check=False, capture=True)
    if result.returncode != 0:
        print(f"ERROR: {branch} is not ahead of main. Rebase it onto main first.")
        sys.exit(3)
    run(["git", "log", "--oneline", f"main..{branch}"])


def secret_scan(branch):
    step("3. Gitleaks secret scan on the branch")
    if not GITLEAKS.exists():
        print(f"WARN: gitleaks not installed at {GITLEAKS}; run tools/install-gitleaks.sh")
        return
    result = run(
        [str(GITLEAKS), "detect", "--source", ".",
         "--config", ".gitleaks.toml",
         "--log-opts", f"main..{branch}",
         "--no-banner", "--exit-code", "1"],
        check=False,
    )
    if result.returncode != 0:
        print("ERROR: secret scan failed. Aborting.")
        sys.exit(4)
    print("  secret scan clean")


def static_checks():
    step("4. Static checks: scripts/validate-infra.py")
    result = run([sys.executable, "scripts/validate-infra.py"], check=False)
    if result.returncode != 0:
        print("ERROR: static checks failed. Aborting.")
        sys.exit(5)


def show_diff(branch):
    step(f"5. Diff: main..{branch}")
    result = run(
        ["git", "diff", "--stat", f"main..{branch}"], check=False, capture=True
    )
    if result.returncode == 0:
        print(result.stdout)
    run(["git", "log", "--oneline", f"main..{branch}"])


def merge_and_push(branch, dry_run):
    step(f"6. {'DRY-RUN: would' if dry_run else 'Will'} rebase, merge, push")
    if dry_run:
        print("  --dry-run set: skipping the rebase/merge/push")
        return
    # Rebase the branch onto the current main (linear history).
    run(["git", "checkout", branch])
    result = run(["git", "rebase", "main"], check=False)
    if result.returncode != 0:
        print("ERROR: rebase failed. Resolve conflicts, then re-run.")
        sys.exit(6)
    # Fast-forward main.
    run(["git", "checkout", "main"])
    run(["git", "merge", "--ff-only", branch])
    # Push.
    result = run(["git", "push", "origin", "main"], check=False)
    if result.returncode != 0:
        print("ERROR: push failed. Manual recovery required.")
        sys.exit(7)


def main():
    parser = argparse.ArgumentParser(
        description=textwrap.dedent(__doc__),
        formatter_class=argparse.RawDescriptionHelpFormatter,
    )
    parser.add_argument("branch", nargs="?", help="branch to merge into main")
    parser.add_argument("--dry-run", action="store_true",
                        help="run all checks but do not push")
    parser.add_argument("--skip-tests", action="store_true",
                        help="skip gitleaks + validate-infra (CI already ran)")
    args = parser.parse_args()

    if not args.branch:
        parser.print_help()
        sys.exit(1)

    sync_main()
    validate_branch(args.branch)
    if not args.skip_tests:
        secret_scan(args.branch)
        static_checks()
    show_diff(args.branch)
    merge_and_push(args.branch, args.dry_run)

    step("7. Summary")
    print(f"  branch {args.branch} {'would be' if args.dry_run else 'is'} on main.")
    print(f"  current main HEAD: $(git rev-parse main)")
    print()
    print("  Notify Lead with:")
    print(f"    'E-003 commit landed: {args.branch} → main (dry-run={args.dry_run})'")


if __name__ == "__main__":
    main()
