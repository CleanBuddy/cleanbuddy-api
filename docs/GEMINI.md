# CleanBuddy API - Development Rules for Gemini

This document outlines the development rules and best practices for the CleanBuddy API, specifically tailored for interaction with the Gemini AI agent. Following these guidelines will ensure that Gemini can effectively understand, assist, and contribute to the project in a consistent and high-quality manner.

## 🤖 INTERACTING WITH GEMINI

To get the most out of Gemini, consider the following when making requests:

*   **Be Clear and Specific:** Clearly articulate your goal, the context, and any specific requirements. Avoid ambiguity.
*   **Provide Context:** When referring to code, provide relevant file paths or code snippets. Explain the "why" behind your request.
*   **State the Desired Outcome:** What do you expect Gemini to achieve? (e.g., "fix a bug," "implement a feature," "refactor this code to improve performance").
*   **Reference Existing Rules:** If your request relates to a specific rule in this document, mention it.
*   **Verify Changes:** Gemini will provide its changes. Always review and verify them.

---

## 🏗️ ARCHITECTURE: SERVICE-ORIENTED DESIGN (MANDATORY)

### Golden Rule: Thin Resolvers, Rich Services

```
❌ NEVER put business logic in resolvers
✅ ALWAYS create services for business logic
```

### Resolver Responsibilities (ONLY)
1.  **Authentication/authorization checks** - Verify user identity and permissions
2.  **Input validation** - Check GraphQL input parameters
3.  **Service initialization and orchestration** - Create and call appropriate services
4.  **Response formatting** - Convert service results to GraphQL types
5.  **Error translation** - Map service errors to user-friendly GraphQL errors

**Maximum Resolver Size: ~50 lines**

### Service Responsibilities (EVERYTHING ELSE)
1.  **ALL business logic and domain rules**
2.  **Data orchestration** across multiple stores
3.  **External API integration** (email, notifications, etc.)
4.  **Complex calculations and transformations**
5.  **Progress tracking and status updates**
6.  **Detailed error logging**

---

## 📋 SERVICE PATTERN (ENFORCE STRICTLY)

### 1. Interface-First Design

```go
// ALWAYS define interface first in interface.go
package myservice

type MyService interface {
    DoSomething(ctx context.Context, id string) (*Result, error)
    DoAnotherThing(ctx context.Context, input Input) error
}

// Implementation in service.go (unexported struct)
type service struct {
    store       store.Store
    otherSvc    OtherService
    logger      *log.Logger
}

// Factory function returns interface
func NewService(store store.Store, otherSvc OtherService, logger *log.Logger) MyService {
    return &service{
        store:    store,
        otherSvc: otherSvc,
        logger:   logger,
    }
}
```

**Why:**
- Easy mocking for tests
- Clear contracts
- Flexibility to swap implementations

### 2. Dependency Injection (MANDATORY)

```go
✅ GOOD: All dependencies via constructor
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

❌ BAD: Creating dependencies internally
func NewService() MyService {
    store := postgresql.New() // NEVER DO THIS
    return &service{store: store}
}
```

**Why:**
- Testability (inject mocks)
- Flexibility (swap implementations)
- Clear dependencies
- No hidden coupling

### 3. Service Composition

```go
// Services can compose other services
type EmailVerificationService struct {
    Store           store.Store
    AuthService     auth.Auth
    MailService     mail.MailService
    NotificationSvc notification.NotificationService
    Logger          *log.Logger
}

func (s *EmailVerificationService) SendVerification(ctx context.Context, userID string) error {
    // Orchestrate multiple services
    user, err := s.Store.Users().Get(ctx, userID)
    if err != nil {
        s.Logger.Printf("Failed to get user: %v", err)
        return fmt.Errorf("user not found")
    }

    token, err := s.AuthService.GenerateAccessToken(user.ID)
    if err != nil {
        s.Logger.Printf("Failed to generate token: %v", err)
        return fmt.Errorf("token generation failed")
    }

    err = s.MailService.SendVerificationEmail(ctx, user.Email, token)
    if err != nil {
        s.Logger.Printf("Failed to send email: %v", err)
        return fmt.Errorf("email sending failed")
    }

    if s.NotificationSvc != nil {
        s.NotificationSvc.NotifyNewVerification(ctx, user.Email)
    }

    return nil
}
```

