# Service Layer Guide

Complete guide to creating, using, and testing services in the SaaS Starter API.

## Table of Contents

- [What is a Service](#what-is-a-service)
- [When to Create a Service](#when-to-create-a-service)
- [Service Pattern](#service-pattern)
- [Dependency Injection](#dependency-injection)
- [Service Composition](#service-composition)
- [Examples from Codebase](#examples-from-codebase)
- [Creating New Services](#creating-new-services)
- [Testing Services](#testing-services)
- [Best Practices](#best-practices)

## What is a Service

A **service** is a component that encapsulates business logic and coordinates operations across multiple layers of the application.

### Service vs. Resolver vs. Store

```
┌────────────────────────────────────────────────────────┐
│ RESOLVER (Thin - ~50 lines)                           │
│ • Authentication checks                                │
│ • Input validation                                     │
│ • Access control                                       │
│ • Service delegation                                   │
│ • Response formatting                                  │
└────────────────────────────────────────────────────────┘
                        ↓ delegates to
┌────────────────────────────────────────────────────────┐
│ SERVICE (Rich - business logic)                        │
│ • ALL business logic                                   │
│ • Data orchestration                                   │
│ • External API integration                             │
│ • Complex calculations                                 │
│ • Progress tracking                                    │
│ • Detailed error logging                               │
└────────────────────────────────────────────────────────┘
                        ↓ uses
┌────────────────────────────────────────────────────────┐
│ STORE (Data access)                                    │
│ • CRUD operations                                      │
│ • Query building                                       │
│ • Transaction management                               │
│ • Data mapping                                         │
└────────────────────────────────────────────────────────┘
```

### Key Characteristics

1. **Interface-First** - Always define interface before implementation
2. **Dependency Injection** - All dependencies passed via constructor
3. **Composable** - Services can use other services
4. **Testable** - Easy to mock and test in isolation
5. **Reusable** - Can be used by multiple resolvers or other services

## When to Create a Service

Create a service when:

### 1. Business Logic Complexity

Logic spans more than 50 lines or involves multiple steps:

```go
// ❌ Too complex for resolver
func (mr *mutationResolver) ProcessOrder(ctx context.Context, orderID string) {
    // 200 lines of order processing logic
    // calculate prices, apply discounts, validate inventory,
    // process payment, send emails, update analytics...
}

// ✅ Create OrderService
type OrderService interface {
    ProcessOrder(ctx context.Context, orderID string) (*Order, error)
}
```

### 2. Multiple Store Operations

Need to coordinate 2+ store calls:

```go
// ✅ Service orchestrates multiple stores
func (s *service) CreateProject(ctx context.Context, name, teamID string) error {
    // 1. Get team
    team, err := s.store.Teams().Get(ctx, teamID)

    // 2. Check subdomain availability
    exists, err := s.store.Projects().SubdomainExists(ctx, subdomain)

    // 3. Create project
    err = s.store.Projects().Create(ctx, project)

    // 4. Create default settings
    err = s.store.ProjectSettings().Create(ctx, settings)

    return nil
}
```

### 3. External API Integration

Calling external services (email, webhooks, etc.):

```go
// ✅ Service handles external integration
func (s *service) SendWelcomeEmail(ctx context.Context, userID string) error {
    user, err := s.store.Users().Get(ctx, userID)
    if err != nil {
        return err
    }

    // External API call
    return s.mailService.SendWelcomeEmail(ctx, user.Email, user.DisplayName)
}
```

### 4. Reusable Logic

Same logic used in multiple resolvers:

```go
// ✅ Service provides reusable logic
type AuthService interface {
    ValidateAndCreateUser(ctx context.Context, email string) (*User, error)
}

// Used by multiple resolvers:
// - SignUp mutation
// - OAuth callback mutation
// - Admin create user mutation
```

### 5. Testability

Complex logic that needs isolated testing:

```go
// ✅ Easy to test with mocks
func TestUserService_CreateUser(t *testing.T) {
    mockStore := &MockStore{}
    service := NewUserService(mockStore, mockLogger)

    user, err := service.CreateUser(ctx, "test@example.com")

    assert.NoError(t, err)
    assert.Equal(t, "test@example.com", user.Email)
}
```

### 6. Domain Boundaries

Clear domain concept (User management, Authentication, Billing, etc.):

```go
// ✅ Clear domain services
type UserService interface { ... }
type TeamService interface { ... }
type BillingService interface { ... }
type AnalyticsService interface { ... }
```

### When NOT to Create a Service

Don't create a service for:

- **Simple CRUD operations** - Can stay in resolver
- **Single store call with no logic** - Resolver can call store directly
- **Pure data transformation** - Use helper functions

```go
// ✅ No service needed - simple CRUD
func (qr *queryResolver) User(ctx context.Context, id string) (*gen.User, error) {
    currentUser := middleware.GetCurrentUser(ctx)
    if currentUser == nil {
        return nil, errors.New("unauthorized")
    }

    user, err := qr.Store.Users().Get(ctx, id)
    if err != nil {
        return nil, errors.New("user not found")
    }

    return toGraphQLUser(user), nil
}
```

## Service Pattern

### 1. Interface-First Design

Always define the interface first in `interface.go`:

```go
// res/userservice/interface.go
package userservice

import (
    "context"
    "saas-starter-api/res/store"
)

// UserService handles user-related business logic
type UserService interface {
    // CreateUser creates a new user with validation
    CreateUser(ctx context.Context, email, displayName string) (*store.User, error)

    // UpdateUser updates user information
    UpdateUser(ctx context.Context, userID string, updates *UserUpdates) (*store.User, error)

    // DeleteUser soft deletes a user and cleans up related data
    DeleteUser(ctx context.Context, userID string) error

    // GetUserStats calculates user statistics
    GetUserStats(ctx context.Context, userID string) (*UserStats, error)
}

// UserUpdates represents fields that can be updated
type UserUpdates struct {
    DisplayName *string
    Status      *string
}

// UserStats represents calculated user statistics
type UserStats struct {
    TeamCount    int
    ProjectCount int
    MemberOf     int
}
```

**Benefits:**
- Clear contract - consumers know what to expect
- Easy to mock - interface can be stubbed in tests
- Flexible - can swap implementations
- Documentation - interface is self-documenting

### 2. Implementation in service.go

Create unexported struct and factory function:

```go
// res/userservice/service.go
package userservice

import (
    "context"
    "fmt"
    "log"
    "strings"

    "saas-starter-api/res/mail"
    "saas-starter-api/res/notification"
    "saas-starter-api/res/store"
)

// service implements UserService interface
// Unexported - consumers use the interface
type service struct {
    store              store.Store
    mailService        mail.MailService
    notificationService notification.NotificationService
    logger             *log.Logger
}

// NewService creates a new UserService
// Returns interface, not concrete type
func NewService(
    store store.Store,
    mailService mail.MailService,
    notificationService notification.NotificationService,
    logger *log.Logger,
) UserService {
    return &service{
        store:              store,
        mailService:        mailService,
        notificationService: notificationService,
        logger:             logger,
    }
}

// CreateUser creates a new user with validation
func (s *service) CreateUser(
    ctx context.Context,
    email string,
    displayName string,
) (*store.User, error) {
    // 1. Validate inputs
    email = strings.TrimSpace(strings.ToLower(email))
    if !isValidEmail(email) {
        s.logger.Printf("Invalid email format: %s", email)
        return nil, fmt.Errorf("invalid email format")
    }

    if len(displayName) == 0 || len(displayName) > 50 {
        s.logger.Printf("Invalid display name length: %d", len(displayName))
        return nil, fmt.Errorf("display name must be 1-50 characters")
    }

    // 2. Check if user already exists
    existing, err := s.store.Users().GetByEmail(ctx, email)
    if err != nil && err != store.ErrNotFound {
        s.logger.Printf("Error checking existing user: %v", err)
        return nil, fmt.Errorf("failed to check existing user: %w", err)
    }
    if existing != nil {
        s.logger.Printf("User already exists: %s", email)
        return nil, fmt.Errorf("user already exists")
    }

    // 3. Create user
    user := &store.User{
        ID:          xid.New().String(),
        Email:       email,
        DisplayName: displayName,
        Status:      "ACTIVE",
    }

    if err := s.store.Users().Create(ctx, user); err != nil {
        s.logger.Printf("Error creating user: %v", err)
        return nil, fmt.Errorf("failed to create user: %w", err)
    }

    // 4. Register with mail service (optional)
    if s.mailService != nil {
        err := s.mailService.RegisterUser(ctx, user.ID, user.Email, user.DisplayName)
        if err != nil {
            s.logger.Printf("Failed to register user with mail service (non-fatal): %v", err)
            // Don't fail the request
        }
    }

    // 5. Send notification (optional)
    if s.notificationService != nil {
        go func() {
            err := s.notificationService.NotifyNewUserSignup(
                context.Background(),
                user.Email,
                user.DisplayName,
                user.ID,
            )
            if err != nil {
                s.logger.Printf("Failed to send signup notification: %v", err)
            }
        }()
    }

    s.logger.Printf("User created successfully: %s", user.ID)
    return user, nil
}

// UpdateUser updates user information
func (s *service) UpdateUser(
    ctx context.Context,
    userID string,
    updates *UserUpdates,
) (*store.User, error) {
    // Get existing user
    user, err := s.store.Users().Get(ctx, userID)
    if err != nil {
        s.logger.Printf("User not found: %s", userID)
        return nil, fmt.Errorf("user not found")
    }

    // Apply updates
    if updates.DisplayName != nil {
        if len(*updates.DisplayName) == 0 || len(*updates.DisplayName) > 50 {
            return nil, fmt.Errorf("display name must be 1-50 characters")
        }
        user.DisplayName = *updates.DisplayName
    }

    if updates.Status != nil {
        validStatuses := []string{"ACTIVE", "SUSPENDED", "PENDING"}
        if !contains(validStatuses, *updates.Status) {
            return nil, fmt.Errorf("invalid status")
        }
        user.Status = *updates.Status
    }

    // Save
    if err := s.store.Users().Update(ctx, user); err != nil {
        s.logger.Printf("Error updating user: %v", err)
        return nil, fmt.Errorf("failed to update user: %w", err)
    }

    return user, nil
}

// DeleteUser soft deletes a user and cleans up related data
func (s *service) DeleteUser(ctx context.Context, userID string) error {
    // 1. Get user
    user, err := s.store.Users().Get(ctx, userID)
    if err != nil {
        return fmt.Errorf("user not found")
    }

    // 2. Check if user owns any teams
    teams, err := s.store.Teams().ListByOwner(ctx, userID)
    if err != nil {
        s.logger.Printf("Error listing teams: %v", err)
        return fmt.Errorf("failed to check team ownership")
    }

    if len(teams) > 0 {
        s.logger.Printf("User owns teams, cannot delete: %s", userID)
        return fmt.Errorf("cannot delete user who owns teams")
    }

    // 3. Remove from mail service (optional, best effort)
    if s.mailService != nil {
        err := s.mailService.RemoveUserByEmail(ctx, user.Email)
        if err != nil {
            s.logger.Printf("Failed to remove user from mail service: %v", err)
            // Continue anyway
        }
    }

    // 4. Delete user (cascades to sessions, team memberships, etc.)
    if err := s.store.Users().Delete(ctx, userID); err != nil {
        s.logger.Printf("Error deleting user: %v", err)
        return fmt.Errorf("failed to delete user: %w", err)
    }

    s.logger.Printf("User deleted successfully: %s", userID)
    return nil
}

// GetUserStats calculates user statistics
func (s *service) GetUserStats(ctx context.Context, userID string) (*UserStats, error) {
    // Verify user exists
    _, err := s.store.Users().Get(ctx, userID)
    if err != nil {
        return nil, fmt.Errorf("user not found")
    }

    // Get owned teams count
    ownedTeams, err := s.store.Teams().ListByOwner(ctx, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to get owned teams: %w", err)
    }

    // Get member teams count
    memberTeams, err := s.store.Teams().ListByMember(ctx, userID)
    if err != nil {
        return nil, fmt.Errorf("failed to get member teams: %w", err)
    }

    // Get projects count across all teams
    projectCount := 0
    for _, team := range ownedTeams {
        projects, _ := s.store.Projects().ListByTeam(ctx, team.ID, 1000, 0)
        projectCount += len(projects)
    }

    return &UserStats{
        TeamCount:    len(ownedTeams),
        ProjectCount: projectCount,
        MemberOf:     len(memberTeams),
    }, nil
}

// Private helpers

func isValidEmail(email string) bool {
    // Simple email validation
    return strings.Contains(email, "@") && len(email) >= 3
}

func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}
```

### 3. Using Services in Resolvers

Inject service via resolver config and use in thin resolvers:

```go
// sys/graphql/graphql.go
type Config struct {
    Store               store.Store
    UserService         userservice.UserService  // Add service
    // ... other services
    Logger              *log.Logger
}

type Resolver struct {
    cfg *Config
}

func NewResolver(cfg *Config) *Resolver {
    return &Resolver{cfg: cfg}
}

// sys/graphql/user.go
func (mr *mutationResolver) UpdateCurrentUser(
    ctx context.Context,
    input gen.UpdateCurrentUserInput,
) (*gen.User, error) {
    // 1. Auth check
    currentUser := middleware.GetCurrentUser(ctx)
    if currentUser == nil {
        return nil, errors.New("unauthorized")
    }

    // 2. Build updates
    updates := &userservice.UserUpdates{
        DisplayName: input.DisplayName,
        Status:      input.Status,
    }

    // 3. Delegate to service
    user, err := mr.cfg.UserService.UpdateUser(ctx, currentUser.ID, updates)
    if err != nil {
        mr.cfg.Logger.Printf("Error updating user: %v", err)
        return nil, errors.New("failed to update user")
    }

    // 4. Format response
    return &gen.User{
        ID:          user.ID,
        Email:       user.Email,
        DisplayName: user.DisplayName,
        Status:      gen.UserStatus(user.Status),
    }, nil
}
```

## Dependency Injection

### Explicit Dependencies

All dependencies must be passed via constructor:

```go
// ✅ GOOD: Explicit dependencies
func NewService(
    store store.Store,
    mailService mail.MailService,
    notificationService notification.NotificationService,
    logger *log.Logger,
) UserService {
    return &service{
        store:              store,
        mailService:        mailService,
        notificationService: notificationService,
        logger:             logger,
    }
}

// ❌ BAD: Hidden dependencies
func NewService() UserService {
    // NEVER create dependencies internally
    store := postgresql.New(...)
    mail := sidemail.New(...)

    return &service{
        store: store,
        mail:  mail,
    }
}
```

### Optional Dependencies

Optional services should be nullable and checked before use:

```go
type service struct {
    store              store.Store              // Required
    mailService        mail.MailService        // Optional
    notificationService notification.NotificationService  // Optional
    logger             *log.Logger              // Required
}

func (s *service) CreateUser(ctx context.Context, email string) (*User, error) {
    // ... create user ...

    // Optional service - check before use
    if s.mailService != nil {
        err := s.mailService.RegisterUser(ctx, user.ID, user.Email)
        if err != nil {
            s.logger.Printf("Mail service failed (non-fatal): %v", err)
            // Don't return error - graceful degradation
        }
    } else {
        s.logger.Printf("Mail service not configured, skipping registration")
    }

    return user, nil
}
```

### Wiring Dependencies

Dependencies are wired in the main entry point:

```go
// cmd/main.go or api/api.go
func setupServices() {
    // Initialize store
    store := postgresql.NewStore(db)

    // Initialize optional services
    var mailService mail.MailService
    if apiKey := os.Getenv("SIDEMAIL_API_KEY"); apiKey != "" {
        mailService = sidemail.NewMailService(apiKey)
    }

    var notificationService notification.NotificationService
    if webhookURL := os.Getenv("SLACK_WEBHOOK_URL"); webhookURL != "" {
        notificationService = slack.NewNotificationService(webhookURL)
    }

    // Create services with dependencies
    userService := userservice.NewService(
        store,
        mailService,
        notificationService,
        logger,
    )

    // Create resolver with services
    resolver := graphql.NewResolver(&graphql.Config{
        Store:               store,
        UserService:         userService,
        // ... other services
        Logger:              logger,
    })
}
```

## Service Composition

Services can use other services to build complex workflows:

### Example: Email Verification Service

```go
// res/verification/interface.go
package verification

type VerificationService interface {
    SendVerificationEmail(ctx context.Context, userID string) error
    VerifyEmail(ctx context.Context, token string) (*store.User, error)
}

// res/verification/service.go
type service struct {
    store       store.Store
    auth        auth.Auth              // Compose auth service
    mailService mail.MailService       // Compose mail service
    userService userservice.UserService  // Compose user service
    logger      *log.Logger
}

func NewService(
    store store.Store,
    auth auth.Auth,
    mailService mail.MailService,
    userService userservice.UserService,
    logger *log.Logger,
) VerificationService {
    return &service{
        store:       store,
        auth:        auth,
        mailService: mailService,
        userService: userService,
        logger:      logger,
    }
}

func (s *service) SendVerificationEmail(
    ctx context.Context,
    userID string,
) error {
    // 1. Use user service to get user
    user, err := s.store.Users().Get(ctx, userID)
    if err != nil {
        return fmt.Errorf("user not found")
    }

    // 2. Use auth service to generate token
    token, err := s.auth.GenerateAccessToken(userID)
    if err != nil {
        s.logger.Printf("Failed to generate token: %v", err)
        return fmt.Errorf("token generation failed")
    }

    // 3. Use mail service to send email
    if s.mailService == nil {
        return fmt.Errorf("mail service not configured")
    }

    err = s.mailService.SendVerificationEmail(ctx, user.Email, token)
    if err != nil {
        s.logger.Printf("Failed to send verification email: %v", err)
        return fmt.Errorf("email sending failed")
    }

    s.logger.Printf("Verification email sent to: %s", user.Email)
    return nil
}

func (s *service) VerifyEmail(
    ctx context.Context,
    token string,
) (*store.User, error) {
    // 1. Use auth service to validate token
    claims := &jwt.StandardClaims{}
    err := s.auth.ValidateToken(token, claims)
    if err != nil {
        return nil, fmt.Errorf("invalid token")
    }

    // 2. Use user service to update status
    updates := &userservice.UserUpdates{
        Status: ptr("ACTIVE"),
    }

    user, err := s.userService.UpdateUser(ctx, claims.Subject, updates)
    if err != nil {
        s.logger.Printf("Failed to update user: %v", err)
        return nil, fmt.Errorf("verification failed")
    }

    return user, nil
}
```

### Benefits of Composition

- **Reusability** - Don't duplicate logic
- **Modularity** - Each service does one thing well
- **Testability** - Mock composed services
- **Maintainability** - Changes isolated to one service

## Examples from Codebase

### 1. Auth Service (`res/auth/`)

Handles authentication and token generation:

```go
type Auth interface {
    ValidateToken(token string, claims jwt.Claims) error
    GenerateAccessToken(userID string) (string, error)
    GenerateRefreshToken(userID, refreshTokenValue string) (string, error)
    AuthorizationWithGoogle(ctx context.Context, code string) (*AuthUserMetadata, error)
}
```

**Used by:**
- Auth resolvers for OAuth flow
- Middleware for token validation
- Other services for token generation

### 2. Mail Service (`res/mail/`)

Interface for email operations:

```go
type MailService interface {
    RegisterUser(ctx context.Context, userID, email, displayName string) error
    RemoveUserByEmail(ctx context.Context, email string) error
    UpdateContactProperty(ctx context.Context, email, propertyName, propertyValue string) error
}
```

**Implementation:** `res/mail/sidemail/`

**Used by:**
- User service for user lifecycle events
- Other services for email notifications

### 3. Notification Service (`res/notification/`)

Interface for notifications (Slack, etc.):

```go
type NotificationService interface {
    NotifyNewUserSignup(ctx context.Context, email, displayName, userID string) error
    SendFeedback(ctx context.Context, message, userID, userEmail string) error
}
```

**Implementation:** `res/notification/slack/`

**Used by:**
- User service for signup notifications
- Other services for important events

## Creating New Services

### Step-by-Step Guide

Let's create a `ProjectService` as an example:

#### Step 1: Create Directory

```bash
mkdir res/projectservice
```

#### Step 2: Define Interface

```go
// res/projectservice/interface.go
package projectservice

import (
    "context"
    "saas-starter-api/res/store"
)

type ProjectService interface {
    CreateProject(ctx context.Context, displayName, teamID string) (*store.Project, error)
    UpdateProject(ctx context.Context, projectID string, updates *ProjectUpdates) (*store.Project, error)
    DeleteProject(ctx context.Context, projectID string) error
    GetProjectWithStats(ctx context.Context, projectID string) (*ProjectWithStats, error)
}

type ProjectUpdates struct {
    DisplayName *string
    Subdomain   *string
}

type ProjectWithStats struct {
    Project    *store.Project
    MemberCount int
    TaskCount   int
}
```

#### Step 3: Implement Service

```go
// res/projectservice/service.go
package projectservice

import (
    "context"
    "fmt"
    "log"
    "strings"

    "github.com/rs/xid"

    "saas-starter-api/res/notification"
    "saas-starter-api/res/store"
)

type service struct {
    store              store.Store
    notificationService notification.NotificationService
    logger             *log.Logger
}

func NewService(
    store store.Store,
    notificationService notification.NotificationService,
    logger *log.Logger,
) ProjectService {
    return &service{
        store:              store,
        notificationService: notificationService,
        logger:             logger,
    }
}

func (s *service) CreateProject(
    ctx context.Context,
    displayName string,
    teamID string,
) (*store.Project, error) {
    // 1. Validate
    if len(displayName) == 0 || len(displayName) > 50 {
        return nil, fmt.Errorf("display name must be 1-50 characters")
    }

    // 2. Verify team exists
    team, err := s.store.Teams().Get(ctx, teamID)
    if err != nil {
        return nil, fmt.Errorf("team not found")
    }

    // 3. Generate subdomain
    subdomain := generateSubdomain(displayName)

    // 4. Check availability
    exists, err := s.store.Projects().SubdomainExists(ctx, subdomain)
    if err != nil {
        s.logger.Printf("Error checking subdomain: %v", err)
        return nil, fmt.Errorf("subdomain check failed")
    }
    if exists {
        subdomain = generateUniqueSubdomain(displayName)
    }

    // 5. Create project
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

    // 6. Optional notification
    if s.notificationService != nil {
        go s.notifyNewProject(context.Background(), project, team)
    }

    s.logger.Printf("Project created: %s", project.ID)
    return project, nil
}

// Helper methods
func generateSubdomain(displayName string) string {
    subdomain := strings.ToLower(displayName)
    subdomain = strings.ReplaceAll(subdomain, " ", "-")
    // Remove special characters...
    return subdomain
}

func generateUniqueSubdomain(displayName string) string {
    base := generateSubdomain(displayName)
    return fmt.Sprintf("%s-%s", base, xid.New().String()[:6])
}

func (s *service) notifyNewProject(ctx context.Context, project *store.Project, team *store.Team) {
    // Implementation...
}

// Implement other methods...
```

#### Step 4: Wire Into Resolver Config

```go
// sys/graphql/graphql.go
type Config struct {
    Store               store.Store
    ProjectService      projectservice.ProjectService  // Add
    // ...
}

// cmd/main.go or api/api.go
projectService := projectservice.NewService(store, notificationService, logger)

resolver := graphql.NewResolver(&graphql.Config{
    Store:          store,
    ProjectService: projectService,
    // ...
})
```

#### Step 5: Use in Resolver

```go
// sys/graphql/project.go
func (mr *mutationResolver) CreateProject(
    ctx context.Context,
    displayName string,
    teamID string,
) (*gen.Project, error) {
    currentUser := middleware.GetCurrentUser(ctx)
    if currentUser == nil {
        return nil, errors.New("unauthorized")
    }

    if err := mr.HasTeamAccess(ctx, teamID); err != nil {
        return nil, err
    }

    project, err := mr.cfg.ProjectService.CreateProject(ctx, displayName, teamID)
    if err != nil {
        mr.cfg.Logger.Printf("Error: %v", err)
        return nil, errors.New("failed to create project")
    }

    return toGraphQLProject(project), nil
}
```

## Testing Services

### Unit Testing with Mocks

```go
// res/projectservice/service_test.go
package projectservice

import (
    "context"
    "log"
    "os"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"

    "saas-starter-api/res/store"
)

// Mock store
type MockStore struct {
    mock.Mock
}

func (m *MockStore) Projects() store.ProjectStore {
    args := m.Called()
    return args.Get(0).(store.ProjectStore)
}

// Mock project store
type MockProjectStore struct {
    mock.Mock
}

func (m *MockProjectStore) Create(ctx context.Context, project *store.Project) error {
    args := m.Called(ctx, project)
    return args.Error(0)
}

func (m *MockProjectStore) SubdomainExists(ctx context.Context, subdomain string) (bool, error) {
    args := m.Called(ctx, subdomain)
    return args.Bool(0), args.Error(1)
}

// Test
func TestService_CreateProject(t *testing.T) {
    // Setup
    mockStore := &MockStore{}
    mockProjectStore := &MockProjectStore{}
    logger := log.New(os.Stdout, "", log.LstdFlags)

    mockStore.On("Projects").Return(mockProjectStore)
    mockProjectStore.On("SubdomainExists", mock.Anything, mock.Anything).Return(false, nil)
    mockProjectStore.On("Create", mock.Anything, mock.Anything).Return(nil)

    service := NewService(mockStore, nil, logger)

    // Execute
    project, err := service.CreateProject(
        context.Background(),
        "Test Project",
        "team123",
    )

    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, project)
    assert.Equal(t, "Test Project", project.DisplayName)
    assert.NotEmpty(t, project.Subdomain)

    mockStore.AssertExpectations(t)
    mockProjectStore.AssertExpectations(t)
}

func TestService_CreateProject_InvalidName(t *testing.T) {
    mockStore := &MockStore{}
    logger := log.New(os.Stdout, "", log.LstdFlags)
    service := NewService(mockStore, nil, logger)

    // Test empty name
    _, err := service.CreateProject(context.Background(), "", "team123")
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "display name")

    // Test too long name
    longName := strings.Repeat("a", 51)
    _, err = service.CreateProject(context.Background(), longName, "team123")
    assert.Error(t, err)
}

func TestService_CreateProject_SubdomainConflict(t *testing.T) {
    mockStore := &MockStore{}
    mockProjectStore := &MockProjectStore{}
    logger := log.New(os.Stdout, "", log.LstdFlags)

    mockStore.On("Projects").Return(mockProjectStore)
    // First subdomain exists, should generate unique one
    mockProjectStore.On("SubdomainExists", mock.Anything, "test-project").Return(true, nil)
    mockProjectStore.On("Create", mock.Anything, mock.Anything).Return(nil)

    service := NewService(mockStore, nil, logger)

    project, err := service.CreateProject(
        context.Background(),
        "Test Project",
        "team123",
    )

    assert.NoError(t, err)
    // Should have generated unique subdomain
    assert.NotEqual(t, "test-project", project.Subdomain)
    assert.Contains(t, project.Subdomain, "test-project-")
}
```

### Integration Testing

```go
func TestService_CreateProject_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Setup real database
    db := setupTestDatabase(t)
    defer cleanupTestDatabase(t, db)

    store := postgresql.NewStore(db)
    service := NewService(store, nil, logger)

    // Create test team first
    team := createTestTeam(t, store)

    // Test project creation
    project, err := service.CreateProject(
        context.Background(),
        "Integration Test Project",
        team.ID,
    )

    assert.NoError(t, err)
    assert.NotNil(t, project)

    // Verify in database
    retrieved, err := store.Projects().Get(context.Background(), project.ID)
    assert.NoError(t, err)
    assert.Equal(t, project.DisplayName, retrieved.DisplayName)
}
```

## Best Practices

### 1. Keep Services Focused

Each service should have a single, clear responsibility:

```go
// ✅ GOOD: Focused service
type UserService interface {
    CreateUser(...)
    UpdateUser(...)
    DeleteUser(...)
}

// ❌ BAD: God service
type ApplicationService interface {
    CreateUser(...)
    ProcessPayment(...)
    SendEmail(...)
    GenerateReport(...)
    // ... 50 more methods
}
```

### 2. Log Extensively

Services should log detailed information for debugging:

```go
func (s *service) CreateProject(ctx context.Context, name, teamID string) (*Project, error) {
    s.logger.Printf("Creating project: name=%s, teamID=%s", name, teamID)

    if err := s.validateName(name); err != nil {
        s.logger.Printf("Validation failed: %v", err)
        return nil, err
    }

    project, err := s.store.Projects().Create(ctx, &Project{...})
    if err != nil {
        s.logger.Printf("Database error creating project: %v", err)
        return nil, fmt.Errorf("failed to create project: %w", err)
    }

    s.logger.Printf("Project created successfully: id=%s", project.ID)
    return project, nil
}
```

### 3. Handle Errors Gracefully

Wrap errors with context, handle optional services gracefully:

```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to create project for team %s: %w", teamID, err)
}

// Optional services don't break main flow
if s.mailService != nil {
    err := s.mailService.SendNotification(...)
    if err != nil {
        s.logger.Printf("Mail notification failed (non-fatal): %v", err)
        // Don't return error
    }
}
```

### 4. Use Transactions for Multi-Step Operations

When operations must be atomic:

```go
func (s *service) TransferOwnership(ctx context.Context, teamID, newOwnerID string) error {
    return s.store.WithTransaction(ctx, func(tx store.Store) error {
        // 1. Update team owner
        if err := tx.Teams().UpdateOwner(ctx, teamID, newOwnerID); err != nil {
            return err
        }

        // 2. Remove old owner from members
        if err := tx.Teams().RemoveMember(ctx, teamID, oldOwnerID); err != nil {
            return err
        }

        // 3. Add new owner notification
        // ...

        return nil
    })
}
```

### 5. Make Services Composable

Build complex services from simpler ones:

```go
type OnboardingService struct {
    userService    UserService
    teamService    TeamService
    projectService ProjectService
    mailService    mail.MailService
}

func (s *OnboardingService) OnboardNewUser(ctx context.Context, email string) error {
    // Compose multiple services
    user, _ := s.userService.CreateUser(ctx, email)
    team, _ := s.teamService.CreateTeam(ctx, "My Team", user.ID)
    project, _ := s.projectService.CreateProject(ctx, "My First Project", team.ID)
    _ = s.mailService.SendWelcomeEmail(ctx, user.Email)

    return nil
}
```

### 6. Return Domain Errors

Define domain-specific errors:

```go
var (
    ErrProjectNotFound    = errors.New("project not found")
    ErrSubdomainTaken     = errors.New("subdomain already taken")
    ErrInvalidDisplayName = errors.New("invalid display name")
)

func (s *service) CreateProject(...) (*Project, error) {
    if exists {
        return nil, ErrSubdomainTaken
    }
    // ...
}

// Resolvers can check error types
if err == projectservice.ErrSubdomainTaken {
    return nil, errors.New("That subdomain is already taken. Please choose another.")
}
```

### 7. Document Public Interfaces

Add clear documentation to interfaces:

```go
// UserService handles user lifecycle and management.
type UserService interface {
    // CreateUser creates a new user with the given email and display name.
    // Returns an error if the email is already registered or invalid.
    CreateUser(ctx context.Context, email, displayName string) (*store.User, error)

    // UpdateUser updates user information. Only non-nil fields in updates are applied.
    // Returns an error if the user is not found or updates are invalid.
    UpdateUser(ctx context.Context, userID string, updates *UserUpdates) (*store.User, error)

    // DeleteUser permanently deletes a user and associated data.
    // Returns an error if the user owns teams (must transfer ownership first).
    DeleteUser(ctx context.Context, userID string) error
}
```

### 8. Keep Helpers Private

Helper methods should be unexported:

```go
// Public interface method
func (s *service) CreateProject(ctx context.Context, name string) (*Project, error) {
    subdomain := s.generateSubdomain(name)  // Private helper
    // ...
}

// Private helper (unexported)
func (s *service) generateSubdomain(name string) string {
    // Implementation
}
```

## Summary

Services are the heart of your business logic:

- **Define interfaces first** - Clear contracts
- **Inject dependencies** - No hidden coupling
- **Compose services** - Build complex from simple
- **Log extensively** - Help future debugging
- **Handle errors gracefully** - Don't break on optional failures
- **Test with mocks** - Easy unit testing
- **Keep focused** - Single responsibility

Follow these patterns and your services will be maintainable, testable, and scalable.

## Next Steps

- **Read [QUEUE_SYSTEM.md](QUEUE_SYSTEM.md)** - Learn async processing
- **Read [DEVELOPMENT.md](DEVELOPMENT.md)** - Development workflow
- **Study existing services** - See patterns in action
- **Build your first service** - Practice makes perfect

Services are your power tool. Use them wisely.
