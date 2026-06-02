# BigBase Ubiquitous Language

## Core Domain

| Term | Definition |
|------|-----------|
| **BigBase** | The single-binary, open-source BaaS platform itself |
| **Entity** | The running BigBase server instance (ECC pattern) |
| **Component** | An independent submodule with its own lifecycle implementing `kernel.Component` interface |
| **Construct** | The configuration that decides which components run together |
| **Kernel** | Core orchestrator: discovers components, resolves dependencies, manages lifecycle |
| **Event Bus** | Pub/sub system within the Kernel for hook-based component communication |
| **Hook** | A named event point where a component subscribes (e.g., `onMutation`, `onRequest`, `onPush`) |
| **HookDef** | The struct defining a hook — its name, priority, and subscribe/unsubscribe methods |

## Infrastructure

| Term | Definition |
|------|-----------|
| **Proxy** | HTTP server component that routes requests to internal handlers |
| **DB** | Database component wrapping SQLite access with migration support |
| **Collection** | A named table in the database, auto-exposed via REST API |
| **Migration** | Versioned SQL DDL that transforms the database schema idempotently |

## Auth Domain

| Term | Definition |
|------|-----------|
| **JWT** | JSON Web Token (HS256) for stateless authentication |
| **Claims** | The payload of a JWT: `user_id`, `email`, `exp`, `iat` |
| **bcrypt** | Adaptive password hashing algorithm used for credential storage |
| **GoogleVerifier** | Interface for mockable Google OAuth token verification |
| **Token** | An authentication token (JWT access token or OAuth code) |

## Developer Tools Domain

| Term | Definition |
|------|-----------|
| **Forge** | Project management component — issues, labels, comments, kanban, wiki |
| **CICI** | CI/CD pipeline component — workflows, runs, steps |
| **Function** | A serverless JavaScript function (goja runtime) |
| **Runtime** | Execution sandbox for functions with timeout and console capture |
| **Realtime** | WebSocket component for live database mutation events |
| **Messaging** | Email, SMS, and push notification dispatching component |
| **Hub** | Connection manager inside Realtime — tracks WebSocket clients and channel subscriptions |
| **Channel** | Named subscription topic in Realtime (e.g., `collection:posts`) |

## Storage Domain

| Term | Definition |
|------|-----------|
| **Storage File** | An uploaded file with metadata (name, size, MIME, UUID) |
| **Thumbnail** | Auto-generated smaller version of an uploaded image |
| **Bucket** | Logical namespace for file organization (future) |

## Deploy / Sites Domain

| Term | Definition |
|------|-----------|
| **Deploy** | Process of building and running a web application from a Git repository |
| **Deployment** | A single running instance of a deployed app with its own port and URL |
| **Site** | A web application deployed from GitHub with automatic updates |
| **Preview URL** | Temporary URL for a deployment (UUID-based) |
| **Build Detection** | Auto-detection of project type (Node.js, Go, Python, Static) from repo contents |

## Git / GitHub Domain

| Term | Definition |
|------|-----------|
| **Git Repo** | An internal bare Git repository managed by the Git component |
| **GitHub App** | External GitHub App for listing private repos and receiving webhooks |
| **Webhook** | Inbound HTTP callback from GitHub (push events, PR events) |
| **Mirror** | A bare clone of a GitHub repository into internal Git storage |

## Monitoring Domain

| Term | Definition |
|------|-----------|
| **Metrics** | Numerical measurements of system and request activity |
| **MetricsCollector** | In-memory structure tracking request counts, latencies, status codes per endpoint |
| **Alert** | Configurable threshold-based notification rule |
| **Log Entry** | Structured log record stored in SQLite with level, message, and timestamp |
| **Health** | Liveness check endpoint (`/api/monitoring/health`) |

## Admin Domain

| Term | Definition |
|------|-----------|
| **Admin UI** | React SPA embedded in the Go binary for browser-based management |
| **Dashboard** | Home page of the Admin UI showing stats, navigation, and component status |
| **Data Studio** | UI for browsing and editing database collections and records |
| **Design Token** | CSS variable defining a semantic color, spacing, or typography value |
