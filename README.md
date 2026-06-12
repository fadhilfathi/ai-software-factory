# AI Software Factory

A multi-agent software development platform that orchestrates specialized AI agents (PM, Architect, Developer, Reviewer, QA, DevOps) to autonomously build software projects from a user's description.

## Architecture

- **Backend:** Go 1.25+ using Gin framework for high-performance REST APIs.
- **Frontend:** Next.js 16 (React 19) with TypeScript and Tailwind CSS 4.
- **Data:** PostgreSQL (Primary), Redis (Cache), S3-compatible object storage.
- **Infrastructure:** Docker and Docker Compose (development), Kubernetes (production).

## Documentation

- [System Architecture](docs/architecture.md)
- [Security Architecture](docs/security.md)
- [Developer Guide](docs/developer-guide.md)
- [User Guide](docs/user-guide.md)
- [Environment Setup](docs/environment-setup.md)
- [API Specification](docs/api-spec.md)

## Getting Started

See [Environment Setup](docs/environment-setup.md) for prerequisites and local development instructions.

```bash
# Clone the repository
git clone https://github.com/fadhilfathi/AI-Software-Factory.git
cd AI-Software-Factory

# Configure environment
cp .env.example .env

# Start the stack
docker compose up -d --build
```