**Benefits:**
- Complex workflows hidden behind simple interfaces
- Reusable service components
- Clear separation of concerns

---

## 🎯 RESOLVER PATTERN (KEEP THIN)

### Example of Thin Resolver (Good)

```go
func (mr *mutationResolver) ProcessData(ctx context.Context, id string) (*gen.Result, error) {
    // 1. Auth check (5 lines)
    currentUser := middleware.GetCurrentUser(ctx)
    if currentUser == nil {
        return nil, errors.New("unauthorized")
    }

    // 2. Access check (5 lines)
    if err := mr.HasAccess(ctx, id); err != nil {
        return nil, err
    }

    // 3. Delegate to service (1 line)
    result, err := mr.MyService.ProcessData(ctx, id)
    if err != nil {
        mr.Logger.Printf("Error processing: %v", err)
        return nil, errors.New("processing failed")
    }

    // 4. Format response (5 lines)
    return toGQLResult(result), nil
}
```

### Example of Fat Resolver (BAD - Don't Do This)

```go
❌ DON'T DO THIS
func (mr *mutationResolver) ProcessData(ctx context.Context, id string) (*gen.Result, error) {
    currentUser := middleware.GetCurrentUser(ctx)

    // Direct store access (should be in service)
    data, _ := mr.Store.Data().Get(ctx, id)

    // Business logic (should be in service)
    if data.Status == "pending" {
        processed := complexProcessing(data) // 50 lines of logic
        data.ProcessedText = processed
    }

    // More business logic (should be in service)
    if shouldNotify(data) {
        mr.NotificationService.Notify(ctx, data)
    }

    // Direct save (should be in service)
    mr.Store.Data().Update(ctx, data)

    return toGQLResult(data), nil
}
```

---

## 🚀 QUEUE SYSTEM (FOR ASYNC OPERATIONS)

### When to Use Queue

Use the queue system for operations that:
- Take > 2 seconds
- Require background processing
- Are batch operations
- Need retry logic
- Require progress tracking for UI

### Queue Pattern

```go
// In resolver: Enqueue task
func (mr *mutationResolver) StartImport(ctx context.Context, projectID string) (*gen.Task, error) {
    // Auth and permission checks
    currentUser := middleware.GetCurrentUser(ctx)
    if err := mr.HasProjectAccess(ctx, projectID); err != nil {
        return nil, err
    }

    // Enqueue task
    payload := map[string]string{"project_id": projectID}
    taskID, err := mr.QueueManager.Enqueue(ctx, queue.TaskTypeImport, payload)
    if err != nil {
        mr.Logger.Printf("Failed to enqueue task: %v", err)
        return nil, errors.New("failed to start import")
    }

    return &gen.Task{ID: taskID, Status: "PENDING"}, nil
}

// In service: Process with progress updates
func (s *ImportService) ProcessImport(ctx context.Context, task *queue.Task) error {
    items, err := s.fetchItems(ctx)
    if err != nil {
        return fmt.Errorf("failed to fetch items: %w", err)
    }

    total := len(items)

    for i, item := range items {
        // Process individual item
        if err := s.processItem(ctx, item); err != nil {
            s.Logger.Printf("Failed to process item %d: %v", i, err)
            continue // Don't fail entire batch
        }

        // Update progress for UI
        progress := int(float64(i+1) / float64(total) * 100)
        s.QueueManager.UpdateProgress(ctx, task.ID, progress)
    }

    return s.QueueManager.CompleteTask(ctx, task.ID)
}
```

