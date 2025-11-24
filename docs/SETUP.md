# Setup Guide

Complete guide to setting up the CleanBuddy API for local development.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Database Setup](#database-setup)
- [Environment Variables](#environment-variables)
- [Running Locally](#running-locally)
- [Verifying Installation](#verifying-installation)
- [Development Tools](#development-tools)
- [Troubleshooting](#troubleshooting)

## Prerequisites

Before you begin, ensure you have the following installed:

### Required

- **Go 1.23 or higher**
  ```bash
  go version
  # Should output: go version go1.23.x or higher
  ```

  Install from: https://golang.org/dl/

- **PostgreSQL 14 or higher**
  ```bash
  psql --version
  # Should output: psql (PostgreSQL) 14.x or higher
  ```

  Install:
  - **macOS**: `brew install postgresql@14`
  - **Ubuntu**: `sudo apt-get install postgresql-14`
  - **Windows**: Download from https://www.postgresql.org/download/windows/

- **Google OAuth2 Credentials**

  You'll need a Google Cloud project with OAuth2 credentials:

  1. Go to [Google Cloud Console](https://console.cloud.google.com/)
  2. Create a new project or select existing one
  3. Enable Google+ API
  4. Go to "Credentials" → "Create Credentials" → "OAuth 2.0 Client IDs"
  5. Configure OAuth consent screen
  6. Create OAuth client ID (Application type: Web application)
  7. Add authorized redirect URIs:
     - Development: `http://localhost:3000/auth/callback`
     - Production: `https://yourdomain.com/auth/callback`
  8. Copy Client ID and Client Secret

### Optional (but recommended)

- **Air** - For hot reload during development
  ```bash
  go install github.com/cosmtrek/air@latest
  ```

- **psql** - PostgreSQL command-line client (usually comes with PostgreSQL)

- **Git** - For version control

## Installation

### 1. Clone the Repository

```bash
git clone <your-repository-url>
cd saas-starter-api
```

### 2. Install Go Dependencies

```bash
go mod download
```

This will download all required Go packages including:
- gqlgen (GraphQL code generation)
- GORM (ORM)
- JWT libraries
- OAuth2 libraries

### 3. Verify Installation

```bash
go mod verify
```

Should output: `all modules verified`

## Database Setup

### 1. Start PostgreSQL

**macOS (Homebrew):**
```bash
brew services start postgresql@14
```

**Ubuntu:**
```bash
sudo systemctl start postgresql
```

**Windows:**
PostgreSQL should start automatically as a service.

### 2. Create Database User (if needed)

```bash
# Connect to PostgreSQL as superuser
psql postgres

# Create user
CREATE USER saas_user WITH PASSWORD 'your_secure_password';

# Grant privileges
ALTER USER saas_user CREATEDB;

# Exit
\q
```

### 3. Create Database

```bash
# Using createdb command
createdb -U saas_user saas_starter_db

# OR using psql
psql -U postgres -c "CREATE DATABASE saas_starter_db OWNER saas_user;"
```

### 4. Run Migrations

The migrations are located in `res/store/migrations/`. Run them in order:

```bash
# Set your database URL
export DATABASE_URL="postgresql://saas_user:your_secure_password@localhost:5432/saas_starter_db"

# Run each migration in order
psql $DATABASE_URL -f res/store/migrations/001_create_users_table.sql
psql $DATABASE_URL -f res/store/migrations/002_create_auth_sessions_table.sql
psql $DATABASE_URL -f res/store/migrations/003_create_teams_table.sql
psql $DATABASE_URL -f res/store/migrations/004_create_team_members_table.sql
psql $DATABASE_URL -f res/store/migrations/005_create_team_member_invites_table.sql
psql $DATABASE_URL -f res/store/migrations/006_create_projects_table.sql
psql $DATABASE_URL -f res/store/migrations/007_create_tasks_table.sql
psql $DATABASE_URL -f res/store/migrations/008_create_invitation_codes_table.sql
```

**Or use a shell script:**

```bash
#!/bin/bash
# run-migrations.sh

DATABASE_URL="postgresql://saas_user:your_secure_password@localhost:5432/saas_starter_db"

for file in res/store/migrations/*.sql; do
    echo "Running migration: $file"
    psql $DATABASE_URL -f "$file"
    if [ $? -ne 0 ]; then
        echo "Migration failed: $file"
        exit 1
    fi
done

echo "All migrations completed successfully"
```

### 5. Verify Tables

```bash
psql $DATABASE_URL -c "\dt"
```

You should see these tables:
- `users`
- `auth_sessions`
- `teams`
- `team_members`
- `team_member_invites`
- `projects`
- `tasks`
- `invitation_codes`

## Environment Variables

### 1. Create Environment File

Create a `.env` file in the project root (optional, you can also export variables):

```bash
# .env
PORT=8080
ENVIRONMENT=development

# Database
DATABASE_URL=postgresql://saas_user:your_secure_password@localhost:5432/saas_starter_db

# Authentication
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production
GOOGLE_CLIENT_ID=your-google-client-id.apps.googleusercontent.com
GOOGLE_CLIENT_SECRET=your-google-client-secret
GOOGLE_REDIRECT_URL=http://localhost:3000/auth/callback

# Optional: Email Service (Sidemail)
SIDEMAIL_API_KEY=your-sidemail-api-key

# Optional: Slack Notifications
SLACK_WEBHOOK_URL=https://hooks.slack.com/services/YOUR/WEBHOOK/URL
```

### 2. Environment Variable Reference

#### Required Variables

**`PORT`** (required)
- Port number for the server
- Example: `8080`
- Used by: Server startup

**`ENVIRONMENT`** (required)
- Current environment
- Values: `development` or `production`
- Effect:
  - `development`: Enables GraphQL playground at `/api/playground`
  - `production`: Disables playground, stricter security

**`DATABASE_URL`** (required)
- PostgreSQL connection string
- Format: `postgresql://username:password@host:port/database`
- Example: `postgresql://user:pass@localhost:5432/dbname`
- Used by: All database operations

**`JWT_SECRET`** (required)
- Secret key for signing JWT tokens
- Should be a long, random string
- Example: `openssl rand -base64 32`
- IMPORTANT: Keep this secret and never commit to version control

**`GOOGLE_CLIENT_ID`** (required)
- Google OAuth2 client ID
- Format: `xxxxx.apps.googleusercontent.com`
- Get from: Google Cloud Console
- Used by: OAuth2 authentication flow

**`GOOGLE_CLIENT_SECRET`** (required)
- Google OAuth2 client secret
- Get from: Google Cloud Console
- Used by: OAuth2 token exchange

**`GOOGLE_REDIRECT_URL`** (required)
- OAuth2 callback URL
- Must match URL configured in Google Console
- Development: `http://localhost:3000/auth/callback`
- Production: `https://yourdomain.com/auth/callback`

#### Optional Variables

**`SIDEMAIL_API_KEY`** (optional)
- API key for Sidemail email service
- Get from: https://sidemail.io
- Used by: Email service (`res/mail/sidemail/`)
- If not set: Email functionality will be disabled (graceful degradation)

**`SLACK_WEBHOOK_URL`** (optional)
- Slack incoming webhook URL
- Get from: Slack App configuration
- Format: `https://hooks.slack.com/services/T00/B00/XXX`
- Used by: Notification service (`res/notification/slack/`)
- If not set: Slack notifications will be disabled

### 3. Load Environment Variables

**Option A: Using .env file**

Install a .env loader:
```bash
go get github.com/joho/godotenv
```

**Option B: Export manually**

```bash
export PORT=8080
export ENVIRONMENT=development
export DATABASE_URL="postgresql://user:pass@localhost:5432/dbname"
# ... etc
```

**Option C: Use a script**

```bash
# dev.sh
#!/bin/bash
source .env
go run cmd/main.go
```

## Running Locally

### Standard Run

```bash
go run cmd/main.go
```

You should see output like:
```
(cmd/main.go) GraphQL playground enabled at /api/playground
(cmd/main.go) Starting server on :8080 (environment: development)
```

### With Hot Reload (Recommended)

Using Air for automatic reloading on code changes:

1. Install Air (if not already installed):
```bash
go install github.com/cosmtrek/air@latest
```

2. Create `.air.toml` configuration:
```toml
root = "."
testdata_dir = "testdata"
tmp_dir = "tmp"

[build]
  args_bin = []
  bin = "./tmp/main"
  cmd = "go build -o ./tmp/main ./cmd/main.go"
  delay = 1000
  exclude_dir = ["assets", "tmp", "vendor", "testdata"]
  exclude_file = []
  exclude_regex = ["_test.go"]
  exclude_unchanged = false
  follow_symlink = false
  full_bin = ""
  include_dir = []
  include_ext = ["go", "tpl", "tmpl", "html", "graphql"]
  include_file = []
  kill_delay = "0s"
  log = "build-errors.log"
  poll = false
  poll_interval = 0
  rerun = false
  rerun_delay = 500
  send_interrupt = false
  stop_on_error = false

[color]
  app = ""
  build = "yellow"
  main = "magenta"
  runner = "green"
  watcher = "cyan"

[log]
  main_only = false
  time = false

[misc]
  clean_on_exit = false

[screen]
  clear_on_rebuild = false
  keep_scroll = true
```

3. Run with Air:
```bash
air
```

Air will watch for file changes and automatically rebuild/restart the server.

## Verifying Installation

### 1. Check Server is Running

```bash
curl http://localhost:8080/api
```

Should return a GraphQL response (likely an error since no query was provided, but confirms server is up).

### 2. Access GraphQL Playground

Open browser to: `http://localhost:8080/api/playground`

You should see the GraphQL Playground interface.

### 3. Test a Query

In the playground, try this query:

```graphql
query {
  currentUser {
    id
    email
    displayName
  }
}
```

Expected result (if not authenticated):
```json
{
  "data": {
    "currentUser": null
  }
}
```

This is correct - you need to authenticate first to get user data.

### 4. Test Database Connection

Check that migrations ran successfully:

```bash
psql $DATABASE_URL -c "SELECT COUNT(*) FROM users;"
```

Should return `0` (no users yet, but table exists).

## Development Tools

### gqlgen - GraphQL Code Generator

Generate GraphQL code after schema changes:

```bash
# From project root
go run github.com/99designs/gqlgen generate
```

Configuration is in `sys/graphql/gqlgen.yml`.

### Air - Hot Reload

As mentioned above, Air provides automatic reloading:

```bash
air
```

### PostgreSQL Tools

**psql - Interactive terminal**
```bash
psql $DATABASE_URL
```

Common commands:
- `\dt` - List tables
- `\d tablename` - Describe table
- `\q` - Quit

**pg_dump - Backup database**
```bash
pg_dump $DATABASE_URL > backup.sql
```

**psql - Restore database**
```bash
psql $DATABASE_URL < backup.sql
```

### Testing

Run all tests:
```bash
go test ./...
```

Run tests with coverage:
```bash
go test -cover ./...
```

Run tests with verbose output:
```bash
go test -v ./...
```

## Troubleshooting

### Database Connection Issues

**Error: `connection refused`**

Solution:
```bash
# Check if PostgreSQL is running
brew services list  # macOS
sudo systemctl status postgresql  # Linux

# Start PostgreSQL
brew services start postgresql@14  # macOS
sudo systemctl start postgresql  # Linux
```

**Error: `database does not exist`**

Solution:
```bash
# Create the database
createdb -U saas_user saas_starter_db

# Or with psql
psql -U postgres -c "CREATE DATABASE saas_starter_db;"
```

**Error: `password authentication failed`**

Solution:
- Verify username/password in DATABASE_URL
- Check PostgreSQL authentication in `pg_hba.conf`
- Try resetting password:
  ```bash
  psql postgres
  ALTER USER saas_user WITH PASSWORD 'newpassword';
  ```

### Google OAuth Issues

**Error: `invalid_client`**

Solution:
- Verify GOOGLE_CLIENT_ID and GOOGLE_CLIENT_SECRET are correct
- Check credentials in Google Cloud Console
- Ensure you're using OAuth 2.0 Client ID, not API key

**Error: `redirect_uri_mismatch`**

Solution:
- Verify GOOGLE_REDIRECT_URL matches exactly what's configured in Google Console
- Include protocol (http:// or https://)
- Check for trailing slashes
- Add both localhost and production URLs to Google Console

**Error: `access_denied`**

Solution:
- Check OAuth consent screen configuration
- Ensure required scopes are enabled
- Verify email is authorized (if app is in testing mode)

### JWT Token Issues

**Error: `signature is invalid`**

Solution:
- Verify JWT_SECRET is the same across all instances
- Don't change JWT_SECRET while users have active sessions
- Ensure JWT_SECRET is not empty

### Port Already in Use

**Error: `bind: address already in use`**

Solution:
```bash
# Find process using port 8080
lsof -i :8080

# Kill the process
kill -9 <PID>

# Or use a different port
PORT=8081 go run cmd/main.go
```

### Module Download Issues

**Error: `go: module ... not found`**

Solution:
```bash
# Clean module cache
go clean -modcache

# Re-download modules
go mod download

# Verify modules
go mod verify

# Tidy up (remove unused, add missing)
go mod tidy
```

### GraphQL Playground Not Available

**Issue: 404 on `/api/playground`**

Solution:
- Check ENVIRONMENT variable is set to `development`
- Playground is disabled in production for security
- Try: `ENVIRONMENT=development go run cmd/main.go`

### Migration Failures

**Error: `relation already exists`**

Solution:
- Migrations have already been run
- To reset database:
  ```bash
  dropdb saas_starter_db
  createdb saas_starter_db
  # Run migrations again
  ```

**Error: `syntax error at or near...`**

Solution:
- Check SQL syntax in migration file
- Ensure you're running migrations in order
- Some migrations may depend on previous ones

### Code Generation Issues

**Error: `unable to load schema`**

Solution:
```bash
# Verify GraphQL schema files exist
ls sys/graphql/*.graphql

# Check gqlgen.yml configuration
cat sys/graphql/gqlgen.yml

# Try regenerating
go run github.com/99designs/gqlgen generate
```

### Common Setup Mistakes

1. **Forgetting to create database** - Run `createdb` first
2. **Wrong DATABASE_URL format** - Use `postgresql://` not `postgres://`
3. **Migrations out of order** - Run migrations 001, 002, 003... in sequence
4. **Missing environment variables** - All required vars must be set
5. **Wrong Google redirect URL** - Must exactly match Google Console config
6. **Weak JWT_SECRET** - Use a strong random string (32+ characters)

### Getting Help

If you're still stuck:

1. Check server logs for detailed error messages
2. Verify all environment variables are set correctly
3. Ensure database is running and accessible
4. Try running with verbose logging
5. Check [CLAUDE_RULES.md](../CLAUDE_RULES.md) for development patterns
6. Review existing code for examples

### Useful Commands Summary

```bash
# Check Go version
go version

# Check PostgreSQL version
psql --version

# Test database connection
psql $DATABASE_URL -c "SELECT 1;"

# List all tables
psql $DATABASE_URL -c "\dt"

# Check running processes
lsof -i :8080

# Test API endpoint
curl http://localhost:8080/api

# Run with environment variable override
PORT=8081 go run cmd/main.go

# Clean and rebuild
go clean && go build ./cmd/main.go

# Run tests
go test ./...

# Check for module issues
go mod verify && go mod tidy
```

## Next Steps

Once you have the server running:

1. **Explore the API** - Use GraphQL playground to test queries
2. **Read the Architecture** - See [ARCHITECTURE.md](ARCHITECTURE.md)
3. **Understand Services** - Read [SERVICES.md](SERVICES.md)
4. **Review Development Workflow** - Check [DEVELOPMENT.md](DEVELOPMENT.md)
5. **Build Your First Feature** - Follow patterns in [CLAUDE_RULES.md](../CLAUDE_RULES.md)

You're all set up! Time to start building.
