# GraphQL API Reference

Complete reference for the SaaS Starter API GraphQL endpoint.

## Table of Contents

- [Overview](#overview)
- [Authentication](#authentication)
- [User Operations](#user-operations)
- [Team Operations](#team-operations)
- [Project Operations](#project-operations)
- [Invitation Codes](#invitation-codes)
- [Pagination](#pagination)
- [Error Handling](#error-handling)
- [Example Queries and Mutations](#example-queries-and-mutations)

## Overview

The API is a GraphQL endpoint available at `/api`.

**Base URL:** `http://localhost:8080/api` (development)

**GraphQL Playground:** `http://localhost:8080/api/playground` (development only)

### Key Features

- **Type-safe** - Strongly typed schema with code generation
- **Flexible** - Request exactly the data you need
- **Efficient** - No over-fetching or under-fetching
- **Real-time** - Support for subscriptions (if needed)
- **Introspective** - Schema is self-documenting

### Common Directives

**`@authRequired`** - Requires authenticated user

```graphql
type User {
    id: ID!
    email: String! @authRequired
}
```

**`@goField`** - Force resolver implementation (internal)

### Common Scalars

- **`Time`** - RFC3339 timestamp (e.g., `2024-01-15T10:30:00Z`)
- **`Void`** - Represents no return value (for deletions)
- **`JSON`** - Arbitrary JSON data
- **`TimeInterval`** - Duration (e.g., `1h30m`)
- **`Upload`** - File upload (if needed)

## Authentication

### authWithIdentityProvider

Initial sign-up and sign-in using OAuth2 providers (currently Google).

**Mutation:**

```graphql
mutation AuthWithGoogle($code: String!) {
    authWithIdentityProvider(code: $code, kind: GoogleOAuth2) {
        accessToken
        refreshToken
    }
}
```

**Input:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `code` | String | Yes | OAuth2 authorization code from provider |
| `kind` | AuthIdentityKind | Yes | Provider type (GoogleOAuth2) |

**Response:**

```graphql
type AuthResult {
    accessToken: String!   # JWT access token (3 days)
    refreshToken: String!  # JWT refresh token (2 weeks)
}
```

**Example:**

```graphql
mutation {
    authWithIdentityProvider(
        code: "4/0AY0e-g7X...",
        kind: GoogleOAuth2
    ) {
        accessToken
        refreshToken
    }
}
```

**Response:**

```json
{
    "data": {
        "authWithIdentityProvider": {
            "accessToken": "eyJhbGciOiJIUzI1NiIs...",
            "refreshToken": "eyJhbGciOiJIUzI1NiIs..."
        }
    }
}
```

**Flow:**

1. Frontend redirects to Google OAuth
2. User authorizes
3. Google redirects back with `code`
4. Frontend calls `authWithIdentityProvider` with code
5. Backend exchanges code for user info
6. Backend creates user (if new) or loads existing
7. Backend returns access + refresh tokens

### authWithRefreshToken

Refresh an expired access token using refresh token.

**Mutation:**

```graphql
mutation RefreshToken($token: String!) {
    authWithRefreshToken(token: $token) {
        accessToken
        refreshToken
    }
}
```

**Input:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `token` | String | Yes | Current refresh token |

**Response:** Same as `authWithIdentityProvider`

**Example:**

```graphql
mutation {
    authWithRefreshToken(token: "eyJhbGciOiJIUzI1NiIs...") {
        accessToken
        refreshToken
    }
}
```

### Using Access Tokens

Include access token in `Authorization` header:

```
Authorization: Bearer eyJhbGciOiJIUzI1NiIs...
```

**HTTP Example:**

```bash
curl -X POST http://localhost:8080/api \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIs..." \
  -d '{"query": "{ currentUser { id email } }"}'
```

## User Operations

### currentUser

Get the currently authenticated user.

**Query:**

```graphql
query {
    currentUser {
        id
        displayName
        email
        status
        ownedTeamsConnection {
            edges {
                node {
                    id
                    displayName
                }
            }
            totalCount
        }
        memberTeamsConnection {
            edges {
                node {
                    id
                    displayName
                }
            }
            totalCount
        }
        invitationCodes
    }
}
```

**Type:**

```graphql
type User {
    id: ID!
    displayName: String!
    status: UserStatus!
    email: String!
    invitationCodes: [String!] @authRequired
    ownedTeamsConnection: TeamConnection!
    memberTeamsConnection: TeamConnection!
}

enum UserStatus {
    PENDING
    ACTIVE
    SUSPENDED
}
```

**Response:**

```json
{
    "data": {
        "currentUser": {
            "id": "abc123",
            "displayName": "John Doe",
            "email": "john@example.com",
            "status": "ACTIVE",
            "ownedTeamsConnection": {
                "edges": [
                    {
                        "node": {
                            "id": "team1",
                            "displayName": "My Team"
                        }
                    }
                ],
                "totalCount": 1
            },
            "memberTeamsConnection": {
                "edges": [],
                "totalCount": 0
            },
            "invitationCodes": ["CODE123"]
        }
    }
}
```

### updateCurrentUser

Update current user's information.

**Mutation:**

```graphql
mutation UpdateUser($input: UpdateCurrentUserInput!) {
    updateCurrentUser(input: $input) {
        id
        displayName
        status
    }
}
```

**Input:**

```graphql
input UpdateCurrentUserInput {
    displayName: String
    status: UserStatus
}
```

**Example:**

```graphql
mutation {
    updateCurrentUser(input: { displayName: "Jane Doe" }) {
        id
        displayName
        status
    }
}
```

### deleteCurrentUser

Delete current user account and all associated data.

**Mutation:**

```graphql
mutation {
    deleteCurrentUser
}
```

**Response:** `Void` (null on success, error on failure)

**Note:** User must not own any teams. Transfer ownership first.

## Team Operations

### team

Get a specific team by ID.

**Query:**

```graphql
query GetTeam($id: ID!) {
    team(id: $id) {
        id
        displayName
        owner {
            id
            displayName
            email
        }
        membersConnection {
            edges {
                node {
                    id
                    user {
                        id
                        displayName
                        email
                    }
                    roles
                    createdAt
                }
            }
            totalCount
        }
        memberInvitesConnection {
            edges {
                node {
                    id
                    inviteeEmail
                    invitedBy {
                        id
                        displayName
                    }
                    createdAt
                }
            }
            totalCount
        }
        projectsConnection {
            edges {
                node {
                    id
                    displayName
                    subdomain
                }
            }
            totalCount
        }
    }
}
```

**Type:**

```graphql
type Team {
    id: ID!
    displayName: String!
    owner: User!
    membersConnection: TeamMemberConnection!
    memberInvitesConnection: TeamMemberInviteConnection!
    projectsConnection: ProjectConnection!
}

type TeamMember {
    id: ID!
    user: User!
    roles: [TeamMemberRole!]!
    createdAt: Time!
}

enum TeamMemberRole {
    BILLING
}
```

### createTeam

Create a new team owned by current user.

**Mutation:**

```graphql
mutation CreateTeam($displayName: String!) {
    createTeam(displayName: $displayName) {
        id
        displayName
        owner {
            id
            displayName
        }
    }
}
```

**Input:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `displayName` | String | Yes | Team name (1-50 characters) |

**Example:**

```graphql
mutation {
    createTeam(displayName: "My Startup") {
        id
        displayName
    }
}
```

**Response:**

```json
{
    "data": {
        "createTeam": {
            "id": "team123",
            "displayName": "My Startup"
        }
    }
}
```

### updateTeam

Update team information.

**Mutation:**

```graphql
mutation UpdateTeam($id: ID!, $displayName: String!) {
    updateTeam(id: $id, displayName: $displayName) {
        id
        displayName
    }
}
```

**Input:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | ID | Yes | Team ID |
| `displayName` | String | Yes | New team name |

**Authorization:** Must be team owner

### deleteTeam

Delete a team and all associated projects.

**Mutation:**

```graphql
mutation DeleteTeam($id: ID!) {
    deleteTeam(id: $id)
}
```

**Authorization:** Must be team owner

**Note:** This cascades to delete all projects, members, and invites.

### addTeamMember

Add an existing user to team by email.

**Mutation:**

```graphql
mutation AddMember($teamID: ID!, $email: String!) {
    addTeamMember(teamID: $teamID, email: $email) {
        id
        displayName
        email
    }
}
```

**Input:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `teamID` | ID | Yes | Team ID |
| `email` | String | Yes | User email address |

**Authorization:** Must be team owner

**Note:** User must already have an account.

### removeTeamMember

Remove a user from team.

**Mutation:**

```graphql
mutation RemoveMember($teamID: ID!, $userID: ID!) {
    removeTeamMember(teamID: $teamID, userID: $userID)
}
```

**Authorization:** Must be team owner

### updateTeamMember

Update team member roles.

**Mutation:**

```graphql
mutation UpdateMemberRoles($teamID: ID!, $userID: ID!, $roles: [TeamMemberRole!]!) {
    updateTeamMember(teamID: $teamID, userID: $userID, roles: $roles) {
        id
        user {
            id
            displayName
        }
        roles
    }
}
```

**Example:**

```graphql
mutation {
    updateTeamMember(
        teamID: "team123",
        userID: "user456",
        roles: [BILLING]
    ) {
        id
        roles
    }
}
```

### addTeamMemberInvite

Send email invitation to join team.

**Mutation:**

```graphql
mutation InviteMember($teamID: ID!, $email: String!) {
    addTeamMemberInvite(teamID: $teamID, email: $email) {
        id
        inviteeEmail
        invitedBy {
            id
            displayName
        }
        createdAt
    }
}
```

**Authorization:** Must be team owner

**Note:** Sends invitation email to the provided address.

### removeTeamMemberInvite

Cancel a pending invitation.

**Mutation:**

```graphql
mutation CancelInvite($inviteID: ID!, $inviteeEmail: String!) {
    removeTeamMemberInvite(inviteID: $inviteID, inviteeEmail: $inviteeEmail)
}
```

**Authorization:** Must be team owner

## Project Operations

### project

Get a specific project by ID.

**Query:**

```graphql
query GetProject($id: ID!) {
    project(id: $id) {
        id
        displayName
        subdomain
        team {
            id
            displayName
            owner {
                id
                displayName
            }
        }
    }
}
```

**Type:**

```graphql
type Project {
    id: ID!
    displayName: String!
    subdomain: String!
    team: Team! @authRequired
}
```

### projectBySubdomain

Get a project by its subdomain (public lookup).

**Query:**

```graphql
query GetProjectBySubdomain($subdomain: String!) {
    projectBySubdomain(subdomain: $subdomain) {
        id
        displayName
        subdomain
    }
}
```

**Example:**

```graphql
query {
    projectBySubdomain(subdomain: "my-app") {
        id
        displayName
    }
}
```

**Note:** No authentication required. Used for public project lookups.

### createProject

Create a new project under a team.

**Mutation:**

```graphql
mutation CreateProject($displayName: String!, $teamID: ID!) {
    createProject(displayName: $displayName, teamID: $teamID) {
        id
        displayName
        subdomain
        team {
            id
            displayName
        }
    }
}
```

**Input:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `displayName` | String | Yes | Project name (1-50 characters) |
| `teamID` | ID | Yes | Team that owns the project |

**Authorization:** Must be team owner or member

**Example:**

```graphql
mutation {
    createProject(displayName: "My App", teamID: "team123") {
        id
        displayName
        subdomain
    }
}
```

**Response:**

```json
{
    "data": {
        "createProject": {
            "id": "proj123",
            "displayName": "My App",
            "subdomain": "my-app"
        }
    }
}
```

**Note:** Subdomain is auto-generated from display name and guaranteed unique.

### updateProject

Update project information.

**Mutation:**

```graphql
mutation UpdateProject($id: ID!, $displayName: String!, $subdomain: String) {
    updateProject(id: $id, displayName: $displayName, subdomain: $subdomain) {
        id
        displayName
        subdomain
    }
}
```

**Input:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `id` | ID | Yes | Project ID |
| `displayName` | String | Yes | New project name |
| `subdomain` | String | No | Custom subdomain (must be unique) |

**Authorization:** Must be team owner or member

### deleteProject

Delete a project.

**Mutation:**

```graphql
mutation DeleteProject($id: ID!) {
    deleteProject(id: $id)
}
```

**Authorization:** Must be team owner

## Invitation Codes

Invitation codes enable closed beta/early access control.

### redeemInvitationCode

Redeem an invitation code (grants access).

**Mutation:**

```graphql
mutation RedeemCode($code: String!) {
    redeemInvitationCode(code: $code)
}
```

**Input:**

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `code` | String | Yes | Invitation code |

**Response:** `Boolean` - true if redeemed successfully

**Example:**

```graphql
mutation {
    redeemInvitationCode(code: "BETA2024")
}
```

**Response:**

```json
{
    "data": {
        "redeemInvitationCode": true
    }
}
```

**Note:** Each code can only be used once per user.

## Pagination

The API uses cursor-based pagination with connections.

### Connection Pattern

```graphql
type TeamConnection {
    edges: [TeamEdge!]!
    totalCount: Int!
}

type TeamEdge {
    node: Team!
    cursor: ID!
}
```

### Forward Pagination Input

```graphql
input ForwardPaginationInput {
    first: Int      # Number of items to fetch
    after: ID       # Cursor to start after
}
```

### Example: Paginated Query

```graphql
query GetTeams {
    currentUser {
        ownedTeamsConnection {
            edges {
                cursor
                node {
                    id
                    displayName
                }
            }
            totalCount
        }
    }
}
```

**Response:**

```json
{
    "data": {
        "currentUser": {
            "ownedTeamsConnection": {
                "edges": [
                    {
                        "cursor": "dGVhbTEyMw==",
                        "node": {
                            "id": "team123",
                            "displayName": "Team 1"
                        }
                    },
                    {
                        "cursor": "dGVhbTQ1Ng==",
                        "node": {
                            "id": "team456",
                            "displayName": "Team 2"
                        }
                    }
                ],
                "totalCount": 2
            }
        }
    }
}
```

### Loading More Results

Use `after` cursor to fetch next page:

```graphql
query GetMoreTeams($after: ID!) {
    currentUser {
        ownedTeamsConnection(first: 10, after: $after) {
            edges {
                cursor
                node {
                    id
                    displayName
                }
            }
            totalCount
        }
    }
}
```

## Error Handling

### Error Response Format

```json
{
    "errors": [
        {
            "message": "unauthorized",
            "path": ["currentUser"],
            "extensions": {
                "code": "UNAUTHENTICATED"
            }
        }
    ],
    "data": {
        "currentUser": null
    }
}
```

### Common Errors

**Unauthorized (401)**

```json
{
    "errors": [
        {
            "message": "unauthorized",
            "extensions": { "code": "UNAUTHENTICATED" }
        }
    ]
}
```

**Cause:** No authentication token or invalid token

**Solution:** Provide valid access token in Authorization header

**Forbidden (403)**

```json
{
    "errors": [
        {
            "message": "access denied",
            "extensions": { "code": "FORBIDDEN" }
        }
    ]
}
```

**Cause:** User doesn't have permission for operation

**Solution:** Ensure user owns/is member of the resource

**Not Found (404)**

```json
{
    "errors": [
        {
            "message": "team not found"
        }
    ]
}
```

**Cause:** Resource doesn't exist

**Validation Error (400)**

```json
{
    "errors": [
        {
            "message": "team name must be between 1 and 50 characters"
        }
    ]
}
```

**Cause:** Input validation failed

**Internal Server Error (500)**

```json
{
    "errors": [
        {
            "message": "failed to create project",
            "extensions": { "code": "INTERNAL_SERVER_ERROR" }
        }
    ]
}
```

**Cause:** Server-side error (check logs)

### Error Handling Best Practices

**Frontend:**

```typescript
async function createTeam(name: string) {
    const response = await fetch('/api', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': `Bearer ${accessToken}`,
        },
        body: JSON.stringify({
            query: `
                mutation CreateTeam($name: String!) {
                    createTeam(displayName: $name) {
                        id
                        displayName
                    }
                }
            `,
            variables: { name },
        }),
    });

    const result = await response.json();

    if (result.errors) {
        const error = result.errors[0];

        if (error.extensions?.code === 'UNAUTHENTICATED') {
            // Refresh token or redirect to login
            await refreshAccessToken();
            return createTeam(name); // Retry
        }

        // Show error to user
        throw new Error(error.message);
    }

    return result.data.createTeam;
}
```

## Example Queries and Mutations

### Complete Authentication Flow

```graphql
# 1. Initial authentication
mutation {
    authWithIdentityProvider(
        code: "4/0AY0e-g7X...",
        kind: GoogleOAuth2
    ) {
        accessToken
        refreshToken
    }
}

# 2. Use access token for requests
query {
    currentUser {
        id
        email
        displayName
    }
}

# 3. When access token expires, refresh
mutation {
    authWithRefreshToken(token: "eyJhbGciOiJIUzI1NiIs...") {
        accessToken
        refreshToken
    }
}
```

### Create Team and Project

```graphql
# 1. Create team
mutation {
    createTeam(displayName: "My Startup") {
        id
        displayName
    }
}

# Response: { "id": "team123", "displayName": "My Startup" }

# 2. Create project under team
mutation {
    createProject(displayName: "Mobile App", teamID: "team123") {
        id
        displayName
        subdomain
    }
}

# Response: { "id": "proj456", "displayName": "Mobile App", "subdomain": "mobile-app" }
```

### Manage Team Members

```graphql
# 1. Invite member
mutation {
    addTeamMemberInvite(
        teamID: "team123",
        email: "jane@example.com"
    ) {
        id
        inviteeEmail
        createdAt
    }
}

# 2. Add existing user
mutation {
    addTeamMember(
        teamID: "team123",
        email: "john@example.com"
    ) {
        id
        email
        displayName
    }
}

# 3. Update member roles
mutation {
    updateTeamMember(
        teamID: "team123",
        userID: "user789",
        roles: [BILLING]
    ) {
        id
        roles
    }
}

# 4. Remove member
mutation {
    removeTeamMember(teamID: "team123", userID: "user789")
}
```

### Query Full Team Details

```graphql
query GetTeamDetails($teamId: ID!) {
    team(id: $teamId) {
        id
        displayName
        owner {
            id
            displayName
            email
        }
        membersConnection {
            edges {
                node {
                    id
                    user {
                        id
                        displayName
                        email
                    }
                    roles
                    createdAt
                }
            }
            totalCount
        }
        projectsConnection {
            edges {
                node {
                    id
                    displayName
                    subdomain
                }
            }
            totalCount
        }
        memberInvitesConnection {
            edges {
                node {
                    id
                    inviteeEmail
                    invitedBy {
                        id
                        displayName
                    }
                    createdAt
                }
            }
            totalCount
        }
    }
}
```

### Update User Profile

```graphql
mutation {
    updateCurrentUser(input: {
        displayName: "John Smith"
    }) {
        id
        displayName
        status
    }
}
```

### Delete Account

```graphql
# Must not own any teams
mutation {
    deleteCurrentUser
}
```

### Redeem Invitation Code

```graphql
mutation {
    redeemInvitationCode(code: "BETA2024")
}
```

## Testing with GraphQL Playground

### Access Playground

Navigate to: `http://localhost:8080/api/playground`

(Only available in development mode)

### Set Authentication Header

In playground, add header:

```json
{
    "Authorization": "Bearer eyJhbGciOiJIUzI1NiIs..."
}
```

### Explore Schema

Use the "Docs" tab to browse:
- Available queries
- Available mutations
- Type definitions
- Field documentation

### Example Session

```graphql
# Get current user
query {
    currentUser {
        id
        email
        ownedTeamsConnection {
            edges {
                node {
                    id
                    displayName
                    projectsConnection {
                        edges {
                            node {
                                id
                                displayName
                                subdomain
                            }
                        }
                    }
                }
            }
        }
    }
}
```

## Rate Limiting

Currently no rate limiting is implemented, but consider:

- **Per-IP limits** - Prevent abuse
- **Per-user limits** - Fair usage
- **Complexity-based** - Query cost analysis

## Versioning

API versioning strategy:

- **Current:** No versioning (v1 implicit)
- **Future:** Add `/api/v2` if breaking changes needed
- **Deprecation:** Deprecate fields with `@deprecated` directive

## Best Practices

### Query Only What You Need

```graphql
# ✅ GOOD - Request only needed fields
query {
    currentUser {
        id
        displayName
    }
}

# ❌ BAD - Request everything
query {
    currentUser {
        id
        displayName
        email
        status
        invitationCodes
        ownedTeamsConnection { ... }
        memberTeamsConnection { ... }
    }
}
```

### Use Fragments for Reusability

```graphql
fragment UserInfo on User {
    id
    displayName
    email
}

query {
    currentUser {
        ...UserInfo
    }
}
```

### Use Variables

```graphql
# ✅ GOOD - Use variables
mutation CreateTeam($name: String!) {
    createTeam(displayName: $name) {
        id
    }
}

# Variables: { "name": "My Team" }

# ❌ BAD - Hardcode values
mutation {
    createTeam(displayName: "My Team") {
        id
    }
}
```

### Handle Errors Gracefully

Always check for `errors` in response before accessing `data`.

### Cache Responses

Use GraphQL client caching (Apollo, Relay, urql) for optimal performance.

## Summary

The API provides:

- **Authentication** - OAuth2 + JWT tokens
- **User Management** - Profile, preferences
- **Team Management** - Ownership, members, invites
- **Project Management** - CRUD, subdomain routing
- **Invitation System** - Closed beta access

All operations are type-safe, well-documented, and follow GraphQL best practices.

## Next Steps

- **Read [DEVELOPMENT.md](DEVELOPMENT.md)** - Learn development workflow
- **Read [DEPLOYMENT.md](DEPLOYMENT.md)** - Deploy to production
- **Explore Playground** - Test queries interactively
- **Build Your Frontend** - Integrate with your client app

Happy querying!
