# Architecture Documentation

Comprehensive guide to the SaaS Starter API architecture, design patterns, and system organization.

## Table of Contents

- [Overview](#overview)
- [Service-Oriented Architecture](#service-oriented-architecture)
- [Layer Breakdown](#layer-breakdown)
- [Request Flow](#request-flow)
- [Service Pattern](#service-pattern)
- [Dependency Injection](#dependency-injection)
- [Multi-Tenant Architecture](#multi-tenant-architecture)
- [Queue System Architecture](#queue-system-architecture)
- [Why Thin Resolvers](#why-thin-resolvers)
- [Design Principles](#design-principles)

## Overview

This template follows a **service-oriented architecture** with clear separation of concerns. The architecture is designed to be:

- **Maintainable** - Clear separation of concerns
- **Testable** - Interface-based design enables mocking
- **Scalable** - Horizontal scaling through stateless design
- **Extensible** - Easy to add new features without breaking existing code

### Core Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                         HTTP Layer                          │
│  ┌───────────────────────────────────────────────────────┐  │
│  │              Middleware                               │  │
│  │  • Authentication (JWT verification)                  │  │
│  │  • CORS (Cross-Origin Resource Sharing)              │  │
│  │  • CSP (Content Security Policy)                     │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│                      GraphQL Layer                          │
│  ┌───────────────────────────────────────────────────────┐  │
│  │         Resolvers (Thin - 50 lines max)              │  │
│  │  • Authentication checks                             │  │
│  │  • Input validation                                  │  │
│  │  • Access control                                    │  │
│  │  • Service delegation                                │  │
│  │  • Response formatting                               │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│                      Service Layer                          │
│  ┌───────────────────────────────────────────────────────┐  │
│  │         Business Logic (Rich Services)               │  │
│  │  • ALL business logic                                │  │
│  │  • Data orchestration                                │  │
│  │  • External API integration                          │  │
│  │  • Complex calculations                              │  │
│  │  • Progress tracking                                 │  │
│  │  • Error handling and logging                        │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│                       Store Layer                           │
│  ┌───────────────────────────────────────────────────────┐  │
│  │         Data Access (GORM + PostgreSQL)              │  │
│  │  • CRUD operations                                   │  │
│  │  • Query building                                    │  │
│  │  • Transaction management                            │  │
│  │  • Data mapping                                      │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                           ↓
┌─────────────────────────────────────────────────────────────┐
│                    PostgreSQL Database                      │
└─────────────────────────────────────────────────────────────┘
```

## Service-Oriented Architecture

### Golden Rule: Thin Resolvers, Rich Services

The fundamental principle of this architecture is:

```
❌ NEVER put business logic in resolvers
✅ ALWAYS create services for business logic
```

### Why Service-Oriented?

1. **Testability** - Services can be tested independently with mock stores
2. **Reusability** - Services can be composed and reused across resolvers
3. **Maintainability** - Business logic is centralized, not scattered
4. **Flexibility** - Easy to change implementations without affecting consumers
5. **Clear Boundaries** - Each layer has a single, well-defined responsibility

## Layer Breakdown

### 1. HTTP Layer (`sys/http/`)

**Responsibility:** Handle incoming HTTP requests and apply middleware

**Components:**
- `middleware/authmiddleware.go` - JWT token verification, extract user from context
- `middleware/corsmiddleware.go` - CORS headers for cross-origin requests
- `middleware/cspmiddleware.go` - Content Security Policy headers

**Example Flow:**
```
1. Request arrives → CORS middleware (add headers)
2. → Auth middleware (verify JWT, extract user)
3. → CSP middleware (security headers)
4. → GraphQL handler
```

**Code Location:**
```
sys/http/
├── middleware/
│   ├── authmiddleware.go    # JWT verification
│   ├── corsmiddleware.go    # CORS configuration
│   ├── cspmiddleware.go     # CSP headers
│   └── util.go              # Shared utilities
```

### 2. GraphQL Layer (`sys/graphql/`)

**Responsibility:** Handle GraphQL operations (queries/mutations) with minimal logic

**Components:**
- Schema files (`*.graphql`) - Type definitions
- Resolver files (`*.go`) - Thin resolver implementations
- Directives (`directive/`) - Custom GraphQL directives
- Scalars (`scalar/`) - Custom scalar types
- Generated code (`gen/`) - gqlgen output

**Resolver Responsibilities (ONLY):**
1. Authentication checks - Verify user is logged in
2. Input validation - Check parameters are valid
3. Access control - Verify user can perform action
4. Service delegation - Call appropriate service methods
5. Response formatting - Convert service results to GraphQL types

**Maximum Resolver Size: ~50 lines**

**Example Resolver:**
```go
// sys/graphql/project.go
func (mr *mutationResolver) CreateProject(
    ctx context.Context,
    displayName string,
    teamID string,
) (*gen.Project, error) {
    // 1. Auth check (5 lines)
    currentUser := middleware.GetCurrentUser(ctx)
    if currentUser == nil {
        return nil, errors.New("unauthorized")
    }

    // 2. Input validation (5 lines)
    if len(displayName) == 0 || len(displayName) > 50 {
        return nil, errors.New("invalid project name")
    }

    // 3. Access check (5 lines)
    if err := mr.HasTeamAccess(ctx, teamID); err != nil {
        return nil, err
    }

    // 4. Delegate to service (10 lines)
    project, err := mr.ProjectService.CreateProject(
        ctx,
        displayName,
        teamID,
    )
    if err != nil {
        mr.Logger.Printf("Error creating project: %v", err)
        return nil, errors.New("failed to create project")
    }

    // 5. Format response (5 lines)
    return &gen.Project{
        ID:          project.ID,
        DisplayName: project.DisplayName,
        Subdomain:   project.Subdomain,
    }, nil
}
```

**Code Location:**
```
sys/graphql/
├── *.graphql              # Schema definitions
├── auth.go                # Auth resolvers
├── user.go                # User resolvers
├── team.go                # Team resolvers
├── project.go             # Project resolvers
├── invitationcode.go      # Invitation code resolvers
├── access_helpers.go      # Shared access checks
├── gen/                   # Generated code
│   ├── gen.go
│   └── model.go
├── directive/
│   └── authrequired.go    # @authRequired directive
└── scalar/
    ├── void.go            # Void scalar
    └── timeinterval.go    # TimeInterval scalar
```

### 3. Service Layer (`res/`)

**Responsibility:** ALL business logic, data orchestration, and external integrations

**Service Responsibilities:**
1. ALL business logic and domain rules
2. Data orchestration across multiple stores
3. External API integration (email, notifications, etc.)
4. Complex calculations and transformations
5. Progress tracking and status updates
6. Detailed error logging
7. Transaction coordination

**Example Service:**
```go
// res/projectservice/service.go
type service struct {
    store              store.Store
    mailService        mail.MailService
    notificationService notification.NotificationService
    queueManager       *queue.Manager
    logger             *log.Logger
}

func (s *service) CreateProject(
    ctx context.Context,
    displayName string,
    teamID string,
) (*store.Project, error) {
    // 1. Validate business rules
    if err := s.validateProjectName(displayName); err != nil {
        s.logger.Printf("Invalid project name: %s", displayName)
        return nil, err
    }

    // 2. Generate subdomain
    subdomain := s.generateSubdomain(displayName)

    // 3. Check subdomain availability
    exists, err := s.store.Projects().SubdomainExists(ctx, subdomain)
    if err != nil {
        s.logger.Printf("Error checking subdomain: %v", err)
        return nil, fmt.Errorf("subdomain check failed: %w", err)
    }
    if exists {
        subdomain = s.generateUniqueSubdomain(displayName)
    }

    // 4. Create project
    project := &store.Project{
        ID:          xid.New().String(),
        DisplayName: displayName,
        Subdomain:   subdomain,
        TeamID:      teamID,
    }

    if err := s.store.Projects().Create(ctx, project); err != nil {
        s.logger.Printf("Error creating project: %v", err)
        return nil, fmt.Errorf("failed to create project: %w", err)
    }

    // 5. Send notifications (optional, non-blocking)
    if s.notificationService != nil {
        go s.notificationService.NotifyNewProject(ctx, project)
    }

    // 6. Register with mail service (optional)
    if s.mailService != nil {
        err := s.mailService.UpdateContactProperty(
            ctx,
            project.ID,
            "project_count",
            "1",
        )
        if err != nil {
            s.logger.Printf("Failed to update mail service: %v", err)
            // Don't fail the request, just log
        }
    }

    return project, nil
}
```

**Code Location:**
```
res/
├── auth/                  # Authentication service
│   ├── auth.go
│   ├── google.go
│   └── jwt.go
├── mail/                  # Email service
│   ├── mail.go           # Interface
│   └── sidemail/         # Sidemail implementation
│       └── mail.go
├── notification/          # Notification service
│   ├── interface.go
│   └── slack/            # Slack implementation
│       └── slack.go
└── queue/                 # Queue system
    └── queue.go
```

### 4. Store Layer (`res/store/`)

**Responsibility:** Data access and persistence (database operations)

**Store Responsibilities:**
1. CRUD operations
2. Query building and execution
3. Transaction management
4. Data mapping (DB ↔ Go structs)
5. Database-specific logic

**Design Pattern:**
- Interface definitions in `res/store/*.go`
- PostgreSQL implementation in `res/store/postgresql/*.go`
- Easy to swap implementations (e.g., add MySQL, MongoDB)

**Example Store Interface:**
```go
// res/store/project.go
type ProjectStore interface {
    Create(ctx context.Context, project *Project) error
    Get(ctx context.Context, id string) (*Project, error)
    Update(ctx context.Context, project *Project) error
    Delete(ctx context.Context, id string) error

    GetBySubdomain(ctx context.Context, subdomain string) (*Project, error)
    SubdomainExists(ctx context.Context, subdomain string) (bool, error)
    ListByTeam(ctx context.Context, teamID string, limit, offset int) ([]*Project, error)
}
```

**Example Store Implementation:**
```go
// res/store/postgresql/project.go
type projectStore struct {
    db *gorm.DB
}

func (s *projectStore) Create(ctx context.Context, project *Project) error {
    return s.db.WithContext(ctx).Create(project).Error
}

func (s *projectStore) Get(ctx context.Context, id string) (*Project, error) {
    var project Project
    err := s.db.WithContext(ctx).First(&project, "id = ?", id).Error
    if err == gorm.ErrRecordNotFound {
        return nil, ErrNotFound
    }
    return &project, err
}
```

**Code Location:**
```
res/store/
├── store.go               # Main store interface
├── user.go                # User store interface
├── team.go                # Team store interface
├── project.go             # Project store interface
├── authsession.go         # Auth session store interface
├── invitationcode.go      # Invitation code store interface
├── task.go                # Task store interface
├── errors.go              # Store errors
├── postgresql/            # PostgreSQL implementation
│   ├── store.go          # Main store implementation
│   ├── user.go           # User CRUD
│   ├── team.go           # Team CRUD
│   ├── project.go        # Project CRUD
│   ├── authsession.go    # Auth session CRUD
│   ├── invitationcode.go # Invitation code CRUD
│   ├── task_store.go     # Task CRUD
│   ├── *getter.go        # Optimized getters
│   └── ...
└── migrations/            # Database migrations
    ├── 001_create_users_table.sql
    ├── 002_create_auth_sessions_table.sql
    └── ...
```

### 5. Database Layer (PostgreSQL)

**Responsibility:** Data persistence and integrity

**Features:**
- ACID transactions
- Foreign key constraints
- Indexes for performance
- Cascading deletes
- JSONB for flexible data (task payloads)

**Schema Overview:**
```sql
-- User → Team → Project hierarchy
users
  ├── auth_sessions (FK: user_id)
  ├── teams (FK: owner_id)
  ├── team_members (FK: user_id)
  └── team_member_invites (FK: invited_by)

teams
  ├── projects (FK: team_id)
  └── team_members (FK: team_id)

-- Other tables
tasks (for queue system)
invitation_codes (for beta access)
```

## Request Flow

### Query Flow Example: Get Current User

```
1. HTTP Request
   GET /api
   Headers: Authorization: Bearer <JWT>
   Body: { query: "{ currentUser { id email } }" }

2. Middleware Layer
   ├─ CORS: Add CORS headers
   ├─ Auth: Verify JWT token
   │  └─ Extract user ID from token
   │  └─ Load user from database
   │  └─ Store in context.Context
   └─ CSP: Add security headers

3. GraphQL Layer
   Query: currentUser
   Resolver: sys/graphql/user.go:CurrentUser()
   ├─ Get user from context (set by middleware)
   ├─ Return user (already loaded)
   └─ Format as GraphQL response

4. HTTP Response
   Status: 200 OK
   Body: {
     "data": {
       "currentUser": {
         "id": "abc123",
         "email": "user@example.com"
       }
     }
   }
```

### Mutation Flow Example: Create Project

```
1. HTTP Request
   POST /api
   Headers: Authorization: Bearer <JWT>
   Body: {
     mutation: "createProject(displayName: 'My App', teamID: 'team123')"
   }

2. Middleware Layer
   ├─ Verify JWT
   ├─ Load current user
   └─ Store in context

3. GraphQL Resolver (sys/graphql/project.go)
   Mutation: createProject
   ├─ Extract current user from context
   ├─ Validate inputs (name length, etc.)
   ├─ Check team access (user owns/member of team)
   └─ Delegate to service ↓

4. Service Layer (res/projectservice/)
   Method: CreateProject()
   ├─ Validate business rules
   ├─ Generate subdomain
   ├─ Check subdomain availability (call store)
   ├─ Create project (call store)
   ├─ Send notifications (optional)
   ├─ Update mail service (optional)
   └─ Return project ↑

5. Store Layer (res/store/postgresql/)
   Method: Projects().Create()
   ├─ Map Go struct to database model
   ├─ Execute INSERT query
   ├─ Handle errors (unique constraint, etc.)
   └─ Return result ↑

6. Database (PostgreSQL)
   ├─ Validate constraints
   ├─ Insert row
   ├─ Check foreign keys
   └─ Commit transaction

7. Response Path (back up)
   Database ↑ Store ↑ Service ↑ Resolver ↑ GraphQL ↑ HTTP

8. HTTP Response
   Status: 200 OK
   Body: {
     "data": {
       "createProject": {
         "id": "proj123",
         "displayName": "My App",
         "subdomain": "my-app"
       }
     }
   }
```

### Background Job Flow Example: Data Export

```
1. HTTP Request
   Mutation: startDataExport(projectID: "proj123")

2. Resolver
   ├─ Auth & access checks
   └─ Enqueue task ↓

3. Queue Manager (res/queue/)
   Method: Enqueue()
   ├─ Create task record in database
   ├─ Set status: PENDING
   ├─ Store payload as JSONB
   └─ Return task ID ↑

4. Response (immediate)
   Return: { taskID: "task456", status: "PENDING" }

5. Background Worker (separate process)
   ├─ Poll queue for PENDING tasks
   ├─ Dequeue task (SKIP LOCKED)
   ├─ Set status: IN_PROGRESS
   └─ Process task ↓

6. Service Layer
   Method: ProcessDataExport()
   ├─ Fetch data from database
   ├─ Transform data
   ├─ Update progress: 25%
   ├─ Generate export file
   ├─ Update progress: 50%
   ├─ Upload to storage
   ├─ Update progress: 75%
   ├─ Send email notification
   ├─ Update progress: 100%
   └─ Mark task as COMPLETED

7. Client Polling
   Query: task(id: "task456") { status, progress }
   Response: { status: "COMPLETED", progress: 100 }
```

## Service Pattern

### Interface-First Design

Always define the interface before implementation:

```go
// res/myservice/interface.go
package myservice

type MyService interface {
    DoSomething(ctx context.Context, id string) (*Result, error)
}

// res/myservice/service.go
package myservice

// Unexported implementation
type service struct {
    store  store.Store
    logger *log.Logger
}

// Factory function returns interface
func NewService(store store.Store, logger *log.Logger) MyService {
    return &service{
        store:  store,
        logger: logger,
    }
}

func (s *service) DoSomething(ctx context.Context, id string) (*Result, error) {
    // Implementation
}
```

**Benefits:**
- Easy to mock for testing
- Clear contracts
- Flexibility to swap implementations
- Forces thinking about API design

### Service Composition

Services can compose other services:

```go
type EmailVerificationService struct {
    Store           store.Store
    AuthService     auth.Auth
    MailService     mail.MailService
    NotificationSvc notification.NotificationService
    Logger          *log.Logger
}

func (s *EmailVerificationService) SendVerification(
    ctx context.Context,
    userID string,
) error {
    // Orchestrate multiple services
    user, err := s.Store.Users().Get(ctx, userID)
    if err != nil {
        return err
    }

    token, err := s.AuthService.GenerateAccessToken(user.ID)
    if err != nil {
        return err
    }

    err = s.MailService.SendVerificationEmail(ctx, user.Email, token)
    if err != nil {
        return err
    }

    // Optional notification
    if s.NotificationSvc != nil {
        s.NotificationSvc.NotifyNewVerification(ctx, user.Email)
    }

    return nil
}
```

## Dependency Injection

All dependencies are passed via constructors:

```go
// ✅ GOOD: Explicit dependencies
func NewService(
    store store.Store,
    mailService mail.MailService,
    logger *log.Logger,
) MyService {
    return &service{
        store:       store,
        mailService: mailService,
        logger:      logger,
    }
}

// ❌ BAD: Hidden dependencies
func NewService() MyService {
    store := postgresql.New()  // NEVER DO THIS
    return &service{store: store}
}
```

**Benefits:**
- Testability - inject mocks
- Flexibility - swap implementations
- Clear dependencies - no hidden coupling
- No global state

## Multi-Tenant Architecture

### Hierarchy Model

```
User (owns)
  └── Team
      ├── Team Members (users)
      └── Projects
```

### Access Control Pattern

Every mutation that affects team/project resources must verify access:

```go
// 1. Get the resource
project, err := mr.Store.Projects().Get(ctx, projectID)
if err != nil {
    return nil, errors.New("project not found")
}

// 2. Get the team
team, err := mr.Store.Teams().Get(ctx, project.TeamID)
if err != nil {
    return nil, errors.New("team not found")
}

// 3. Check ownership or membership
currentUser := middleware.GetCurrentUser(ctx)

if team.OwnerID != currentUser.ID {
    // Not owner, check membership
    member, _ := mr.Store.Teams().GetMember(ctx, team.ID, currentUser.ID)
    if member == nil {
        return nil, errors.New("access denied")
    }
}

// 4. Proceed with mutation
```

### Database Relationships

```sql
-- User owns teams
CREATE TABLE teams (
    id TEXT PRIMARY KEY,
    owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ...
);

-- Users can be members of teams
CREATE TABLE team_members (
    id TEXT PRIMARY KEY,
    team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    ...
);

-- Projects belong to teams
CREATE TABLE projects (
    id TEXT PRIMARY KEY,
    team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    ...
);
```

## Queue System Architecture

### Design

Database-backed task queue with:
- Task persistence in PostgreSQL
- Status tracking (pending → in_progress → completed/failed)
- Progress updates (0-100%)
- Automatic retry logic
- SKIP LOCKED for worker coordination

### Task Lifecycle

```
1. ENQUEUE
   Client → Resolver → QueueManager.Enqueue()
   ├─ Create task record
   ├─ Status: PENDING
   ├─ Progress: 0
   └─ Return task ID

2. DEQUEUE
   Worker → QueueManager.Dequeue()
   ├─ SELECT ... FOR UPDATE SKIP LOCKED
   ├─ Update status: IN_PROGRESS
   ├─ Set started_at timestamp
   └─ Return task

3. PROCESSING
   Worker → Service.ProcessTask()
   ├─ Process items
   ├─ Update progress periodically
   │  └─ QueueManager.UpdateProgress(taskID, 50)
   └─ Complete or fail

4. COMPLETION
   Success: QueueManager.CompleteTask()
   ├─ Status: COMPLETED
   ├─ Progress: 100
   └─ Set completed_at timestamp

   Failure: QueueManager.FailTask()
   ├─ Increment retry_count
   ├─ If retry_count < max_retries:
   │  └─ Status: PENDING (retry)
   ├─ Else:
   │  └─ Status: FAILED (permanent)
   └─ Set error_message
```

### When to Use Queue

Use for operations that:
- Take > 2 seconds
- Require background processing
- Are batch operations
- Need retry logic
- Require progress tracking for UI

See [QUEUE_SYSTEM.md](QUEUE_SYSTEM.md) for detailed documentation.

## Why Thin Resolvers

### Problems with Fat Resolvers

```go
// ❌ BAD: Fat resolver (200+ lines)
func (mr *mutationResolver) CreateProject(...) {
    // Direct store access
    team, _ := mr.Store.Teams().Get(ctx, teamID)

    // Business logic
    subdomain := generateSubdomain(displayName)
    if exists, _ := mr.Store.Projects().SubdomainExists(ctx, subdomain); exists {
        subdomain = generateUniqueSubdomain(displayName)
    }

    // More business logic
    project := &store.Project{...}
    mr.Store.Projects().Create(ctx, project)

    // External API calls
    mr.MailService.RegisterProject(ctx, project)
    mr.SlackService.NotifyNewProject(ctx, project)

    // Even more logic...
}
```

**Problems:**
- Hard to test (need to mock everything)
- Logic scattered across resolvers
- Difficult to reuse
- Tight coupling
- Hard to maintain

### Solution: Thin Resolvers + Rich Services

```go
// ✅ GOOD: Thin resolver (30 lines)
func (mr *mutationResolver) CreateProject(...) {
    // Auth check
    currentUser := middleware.GetCurrentUser(ctx)
    if currentUser == nil {
        return nil, errors.New("unauthorized")
    }

    // Access check
    if err := mr.HasTeamAccess(ctx, teamID); err != nil {
        return nil, err
    }

    // Delegate to service
    project, err := mr.ProjectService.CreateProject(ctx, displayName, teamID)
    if err != nil {
        mr.Logger.Printf("Error: %v", err)
        return nil, errors.New("failed to create project")
    }

    return toGraphQLProject(project), nil
}

// Service handles ALL business logic
func (s *projectService) CreateProject(...) {
    // ALL logic here
}
```

**Benefits:**
- Easy to test services with mocks
- Logic centralized and reusable
- Clear separation of concerns
- Loose coupling
- Maintainable codebase

## Design Principles

### 1. Single Responsibility

Each layer has ONE job:
- HTTP: Handle requests, apply middleware
- GraphQL: Route operations, validate, delegate
- Services: Business logic
- Stores: Data access

### 2. Interface Segregation

Define small, focused interfaces:

```go
// ✅ GOOD: Specific interfaces
type UserReader interface {
    Get(ctx context.Context, id string) (*User, error)
}

type UserWriter interface {
    Create(ctx context.Context, user *User) error
    Update(ctx context.Context, user *User) error
}

// ❌ BAD: God interface
type UserStore interface {
    Get(...)
    Create(...)
    Update(...)
    Delete(...)
    List(...)
    Search(...)
    // ... 20 more methods
}
```

### 3. Dependency Inversion

Depend on interfaces, not concrete types:

```go
// ✅ GOOD
type Service struct {
    Store store.Store  // Interface
}

// ❌ BAD
type Service struct {
    Store *postgresql.Store  // Concrete type
}
```

### 4. Open/Closed Principle

Open for extension, closed for modification:

```go
// Easy to add new implementations
type NotificationService interface {
    Notify(ctx context.Context, message string) error
}

// Add Slack implementation
type SlackNotifier struct {}
func (s *SlackNotifier) Notify(...) error {}

// Add Email implementation
type EmailNotifier struct {}
func (e *EmailNotifier) Notify(...) error {}

// Compose them
type MultiNotifier struct {
    notifiers []NotificationService
}
```

### 5. Explicit Over Implicit

Be explicit about dependencies and behavior:

```go
// ✅ GOOD: Clear what's needed
func NewService(
    store store.Store,
    mail mail.MailService,
    logger *log.Logger,
) Service

// ❌ BAD: Hidden dependencies
func NewService() Service
```

### 6. Fail Fast

Validate early, fail with clear errors:

```go
// Validate in resolver
if len(name) == 0 {
    return nil, errors.New("name is required")
}

// Validate in service
if !isValidEmail(email) {
    return fmt.Errorf("invalid email: %s", email)
}
```

### 7. Graceful Degradation

Optional services should not break core functionality:

```go
// ✅ GOOD: Optional service
if s.NotificationService != nil {
    err := s.NotificationService.Notify(ctx, msg)
    if err != nil {
        s.Logger.Printf("Notification failed (non-fatal): %v", err)
        // Continue, don't return error
    }
}

// ❌ BAD: Required optional service
err := s.NotificationService.Notify(ctx, msg)
if err != nil {
    return err  // Breaks main flow!
}
```

## Summary

This architecture provides:

- **Clear Structure** - Each layer has a well-defined role
- **Testability** - Interface-based design enables easy mocking
- **Maintainability** - Business logic is centralized in services
- **Scalability** - Stateless design enables horizontal scaling
- **Flexibility** - Easy to add new features or swap implementations

Follow the patterns in this document and [CLAUDE_RULES.md](../CLAUDE_RULES.md) to maintain architectural consistency.

## Next Steps

- **Read [SERVICES.md](SERVICES.md)** - Learn how to create and use services
- **Read [QUEUE_SYSTEM.md](QUEUE_SYSTEM.md)** - Understand background job processing
- **Read [DEVELOPMENT.md](DEVELOPMENT.md)** - Learn the development workflow
- **Study existing code** - See patterns in action

The architecture is your foundation. Build on it wisely.
