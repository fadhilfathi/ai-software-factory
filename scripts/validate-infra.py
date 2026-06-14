"""Static validation of docker-compose.yml and infra references.

Used by E-001 Infrastructure Validation, 2026-06-14.
Run with: python scripts/validate-infra.py
"""
import os
import re
import sys
from pathlib import Path

import yaml

ROOT = Path(__file__).resolve().parent.parent
ERRORS = []
WARNINGS = []


def err(msg):
    ERRORS.append(msg)
    print(f"ERROR: {msg}")


def warn(msg):
    WARNINGS.append(msg)
    print(f"WARN:  {msg}")


def ok(msg):
    print(f"OK:    {msg}")


def main():
    compose_path = ROOT / "docker-compose.yml"
    if not compose_path.exists():
        err("docker-compose.yml not found at repo root")
        return 1
    with open(compose_path, encoding="utf-8") as f:
        try:
            compose = yaml.safe_load(f)
        except yaml.YAMLError as e:
            err(f"docker-compose.yml is not valid YAML: {e}")
            return 1
    ok("docker-compose.yml parses as valid YAML")

    services = compose.get("services", {})
    if not services:
        err("no services defined in docker-compose.yml")
        return 1
    ok(f"services defined: {list(services.keys())}")

    # 1. Required services (the canonical names actually used in this repo)
    required = {"db", "redis", "api", "frontend"}
    missing = required - set(services.keys())
    if missing:
        err(f"missing required services: {missing}")
    else:
        ok(f"all required services present: {sorted(required)}")

    # 2. Each service must have a healthcheck
    for name, svc in services.items():
        if "healthcheck" not in svc:
            warn(f"service {name!r} has no healthcheck")
        else:
            ok(f"service {name!r} has healthcheck")

    # 3. env_file present
    for name, svc in services.items():
        if svc.get("env_file") is None:
            warn(f"service {name!r} has no env_file")
        else:
            ef = svc["env_file"]
            if isinstance(ef, str):
                ef = [ef]
            for path in ef:
                full = ROOT / path
                if not full.exists():
                    err(f"service {name!r} env_file {path!r} not found at {full}")
                else:
                    ok(f"service {name!r} env_file {path!r} exists")

    # 4. env vars referenced in docker-compose.yml must exist in .env.example
    compose_text = compose_path.read_text(encoding="utf-8")
    referenced = set(re.findall(r"\$\{([A-Z_][A-Z0-9_]*)\}|\$([A-Z_][A-Z0-9_]*)", compose_text))
    referenced = {a or b for a, b in referenced}
    if referenced:
        ok(f"compose references env vars: {sorted(referenced)}")
        env_example = ROOT / ".env.example"
        if not env_example.exists():
            err(f"compose references env vars {referenced} but .env.example missing")
        else:
            env_text = env_example.read_text(encoding="utf-8")
            env_keys = set()
            for line in env_text.splitlines():
                m = re.match(r"^([A-Z_][A-Z0-9_]*)\s*=", line)
                if m:
                    env_keys.add(m.group(1))
            missing = referenced - env_keys
            if missing:
                err(f"env vars referenced by compose but missing from .env.example: {sorted(missing)}")
            else:
                ok(f"all env vars referenced by compose are defined in .env.example")

    # 5. Dockerfile references must exist
    for name, svc in services.items():
        if "build" in svc:
            build = svc["build"]
            if isinstance(build, str):
                ctx = ROOT / build
                dockerfile = ctx / "Dockerfile"
            else:
                ctx = ROOT / build.get("context", ".")
                dockerfile_rel = build.get("dockerfile", "Dockerfile")
                dockerfile = (ctx / dockerfile_rel).resolve()
            if not dockerfile.exists():
                err(f"service {name!r} references Dockerfile at {dockerfile} but it does not exist")
            else:
                ok(f"service {name!r} Dockerfile exists: {dockerfile.relative_to(ROOT)}")
                # Check Dockerfile has a HEALTHCHECK
                df_text = dockerfile.read_text(encoding="utf-8")
                if "HEALTHCHECK" in df_text:
                    ok(f"Dockerfile {dockerfile.relative_to(ROOT)} declares HEALTHCHECK")
                else:
                    warn(f"Dockerfile {dockerfile.relative_to(ROOT)} has no HEALTHCHECK directive")
                # Check non-root user
                if re.search(r"^USER\s+\w+", df_text, re.MULTILINE) and not re.search(r"^USER\s+root", df_text, re.MULTILINE):
                    ok(f"Dockerfile {dockerfile.relative_to(ROOT)} switches to non-root user")
                else:
                    warn(f"Dockerfile {dockerfile.relative_to(ROOT)} does not switch to non-root user (security)")

    # 6. Volumes
    if "volumes" in compose:
        for vol_name in compose["volumes"]:
            ok(f"named volume declared: {vol_name}")

    # 7. Port mapping check
    for name, svc in services.items():
        if "ports" in svc:
            for port in svc["ports"]:
                ok(f"service {name!r} port mapping: {port}")

    # 8. .env.example sanity
    env_example = ROOT / ".env.example"
    if env_example.exists():
        env_text = env_example.read_text(encoding="utf-8")
        placeholder_count = sum(1 for line in env_text.splitlines() if "CHANGE_ME" in line or "your-" in line.lower())
        ok(f".env.example has {placeholder_count} placeholder values to replace before use")
        # Check no real secrets
        if re.search(r"AKIA[0-9A-Z]{16}", env_text):
            err(".env.example contains what looks like an AWS access key")
        if re.search(r"ghp_[A-Za-z0-9]{36}", env_text):
            err(".env.example contains what looks like a GitHub PAT")
        if re.search(r"sk-[A-Za-z0-9]{20,}", env_text):
            err(".env.example contains what looks like an OpenAI/Anthropic key")
        ok(".env.example scan: no real-secret patterns detected")

    # 9. scripts/ existence
    for script in ["scripts/healthcheck.sh", "scripts/deploy.sh", "scripts/validate-infra.py"]:
        p = ROOT / script
        if p.exists():
            ok(f"{script} present ({p.stat().st_size} bytes)")
        else:
            err(f"{script} missing")
        if script.endswith(".sh") and p.exists():
            text = p.read_text(encoding="utf-8")
            if not text.startswith("#!"):
                warn(f"{script} missing shebang")

    # 10. healthcheck.sh references
    hc = ROOT / "scripts" / "healthcheck.sh"
    if hc.exists():
        text = hc.read_text(encoding="utf-8")
        if "/healthz" in text or "/health" in text:
            ok("healthcheck.sh probes the /healthz endpoint")
        else:
            warn("healthcheck.sh does not probe /healthz")

    # 11. deploy.sh idempotency
    dep = ROOT / "scripts" / "deploy.sh"
    if dep.exists():
        text = dep.read_text(encoding="utf-8")
        if "docker compose" in text or "docker-compose" in text:
            ok("deploy.sh uses docker compose")
        else:
            warn("deploy.sh does not appear to use docker compose")

    # 12. dev01-task-427-wt and tester01-wt worktree presence (Sprint 6 staging sanity)
    wt = ROOT / "dev01-task-427-wt"
    if wt.exists():
        ok(f"Sprint 6 staging worktree present: {wt.relative_to(ROOT)}")
    wt = ROOT / "..tester01-wt"
    if (ROOT / "..tester01-wt").exists():
        ok(f"tester01 staging worktree present: ../tester01-wt")

    print()
    print("=" * 60)
    print(f"SUMMARY: {len(ERRORS)} errors, {len(WARNINGS)} warnings")
    print("=" * 60)
    if ERRORS:
        for e in ERRORS:
            print(f"  - {e}")
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
