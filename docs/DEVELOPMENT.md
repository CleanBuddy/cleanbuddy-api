# Development Guide

Complete guide to developing features and maintaining the SaaS Starter API.

## Table of Contents

- [Development Workflow](#development-workflow)
- [Adding New Features](#adding-new-features)
- [Creating Migrations](#creating-migrations)
- [Adding Services](#adding-services)
- [Updating GraphQL Schema](#updating-graphql-schema)
- [Running Tests](#running-tests)
- [Debugging Tips](#debugging-tips)
- [Code Generation](#code-generation)
- [Git Workflow](#git-workflow)
- [Code Review Checklist](#code-review-checklist)

## Development Workflow

### Daily Development Flow

```bash
# 1. Start PostgreSQL
brew services start postgresql@14  # macOS
sudo systemctl start postgresql     # Linux

# 2. Start dev server with hot reload
air

# 3. Open GraphQL Playground
# http://localhost:8080/api/playground

# 4. Make changes and test
# Air automatically reloads on file changes

# 5. Run tests before committing
go test ./...

# 6. Commit changes
git add .
git commit -m "feat: add new feature"
```

### Project Structure Reference

```
saas-starter-api/
├── cmd/                    # Entry points
│   └── main.go            # Main server
├── api/                    # HTTP handlers
│   └── api.go             # GraphQL endpoint
├── sys/                    # System layer
│   ├── http/middleware/   # Middleware
│   └── graphql/           # GraphQL resolvers & schema
├── res/                    # Resources
│   ├── auth/              # Auth service
│   ├── mail/              # Mail service
│   ├── notification/      # Notification service
│   ├── queue/             # Queue system
│   └── store/             # Data layer
└── docs/                   # Documentation
```

## Adding New Features

Follow this process for adding any new feature:

### 1. Plan the Feature

Ask yourself:

- What does this feature do?
- What data does it need?
- Where does business logic belong? (Service!)
- Does it need database changes? (Migration!)
- How will users access it? (GraphQL schema!)

### 2. Create Database Migration (if needed)

If the feature requires database changes:

```bash
# Create migration file
touch res/store/migrations/009_add_feature_table.sql
```

```sql
-- res/store/migrations/009_add_feature_table.sql
CREATE TABLE feature_data (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_feature_data_user_id ON feature_data(user_id);
```

Run migration:

```bash
psql $DATABASE_URL -f res/store/migrations/009_add_feature_table.sql
```

### 3. Update Store Layer

**Define store interface:**

```go
// res/store/feature.go
package store

type FeatureData struct {
    ID        string
    Name      string
    UserID    string
    CreatedAt time.Time
}

type FeatureStore interface {
    Create(ctx context.Context, data *FeatureData) error
    Get(ctx context.Context, id string) (*FeatureData, error)
    Update(ctx context.Context, data *FeatureData) error
    Delete(ctx context.Context, id string) error
    ListByUser(ctx context.Context, userID string) ([]*FeatureData, error)
}
```

**Implement in PostgreSQL:**

```go
// res/store/postgresql/feature.go
package postgresql

type featureStore struct {
    db *gorm.DB
}

func (s *featureStore) Create(ctx context.Context, data *store.FeatureData) error {
    return s.db.WithContext(ctx).Create(data).Error
}

func (s *featureStore) Get(ctx context.Context, id string) (*store.FeatureData, error) {
    var data store.FeatureData
    err := s.db.WithContext(ctx).First(&data, "id = ?", id).Error
    if err == gorm.ErrRecordNotFound {
        return nil, store.ErrNotFound
    }
    return &data, err
}

// ... implement other methods
```

**Add to main store:**

```go
// res/store/store.go
type Store interface {
    // ... existing methods
    Features() FeatureStore
}

// res/store/postgresql/store.go
func (s *store) Features() store.FeatureStore {
    return &featureStore{db: s.db}
}
```

### 4. Create Service (if needed)

If business logic is complex (see [SERVICES.md](SERVICES.md)):

**Define interface:**

```go
// res/featureservice/interface.go
package featureservice

type FeatureService interface {
    CreateFeature(ctx context.Context, name, userID string) (*store.FeatureData, error)
    GetUserFeatures(ctx context.Context, userID string) ([]*store.FeatureData, error)
}
```

**Implement service:**

```go
// res/featureservice/service.go
package featureservice

type service struct {
    store  store.Store
    logger *log.Logger
}

func NewService(store store.Store, logger *log.Logger) FeatureService {
    return &service{
        store:  store,
        logger: logger,
    }
}

func (s *service) CreateFeature(
    ctx context.Context,
    name string,
    userID string,
) (*store.FeatureData, error) {
    // Validate
    if len(name) == 0 || len(name) > 100 {
        return nil, fmt.Errorf("invalid name length")
    }

    // Create
    feature := &store.FeatureData{
        ID:     xid.New().String(),
        Name:   name,
        UserID: userID,
    }

    if err := s.store.Features().Create(ctx, feature); err != nil {
        s.logger.Printf("Error creating feature: %v", err)
        return nil, fmt.Errorf("failed to create feature: %w", err)
    }

    return feature, nil
}

func (s *service) GetUserFeatures(
    ctx context.Context,
    userID string,
) ([]*store.FeatureData, error) {
    features, err := s.store.Features().ListByUser(ctx, userID)
    if err != nil {
        s.logger.Printf("Error listing features: %v", err)
        return nil, fmt.Errorf("failed to list features: %w", err)
    }

    return features, nil
}
```

### 5. Update GraphQL Schema

**Add to schema:**

```graphql
# sys/graphql/feature.graphql
type Feature {
    id: ID!
    name: String!
    createdAt: Time!
}

extend type Query {
    myFeatures: [Feature!]! @authRequired
}

extend type Mutation {
    createFeature(name: String!): Feature! @authRequired
}
```

**Generate code:**

```bash
go run github.com/99designs/gqlgen generate
```

### 6. Implement Resolver

**Create resolver file:**

```go
// sys/graphql/feature.go
package graphql

import (
    "context"
    "errors"

    "saas-starter-api/sys/graphql/gen"
    "saas-starter-api/sys/http/middleware"
)

// Query resolver
func (qr *queryResolver) MyFeatures(ctx context.Context) ([]*gen.Feature, error) {
    // Auth check
    currentUser := middleware.GetCurrentUser(ctx)
    if currentUser == nil {
        return nil, errors.New("unauthorized")
    }

    // Delegate to service
    features, err := qr.cfg.FeatureService.GetUserFeatures(ctx, currentUser.ID)
    if err != nil {
        qr.cfg.Logger.Printf("Error getting features: %v", err)
        return nil, errors.New("failed to get features")
    }

    // Format response
    result := make([]*gen.Feature, len(features))
    for i, f := range features {
        result[i] = &gen.Feature{
            ID:        f.ID,
            Name:      f.Name,
            CreatedAt: f.CreatedAt,
        }
    }

    return result, nil
}

// Mutation resolver
func (mr *mutationResolver) CreateFeature(
    ctx context.Context,
    name string,
) (*gen.Feature, error) {
    // Auth check
    currentUser := middleware.GetCurrentUser(ctx)
    if currentUser == nil {
        return nil, errors.New("unauthorized")
    }

    // Input validation
    if len(name) == 0 || len(name) > 100 {
        return nil, errors.New("name must be 1-100 characters")
    }

    // Delegate to service
    feature, err := mr.cfg.FeatureService.CreateFeature(ctx, name, currentUser.ID)
    if err != nil {
        mr.cfg.Logger.Printf("Error creating feature: %v", err)
        return nil, errors.New("failed to create feature")
    }

    // Format response
    return &gen.Feature{
        ID:        feature.ID,
        Name:      feature.Name,
        CreatedAt: feature.CreatedAt,
    }, nil
}
```

### 7. Wire Dependencies

**Update resolver config:**

```go
// sys/graphql/graphql.go
type Config struct {
    Store          store.Store
    FeatureService featureservice.FeatureService  // Add
    // ... other services
    Logger         *log.Logger
}
```

**Wire in main:**

```go
// api/api.go or cmd/main.go
featureService := featureservice.NewService(store, logger)

resolver := graphql.NewResolver(&graphql.Config{
    Store:          store,
    FeatureService: featureService,
    // ...
    Logger:         logger,
})
```

### 8. Test

**Unit test service:**

```go
// res/featureservice/service_test.go
func TestService_CreateFeature(t *testing.T) {
    mockStore := &MockStore{}
    service := NewService(mockStore, logger)

    feature, err := service.CreateFeature(ctx, "Test Feature", "user123")

    assert.NoError(t, err)
    assert.Equal(t, "Test Feature", feature.Name)
}
```

**Test via GraphQL:**

```graphql
mutation {
    createFeature(name: "My Feature") {
        id
        name
        createdAt
    }
}

query {
    myFeatures {
        id
        name
        createdAt
    }
}
```

## Creating Migrations

### Migration Naming Convention

```
XXX_descriptive_name.sql

Examples:
001_create_users_table.sql
002_add_user_status_column.sql
003_create_teams_table.sql
010_add_feature_table.sql
```

### Migration Best Practices

**1. Make Migrations Idempotent**

Use `IF NOT EXISTS` and `IF EXISTS`:

```sql
-- Safe to run multiple times
CREATE TABLE IF NOT EXISTS feature_data (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL
);

ALTER TABLE users ADD COLUMN IF NOT EXISTS status TEXT DEFAULT 'active';
```

**2. Add Indexes for Foreign Keys**

```sql
CREATE TABLE team_members (
    id TEXT PRIMARY KEY,
    team_id TEXT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE
);

-- Add indexes
CREATE INDEX idx_team_members_team_id ON team_members(team_id);
CREATE INDEX idx_team_members_user_id ON team_members(user_id);
```

**3. Use Appropriate CASCADE Rules**

```sql
-- Delete user → delete their data
FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE

-- Delete parent → keep child but nullify reference
FOREIGN KEY (parent_id) REFERENCES parents(id) ON DELETE SET NULL

-- Prevent deletion if children exist
FOREIGN KEY (parent_id) REFERENCES parents(id) ON DELETE RESTRICT
```

**4. Set Sensible Defaults**

```sql
CREATE TABLE tasks (
    id SERIAL PRIMARY KEY,
    status TEXT DEFAULT 'pending',
    progress INTEGER DEFAULT 0,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**5. Use Proper Data Types**

```sql
-- ✅ GOOD
email TEXT NOT NULL              -- Variable length
status TEXT DEFAULT 'active'     -- Enum-like values
created_at TIMESTAMP             -- Dates with time
amount NUMERIC(10, 2)            -- Money (avoid FLOAT)
metadata JSONB                   -- Flexible data

-- ❌ BAD
email VARCHAR(255)               -- Unnecessary length limit
created_at INT                   -- Unix timestamps (use TIMESTAMP)
amount FLOAT                     -- Imprecise for money
```

### Running Migrations

**Development:**

```bash
# Run single migration
psql $DATABASE_URL -f res/store/migrations/010_new_migration.sql

# Run all migrations in order
for file in res/store/migrations/*.sql; do
    echo "Running $file"
    psql $DATABASE_URL -f "$file"
done
```

**Production:**

Use a migration tool like:
- [golang-migrate](https://github.com/golang-migrate/migrate)
- [goose](https://github.com/pressly/goose)
- [dbmate](https://github.com/amacneil/dbmate)

## Adding Services

See complete guide in [SERVICES.md](SERVICES.md).

Quick checklist:

1. ✅ Define interface in `interface.go`
2. ✅ Implement in `service.go`
3. ✅ Use dependency injection (no global state)
4. ✅ Log extensively
5. ✅ Handle errors gracefully
6. ✅ Write tests
7. ✅ Wire into resolver config

## Updating GraphQL Schema

### 1. Modify Schema Files

Edit `*.graphql` files in `sys/graphql/`:

```graphql
# Add new type
type NewType {
    id: ID!
    name: String!
}

# Add new query
extend type Query {
    newThing(id: ID!): NewType! @authRequired
}

# Add new mutation
extend type Mutation {
    createNewThing(name: String!): NewType! @authRequired
}
```

### 2. Update gqlgen Config (if needed)

Edit `sys/graphql/gqlgen.yml` if you need custom mappings:

```yaml
models:
  NewType:
    model: saas-starter-api/res/store.NewType
```

### 3. Generate Code

```bash
go run github.com/99designs/gqlgen generate
```

This generates:
- `sys/graphql/gen/gen.go` - Resolver interfaces
- `sys/graphql/gen/model.go` - GraphQL types

### 4. Implement Resolvers

Implement the generated resolver interfaces in your resolver files.

### 5. Test in Playground

```graphql
query {
    newThing(id: "123") {
        id
        name
    }
}
```

### Common Schema Patterns

**Input types for mutations:**

```graphql
input CreateThingInput {
    name: String!
    description: String
}

extend type Mutation {
    createThing(input: CreateThingInput!): Thing! @authRequired
}
```

**Connection types for pagination:**

```graphql
type Thing {
    id: ID!
    name: String!
}

type ThingEdge {
    node: Thing!
    cursor: ID!
}

type ThingConnection {
    edges: [ThingEdge!]!
    totalCount: Int!
}
```

**Enums:**

```graphql
enum ThingStatus {
    ACTIVE
    INACTIVE
    PENDING
}

type Thing {
    id: ID!
    status: ThingStatus!
}
```

## Running Tests

### Run All Tests

```bash
go test ./...
```

### Run Tests with Coverage

```bash
go test -cover ./...
```

### Run Specific Package Tests

```bash
go test ./res/store/postgresql/
```

### Run Specific Test

```bash
go test -run TestUserService_CreateUser ./res/userservice/
```

### Run Tests with Verbose Output

```bash
go test -v ./...
```

### Test Coverage Report

```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Writing Tests

**Service tests with mocks:**

```go
func TestService_CreateThing(t *testing.T) {
    // Setup
    mockStore := &MockStore{}
    mockThingStore := &MockThingStore{}
    logger := log.New(os.Stdout, "", log.LstdFlags)

    mockStore.On("Things").Return(mockThingStore)
    mockThingStore.On("Create", mock.Anything, mock.Anything).Return(nil)

    service := NewService(mockStore, logger)

    // Execute
    thing, err := service.CreateThing(context.Background(), "Test Thing")

    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, thing)
    assert.Equal(t, "Test Thing", thing.Name)

    mockStore.AssertExpectations(t)
    mockThingStore.AssertExpectations(t)
}
```

**Store tests with test database:**

```go
func TestThingStore_Create(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    // Setup test database
    db := setupTestDB(t)
    defer cleanupTestDB(t, db)

    store := postgresql.NewStore(db)

    // Test
    thing := &store.Thing{
        ID:   xid.New().String(),
        Name: "Test Thing",
    }

    err := store.Things().Create(context.Background(), thing)

    assert.NoError(t, err)

    // Verify
    retrieved, err := store.Things().Get(context.Background(), thing.ID)
    assert.NoError(t, err)
    assert.Equal(t, thing.Name, retrieved.Name)
}
```

## Debugging Tips

### Enable Detailed Logging

```go
logger := log.New(os.Stdout, "[DEBUG] ", log.LstdFlags|log.Lshortfile)
```

### Debug GraphQL Queries

Use playground to:
- Test queries in isolation
- Inspect request/response
- View schema documentation

### Debug Database Queries

**Enable GORM logging:**

```go
db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
    Logger: logger.Default.LogMode(logger.Info),
})
```

**Use EXPLAIN:**

```sql
EXPLAIN ANALYZE
SELECT * FROM users WHERE email = 'test@example.com';
```

### Debug HTTP Requests

```bash
# Verbose curl
curl -v -X POST http://localhost:8080/api \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"query": "{ currentUser { id } }"}'
```

### Common Issues

**"no such table" error**

- Run migrations
- Verify DATABASE_URL is correct

**"unauthorized" error**

- Check Authorization header format
- Verify JWT_SECRET matches
- Check token hasn't expired

**GraphQL parse error**

- Validate GraphQL syntax
- Ensure all types are defined
- Run `gqlgen generate`

**Import cycle error**

- Check for circular dependencies
- Use interfaces to break cycles

## Code Generation

### GraphQL Code Generation

```bash
# Generate resolvers and types
go run github.com/99designs/gqlgen generate

# Generate with verbose output
go run github.com/99designs/gqlgen generate -v
```

**When to regenerate:**

- After modifying `*.graphql` files
- After updating `gqlgen.yml`
- After adding new types or operations

### What Gets Generated

- `sys/graphql/gen/gen.go` - Resolver interfaces, executor
- `sys/graphql/gen/model.go` - GraphQL types as Go structs

### Customizing Generation

Edit `sys/graphql/gqlgen.yml`:

```yaml
# Map GraphQL types to Go types
models:
  User:
    model: saas-starter-api/res/store.User

# Skip generation for certain types
  Team:
    model: saas-starter-api/res/store.Team

# Custom scalar mappings
  Time:
    model: time.Time
```

## Git Workflow

### Branch Naming

```
feature/feature-name
fix/bug-description
refactor/what-changed
docs/documentation-update
```

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add user profile feature
fix: resolve team deletion cascade issue
refactor: extract service from resolver
docs: update API documentation
test: add tests for project service
chore: update dependencies
```

### Development Flow

```bash
# 1. Create feature branch
git checkout -b feature/my-feature

# 2. Make changes and commit
git add .
git commit -m "feat: add my feature"

# 3. Run tests
go test ./...

# 4. Push branch
git push origin feature/my-feature

# 5. Create pull request
# (Use GitHub, GitLab, etc.)

# 6. After review and merge, update main
git checkout main
git pull origin main
```

### Pre-Commit Checklist

Before committing:

- [ ] Tests pass: `go test ./...`
- [ ] Code formatted: `go fmt ./...`
- [ ] No linting errors: `go vet ./...`
- [ ] GraphQL generated: `go run github.com/99designs/gqlgen generate`
- [ ] Migrations created (if DB changes)
- [ ] Documentation updated

## Code Review Checklist

### Architecture

- [ ] Business logic is in services, not resolvers
- [ ] Services use dependency injection
- [ ] Resolvers are thin (~50 lines max)
- [ ] Interface-first design for new services

### Security

- [ ] Authentication checks present
- [ ] Authorization/access control verified
- [ ] Input validation added
- [ ] No sensitive data in logs
- [ ] SQL injection prevented (using GORM/parameterized queries)

### Database

- [ ] Migrations created for schema changes
- [ ] Foreign keys have appropriate CASCADE rules
- [ ] Indexes added for foreign keys
- [ ] Appropriate data types used

### Code Quality

- [ ] Code is readable and well-organized
- [ ] Functions have single responsibility
- [ ] No code duplication
- [ ] Error handling is appropriate
- [ ] Logging is adequate

### Testing

- [ ] Unit tests added for services
- [ ] Integration tests for critical paths
- [ ] Tests cover error cases
- [ ] Mocks used appropriately

### Documentation

- [ ] GraphQL schema documented
- [ ] Complex functions have comments
- [ ] README updated if needed
- [ ] API changes documented

### Performance

- [ ] No N+1 query issues
- [ ] Appropriate indexes on database
- [ ] Long operations use queue system
- [ ] No blocking operations in resolvers

## Development Commands Reference

```bash
# Start development server
air

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Format code
go fmt ./...

# Lint code
go vet ./...

# Generate GraphQL code
go run github.com/99designs/gqlgen generate

# Run migrations
psql $DATABASE_URL -f res/store/migrations/XXX_migration.sql

# Build for production
go build -o bin/server ./cmd/main.go

# Run production build
./bin/server

# Clean build cache
go clean -cache

# Update dependencies
go mod tidy

# Verify dependencies
go mod verify
```

## Environment-Specific Configuration

### Development

```bash
export ENVIRONMENT=development
export PORT=8080
export DATABASE_URL="postgresql://user:pass@localhost:5432/dbname"
# ... other vars
```

### Testing

```bash
export ENVIRONMENT=test
export PORT=8081
export DATABASE_URL="postgresql://user:pass@localhost:5432/test_dbname"
```

### Production

See [DEPLOYMENT.md](DEPLOYMENT.md) for production configuration.

## Common Development Tasks

### Add a New GraphQL Query

1. Add to schema (`sys/graphql/*.graphql`)
2. Run `go run github.com/99designs/gqlgen generate`
3. Implement resolver
4. Test in playground

### Add a New Service

1. Create `res/myservice/interface.go`
2. Create `res/myservice/service.go`
3. Wire into resolver config
4. Write tests

### Add a Database Table

1. Create migration (`res/store/migrations/XXX_*.sql`)
2. Run migration
3. Define Go struct in `res/store/`
4. Implement store interface
5. Add to main store

### Debug a Slow Query

1. Enable GORM logging
2. Use `EXPLAIN ANALYZE` in psql
3. Add missing indexes
4. Optimize query structure

## Next Steps

- **Read [DEPLOYMENT.md](DEPLOYMENT.md)** - Deploy to production
- **Read [CLAUDE_RULES.md](../CLAUDE_RULES.md)** - Development patterns
- **Review existing code** - Learn by example
- **Start building** - Add your first feature

Happy developing!
