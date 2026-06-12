# Kanban Acceptance Criteria (TASK-107)

This document defines the quality and functional requirements for the Kanban board drag-and-drop operations.

## 1. Functional Requirements
- **Drag-and-Drop**: Users must be able to drag tasks between status columns (e.g., 'Pending' to 'In Progress').
- **Persistence**: Upon dropping a task, the frontend must immediately initiate an API call to update the task status in the backend.
- **Optimistic UI**: The UI should update immediately, with a rollback mechanism in case the backend API call fails.
- **Validation**:
  - The drop action must be validated against the permitted state transition logic defined in the backend.
  - Backend must return success/failure confirmation.

## 2. Testing Acceptance Criteria
- **Unit Testing (Frontend)**:
  - Verify drag-and-drop component state transitions.
  - Mock the API interaction to verify optimistic update and rollback behavior.
- **Integration Testing (Backend)**:
  - Verify API endpoints correctly handle status updates.
  - Ensure `uuid.UUID` parsing for task and column IDs is validated.
- **E2E Testing (Playwright)**:
  - Verify full end-to-end flow: Drag a task, observe UI update, verify backend status change via API call, and confirm persistence after page refresh.
