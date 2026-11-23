<p align="center">
  <img src="apps/web/public/branding/dasboard_logo.png" alt="AI House Generator Logo" width="180">
</p>

<h1 align="center">AI HOUSE GENERATOR</h1>

<p align="center">
  AI-Powered House Design & Project Management Platform
</p>

---
# AI House Generator

AI House Generator (AHG) is a software platform for creating and managing AI-assisted house-design projects. The repository contains the web application and Go API services, with PostgreSQL for persistent project data and Redis for supporting application services.

> **Project status:** Backend hardening baseline completed on 23 November 2026. The API starts successfully, database migrations are verified, the API runtime connects through a restricted PostgreSQL role, and the Go test and vet suites pass.

## Core capabilities

- Manage house-design projects through the API.
- Organize designs under projects.
- Preserve design revisions using design versions.
- Store application data in PostgreSQL.
- Use Redis for supporting runtime services.
- Provide health and readiness endpoints for operational checks.

The exact AI model providers, generation workflows, and supported CAD/export formats should be documented as those integrations are finalized. This README intentionally does not claim unverified providers, certifications, production deployments, or generation capabilities.

## Technology

| Component | Technology |
|---|---|
| API | Go |
| Database | PostgreSQL 17 |
| Cache / supporting runtime service | Redis |
| Web application | `apps/web` |

## Repository layout

```text
ai-house-generator/
├── apps/
│   └── web/                 # Web application
├── services/
│   └── api/
│       ├── cmd/
│       │   ├── server/      # HTTP API server
│       │   └── worker/      # Background worker entry point
│       ├── internal/
│       │   ├── database/    # PostgreSQL configuration and migrations
│       │   ├── projects/    # Project API and data access
│       │   └── queue/       # Queue integration
│       └── migrations/      # Embedded SQL migrations
└── README.md
```

## Backend requirements

- Go toolchain compatible with the version declared by `services/api/go.mod`.
- PostgreSQL 17.
- Redis.
- Docker Desktop is recommended for running local infrastructure on Windows.

The local development environment described during backend verification used:

| Service | Local endpoint |
|---|---|
| PostgreSQL | `127.0.0.1:55432` |
| Redis | `127.0.0.1:56379` |
| API | `127.0.0.1:8097` |

These are the verified local development ports, not production endpoints.

## Configuration

Create or update the root `.env` file using `.env.example` as the template. Do not commit `.env`, passwords, tokens, database dumps, or other secrets.

The backend separates database credentials by responsibility:

| Environment variable | Purpose |
|---|---|
| `POSTGRES_HOST` | PostgreSQL host |
| `POSTGRES_PORT` | PostgreSQL port |
| `POSTGRES_DB` | Application database |
| `POSTGRES_MIGRATOR_USER` | Migration login |
| `POSTGRES_MIGRATOR_PASSWORD` | Migration password |
| `POSTGRES_RUNTIME_USER` | Restricted API runtime login |
| `POSTGRES_RUNTIME_PASSWORD` | Runtime password |
| `REDIS_PORT` | Redis port |
| `PORT` | HTTP API port; defaults to `8097` |

Use strong, unique passwords in local and deployed environments. Never paste `.env` contents into issues, chat messages, logs, or source control.

## Run the API locally

From the repository root in PowerShell:

```powershell
Push-Location .\services\api
go run ./cmd/server
```

The API listens on `http://127.0.0.1:8097` by default. Start PostgreSQL and Redis before starting the API.

The server runs pending embedded SQL migrations using the migration account, then starts the HTTP service using the restricted runtime account. The migration connection pool is closed after migration execution.

## API operational endpoints

| Method | Endpoint | Purpose |
|---|---|---|
| `GET` | `/healthz` | Basic API health |
| `GET` | `/api/v1/health` | API health route |
| `GET` | `/api/v1/ready` | Readiness check for database and Redis |
| `GET` | `/api/v1/projects` | Project collection endpoint |
| `GET` | `/api/v1/projects/` | Project resource route |

Project routes support the project operations implemented by the current API. Consult the source and API tests for the current request and response schemas.

Example checks:

```powershell
Invoke-RestMethod http://127.0.0.1:8097/healthz
Invoke-RestMethod http://127.0.0.1:8097/api/v1/ready
```

A successful readiness response reports the database and Redis connection status.

## Database security model

The local PostgreSQL setup uses separate roles for administration, object ownership, migrations, and application runtime.

| Role | Responsibility |
|---|---|
| `ahg_app` | Administrative/bootstrap access; not for API runtime |
| `ahg_owner` | `NOLOGIN` owner of application schemas and objects |
| `ahg_migrator` | Migration login; assumes the `ahg_owner` role when applying migrations |
| `ahg_runtime` | Restricted API login for normal application operations |

The API runtime role is not a superuser and does not own application objects. It has the required access to the `core` schema and CRUD operations on the application tables, without `TRUNCATE` permission or access to the internal `app` migration schema.

Do not use the administrative/bootstrap account as the API runtime account.

## Database migrations

SQL migrations are embedded into the API binary using Go's `embed` package. The migration runner:

- Applies migrations in sorted order.
- Uses a PostgreSQL advisory lock to avoid concurrent migration runs.
- Records applied migrations in `app.schema_migrations`.
- Stores SHA-256 checksums and rejects changes to an already-applied migration.
- Runs each migration transactionally.

The initial migration is:

```text
services/api/migrations/0001_initial_core.sql
```

It establishes the `core` schema and the initial project, design, and design-version data structures.

## Development checks

Run from `services/api`:

```powershell
go test ./...
go vet ./...
```

Both commands passed during the backend hardening verification on 23 November 2026. The project CRUD integration test also passed against the dedicated test database.

## Current backend verification

The following checks were completed during the local hardening session:

- PostgreSQL ownership assigned to the non-login `ahg_owner` role.
- Separate migration and runtime database credentials configured.
- Migration runner verified against the test database.
- Migration checksum verified against the main database.
- Project CRUD integration test passed.
- `go test ./...` passed.
- `go vet ./...` passed.
- `/healthz` returned `status: ok`.
- `/api/v1/ready` reported PostgreSQL and Redis as connected.
- PostgreSQL activity confirmed that API connections use `ahg_runtime`.

These results describe the local development environment at the time of verification; they are not a claim of production deployment or production certification.

## Development principles

- Keep dashboard and workspace UI changes separate from backend hardening work.
- Prefer incremental changes with tests.
- Back up databases before schema or role changes.
- Never commit secrets, local environment files, generated binaries, or database dumps.
- Verify migrations and data integrity before applying changes to important databases.

## Roadmap

The next proposed engineering phase is **Project Detail & Data Integrity Hardening**, covering:

1. Project → Design → Design Version relationship validation.
2. Foreign-key and data consistency checks.
3. Transaction boundaries and CRUD error handling.
4. Pagination, filtering, and API edge cases.
5. Automated negative-path and boundary-condition tests.

## License

No license has been specified yet. All rights are reserved unless and until a license is added to this repository.

