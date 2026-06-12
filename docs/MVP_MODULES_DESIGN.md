# MVP Modules Design

This document outlines the high-level design for the key modules implemented during Sprint 2.

## Project Management Module

### Overview
Handles the project lifecycle and task decomposition for the AI agents.

### API Endpoints
- `POST /projects` - Create a new project.
- `GET /projects/:id` - Get project details.
- `PATCH /projects/:id` - Update project name/description.
- `DELETE /projects/:id` - Delete a project.
- `POST /projects/:id/decompose` - Trigger project task decomposition (interfaces with PM Agent).

### Data Model
- Uses `uuid.UUID` for all IDs.
- `Status` is an enum (Initializing, InProgress, Completed, Failed).

### Service Logic
- `CreateProject` initializes project with `uuid.UUID`.
- `DecomposeProject` triggers agent logic asynchronously (event-driven).

## Agent Registry Module

### Overview
Provides a catalog of available agent types and their capabilities.

### Agent Types
- `pm` (Project Manager)
- `developer`
- `reviewer`
- `qa`
- `devops`

### API Endpoints
- `GET /agents/registry` - List available agent types and capabilities.
- `GET /agents/registry/:type` - Get details for a specific agent type.

### Service Logic
- Agent types are seeded on application startup as constants/enums.
