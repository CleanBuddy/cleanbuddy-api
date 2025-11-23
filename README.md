# SaaS Starter API

A production-ready Go backend template for building SaaS applications with GraphQL, PostgreSQL, and a service-oriented architecture.

## Overview

This is a batteries-included backend starter kit designed to help you build scalable SaaS applications quickly. It provides a solid foundation with authentication, multi-tenancy (teams), project management, and an asynchronous queue system for background jobs.

## Key Features

- **GraphQL API** - Type-safe API with code generation using gqlgen
- **Multi-Tenant Architecture** - Built-in User → Team → Project hierarchy
- **Service-Oriented Design** - Clean separation of concerns with thin resolvers and rich services
- **Background Queue System** - Async task processing with progress tracking and retry logic
- **Authentication** - Google OAuth2 with JWT tokens (access + refresh)
- **PostgreSQL + GORM** - Production-ready database layer with migrations
- **Email Integration** - Mail service interface with Sidemail implementation
- **Slack Notifications** - Optional notification service for user events
- **Developer Friendly** - Hot reload support, GraphQL playground, comprehensive tooling

## Tech Stack

- **Language:** Go 1.24+
- **API:** GraphQL (gqlgen)
- **Database:** PostgreSQL 14+ with GORM ORM
- **Authentication:** JWT with OAuth2 (Google)
- **Queue:** Database-backed task queue with progress tracking
- **Deployment:** Vercel-ready serverless functions

## Quick Start

### Prerequisites

- Go 1.23 or higher
- PostgreSQL 14 or higher
- Google OAuth2 credentials (for authentication)

### Installation

1. **Clone the repository**
```bash
git clone <your-repo-url>
cd saas-starter-api
```

2. **Install dependencies**
```bash
go mod download
```

3. **Set up environment variables**

Create a `.env` file or set the following environment variables:

```bash
# Server
PORT=8080
ENVIRONMENT=development

# Database
DATABASE_URL=postgresql://user:password@localhost:5432/dbname

# Authentication
JWT_SECRET=your-secret-key-here
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret
GOOGLE_REDIRECT_URL=http://localhost:3000/auth/callback

# Optional: Email (Sidemail)
SIDEMAIL_API_KEY=your-sidemail-key

# Optional: Notifications (Slack)
SLACK_WEBHOOK_URL=your-slack-webhook-url
```

4. **Create the database**
```bash
createdb your_database_name
```

5. **Run migrations**
```bash
# Migrations are in res/store/migrations/
# Run them in order using psql or your preferred migration tool
psql $DATABASE_URL -f res/store/migrations/001_create_users_table.sql
psql $DATABASE_URL -f res/store/migrations/002_create_auth_sessions_table.sql
# ... run all migration files in order
```

6. **Start the development server**
```bash
go run cmd/main.go
```

The API will be available at `http://localhost:8080/api`

GraphQL Playground (dev only): `http://localhost:8080/api/playground`

## Project Structure

```
saas-starter-api/
├── cmd/                      # Application entry points
│   └── main.go              # Main server entry point
├── api/                      # HTTP handlers
│   ├── api.go               # Main GraphQL endpoint handler
│   └── playground/          # GraphQL playground (dev only)
├── sys/                      # System layer (HTTP, GraphQL)
│   ├── http/
│   │   └── middleware/      # Auth, CORS, CSP middleware
│   └── graphql/             # GraphQL resolvers and schema
│       ├── *.graphql        # GraphQL schema files
│       ├── *.go             # Resolver implementations
│       ├── gen/             # Generated GraphQL code
│       ├── directive/       # Custom directives (@authRequired)
│       └── scalar/          # Custom scalar types
├── res/                      # Resources layer (services, stores)
│   ├── auth/                # Authentication service
│   ├── mail/                # Email service (Sidemail)
│   ├── notification/        # Notification service (Slack)
│   ├── queue/               # Task queue system
│   └── store/               # Data access layer
│       ├── *.go             # Store interfaces
│       ├── postgresql/      # PostgreSQL implementation
│       └── migrations/      # Database migrations
├── docs/                     # Documentation
│   ├── SETUP.md             # Setup guide
│   ├── ARCHITECTURE.md      # Architecture overview
│   ├── SERVICES.md          # Service layer guide
│   ├── QUEUE_SYSTEM.md      # Queue system docs
│   ├── API_REFERENCE.md     # GraphQL API reference
│   ├── DEVELOPMENT.md       # Development workflow
│   └── DEPLOYMENT.md        # Deployment guide
├── CLAUDE_RULES.md          # Development patterns and rules
├── go.mod                   # Go module definition
└── go.sum                   # Go module checksums
```

## Architecture Overview

This template follows a **service-oriented architecture** with clear separation of concerns:

```
HTTP Request
    ↓
Middleware (Auth, CORS)
    ↓
GraphQL Resolver (thin - auth & validation only)
    ↓
Service Layer (business logic)
    ↓
Store Layer (database operations)
    ↓
PostgreSQL Database
```

**Key Principles:**

1. **Thin Resolvers** - Resolvers handle only auth, validation, and delegation to services
2. **Rich Services** - All business logic lives in services
3. **Interface-First Design** - Services and stores use interfaces for testability
4. **Dependency Injection** - All dependencies passed via constructors

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for detailed architecture documentation.

## Multi-Tenant Model

The template includes a built-in multi-tenant hierarchy:

```
User (owner)
  └── Team (owned by user)
      ├── Team Members (users with roles)
      └── Projects (belongs to team)
```

