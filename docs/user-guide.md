# AI Software Factory — User Guide

> **Document Version**: 1.0  
> **Last Updated**: 2026-06-12  

---

## Welcome to the AI Software Factory

The AI Software Factory is a platform designed to accelerate software delivery by orchestrating a team of specialized AI agents. This guide will walk you through the core features of the platform, from creating your first project to deploying your finished application.

## Table of Contents

1. [Getting Started](#getting-started)
2. [Managing Projects](#managing-projects)
3. [The AI Agent Team](#the-ai-agent-team)
4. [Tasks & Kanban Board](#tasks--kanban-board)
5. [Review & Deployment](#review--deployment)

---

## Getting Started

To start using the platform:

1. **Sign Up / Log In**: Access the platform and create your account.
2. **Dashboard Overview**: Your dashboard provides a real-time view of all your projects, agent activities, and pending tasks.

---

## Managing Projects

### Creating a New Project

1. Click on **New Project**.
2. Provide a **Name** and a detailed **Description** of your software requirements.
3. Select a **Template** (e.g., `web-app`, `api`, `cli`) to give the AI agents a starting point.
4. The system will automatically spawn a **PM Agent** to analyze your request and generate user stories and tasks.

### Monitoring Progress

Your project dashboard tracks overall progress, artifacts generated (like architecture docs and user stories), and the status of active AI agents.

---

## The AI Agent Team

The platform utilizes specialized agents to mirror a traditional software team:

- **PM Agent**: Breaks down requirements, creates user stories, and prioritizes work.
- **Architect Agent**: Designs system architecture, selects the tech stack, and defines APIs.
- **Developer Agent**: Writes code, implements features, and handles refactoring.
- **Review Agent**: Reviews code quality, enforces standards, and suggests improvements.
- **QA Agent**: Creates test plans, runs tests, and verifies bug fixes.
- **DevOps Agent**: Manages deployments and monitors infrastructure.

---

## Tasks & Kanban Board

Once the PM Agent decomposes your project, work is tracked on the **Kanban Board**.

- **Backlog**: Tasks waiting to be picked up.
- **Ready**: Tasks assigned to specific agents.
- **In Progress**: Agents actively working on code generation or design.
- **Review**: Tasks undergoing automated or human review.
- **Done**: Completed tasks.

You can manually adjust task priorities or provide additional context to guide the agents.

---

## Review & Deployment

### Quality Gates

Critical decisions and code changes pass through Quality Gates. You (the human overseer) have the final say on approvals before major deployments.

### Deployment

Once features pass review, the **DevOps Agent** triggers the CI/CD pipeline to build, test, and deploy the application to the specified environment (e.g., Staging or Production).

---

> For developers looking to integrate with the API or contribute to the platform, see the [Developer Guide](./developer-guide.md).