### Queue Best Practices

- **Always update progress** for long operations (users love feedback)
- **Handle partial failures gracefully** (don't fail entire batch on one error)
- **Log errors before calling FailTask** (helps debugging)
- **Use EnqueueIfNotExists** to prevent duplicate tasks
- **Set appropriate max_retries** (default is 3, but consider your use case)
- **Clean up old tasks** periodically to avoid bloat

---

## 📝 CODE STYLE

### Naming Conventions

```go
// Services
type UserService interface { ... }           // Interface
type userService struct { ... }              // Implementation (unexported)

// Files
interface.go     // Service interface
service.go       // Main implementation
helpers.go       // Private helpers (optional)
types.go         // Domain types (optional)

// Variables
userStore        // Full words, descriptive
projectID        // Not projID or pid
teamMemberRole   // Not tmRole

// Constants
const (
    StatusActive   = "active"   // Exported
    StatusPending  = "pending"
)
```

### File Organization

```go
res/myservice/
├── interface.go          # Public interface
├── service.go           # Main implementation
├── helpers.go           # Private helpers (optional)
└── types.go             # Domain types (optional)
```

### Import Grouping

```go
import (
    // 1. Standard library
    "context"
    "fmt"
    "time"

    // 2. External packages (alphabetical)
    "github.com/rs/xid"
    "gorm.io/gorm"

    // 3. Internal packages (alphabetical)
    "saas-starter-api/res/auth"
    "saas-starter-api/res/store"
)
```

---

## 🔒 SECURITY

### Always Verify Ownership

```go
// Before any mutation
project, err := mr.Store.Projects().Get(ctx, projectID)
if err != nil {
    return nil, errors.New("project not found")
}

// Check if user owns the project's team
team, err := mr.Store.Teams().Get(ctx, project.TeamID)
if err != nil {
    return nil, errors.New("team not found")
}

if team.OwnerID != currentUser.ID {
    // Also check if user is a member
    member, _ := mr.Store.Teams().GetMember(ctx, team.ID, currentUser.ID)
    if member == nil {
        return nil, errors.New("access denied")
    }
}
```

### Never Expose Internal Errors

```go
// In Services: Detailed logging
s.Logger.Printf("Failed to connect to database: connection timeout after 30s, host: %s", dbHost)
return fmt.Errorf("database connection failed: %w", err)

// In Resolvers: Generic user-facing error
mr.Logger.Printf("Error in CreateProject: %v", err)
return nil, errors.New("failed to create project")
```

### Validate All Inputs

```go
func (mr *mutationResolver) CreateTeam(ctx context.Context, displayName string) (*gen.Team, error) {
    // Length validation
    if len(displayName) == 0 || len(displayName) > 50 {
        return nil, errors.New("team name must be between 1 and 50 characters")
    }

    // Character validation
    if !validTeamNameRegex.MatchString(displayName) {
        return nil, errors.New("team name contains invalid characters")
    }

    // Proceed with creation
    ...
}
```

### Use HTTP-Only Cookies for Tokens

```go
// Never send tokens in response body for storage in localStorage
// Use HTTP-only cookies or let the frontend handle token storage securely
```

---

## 🧪 TESTING

### Test Services with Mock Stores

```go
func TestUserService_CreateUser(t *testing.T) {
    // Create mock store
    mockStore := &MockStore{
        users: &MockUserStore{},
    }

    // Create service with mock
    service := NewUserService(mockStore, mockLogger)

    // Test service
    user, err := service.CreateUser(ctx, "test@example.com")
    assert.NoError(t, err)
    assert.Equal(t, "test@example.com", user.Email)
}
```

### Test Resolvers with Mock Services

```go
func TestMutationResolver_CreateTeam(t *testing.T) {
    // Create mock service
    mockService := &MockTeamService{}

    // Create resolver
    resolver := &mutationResolver{
        TeamService: mockService,
        Logger:      mockLogger,
    }

    // Test resolver
    team, err := resolver.CreateTeam(ctx, "Test Team")
    assert.NoError(t, err)
}
```

### Integration Tests for Critical Paths

```go
func TestAuthFlow_Integration(t *testing.T) {
    // Use test database
    testDB := setupTestDatabase(t)
    defer testDB.Cleanup()

    // Test full flow: signup → login → refresh
    // ...
}
```

---

## 🏗️ BUILD & CODE GENERATION

### GraphQL Code Generation

**ALWAYS regenerate after schema changes:**

```bash
cd sys/graphql
go run github.com/99designs/gqlgen generate
```

**When to regenerate:**
- ✅ After modifying any `.graphql` file in `sys/graphql/`
- ✅ After adding new types, queries, or mutations
- ✅ After changing field signatures
- ✅ After pulling git changes that modified schemas

**Generated files (NEVER EDIT MANUALLY):**
- `sys/graphql/gen/gen.go` - Resolver interfaces and types (auto-generated)
- `sys/graphql/gen/model.go` - GraphQL model types (auto-generated)

**Common errors:**
```bash
# Error: resolver not found
# Solution: Implement missing resolver method in appropriate resolver file

# Error: signature mismatch
# Solution: Update resolver signature to match generated interface

# Error: type not found
# Solution: Check GraphQL schema syntax in .graphql files
```

### Development Workflow

**Starting the dev server:**

```bash
# Option 1: With .env loading (recommended)
./run.sh

# Option 2: Direct run
go run cmd/main.go

# Option 3: Hot reload (install first: go install github.com/cosmtrek/air@latest)
air
```

**Development requirements:**
- PostgreSQL must be running
- `.env` file must be configured
- Database migrations must be executed
- GraphQL code must be generated

**GraphQL Playground (development only):**
- Access at `http://localhost:8080/api/playground`
- NOT available in production
- Use for testing queries and mutations

### Build Commands Reference

```bash
# Dependencies
go mod download              # Install dependencies
go mod tidy                  # Clean up go.mod and go.sum

# Code Generation
go run github.com/99designs/gqlgen generate  # Generate GraphQL code

# Development
./run.sh                     # Start with .env
go run cmd/main.go          # Direct start
air                         # Hot reload (if installed)

# Testing
go test ./...               # Run all tests
go test -v ./res/store/...  # Test specific package
go test -cover ./...        # With coverage

# Building
go build -o server cmd/main.go  # Build binary
go clean -cache                  # Clear build cache
```

### Pre-Commit Checklist

Before committing GraphQL changes:

- [ ] Run `go run github.com/99designs/gqlgen generate`
- [ ] Fix any compilation errors
- [ ] Implement all required resolvers
- [ ] Server starts successfully (`go run cmd/main.go`)
- [ ] Test queries/mutations in GraphQL Playground
- [ ] Run tests (`go test ./...`)

### Common Build Issues

**Issue: "cannot find package"**
```bash
# Solution
go mod download
go mod tidy
```

**Issue: "resolver not found"**
```bash
# You forgot to implement a resolver method
# Check sys/graphql/gen/gen.go for required method signature
# Implement in appropriate resolver file (e.g., sys/graphql/user.go)
```

**Issue: Stale generated code**
```bash
# Delete and regenerate
rm -rf sys/graphql/gen/*
go run github.com/99designs/gqlgen generate
```

**Issue: Database connection failed**
```bash
# Check PostgreSQL is running
pg_isready

# Verify DATABASE_POSTGRES_URL in .env
echo $DATABASE_POSTGRES_URL

# Test connection
psql $DATABASE_POSTGRES_URL -c "SELECT 1"
```

---

## 🗄️ DATABASE

### Always Create Migrations

```bash
❌ NEVER: gorm.AutoMigrate() in production
✅ ALWAYS: Create explicit migration files
```

### Migration Naming

```
XXX_descriptive_name.sql

Examples:
001_create_users_table.sql
002_add_user_status_column.sql
003_create_teams_table.sql
```

### Use Appropriate CASCADE Rules

```sql
-- Delete user → delete all their teams
FOREIGN KEY (owner_id) REFERENCES users(id)
    ON DELETE CASCADE

-- Delete team → set owner to NULL (keep record)
FOREIGN KEY (owner_id) REFERENCES users(id)
    ON DELETE SET NULL
```

### Index Foreign Keys

```sql
CREATE INDEX idx_projects_team_id ON projects(team_id);
CREATE INDEX idx_team_members_user_id ON team_members(user_id);
CREATE INDEX idx_team_members_team_id ON team_members(team_id);
```

### Use GORM Tags for Constraints

```go
type User struct {
    ID          string `gorm:"primaryKey;size:50;unique"`
    Email       string `gorm:"size:256;not null;unique"`
    DisplayName string `gorm:"size:50;not null"`
}
```

---

## 🎨 GRAPHQL

### Schema Organization

```
One file per domain:
- auth.graphql       # Authentication
- user.graphql       # User management
- team.graphql       # Team operations
- project.graphql    # Project operations
- gqlcommon.graphql  # Shared types
```

### Use Input Types for Mutations

```graphql
input CreateTeamInput {
    displayName: String!
}

extend type Mutation {
    createTeam(input: CreateTeamInput!): Team! @authRequired
}
```

### Use Connection Types for Lists

```graphql
type TeamConnection {
    edges: [TeamEdge!]!
    totalCount: Int!
}

type TeamEdge {
    node: Team!
    cursor: String!
}
```

### Apply @authRequired Directive

```graphql
type User {
    id: ID!
    email: String! @authRequired
    ownedTeams: [Team!]! @authRequired
}
```

### Document All Fields

```graphql
"""
A team is a group of users working together on projects.
Teams have an owner and can have multiple members.
"""
type Team {
    "Unique identifier for the team"
    id: ID!

    "Display name of the team (1-50 characters)"
    displayName: String!

    "User who owns the team"
    owner: User! @authRequired
}
```

---

## ⚠️ ERROR HANDLING

### In Services: Detailed Errors

```go
// Log with context
s.Logger.Printf("Failed to create user: email=%s, error=%v", email, err)

// Return wrapped error
return fmt.Errorf("failed to create user %s: %w", userID, err)
```

### In Resolvers: Generic User-Facing Errors

```go
// Log detailed error
mr.Logger.Printf("Error in CreateProject: user=%s, project=%s, error=%v",
    currentUser.ID, projectID, err)

// Return generic error
return nil, errors.New("failed to create project")
```

### Graceful Degradation

```go
// Optional services should fail gracefully
if s.MailService != nil {
    err := s.MailService.SendWelcomeEmail(ctx, user.Email)
    if err != nil {
        s.Logger.Printf("Failed to send welcome email (non-fatal): %v", err)
        // Don't return error, continue
    }
} else {
    s.Logger.Printf("Mail service not configured, skipping welcome email")
}
```

---

## 🎯 WHEN TO CREATE A SERVICE

Create a new service when:

1.  **Business Logic Complexity**: Logic spans > 50 lines
2.  **Multiple Store Operations**: Need to coordinate 2+ store calls
3.  **External API Integration**: Calling external services (email, webhooks, etc.)
4.  **Reusable Logic**: Same logic used in multiple resolvers
5.  **Testability**: Complex logic that needs isolated testing
6.  **Domain Boundaries**: Clear domain concept (UserService, TeamService, etc.)

Don't create a service for:
- Simple CRUD operations (can stay in resolver)
- Single store call with no logic
- Pure data transformation

---

## 📚 EXAMPLE: Adding a New Feature

### Scenario: Add email verification

**1. Create Service Interface**

```go
// res/verification/interface.go
package verification

type VerificationService interface {
    SendVerificationEmail(ctx context.Context, userID string) error
    VerifyEmail(ctx context.Context, token string) (*store.User, error)
}
```

**2. Implement Service**

```go
// res/verification/service.go
package verification

type service struct {
    store       store.Store
    auth        auth.Auth
    mailService mail.MailService
    logger      *log.Logger
}

func NewService(
    store store.Store,
    auth auth.Auth,
    mailService mail.MailService,
    logger *log.Logger,
) VerificationService {
    return &service{
        store:       store,
        auth:        auth,
        mailService: mailService,
        logger:      logger,
    }
}

func (s *service) SendVerificationEmail(ctx context.Context, userID string) error {
    user, err := s.store.Users().Get(ctx, userID)
    if err != nil {
        return fmt.Errorf("failed to get user: %w", err)
    }

    token, err := s.auth.GenerateAccessToken(userID)
    if err != nil {
        return fmt.Errorf("failed to generate token: %w", err)
    }

    if s.mailService == nil {
        s.logger.Printf("Mail service not configured")
        return errors.New("mail service unavailable")
    }

    err = s.mailService.SendVerificationEmail(ctx, user.Email, token)
    if err != nil {
        return fmt.Errorf("failed to send email: %w", err)
    }

    return nil
}
```

**3. Add to GraphQL Config**

```go
// sys/graphql/graphql.go
type Config struct {
    ...
    VerificationService verification.VerificationService
}
```

**4. Create Thin Resolver**

```go
// sys/graphql/verification.go
func (mr *mutationResolver) SendVerificationEmail(ctx context.Context) error {
    currentUser := middleware.GetCurrentUser(ctx)
    if currentUser == nil {
        return errors.New("unauthorized")
    }

    err := mr.VerificationService.SendVerificationEmail(ctx, currentUser.ID)
    if err != nil {
        mr.Logger.Printf("Failed to send verification: %v", err)
        return errors.New("failed to send verification email")
    }

    return nil
}
```

---

## 🚨 COMMON MISTAKES TO AVOID

1.  **Fat Resolvers**: Moving business logic into resolvers
2.  **No Service Layer**: Calling store directly from resolvers
3.  **Creating Dependencies**: Services creating their own dependencies
4.  **Exposing Errors**: Returning internal errors to clients
5.  **No Access Checks**: Forgetting to verify ownership
6.  **Synchronous Long Operations**: Not using queue for slow operations
7.  **No Progress Updates**: Long operations without UI feedback
8.  **Hard Dependencies**: Requiring optional services
9.  **No Validation**: Accepting any input without checks
10. **Poor Error Logging**: Not logging enough context

---

## ✅ CHECKLIST FOR NEW FEATURES

Before submitting code, verify:

-   [ ] Business logic is in a service, not resolver
-   [ ] Service has interface-first design
-   [ ] All dependencies injected via constructor
-   [ ] Resolvers are thin (~50 lines max)
-   [ ] Access checks implemented
-   [ ] Input validation added
-   [ ] Errors logged with context
-   [ ] User-facing errors are generic
-   [ ] Optional services handled gracefully
-   [ ] Long operations use queue system
-   [ ] Progress updates implemented
-   [ ] GraphQL schema documented
-   [ ] Migration created (if DB changes)
-   [ ] Tests added (unit + integration)

---

## 🎓 REMEMBER

**The goal is maintainable, testable, scalable code.**

Service-oriented architecture achieves this by:
- Clear separation of concerns
- Easy testing (mock interfaces)
- Reusable components
- Flexible architecture
- Independent scaling

**When in doubt, ask:**
1. Is this logic in the right place?
2. Can I test this easily?
3. Is this resolver thin enough?
4. Are my services composable?

Follow these rules and you'll build a solid, maintainable SaaS API! 🚀
