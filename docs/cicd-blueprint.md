# CI/CD Blueprint: Go/Gin + Next.js

This document outlines the CI/CD pipeline strategy for the AI Software Factory, now utilizing a Go/Gin backend and Next.js frontend, containerized for reliable deployment.

## Overview
The CI/CD pipeline uses GitHub Actions to automate linting, testing, building, and deployment to a staging environment.

## Pipeline Structure

### 1. CI Pipeline (`.github/workflows/ci.yml`)
Runs on every push/PR to `main`.
*   **Linting:** 
    *   **Go (Backend):** `golangci-lint` for code quality; `go fmt` check.
    *   **TypeScript (Frontend):** `npm run lint` for React standards.
*   **Testing:**
    *   **Go:** Runs unit and integration tests using `go test` with race detection.
*   **Builds (Isolated):**
    *   **Backend Build:** Builds production Docker image for Go API.
    *   **Frontend Build:** Performs `npm run build` to create static assets, then builds production Docker image for the frontend.
*   **End-to-End Smoke Test:**
    *   Uses `docker compose` to orchestrate the verified container images (API, Frontend, DB) and verifies that health endpoints are reachable and healthy.

### 2. Deployment Pipeline (`.github/workflows/deploy.yml`)
Runs on push to `main` after successful CI.
*   **Build & Push:** Builds production Docker images and pushes them to GitHub Container Registry (GHCR).
*   **Deploy:** Pulls fresh images, updates `docker-compose.yml` to use the new image tags, and deploys the stack using `docker compose up -d`.
*   **Verification:** Performs a health check after deployment. If the deployment fails health checks, it attempts a rollback.

## Infrastructure Requirements
*   **Runner:** GitHub-hosted `ubuntu-latest` runners are used.
*   **Registry:** GitHub Container Registry (GHCR) for secure image storage.
*   **Docker:** `docker-compose` is used for orchestrating the stack on the staging environment.

## Deployment to Staging
The `deploy.yml` workflow assumes a Docker environment is available on the target machine with credentials configured to pull images from GHCR.

## Future Enhancements
*   Add automated integration tests against the database.
*   Implement deployment to a Kubernetes cluster for production scaling.
*   Add performance/load testing phase.
