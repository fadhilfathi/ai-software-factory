# Test Strategy

This document outlines the testing strategy for the AI-Software-Factory project, utilizing a Go/Gin backend and Next.js/TypeScript frontend.

## 1. Objectives
- Ensure system reliability, stability, and maintainability.
- Facilitate rapid development cycles with high-confidence deployments.
- Validate business logic, API integrity, and user experience.

## 2. Testing Pyramid
We adopt a testing pyramid approach, prioritizing faster, lower-level tests while maintaining sufficient E2E coverage for critical paths.

### 2.1 Backend (Go/Gin)
- **Unit Tests**: Focus on isolated business logic (services), model validation, and utility functions.
  - Framework: `testing` (standard library).
- **Integration Tests**: Verify API endpoints, handler interactions, and database connectivity.
  - Framework: `net/http/httptest` with the Gin router.

### 2.2 Frontend (Next.js/React/TypeScript)
- **Unit/Component Tests**: Focus on React components, hooks, and shared logic.
  - Framework: `Vitest` and `React Testing Library`.
- **End-to-End (E2E) Tests**: Verify critical user flows (e.g., Auth, Task management, Dashboard).
  - Framework: `Playwright`.

## 3. Tooling and Environment
- **Automation**: All tests must be runnable via CI workflows in `.github/workflows/ci.yml`.
- **Coverage**: Aim for >80% coverage for core business logic.

## 5. Implementation Standards
- **Orchestrator Testability**: To ensure the `AgentOrchestrator` (utilizing Docker SDK) is testable, all interactions with external SDKs MUST be abstracted behind mockable Go interfaces.
- **Gin Handler Validation**: All Gin handlers MUST include comprehensive unit tests verifying the correct parsing, validation, and error handling of `uuid.UUID` fields in request parameters and body.

## 6. Execution
- Tests are executed locally during development and in the CI environment upon PR submission.