- **Users** can own multiple teams and be members of other teams
- **Teams** have one owner and multiple members
- **Projects** belong to teams (access controlled by team membership)
- **Invitation Codes** for closed beta access control

## Core Features

### Authentication

- Google OAuth2 authentication flow
- JWT-based access tokens (3 days) and refresh tokens (2 weeks)
- Automatic user creation on first sign-in
- Session management with database-backed auth sessions

### Queue System

Background task processing with:
- Task enqueueing with JSON payloads
- Progress tracking (0-100%)
- Automatic retry logic (configurable max retries)
- Status tracking (pending, in_progress, completed, failed)
- Duplicate prevention with `EnqueueIfNotExists`

See [docs/QUEUE_SYSTEM.md](docs/QUEUE_SYSTEM.md) for details.

### GraphQL API

- Type-safe schema with code generation
- Custom directives (`@authRequired`)
- Connection-based pagination
- Comprehensive error handling
- Optional GraphQL Playground (disabled in production)

See [docs/API_REFERENCE.md](docs/API_REFERENCE.md) for full API documentation.

## Documentation

- **[Setup Guide](docs/SETUP.md)** - Detailed installation and configuration
- **[Architecture](docs/ARCHITECTURE.md)** - System design and patterns
- **[Services](docs/SERVICES.md)** - Service layer development guide
- **[Queue System](docs/QUEUE_SYSTEM.md)** - Background job processing
- **[API Reference](docs/API_REFERENCE.md)** - GraphQL API documentation
- **[Development](docs/DEVELOPMENT.md)** - Development workflow and tools
- **[Deployment](docs/DEPLOYMENT.md)** - Production deployment guide
- **[Development Rules](CLAUDE_RULES.md)** - Code patterns and best practices

## Development

### Hot Reload with Air

Install Air for automatic reloading during development:

```bash
go install github.com/cosmtrek/air@latest
air
```

### Code Generation

After modifying GraphQL schema files:

```bash
go run github.com/99designs/gqlgen generate
```

### Running Tests

```bash
go test ./...
```

### Creating Migrations

```bash
# Create a new migration file
touch res/store/migrations/009_your_migration_name.sql
```

See [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) for complete development workflow.

## Deployment

This template is designed for Vercel serverless deployment but can be deployed anywhere Go runs.

See [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) for complete deployment instructions.

## Environment Variables Reference

| Variable | Required | Description |
|----------|----------|-------------|
| `PORT` | Yes | Server port (e.g., 8080) |
| `ENVIRONMENT` | Yes | Environment (development/production) |
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `JWT_SECRET` | Yes | Secret key for JWT signing |
| `GOOGLE_CLIENT_ID` | Yes | Google OAuth2 client ID |
| `GOOGLE_CLIENT_SECRET` | Yes | Google OAuth2 client secret |
| `GOOGLE_REDIRECT_URL` | Yes | OAuth2 callback URL |
| `SIDEMAIL_API_KEY` | No | Sidemail API key for emails |
| `SLACK_WEBHOOK_URL` | No | Slack webhook for notifications |

## Extending the Template

### Adding a New Feature

1. Define service interface in `res/yourservice/interface.go`
2. Implement service in `res/yourservice/service.go`
3. Update GraphQL schema in `sys/graphql/yourfeature.graphql`
4. Generate code: `go run github.com/99designs/gqlgen generate`
5. Create thin resolver in `sys/graphql/yourfeature.go`
6. Add migrations if database changes are needed

See [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) for detailed examples.

### Using the Queue System

For long-running operations (>2 seconds):

```go
// In resolver: Enqueue task
taskID, err := mr.QueueManager.Enqueue(ctx, queue.TaskTypeYourTask, payload)

// In service: Process with progress updates
for i, item := range items {
    // Process item...
    progress := int(float64(i+1) / float64(total) * 100)
    s.QueueManager.UpdateProgress(ctx, task.ID, progress)
}
```

See [docs/QUEUE_SYSTEM.md](docs/QUEUE_SYSTEM.md) for complete usage.

## Best Practices

- **Keep resolvers thin** (~50 lines max) - delegate to services
- **Use services for business logic** - don't put logic in resolvers
- **Interface-first design** - define interfaces before implementations
- **Dependency injection** - pass dependencies via constructors
- **Validate inputs** - always validate in resolvers before calling services
- **Check access** - verify ownership before mutations
- **Log errors with context** - help debugging in production
- **Use the queue** - for operations taking >2 seconds

See [CLAUDE_RULES.md](CLAUDE_RULES.md) for comprehensive development guidelines.

## Troubleshooting

### Common Issues

**Database connection fails**
- Verify `DATABASE_URL` is correct
- Ensure PostgreSQL is running
- Check database exists: `psql -l`

**Authentication fails**
- Verify Google OAuth2 credentials are correct
- Check redirect URL matches Google Console configuration
- Ensure JWT_SECRET is set

**GraphQL playground not available**
- Playground is disabled in production (`ENVIRONMENT=production`)
- Use `ENVIRONMENT=development` for local development

See [docs/SETUP.md](docs/SETUP.md) for more troubleshooting help.

## Contributing

1. Follow the patterns in [CLAUDE_RULES.md](CLAUDE_RULES.md)
2. Keep resolvers thin, services rich
3. Write tests for new services
4. Create migrations for database changes
5. Update documentation for new features

## License

MIT License - see LICENSE file for details

## Support

- Check the [docs/](docs/) folder for detailed documentation
- Review [CLAUDE_RULES.md](CLAUDE_RULES.md) for development patterns
- Look at existing code for examples

---

Built with Go, GraphQL, and PostgreSQL. Ready for production.
