# Deployment Guide

Complete guide to deploying the SaaS Starter API to production.

## Table of Contents

- [Overview](#overview)
- [Vercel Deployment](#vercel-deployment)
- [Database Hosting](#database-hosting)
- [Environment Variables](#environment-variables)
- [Serverless Functions](#serverless-functions)
- [Production Considerations](#production-considerations)
- [Monitoring and Logging](#monitoring-and-logging)
- [Performance Optimization](#performance-optimization)
- [Security Checklist](#security-checklist)
- [Troubleshooting](#troubleshooting)

## Overview

This template is designed for **serverless deployment**, primarily targeting Vercel, but can be deployed to:

- **Vercel** (Recommended - easy deployment)
- **AWS Lambda** (with API Gateway)
- **Google Cloud Functions**
- **Traditional VPS** (DigitalOcean, Linode, etc.)
- **Docker/Kubernetes** (self-hosted)

This guide focuses on **Vercel deployment** as the primary method.

## Vercel Deployment

Vercel provides serverless function deployment with:
- Automatic HTTPS
- CDN edge caching
- Zero-downtime deployments
- Built-in CI/CD
- Easy environment variable management

### Prerequisites

- Vercel account (free tier available)
- Vercel CLI installed: `npm i -g vercel`
- GitHub/GitLab repository (for auto-deployments)

### Step 1: Create vercel.json

Create `vercel.json` in project root:

```json
{
  "version": 2,
  "builds": [
    {
      "src": "cmd/main.go",
      "use": "@vercel/go"
    }
  ],
  "routes": [
    {
      "src": "/api",
      "dest": "cmd/main.go"
    },
    {
      "src": "/api/playground",
      "dest": "cmd/main.go"
    }
  ],
  "env": {
    "PORT": "8080",
    "ENVIRONMENT": "production"
  }
}
```

### Step 2: Install Vercel CLI

```bash
npm install -g vercel
```

### Step 3: Login to Vercel

```bash
vercel login
```

### Step 4: Configure Environment Variables

Set environment variables in Vercel dashboard or CLI:

```bash
# Using CLI
vercel env add DATABASE_URL
vercel env add JWT_SECRET
vercel env add GOOGLE_CLIENT_ID
vercel env add GOOGLE_CLIENT_SECRET
vercel env add GOOGLE_REDIRECT_URL
vercel env add SIDEMAIL_API_KEY
vercel env add SLACK_WEBHOOK_URL
```

**Or use Vercel Dashboard:**

1. Go to project settings
2. Navigate to "Environment Variables"
3. Add all required variables for Production

### Step 5: Deploy

**Deploy from CLI:**

```bash
# Deploy to preview (development)
vercel

# Deploy to production
vercel --prod
```

**Deploy from Git (Recommended):**

1. Connect repository to Vercel
2. Push to `main` branch
3. Vercel auto-deploys

### Step 6: Configure Custom Domain

In Vercel dashboard:

1. Go to project settings
2. Navigate to "Domains"
3. Add your custom domain
4. Update DNS records as instructed
5. Wait for SSL certificate (automatic)

### Step 7: Update OAuth Redirect URLs

Update Google OAuth Console:

```
Add authorized redirect URI:
https://yourdomain.com/auth/callback
```

Update environment variable:

```bash
GOOGLE_REDIRECT_URL=https://yourdomain.com/auth/callback
```

### Vercel Deployment Options

**vercel.json Configuration Options:**

```json
{
  "version": 2,
  "builds": [
    {
      "src": "cmd/main.go",
      "use": "@vercel/go",
      "config": {
        "maxLambdaSize": "50mb"
      }
    }
  ],
  "routes": [
    {
      "src": "/api",
      "dest": "cmd/main.go",
      "methods": ["GET", "POST", "OPTIONS"]
    }
  ],
  "regions": ["iad1"],  // Region (optional)
  "env": {
    "PORT": "8080",
    "ENVIRONMENT": "production"
  }
}
```

### Continuous Deployment with Git

1. **Connect Repository**
   - Go to Vercel dashboard
   - Click "Import Project"
   - Connect GitHub/GitLab account
   - Select repository

2. **Configure Build Settings**
   - Build Command: (leave empty for Go)
   - Output Directory: (leave empty)
   - Install Command: (leave empty)

3. **Add Environment Variables**
   - Add all production environment variables
   - Mark sensitive variables as "Secret"

4. **Deploy**
   - Push to `main` → deploys to production
   - Push to other branches → deploys to preview URLs

## Database Hosting

Choose a PostgreSQL hosting provider:

### Option 1: Neon (Recommended)

**Pros:**
- Serverless PostgreSQL
- Generous free tier
- Auto-scaling
- Instant branching
- Built for Vercel

**Setup:**

1. Create account at [neon.tech](https://neon.tech)
2. Create project
3. Copy connection string
4. Add to Vercel environment variables as `DATABASE_URL`

**Connection string format:**

```
postgresql://user:password@ep-xxx.us-east-2.aws.neon.tech/dbname
```

### Option 2: Supabase

**Pros:**
- Generous free tier
- Built-in auth, storage, real-time
- Good developer experience

**Setup:**

1. Create project at [supabase.com](https://supabase.com)
2. Go to Settings → Database
3. Copy connection string (Session mode)
4. Add to Vercel as `DATABASE_URL`

### Option 3: Railway

**Pros:**
- Simple setup
- Good free tier
- Fast provisioning

**Setup:**

1. Create project at [railway.app](https://railway.app)
2. Add PostgreSQL service
3. Copy DATABASE_URL from environment variables
4. Add to Vercel environment variables

### Option 4: Heroku Postgres

**Pros:**
- Reliable
- Good tooling
- Well-documented

**Setup:**

1. Create Heroku app
2. Add Heroku Postgres addon
3. Copy DATABASE_URL from config vars
4. Add to Vercel environment variables

### Option 5: AWS RDS

**Pros:**
- Full control
- Scalable
- Enterprise features

**Cons:**
- More expensive
- Requires AWS knowledge

**Setup:**

1. Create RDS PostgreSQL instance
2. Configure security groups (allow Vercel IPs)
3. Copy connection string
4. Add to Vercel environment variables

### Running Migrations in Production

**Option A: Manual via psql**

```bash
# Set production DATABASE_URL
export DATABASE_URL="postgresql://user:pass@host/db"

# Run migrations in order
psql $DATABASE_URL -f res/store/migrations/001_create_users_table.sql
psql $DATABASE_URL -f res/store/migrations/002_create_auth_sessions_table.sql
# ... etc
```

**Option B: Using migration tool**

Use [golang-migrate](https://github.com/golang-migrate/migrate):

```bash
# Install
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Run migrations
migrate -database "$DATABASE_URL" -path res/store/migrations up
```

**Option C: Initialization script**

Create `scripts/migrate.sh`:

```bash
#!/bin/bash
set -e

DATABASE_URL=$1

if [ -z "$DATABASE_URL" ]; then
    echo "Usage: ./migrate.sh <DATABASE_URL>"
    exit 1
fi

echo "Running migrations..."

for file in res/store/migrations/*.sql; do
    echo "Running $file"
    psql $DATABASE_URL -f "$file"
    if [ $? -ne 0 ]; then
        echo "Migration failed: $file"
        exit 1
    fi
done

echo "All migrations completed successfully"
```

**Run:**

```bash
chmod +x scripts/migrate.sh
./scripts/migrate.sh "$DATABASE_URL"
```

## Environment Variables

### Required Production Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `PORT` | Server port | `8080` |
| `ENVIRONMENT` | Environment name | `production` |
| `DATABASE_URL` | PostgreSQL connection string | `postgresql://user:pass@host/db` |
| `JWT_SECRET` | Secret for JWT signing | `<64-char-random-string>` |
| `GOOGLE_CLIENT_ID` | Google OAuth client ID | `xxx.apps.googleusercontent.com` |
| `GOOGLE_CLIENT_SECRET` | Google OAuth secret | `GOCSPX-xxx` |
| `GOOGLE_REDIRECT_URL` | OAuth callback URL | `https://yourdomain.com/auth/callback` |

### Optional Production Variables

| Variable | Description |
|----------|-------------|
| `SIDEMAIL_API_KEY` | Sidemail API key for emails |
| `SLACK_WEBHOOK_URL` | Slack webhook for notifications |

### Generating Secure Secrets

**JWT_SECRET:**

```bash
# Generate 64-character random string
openssl rand -base64 64 | tr -d '\n'
```

**Alternative:**

```bash
# Using /dev/urandom
head -c 64 /dev/urandom | base64 | tr -d '\n'
```

### Setting Variables in Vercel

**Via Dashboard:**

1. Project Settings → Environment Variables
2. Add each variable
3. Select environment (Production/Preview/Development)
4. Mark sensitive values as "Secret"

**Via CLI:**

```bash
# Add variable for production
vercel env add DATABASE_URL production

# Add for all environments
vercel env add JWT_SECRET
```

### Environment Variable Security

- **NEVER commit secrets to git**
- **Use Vercel's "Secret" option** for sensitive values
- **Rotate secrets regularly** (especially JWT_SECRET)
- **Use different values** for dev/staging/production
- **Audit access** to environment variables

## Serverless Functions

### How Vercel Functions Work

Vercel runs your Go application as serverless functions:

1. Request arrives at `/api`
2. Vercel spins up function instance
3. Go application handles request
4. Response returned
5. Function kept warm or shut down

### Function Configuration

**Timeouts:**

Default: 10 seconds (Hobby), 60 seconds (Pro)

Increase in `vercel.json`:

```json
{
  "functions": {
    "cmd/main.go": {
      "maxDuration": 60
    }
  }
}
```

**Memory:**

Default: 1024 MB

Adjust in `vercel.json`:

```json
{
  "functions": {
    "cmd/main.go": {
      "memory": 2048
    }
  }
}
```

**Regions:**

Deploy to specific regions:

```json
{
  "regions": ["iad1", "sfo1"]
}
```

### Cold Start Optimization

**1. Keep Functions Warm**

Use a ping service (e.g., UptimeRobot):
- Ping `/api` every 5 minutes
- Keeps function instance warm

**2. Minimize Dependencies**

Only import what you need:

```go
// ❌ BAD - Imports unused packages
import (
    "saas-starter-api/res/auth"
    "saas-starter-api/res/mail"
    "saas-starter-api/res/notification"
    "saas-starter-api/res/queue"
)

// ✅ GOOD - Only import what's used
import (
    "saas-starter-api/res/auth"
    "saas-starter-api/res/store/postgresql"
)
```

**3. Lazy Loading**

Initialize expensive resources only when needed.

### Connection Pooling

**Database connections:**

Configure GORM connection pool:

```go
sqlDB, err := db.DB()
sqlDB.SetMaxIdleConns(2)
sqlDB.SetMaxOpenConns(10)
sqlDB.SetConnMaxLifetime(time.Hour)
```

For serverless, keep connection pool small:
- MaxIdleConns: 1-2
- MaxOpenConns: 5-10

## Production Considerations

### 1. Database Connection Limits

**Problem:** Each serverless function creates database connections.

**Solution:**

- Use connection pooling
- Consider connection pooler (PgBouncer, Neon's pooler)
- Set appropriate pool limits

**Example with Neon:**

```
DATABASE_URL=postgresql://user:pass@ep-xxx.us-east-2.aws.neon.tech/db?sslmode=require&pooler=true
```

### 2. CORS Configuration

Update CORS middleware for production:

```go
// sys/http/middleware/corsmiddleware.go
func CORSMiddleware() func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            origin := r.Header.Get("Origin")

            // Production: Whitelist specific origins
            allowedOrigins := []string{
                "https://yourdomain.com",
                "https://app.yourdomain.com",
            }

            for _, allowed := range allowedOrigins {
                if origin == allowed {
                    w.Header().Set("Access-Control-Allow-Origin", origin)
                    break
                }
            }

            // Other CORS headers...
        })
    }
}
```

### 3. Rate Limiting

Implement rate limiting to prevent abuse:

**Option A: Vercel Edge Middleware**

Create `middleware.ts`:

```typescript
import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'

export function middleware(request: NextRequest) {
    // Rate limiting logic
    // Use Vercel KV or Upstash Redis
}

export const config = {
    matcher: '/api/:path*',
}
```

**Option B: Application-level**

Use middleware with in-memory or Redis-backed rate limiter.

### 4. Caching

**GraphQL Response Caching:**

Consider caching for expensive queries:

```go
// Pseudo-code
cachedResult := cache.Get(queryHash)
if cachedResult != nil {
    return cachedResult
}

result := executeQuery()
cache.Set(queryHash, result, 5*time.Minute)
return result
```

**Database Query Caching:**

Use GORM's caching or implement Redis cache.

### 5. Error Tracking

Use error tracking service:

- **Sentry** - Full-featured error tracking
- **Rollbar** - Error monitoring
- **Bugsnag** - Exception tracking

**Example with Sentry:**

```go
import "github.com/getsentry/sentry-go"

sentry.Init(sentry.ClientOptions{
    Dsn: os.Getenv("SENTRY_DSN"),
    Environment: os.Getenv("ENVIRONMENT"),
})

// In error handler
sentry.CaptureException(err)
```

### 6. Structured Logging

Use structured logging for better log analysis:

```go
import "go.uber.org/zap"

logger, _ := zap.NewProduction()
defer logger.Sync()

logger.Info("user created",
    zap.String("user_id", userID),
    zap.String("email", email),
)
```

## Monitoring and Logging

### Vercel Logs

**View logs:**

```bash
# Real-time logs
vercel logs --follow

# Logs for production
vercel logs --prod

# Logs for specific deployment
vercel logs <deployment-url>
```

### Application Logging

**Production logging best practices:**

```go
// Use structured logging
logger.Printf("[INFO] user_created user_id=%s email=%s", userID, email)

// Log levels: INFO, WARN, ERROR
logger.Printf("[ERROR] database_error error=%v query=%s", err, query)

// Include context
logger.Printf("[WARN] slow_query duration=%dms query=%s", duration, query)
```

### Monitoring Services

**Recommended services:**

1. **Vercel Analytics** - Built-in analytics
2. **New Relic** - APM and monitoring
3. **Datadog** - Infrastructure monitoring
4. **Prometheus + Grafana** - Self-hosted monitoring

### Health Checks

Add health check endpoint:

```go
// sys/graphql/health.go
http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    // Check database
    if err := db.Ping(); err != nil {
        w.WriteHeader(http.StatusServiceUnavailable)
        json.NewEncoder(w).Encode(map[string]string{
            "status": "unhealthy",
            "error": "database unavailable",
        })
        return
    }

    w.WriteHeader(http.StatusOK)
    json.NewEncoder(w).Encode(map[string]string{
        "status": "healthy",
        "version": "1.0.0",
    })
})
```

**Monitor with UptimeRobot, Pingdom, or similar.**

### Metrics to Track

- **Response times** - API latency
- **Error rates** - 4xx and 5xx responses
- **Database connections** - Connection pool usage
- **Queue depth** - Pending tasks count
- **User signups** - New users per day
- **Active users** - DAU/MAU

## Performance Optimization

### 1. Database Query Optimization

**Use indexes:**

```sql
CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_projects_team_id ON projects(team_id);
CREATE INDEX idx_tasks_status_type ON tasks(status, type);
```

**Avoid N+1 queries:**

Use GORM's `Preload`:

```go
// ❌ BAD - N+1 queries
teams, _ := store.Teams().List(ctx)
for _, team := range teams {
    owner, _ := store.Users().Get(ctx, team.OwnerID) // N queries
}

// ✅ GOOD - Single query with join
db.Preload("Owner").Find(&teams)
```

### 2. Connection Pooling

Configure appropriate pool size:

```go
sqlDB, _ := db.DB()
sqlDB.SetMaxIdleConns(2)      // Serverless: keep low
sqlDB.SetMaxOpenConns(10)     // Adjust based on load
sqlDB.SetConnMaxLifetime(time.Hour)
```

### 3. Caching

**In-memory cache for hot data:**

```go
type Cache struct {
    data sync.Map
    ttl  time.Duration
}

func (c *Cache) Get(key string) interface{} {
    // Implementation
}
```

**Redis for distributed cache:**

Use Redis for multi-region deployments.

### 4. GraphQL Query Complexity

Limit query complexity to prevent abuse:

```go
// In gqlgen config
func (c *Config) Complexity() graphql.ComplexityRoot {
    return graphql.ComplexityRoot{
        Query: graphql.QueryComplexityRoot{
            // Limit nested queries
        },
    }
}
```

### 5. Response Compression

Enable gzip compression:

```go
import "github.com/gorilla/handlers"

http.Handle("/api", handlers.CompressHandler(apiHandler))
```

## Security Checklist

### Before Production Deployment

- [ ] **JWT_SECRET is strong** (64+ random characters)
- [ ] **Database uses SSL/TLS** (`sslmode=require`)
- [ ] **CORS configured** (specific origins, not `*`)
- [ ] **Environment variables secured** (not in git)
- [ ] **HTTPS enabled** (automatic with Vercel)
- [ ] **GraphQL Playground disabled** (check `ENVIRONMENT=production`)
- [ ] **Input validation on all mutations**
- [ ] **SQL injection prevented** (using GORM parameterized queries)
- [ ] **Rate limiting enabled**
- [ ] **Error messages sanitized** (no internal details exposed)
- [ ] **Sensitive logs redacted** (no passwords, tokens in logs)
- [ ] **Dependencies updated** (`go mod tidy`)
- [ ] **OAuth redirect URLs updated** (production domain)
- [ ] **Database backups enabled**
- [ ] **Monitoring/alerting configured**

### Ongoing Security

- **Rotate secrets quarterly**
- **Update dependencies monthly**
- **Review access logs weekly**
- **Audit database queries**
- **Monitor for suspicious activity**

## Troubleshooting

### Deployment Fails

**Error: "Build failed"**

Check build logs:
```bash
vercel logs <deployment-url>
```

Common causes:
- Syntax errors
- Missing dependencies
- Environment variables not set

**Error: "Function timeout"**

Increase timeout in `vercel.json`:
```json
{
  "functions": {
    "cmd/main.go": {
      "maxDuration": 60
    }
  }
}
```

### Database Connection Issues

**Error: "connection refused"**

- Check DATABASE_URL is correct
- Verify database is running
- Check firewall/security groups
- Ensure SSL mode: `?sslmode=require`

**Error: "too many connections"**

- Reduce connection pool size
- Use connection pooler (PgBouncer)
- Check for connection leaks

### GraphQL Errors

**Error: "unauthorized"**

- Check JWT_SECRET matches
- Verify Authorization header format
- Check token hasn't expired

**Error: "Internal server error"**

- Check application logs
- Verify environment variables
- Check database connectivity

### Performance Issues

**Slow response times**

- Check database query performance (EXPLAIN)
- Add missing indexes
- Enable connection pooling
- Use caching

**High memory usage**

- Check for memory leaks
- Reduce connection pool size
- Optimize data structures

## Deployment Checklist

### Pre-Deployment

- [ ] All tests passing
- [ ] Migrations ready
- [ ] Environment variables documented
- [ ] Production database created
- [ ] OAuth credentials updated
- [ ] Domain configured
- [ ] SSL certificate ready

### Deployment

- [ ] Run migrations on production database
- [ ] Set all environment variables in Vercel
- [ ] Deploy application
- [ ] Verify health check endpoint
- [ ] Test authentication flow
- [ ] Test critical operations
- [ ] Check error tracking is working

### Post-Deployment

- [ ] Monitor logs for errors
- [ ] Verify database connections
- [ ] Test from production domain
- [ ] Check performance metrics
- [ ] Set up alerts
- [ ] Document any issues

## Next Steps

After deployment:

1. **Set up monitoring** - Error tracking, performance monitoring
2. **Configure backups** - Database automated backups
3. **Set up alerts** - Downtime, errors, performance degradation
4. **Document runbook** - Common issues and solutions
5. **Create staging environment** - Test before production
6. **Plan scaling strategy** - Handle growth

## Summary

You've deployed a production-ready SaaS API with:

- ✅ Serverless architecture (Vercel)
- ✅ PostgreSQL database (Neon/Supabase)
- ✅ HTTPS and security
- ✅ Environment variable management
- ✅ Monitoring and logging
- ✅ Performance optimization

Your API is now live and ready to serve users!

## Additional Resources

- [Vercel Documentation](https://vercel.com/docs)
- [Neon Documentation](https://neon.tech/docs)
- [Go Best Practices](https://golang.org/doc/effective_go)
- [PostgreSQL Performance](https://wiki.postgresql.org/wiki/Performance_Optimization)

Need help? Check the other docs:
- [SETUP.md](SETUP.md) - Local development
- [ARCHITECTURE.md](ARCHITECTURE.md) - System design
- [DEVELOPMENT.md](DEVELOPMENT.md) - Development workflow

You're production-ready. Ship it!
