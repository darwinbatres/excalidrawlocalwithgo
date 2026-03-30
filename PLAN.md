# Drawgo — Backend Migration Plan

> **Created**: 2026-03-28
> **Status**: Complete
> **Goal**: Migrate backend from Next.js/Prisma/TypeScript to Go, improve architecture, add enterprise capabilities
>
> _This document is kept for historical reference. The migration is fully complete. See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the current system design._

---

## Table of Contents

1. [Technology Stack](#technology-stack)
2. [Framework Selection: Chi vs Gin vs Others](#framework-selection-chi-vs-gin-vs-others)
3. [Architecture Overview](#architecture-overview)
4. [Feature Inventory & Migration Tracker](#feature-inventory--migration-tracker)
5. [Improvements Over Current Implementation](#improvements-over-current-implementation)
6. [Image Storage Redesign](#image-storage-redesign)
7. [Security Hardening](#security-hardening)
8. [Performance Best Practices](#performance-best-practices)
9. [Testing Strategy](#testing-strategy)
10. [Database Backup & Restore](#database-backup--restore)
11. [Documentation Standards](#documentation-standards)
12. [Project Structure](#project-structure)
13. [Migration Phases](#migration-phases)
14. [Frontend Package Upgrades](#frontend-package-upgrades)
15. [Cleanup Tracker](#cleanup-tracker)
16. [Environment Variables](#environment-variables)
17. [Docker Architecture](#docker-architecture)

---

## Technology Stack

### Go Backend (New)

| Component | Library | Version | Rationale |
|---|---|---|---|
| **HTTP Router** | [go-chi/chi](https://github.com/go-chi/chi) | v5.2+ | Lightweight, idiomatic, 100% net/http compatible, 21.9k stars, used by Cloudflare/Heroku. Composable middleware, route groups, sub-routers. |
| **Database Driver** | [jackc/pgx](https://github.com/jackc/pgx) | v5.9+ | Best-in-class PostgreSQL driver. Native pgx interface (not database/sql) for maximum performance. Connection pooling via pgxpool. LISTEN/NOTIFY, COPY, binary protocol support. 13.5k stars. |
| **Database Migrations** | [golang-migrate/migrate](https://github.com/golang-migrate/migrate) | v4.19+ | Industry standard. 18.3k stars. Supports pgx v5 driver, embedded migrations via io/fs, CLI + library usage. Up/down migrations. |
| **JWT Auth** | [golang-jwt/jwt](https://github.com/golang-jwt/jwt) | v5.3+ | De facto standard for JWT in Go. 9k stars, 73.4k dependents. HMAC-SHA256, RSA, ECDSA support. |
| **Structured Logging** | [rs/zerolog](https://github.com/rs/zerolog) | latest | Zero-allocation JSON logger. Fastest Go logger (outperforms zap). 12.3k stars. context.Context integration, net/http middleware (hlog), slog compatibility. |
| **Rate Limiting** | [go-chi/httprate](https://github.com/go-chi/httprate) | v0.15+ | Sliding window counter (CloudFlare pattern). Integrates natively with chi. In-memory per-instance. |
| **Password Hashing** | [golang.org/x/crypto/bcrypt](https://pkg.go.dev/golang.org/x/crypto/bcrypt) | latest | Standard library extension, same bcrypt as current implementation. |
| **Validation** | [go-playground/validator](https://github.com/go-playground/validator) | v10 | Struct tag-based validation, widely adopted. |
| **CORS** | [go-chi/cors](https://github.com/go-chi/cors) | latest | Native chi integration. |
| **Configuration** | [caarlos0/env](https://github.com/caarlos0/env) | v11+ | Simple struct-based env parsing with defaults. |
| **OAuth2** | [golang.org/x/oauth2](https://pkg.go.dev/golang.org/x/oauth2) | latest | Standard library for Google/GitHub OAuth2 flows. |
| **UUID/CUID** | [rs/xid](https://github.com/rs/xid) or [segmentio/ksuid](https://github.com/segmentio/ksuid) | latest | Globally unique, sortable IDs. Compatible with existing CUID data. |
| **Testing** | stdlib `testing` + [stretchr/testify](https://github.com/stretchr/testify) | latest | Standard Go testing with assertions. |
| **Object Storage** | [minio/minio-go](https://github.com/minio/minio-go) | v7 | S3-compatible client for MinIO/S3. For image file storage (replacing base64). |
| **WebSocket** | [coder/websocket](https://github.com/coder/websocket) | v1.8+ | Minimal, idiomatic WebSocket library (formerly nhooyr/websocket). 5.1k stars. Zero deps, net/http compatible, context.Context-first, zero-alloc reads/writes, concurrent writes, permessage-deflate compression. Now maintained by Coder. |
| **Security Headers** | Custom chi middleware | — | HSTS, CSP, X-Content-Type-Options, X-Frame-Options, Referrer-Policy, Permissions-Policy. Reusable middleware component. |
| **Unit Testing** | stdlib `testing` + [stretchr/testify](https://github.com/stretchr/testify) | latest | Standard Go testing with assertions, mocks, and suites. |
| **Integration Testing** | [testcontainers-go](https://github.com/testcontainers/testcontainers-go) | v0.41+ | Real PostgreSQL + MinIO containers in tests. 4.7k stars, 10.3k dependents. No mocks — test against real infrastructure. |
| **Frontend Unit Testing** | [Vitest](https://github.com/vitest-dev/vitest) | v4.1+ | Next-gen test runner powered by Vite. 16.2k stars, 566k dependents. Native ESM, TypeScript, JSX. Jest-compatible API. |
| **E2E Testing** | [Playwright](https://github.com/microsoft/playwright) | v1.58+ | Cross-browser E2E testing. 72k+ stars. Chromium, Firefox, WebKit. Auto-wait, network interception, trace viewer. |
| **DB Backup** | [pg_dump](https://www.postgresql.org/docs/16/app-pgdump.html) + custom Go service | — | Automated PostgreSQL backups with rotation, stored in MinIO. Configurable schedule, frontend management UI. |

### Retained

| Component | Technology | Notes |
|---|---|---|
| **Database** | PostgreSQL 16+ | Same DB, same schema (with improvements) |
| **Frontend** | Next.js (React) | Frontend-only — no backend routes, served as static/SPA |
| **Container** | Docker + docker-compose | Multi-service orchestration |
| **Tunnel** | Cloudflare Tunnel | Optional HTTPS tunneling |

### New Infrastructure (Added)

| Component | Technology | Rationale |
|---|---|---|
| **Object Storage** | MinIO (S3-compatible) | Local S3-compatible storage for images/files — replaces base64 in DB |
| **Reverse Proxy** | Caddy 2 | Single entry point for frontend + backend + WebSocket routing |

---

## Framework Selection: Chi vs Gin vs Others

### Why Chi over Gin?

Gin is the most popular Go web framework (88.3k stars, 294k dependents, v1.12.0) — so why not use it? The decision comes down to **architectural philosophy**, not popularity.

| Criteria | Chi v5 | Gin v1.12 | Winner | Why It Matters |
|---|---|---|---|---|
| **net/http compatibility** | 100% — uses `http.Handler`, `http.HandlerFunc` | Custom `gin.Context`, `gin.HandlerFunc` | **Chi** | Any stdlib or third-party net/http middleware works with chi out of the box. Gin requires Gin-specific middleware or adapters. |
| **Handler signatures** | `func(w http.ResponseWriter, r *http.Request)` | `func(c *gin.Context)` | **Chi** | Standard Go signatures. No vendor lock-in. Easier to test, easier to onboard Go developers. |
| **Middleware ecosystem** | All net/http middleware works (thousands) | Gin-specific middleware only (gin-contrib) | **Chi** | Larger effective ecosystem. zerolog/hlog, pgx tracing, OpenTelemetry — all work natively with chi. |
| **Context handling** | Standard `context.Context` via `r.Context()` | Custom `gin.Context` (wraps stdlib) | **Chi** | Go's standard context propagation patterns work naturally. |
| **Dependencies** | Zero external dependencies | Multiple deps (json-iterator, go-isatty, etc.) | **Chi** | Smaller binary, fewer supply chain attack vectors. |
| **Go 1.22+ alignment** | Builds on stdlib's new routing patterns | Parallel ecosystem with own router | **Chi** | Go's stdlib added method+pattern routing in 1.22. Chi extends this; Gin ignores it. |
| **Raw routing perf** | ~238k ns/op (GitHub API bench) | ~27k ns/op (GitHub API bench) | Gin | Gin is ~8.7x faster at pure routing. |
| **Real-world perf** | Negligible difference | Negligible difference | **Tie** | Routing is <0.1% of request time. DB queries, I/O, and business logic dominate. |
| **Stars / Dependents** | 21.9k / 30.5k | 88.3k / 294k | Gin | Gin is older and more popular by raw numbers. |
| **Learning curve** | Standard Go — if you know net/http, you know chi | Gin-specific patterns (c.JSON, c.Bind, etc.) | **Chi** | Lower barrier for Go developers. |

### The Core Argument: stdlib Compatibility

Chi's entire value proposition is that it adds **just enough** on top of Go's standard library. Every middleware, every handler, every test helper uses standard Go interfaces:

```go
// Chi handler — standard Go
func GetBoard(w http.ResponseWriter, r *http.Request) {
    id := chi.URLParam(r, "id")
    // ... standard net/http code
}

// Gin handler — Gin-specific
func GetBoard(c *gin.Context) {
    id := c.Param("id")
    c.JSON(200, board)  // Gin-specific response
}
```

With chi, switching to another router (or even raw stdlib) requires changing only the route registration. With Gin, every handler must be rewritten.

### Other Frameworks Considered

| Framework | Stars | Verdict |
|---|---|---|
| **Gin** | 88.3k | Most popular, but custom context creates vendor lock-in. Not idiomatic Go. |
| **Echo** | 31k | Similar to Gin (custom context). Smaller ecosystem than Gin with same drawbacks. |
| **Fiber** | 35k | Built on fasthttp (NOT net/http). Incompatible with Go stdlib entirely. Fast but isolated ecosystem. |
| **gorilla/mux** | 21k | Went into archive in 2022, revived but lost community momentum. No built-in middleware. |
| **stdlib (net/http)** | — | Go 1.22+ added method routing. Viable but lacks middleware composition (no groups, no sub-routers). |
| **Chi** ✅ | 21.9k | Best balance: idiomatic Go, stdlib-compatible, rich middleware, composable routing, zero deps. |

### Bottom Line

> **We optimize for long-term maintainability and Go idiom alignment, not GitHub stars.** Chi gives us the full Go ecosystem for free while Gin locks us into a framework-specific world. For an enterprise application that will live for years, stdlib compatibility is the correct bet.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        Docker Compose                           │
│                                                                 │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐          │
│  │   Frontend   │  │  Go Backend  │  │  PostgreSQL  │          │
│  │  (Next.js)   │  │  (chi + pgx) │  │    16+       │          │
│  │  Port 3000   │  │  Port 8080   │  │  Port 5432   │          │
│  └──────┬───────┘  └──────┬───────┘  └──────────────┘          │
│         │                 │                                     │
│         │  REST + WS      │          ┌──────────────┐           │
│         └────────────────►│          │    MinIO      │           │
│                           ├─────────►│  (S3-compat) │           │
│         ┌────────────────►│          │  Port 9000   │           │
│         │  WebSocket      │          └──────────────┘           │
│         │  (live cursors  │                                     │
│         │   + board sync) │          ┌──────────────┐           │
│  ┌──────┴───────┐         │          │  Cloudflare  │           │
│  │   Viewers    │         │          │   Tunnel     │           │
│  │ (read-only)  │         │          │  (optional)  │           │
│  └──────────────┘         │          └──────────────┘           │
└─────────────────────────────────────────────────────────────────┘
```

---

## Feature Inventory & Migration Tracker

### Legend
- ⬜ Not started
- 🔄 In progress
- ✅ Completed
- 🔧 Improved (migrated + enhanced)
- 🆕 New feature (not in original)
- 🗑️ Removed

---

### 1. Authentication & Session Management

| # | Feature | Current Implementation | Go Implementation | Status | Notes |
|---|---|---|---|---|---|
| 1.1 | Email/password login | NextAuth credentials provider, bcrypt(12) | Custom handler, bcrypt(12), JWT issued manually | ✅ | |
| 1.2 | Google OAuth2 login | NextAuth Google provider | golang.org/x/oauth2 + Google endpoint | ✅ | |
| 1.3 | GitHub OAuth2 login | NextAuth GitHub provider | golang.org/x/oauth2 + GitHub endpoint | ✅ | |
| 1.4 | JWT session tokens | NextAuth JWT strategy, 30-day expiry | golang-jwt/jwt v5, httpOnly cookie, 30-day | ✅ | |
| 1.5 | Session cookie management | next-auth.session-token cookie | Custom cookie with same security properties | ✅ | |
| 1.6 | User enumeration prevention | Same error for missing user/wrong password | Same pattern | ✅ | |
| 1.7 | Login rate limiting | rateLimiters.auth: 10 req/min | httprate: 10 req/min on /auth/login | ✅ | |
| 1.8 | Login audit logging | auth.login event logged | Same | ✅ | |
| 1.9 | JWT refresh tokens | ❌ Not implemented | 🆕 Short-lived access + long-lived refresh | ✅ | **IMPROVEMENT** |
| 1.10 | Password reset flow | ❌ Not implemented | 🆕 Email-based password reset | ⬜ | **IMPROVEMENT** (future) |

### 2. Authorization & Access Control

| # | Feature | Current Implementation | Go Implementation | Status | Notes |
|---|---|---|---|---|---|
| 2.1 | Org role hierarchy | OWNER(100) > ADMIN(75) > MEMBER(50) > VIEWER(25) | Same hierarchy, constants in access pkg | ✅ | |
| 2.2 | Board role hierarchy | OWNER(100) > EDITOR(50) > VIEWER(25) | Same | ✅ | |
| 2.3 | hasOrgRole() | Prisma query, role comparison | SQL query, role comparison | ✅ | |
| 2.4 | getBoardPermission() | Owner check → org role → BoardPermission → fallback | Same logic | ✅ | |
| 2.5 | canViewBoard() | Derived from getBoardPermission | Same | ✅ | |
| 2.6 | canEditBoard() | OWNER or EDITOR | Same | ✅ | |
| 2.7 | canDeleteBoard() | OWNER only | Same | ✅ | |
| 2.8 | canManageBoardPermissions() | OWNER only | Same | ✅ | |
| 2.9 | canManageMembers() | ADMIN+ | Same | ✅ | |
| 2.10 | grantBoardPermission() | Upsert BoardPermission | Same | ✅ | |
| 2.11 | revokeBoardPermission() | Delete BoardPermission | Same | ✅ | |
| 2.12 | getAccessibleBoards() | Complex Prisma query | Same logic in SQL | ✅ | |
| 2.13 | Auth middleware | withAuth() extracts session, attaches user | chi middleware, JWT validation, context injection | ✅ | |
| 2.14 | RBAC middleware per route | Manual checks in handlers | 🔧 Declarative chi middleware per route group | ✅ | **IMPROVEMENT** |

### 3. Board Management

| # | Feature | Current Implementation | Go Implementation | Status | Notes |
|---|---|---|---|---|---|
| 3.1 | Create board | POST /api/boards — transaction: Board + BoardVersion + audit | POST /api/v1/boards — pgx transaction | ✅ | |
| 3.2 | List boards (paginated) | GET /api/boards?orgId&query&tags&archived&limit&offset | GET /api/v1/boards?same_params | ✅ | |
| 3.3 | Get board with latest version | GET /api/boards/[id] — joins latestVersion | GET /api/v1/boards/{id} | ✅ | |
| 3.4 | Update board metadata | PATCH /api/boards/[id] — title, description, tags, isArchived | PATCH /api/v1/boards/{id} | ✅ | |
| 3.5 | Delete board | DELETE /api/boards/[id] — cascade | DELETE /api/v1/boards/{id} | ✅ | |
| 3.6 | Archive board (soft delete) | DELETE /api/boards/[id]?archive=true | PATCH /api/v1/boards/{id}/archive | ✅ | 🔧 Separate endpoint |
| 3.7 | Board view audit logging | board.view event on GET | Same | ✅ | |
| 3.8 | Search boards | ILIKE on title, description, searchContent | Same + 🔧 pg_trgm optimization | ✅ | |
| 3.9 | Tag filtering | PostgreSQL array @> operator | Same | ✅ | |
| 3.10 | Board etag (optimistic locking) | CUID-based etag, 409 on mismatch | Same | ✅ | |

### 4. Board Versioning

| # | Feature | Current Implementation | Go Implementation | Status | Notes |
|---|---|---|---|---|---|
| 4.1 | Save new version | POST /api/boards/[id]/save — orphan cleanup + search extract + etag | POST /api/v1/boards/{id}/versions | ✅ | |
| 4.2 | Search content extraction | Text elements + markdown + rich text → searchContent (50KB max) | Same logic in Go | ✅ | |
| 4.3 | Orphaned file cleanup on save | Scans elements for fileId, removes unused files | Same + 🔧 S3 cleanup | ✅ | |
| 4.4 | Optimistic concurrency (etag) | expectedEtag check, 409 Conflict response | Same | ✅ | |
| 4.5 | List versions (paginated) | GET /api/boards/[id]/versions | GET /api/v1/boards/{id}/versions | ✅ | |
| 4.6 | Get specific version | GET /api/boards/[id]/versions/[version] | GET /api/v1/boards/{id}/versions/{version} | ✅ | |
| 4.7 | Restore version | POST /api/boards/[id]/versions/[version] | POST /api/v1/boards/{id}/versions/{version}/restore | ✅ | 🔧 Clearer REST endpoint |
| 4.8 | Thumbnail storage | base64 data URL in Board.thumbnail | 🔧 Upload to MinIO, store URL | ✅ | **IMPROVEMENT** |
| 4.9 | Version label | Optional user-defined label per version | Same | ✅ | |
| 4.10 | 16MB body size limit | Next.js config for save endpoint | Go http.MaxBytesReader | ✅ | |

### 5. Organization Management

| # | Feature | Current Implementation | Go Implementation | Status | Notes |
|---|---|---|---|---|---|
| 5.1 | Create org | POST /api/orgs — transaction: Org + Membership(OWNER) | POST /api/v1/orgs | ✅ | |
| 5.2 | List user's orgs | GET /api/orgs — with memberCount, boardCount | GET /api/v1/orgs | ✅ | |
| 5.3 | Update org (rename) | PATCH /api/orgs/[id] — OWNER only | PATCH /api/v1/orgs/{id} | ✅ | |
| 5.4 | Delete org | DELETE /api/orgs/[id] — OWNER, ≥2 orgs, 0 boards | DELETE /api/v1/orgs/{id} | ✅ | |
| 5.5 | Slug validation | lowercase alphanumeric + hyphens, 3-50 chars | Same validation | ✅ | |
| 5.6 | Slug immutability | Cannot change after creation | Same | ✅ | |

### 6. Member Management

| # | Feature | Current Implementation | Go Implementation | Status | Notes |
|---|---|---|---|---|---|
| 6.1 | List org members | GET /api/orgs/[id]/members — with user details | GET /api/v1/orgs/{id}/members | ✅ | |
| 6.2 | Invite member | POST /api/orgs/[id]/members — by email, default MEMBER | POST /api/v1/orgs/{id}/members | ✅ | |
| 6.3 | Update member role | PATCH /api/orgs/[id]/members — ADMIN only | PATCH /api/v1/orgs/{id}/members/{membershipId} | ✅ | 🔧 RESTful URL |
| 6.4 | Remove member | DELETE /api/orgs/[id]/members — ADMIN only | DELETE /api/v1/orgs/{id}/members/{membershipId} | ✅ | 🔧 RESTful URL |

### 7. Audit Logging

| # | Feature | Current Implementation | Go Implementation | Status | Notes |
|---|---|---|---|---|---|
| 7.1 | Log audit events | auditService.logAuditEvent() — non-blocking, fail-silent | Same: goroutine, non-blocking | ✅ | |
| 7.2 | Audit event schema | orgId, actorId, action, targetType, targetId, ip, userAgent, metadata | Same | ✅ | |
| 7.3 | Query audit logs | GET /api/audit — filters: actor, action, target, date range | GET /api/v1/audit — same filters | ✅ | |
| 7.4 | Audit stats | GET /api/audit?stats=true — aggregated by action | GET /api/v1/audit/stats | ✅ | 🔧 Separate endpoint |
| 7.5 | IP extraction | x-forwarded-for or socket.remoteAddress | chi middleware.RealIP + request context | ✅ | |
| 7.6 | User-Agent tracking | From request headers | Same | ✅ | |
| 7.7 | Audit event actions | 20 action types (board.*, version.*, org.*, member.*, auth.*) | Same actions | ✅ | |
| 7.8 | Audit log export | ❌ Not implemented | 🆕 CSV/JSON export endpoint | ⬜ | **NEW** |

### 8. Storage Management

| # | Feature | Current Implementation | Go Implementation | Status | Notes |
|---|---|---|---|---|---|
| 8.1 | Board storage breakdown | GET /api/boards/[id]/storage — calculates from JSONB | GET /api/v1/boards/{id}/storage | ✅ | |
| 8.2 | Workspace storage | GET /api/orgs/[id]/storage — aggregates all boards | GET /api/v1/orgs/{id}/storage | ✅ | |
| 8.3 | Cleanup orphaned files | POST /api/boards/[id]/cleanup — creates new version | POST /api/v1/boards/{id}/cleanup | ✅ | |
| 8.4 | Storage categories | standard, embedded, markdown, richText, searchText, appState, thumbnail, versions | Same + S3 storage stats | ✅ | |
| 8.5 | Size limits enforcement | Constants: MAX_IMAGE_SIZE(50MB), MAX_BOARD_SIZE(500MB), etc. | Same limits, enforced in Go | ✅ | |

### 9. Rate Limiting

| # | Feature | Current Implementation | Go Implementation | Status | Notes |
|---|---|---|---|---|---|
| 9.1 | API rate limit | 60 req/min, in-memory sliding window | httprate: 60 req/min, sliding window | ✅ | |
| 9.2 | Auth rate limit | 10 req/min for login | httprate: 10 req/min on auth routes | ✅ | |
| 9.3 | Write rate limit | 60 req/min for save operations | httprate: 60 req/min on write routes | ✅ | |
| 9.4 | Rate limit headers | X-RateLimit-Limit, Remaining, Reset, Retry-After | httprate built-in headers | ✅ | |
| 9.5 | IP-based keying | x-forwarded-for or socket | httprate.KeyByIP (with RealIP) | ✅ | |
| 9.6 | Horizontal rate limiting | ❌ Not implemented (in-memory only) | Future: pluggable backend for multi-instance | ⬜ | **IMPROVEMENT** |

### 10. Health & Stats

| # | Feature | Current Implementation | Go Implementation | Status | Notes |
|---|---|---|---|---|---|
| 10.1 | Health check | GET /api/health — DB check, memory stats | GET /api/v1/health — DB ping, memory, goroutines | ✅ | |
| 10.2 | System stats | GET /api/stats — overview, boards, storage, user stats | GET /api/v1/stats | ✅ | |
| 10.3 | Readiness probe | Same as health | 🆕 GET /api/v1/ready — separate from liveness | ✅ | **IMPROVEMENT** |
| 10.4 | Prometheus metrics | ❌ Not implemented | 🆕 GET /metrics — standard Prometheus format | ⬜ | **NEW** (future) |

### 11. Database & Data

| # | Feature | Current Implementation | Go Implementation | Status | Notes |
|---|---|---|---|---|---|
| 11.1 | User model | Prisma schema | SQL migration + Go struct | ✅ | |
| 11.2 | Organization model | Prisma schema | SQL migration + Go struct | ✅ | |
| 11.3 | Membership model | Prisma schema | SQL migration + Go struct | ✅ | |
| 11.4 | Board model | Prisma schema | SQL migration + Go struct | ✅ | |
| 11.5 | BoardVersion model | Prisma schema | SQL migration + Go struct | ✅ | |
| 11.6 | BoardPermission model | Prisma schema | SQL migration + Go struct | ✅ | |
| 11.7 | BoardAsset model | Prisma schema | SQL migration + Go struct (enhanced for S3) | ✅ | |
| 11.8 | AuditEvent model | Prisma schema | SQL migration + Go struct | ✅ | |
| 11.9 | ShareLink model | Prisma schema | SQL migration + Go struct | ✅ | |
| 11.10 | NextAuth Account | Prisma schema | 🔧 Simplified OAuth account table | ✅ | |
| 11.11 | NextAuth Session | Prisma schema | 🗑️ Removed (JWT-only, no DB sessions) | ✅ | |
| 11.12 | VerificationToken | Prisma schema | 🗑️ Removed (or reimplemented if needed) | ✅ | |
| 11.13 | GIN trigram indexes | Migration 20260112000000 | Same indexes, carried over | ✅ | |
| 11.14 | Cascade deletes | Prisma onDelete: Cascade | SQL ON DELETE CASCADE | ✅ | |
| 11.15 | Database seed | seed.ts — demo user | 🔧 Go seed command or SQL script | ✅ | |

### 12. Real-Time & Sharing

| # | Feature | Current Implementation | Go Implementation | Status | Notes |
|---|---|---|---|---|---|
| 12.1 | Share link generation | ShareLink model exists but barely used | 🔧 Full share link system — generate/revoke/expire | ✅ | **IMPROVEMENT** |
| 12.2 | View-only board access | ❌ Not implemented | 🆕 GET /api/v1/share/{token} — returns board read-only | ✅ | **NEW** |
| 12.3 | Share link expiration | ShareLink.expiresAt exists | Enforce expiration + optional never-expire | ✅ | |
| 12.4 | Share link revocation | ❌ Not implemented | 🆕 DELETE /api/v1/boards/{id}/share/{token} | ✅ | **NEW** |
| 12.5 | WebSocket connection | ❌ Not implemented | 🆕 WS /api/v1/ws/boards/{id} — board room via coder/websocket | ✅ | **NEW** |
| 12.6 | Real-time board broadcast | ❌ Not implemented | 🆕 Owner saves → broadcast scene to all viewers in room | ✅ | **NEW** |
| 12.7 | Live cursor tracking | ❌ Not implemented | 🆕 Viewers + owner cursor positions broadcast via WS | ✅ | **NEW** (nice-to-have) |
| 12.8 | Viewer presence list | ❌ Not implemented | 🆕 Show who is currently viewing the board | ✅ | **NEW** |
| 12.9 | Connection management | ❌ Not implemented | 🆕 Hub/room pattern — track per-board connections, cleanup on disconnect | ✅ | **NEW** |
| 12.10 | WS authentication | ❌ Not implemented | 🆕 JWT for owner, share token for viewers — validated on WS upgrade | ✅ | **NEW** |
| 12.11 | WS rate limiting | ❌ Not implemented | 🆕 Max connections per board, max message rate per client | ✅ | **NEW** |
| 12.12 | WS heartbeat/ping-pong | ❌ Not implemented | 🆕 Periodic ping to detect stale connections, auto-cleanup | ✅ | **NEW** |
| 12.13 | Multi-device sync | ❌ Not implemented | 🆕 Same user on multiple devices sees synced state via WS | ✅ | **NEW** |

### 13. Security Hardening

| # | Feature | Current Implementation | Go Implementation | Status | Notes |
|---|---|---|---|---|---|
| 13.1 | HTTPS enforcement | Cloudflare Tunnel handles TLS | 🔧 HSTS header + redirect middleware when tunnel active | ✅ | |
| 13.2 | Security headers middleware | ❌ Not implemented | 🆕 Reusable middleware: CSP, X-Content-Type-Options, X-Frame-Options, Referrer-Policy, Permissions-Policy | ✅ | **NEW** |
| 13.3 | Content Security Policy | ❌ Not implemented | 🆕 Strict CSP: script-src 'self', connect-src to API origin, img-src 'self' + S3 | ✅ | **NEW** |
| 13.4 | CSRF protection | Next.js built-in (same-origin) | 🆕 SameSite=Strict cookies + Origin header validation on mutations | ✅ | **NEW** |
| 13.5 | Cookie security | NextAuth manages cookies | 🔧 httpOnly, Secure (when HTTPS), SameSite=Lax, Path=/, __Host- prefix | ✅ | **IMPROVEMENT** |
| 13.6 | Input sanitization | Zod validation on some endpoints | 🔧 go-playground/validator on all inputs + HTML sanitization on text fields | ✅ | **IMPROVEMENT** |
| 13.7 | SQL injection prevention | Prisma parameterized queries | pgx parameterized queries (never string concatenation) | ✅ | |
| 13.8 | Request size limits | 16MB on save endpoint only | 🔧 Global MaxBytesReader + per-route overrides | ✅ | **IMPROVEMENT** |
| 13.9 | File upload validation | Basic MIME check | 🔧 Magic byte validation + allowlist (image/png, image/jpeg, image/svg+xml, image/gif, image/webp) | ✅ | **IMPROVEMENT** |
| 13.10 | Share token entropy | CUID tokens | 🔧 crypto/rand 32-byte tokens, base64url encoded (256-bit entropy) | ✅ | **IMPROVEMENT** |
| 13.11 | Brute-force protection | 10/min auth rate limit | 🔧 Progressive delays + account lockout after N failures | ✅ | **IMPROVEMENT** |
| 13.12 | Dependency security | npm audit | 🆕 govulncheck in CI + Dependabot for Go modules | ✅ | **NEW** |
| 13.13 | Secrets management | .env file | 🔧 .env + Docker secrets support + never log secrets | ✅ | **IMPROVEMENT** |
| 13.14 | Audit trail for security events | Partial (auth.login only) | 🔧 Log: failed logins, permission denied, rate limit hits, share access | ✅ | **IMPROVEMENT** |
| 13.15 | CORS strict mode | Basic CORS | 🔧 Explicit origin allowlist, no wildcards in production, credentials mode | ✅ | **IMPROVEMENT** |
| 13.16 | WebSocket origin validation | ❌ Not implemented | 🆕 Validate Origin header on WS upgrade against allowed origins | ✅ | **NEW** |
| 13.17 | Clickjacking protection | ❌ Not implemented | 🆕 X-Frame-Options: DENY + CSP frame-ancestors 'none' | ✅ | **NEW** |
| 13.18 | Error information leakage | Stack traces in dev, generic in prod | 🔧 Never expose internal errors to clients; log internally, return safe codes | ✅ | **IMPROVEMENT** |

### 14. Performance

| # | Feature | Current Implementation | Go Implementation | Status | Notes |
|---|---|---|---|---|---|
| 14.1 | Connection pooling | Prisma implicit | 🔧 pgxpool: explicit pool size, idle timeout, max lifetime, health checks | ✅ | **IMPROVEMENT** |
| 14.2 | Response compression | ❌ Not implemented | 🆕 chi middleware.Compress (gzip/brotli) | ✅ | **NEW** |
| 14.3 | HTTP caching headers | ❌ Not implemented | 🆕 ETag + Cache-Control on board reads, immutable on S3 assets | ✅ | **NEW** |
| 14.4 | Database query optimization | Prisma-generated queries | 🔧 Hand-tuned SQL, EXPLAIN ANALYZE verified, proper indexes | ✅ | **IMPROVEMENT** |
| 14.5 | Prepared statements | Prisma handles | 🔧 pgx auto-prepared statements (statement cache) | ✅ | **IMPROVEMENT** |
| 14.6 | JSON serialization | Next.js JSON.stringify | 🔧 segmentio/encoding or stdlib encoding/json/v2 (Go 1.24+) for faster JSON | ✅ | **IMPROVEMENT** |
| 14.7 | Static asset caching | Next.js built-in | Frontend: long Cache-Control for hashed assets | ✅ | |
| 14.8 | Lazy loading board data | Load full scene on GET | 🆕 Load scene metadata first, lazy-load files on demand from S3 | ✅ | **NEW** |
| 14.9 | WebSocket message batching | ❌ Not implemented | 🆕 Batch cursor updates (50ms window) to reduce WS message volume | ✅ | **NEW** |
| 14.10 | Keep-Alive | Default Node.js keep-alive | Go net/http keep-alive by default + configurable idle timeout | ✅ | |
| 14.11 | Request timeout budget | ❌ Not implemented | 🆕 context.WithTimeout per route group: reads 10s, writes 30s, uploads 120s | ✅ | **NEW** |
| 14.12 | Database connection health | Prisma auto-reconnect | 🔧 pgxpool health check, pre-ping, automatic reconnect on pool checkout | ✅ | **IMPROVEMENT** |
| 14.13 | Minimal allocations | N/A (JavaScript) | 🆕 Go's value types + zerolog zero-alloc + sync.Pool for hot paths | ✅ | **NEW** |

### 15. Middleware & Cross-Cutting

| # | Feature | Current Implementation | Go Implementation | Status | Notes |
|---|---|---|---|---|---|
| 15.1 | Request logging | Console logging in API routes | zerolog hlog middleware — structured JSON | ✅ | 🔧 Major improvement |
| 15.2 | Request ID | Not standardized | 🆕 chi middleware.RequestID | ✅ | **NEW** |
| 15.3 | Panic recovery | Try/catch in handlers | chi middleware.Recoverer | ✅ | |
| 15.4 | CORS | Next.js config | go-chi/cors middleware | ✅ | |
| 15.5 | Real IP extraction | Manual x-forwarded-for parsing | chi middleware.RealIP | ✅ | |
| 15.6 | Request timeout | Not implemented | chi middleware.Timeout (per route group) | ✅ | **NEW** |
| 15.7 | Gzip/Brotli compression | Not implemented | chi middleware.Compress | ✅ | **NEW** |
| 15.8 | Content-Type enforcement | Manual checks | chi middleware.AllowContentType | ✅ | **NEW** |
| 15.9 | Graceful shutdown | Not implemented | 🆕 context.Context + signal handling | ✅ | **IMPROVEMENT** |
| 15.10 | Structured error responses | ApiError class, inconsistent format | 🔧 Unified error response envelope | ✅ | **IMPROVEMENT** |
| 15.11 | Security headers | Not implemented | 🆕 Reusable middleware (see Security Hardening §13) | ✅ | **NEW** |

### 16. Testing

| # | Feature | Current Implementation | Go Implementation | Status | Notes |
|---|---|---|---|---|---|
| 16.1 | Go unit tests — auth service | Not applicable | stdlib testing + testify assertions | ✅ | **NEW** |
| 16.2 | Go unit tests — boards service | Not applicable | stdlib testing + testify assertions | ✅ | **NEW** |
| 16.3 | Go unit tests — versions service | Not applicable | stdlib testing + testify assertions | ✅ | **NEW** |
| 16.4 | Go unit tests — orgs/members service | Not applicable | stdlib testing + testify assertions | ✅ | **NEW** |
| 16.5 | Go unit tests — sharing service | Not applicable | stdlib testing + testify assertions | ✅ | **NEW** |
| 16.6 | Go unit tests — storage service | Not applicable | stdlib testing + testify assertions | ✅ | **NEW** |
| 16.7 | Go unit tests — backup service | Not applicable | stdlib testing + testify assertions | ✅ | **NEW** |
| 16.8 | Go unit tests — middleware (auth, rate limit, CORS) | Not applicable | httptest + chi test helpers | ✅ | **NEW** |
| 16.9 | Go integration tests — auth flow | Not applicable | testcontainers-go (real PG + MinIO) | ✅ | **NEW** |
| 16.10 | Go integration tests — board lifecycle | Not applicable | testcontainers-go (create → edit → version → delete) | ✅ | **NEW** |
| 16.11 | Go integration tests — sharing flow | Not applicable | testcontainers-go (share → view-only → revoke) | ✅ | **NEW** |
| 16.12 | Go integration tests — WebSocket cursors | Not applicable | testcontainers-go + coder/websocket client | ⬜ | **NEW** |
| 16.13 | Go integration tests — backup & restore | Not applicable | testcontainers-go (pg_dump → MinIO → pg_restore) | ✅ | **NEW** |
| 16.14 | Frontend unit tests — components | Not applicable | Vitest + React Testing Library | ⬜ | **NEW** |
| 16.15 | Frontend unit tests — API client | Not applicable | Vitest + MSW (Mock Service Worker) | ⬜ | **NEW** |
| 16.16 | E2E — authentication flow | Not applicable | Playwright (Chromium + Firefox + WebKit) | ⬜ | **NEW** |
| 16.17 | E2E — board CRUD operations | Not applicable | Playwright cross-browser | ⬜ | **NEW** |
| 16.18 | E2E — sharing & view-only mode | Not applicable | Playwright multi-tab/multi-context | ⬜ | **NEW** |
| 16.19 | E2E — backup settings & restore | Not applicable | Playwright admin flow | ⬜ | **NEW** |

### 17. Database Backup & Restore

| # | Feature | Current Implementation | Go Implementation | Status | Notes |
|---|---|---|---|---|---|
| 17.1 | Automated scheduled backups | Not implemented | 🆕 Go cron scheduler + pg_dump via os/exec | ✅ | **NEW** |
| 17.2 | On-demand manual backup | Not implemented | 🆕 POST /api/v1/backups trigger | ✅ | **NEW** |
| 17.3 | Backup storage to MinIO/S3 | Not implemented | 🆕 minio-go SDK upload | ✅ | **NEW** |
| 17.4 | Backup encryption at rest | Not implemented | 🆕 AES-256 encryption before upload | ⬜ | **NEW** |
| 17.5 | Backup retention rotation | Not implemented | 🆕 Daily/weekly/monthly rotation policy | ✅ | **NEW** |
| 17.6 | List available backups | Not implemented | 🆕 GET /api/v1/backups with size/date/status | ✅ | **NEW** |
| 17.7 | Download backup file | Not implemented | 🆕 GET /api/v1/backups/:id/download (presigned URL) | ✅ | **NEW** |
| 17.8 | Restore from backup | Not implemented | 🆕 POST /api/v1/backups/:id/restore (pg_restore) | ✅ | **NEW** |
| 17.9 | Backup schedule config from frontend | Not implemented | 🆕 Settings UI → PUT /api/v1/settings/backup | ✅ | **NEW** |
| 17.10 | Backup status & progress tracking | Not implemented | 🆕 SSE/polling for backup job status | ⬜ | **NEW** |
| 17.11 | Backup audit trail | Not implemented | 🆕 Log all backup/restore events to audit table | ✅ | **NEW** |
| 17.12 | Backup health check | Not implemented | 🆕 Verify last backup age in /health endpoint | ✅ | **NEW** |
| 17.13 | Backup failure alerting | Not implemented | 🆕 Log + optional webhook on backup failure | ⬜ | **NEW** |

### 18. Documentation

| # | Feature | Current Implementation | Go Implementation | Status | Notes |
|---|---|---|---|---|---|
| 18.1 | Root README.md | Basic README | 🔧 Full overview, quick start, Mermaid architecture diagram | ✅ | **IMPROVEMENT** |
| 18.2 | Backend README.md | Not applicable | 🆕 Go setup, API overview, testing guide | ✅ | **NEW** |
| 18.3 | API reference (docs/API.md) | Not implemented | 🆕 Full endpoint reference with examples | ✅ | **NEW** |
| 18.4 | Deployment guide (docs/DEPLOYMENT.md) | Not implemented | 🆕 Docker, DigitalOcean, Cloudflare Tunnel, backups | ✅ | **NEW** |
| 18.5 | Architecture doc (docs/ARCHITECTURE.md) | Not implemented | 🆕 System design, ER diagram, data flow diagrams | ✅ | **NEW** |
| 18.6 | CONTRIBUTING.md update | Basic CONTRIBUTING | 🔧 New dev setup, Go + frontend testing, PR guidelines | ✅ | **IMPROVEMENT** |
| 18.7 | Mermaid diagrams for all architecture views | Not implemented | 🆕 Version-controlled diagrams in Markdown | ✅ | **NEW** |

---

## Improvements Over Current Implementation

### Critical Improvements

#### 1. Image Storage: Base64 → Object Storage (MinIO/S3)

**Current Problem**: Images are stored as base64 data URLs inside `sceneJson` JSONB column. This means:
- A 5MB image becomes ~6.7MB when base64-encoded
- Every version save duplicates ALL images in the new version's JSONB
- Database bloats rapidly — a board with 10 images × 50 versions = 10 × 6.7MB × 50 = **3.35 GB** in PostgreSQL
- Slow queries on large JSONB
- Backup size explosion
- Cannot serve images with CDN caching
- Cannot deduplicate identical images across boards

**Solution**: See [Image Storage Redesign](#image-storage-redesign) section below.

#### 2. Structured Logging

**Current**: Console.log-style logging, no correlation, no structured format.
**Improved**: zerolog JSON logging with request IDs, user context, timing, log levels. Ready for log aggregation (ELK, Datadog, etc.).

#### 3. Rate Limiting: In-Memory → Pluggable Backend

**Current**: Single-instance in-memory Map, not usable with load balancers.
**Improved**: httprate with sliding window counter, in-memory per-instance (sufficient for single-server deployments).

#### 4. Graceful Shutdown

**Current**: No graceful shutdown — in-flight requests killed on SIGTERM.
**Improved**: context.Context propagation, signal handling, drain connections before exit.

#### 5. Request Tracing

**Current**: No request tracing or correlation IDs.
**Improved**: Request ID injected into every request, propagated through context, logged in every log line and audit event.

#### 6. API Versioning

**Current**: No API versioning (/api/boards).
**Improved**: Versioned API (/api/v1/boards) for backward-compatible evolution.

#### 7. Unified Error Responses

**Current**: Inconsistent error formats across endpoints.
**Improved**: Standard envelope:
```json
{
  "error": {
    "code": "BOARD_NOT_FOUND",
    "message": "Board not found",
    "details": {}
  }
}
```

#### 8. JWT Refresh Tokens

**Current**: Single long-lived JWT (30 days).
**Improved**: Short-lived access token (15 min) + long-lived refresh token (30 days). Reduces risk exposure from token theft.

#### 9. Health Check Separation

**Current**: Single /api/health endpoint.
**Improved**: Separate liveness (/health) and readiness (/ready) probes for Kubernetes-style orchestration.

#### 10. Connection Pooling

**Current**: Prisma manages connection pool implicitly.
**Improved**: pgxpool with explicit pool size, health checks, idle timeouts, max conn lifetime configuration.

#### 11. View-Only Board Sharing with Live Cursors

**Current**: ShareLink model exists in schema but is not functionally used. No real-time features.
**Improved**: Full share link system with WebSocket-powered real-time viewing. Owners generate a share token → viewers open the link → WebSocket connection established → viewers see live board state + all cursors. Uses coder/websocket (stdlib-compatible, zero deps, maintained by Coder).

Architecture:
```
┌──────────────────────────────────────────┐
│              Go Backend                  │
│                                          │
│  ┌─────────┐    ┌──────────────────┐     │
│  │ REST    │    │   WebSocket Hub  │     │
│  │ Handler │    │                  │     │
│  │         │    │  Board Room 1    │     │
│  │ POST    │───►│   ├─ Owner conn  │     │
│  │ /save   │    │   ├─ Viewer A    │     │
│  │         │    │   └─ Viewer B    │     │
│  │         │    │                  │     │
│  │         │    │  Board Room 2    │     │
│  │         │    │   └─ Owner conn  │     │
│  └─────────┘    └──────────────────┘     │
└──────────────────────────────────────────┘

Message types:
  → scene_update   (owner → all viewers)
  → cursor_move    (any participant → all others)
  → presence_join  (server → all in room)
  → presence_leave (server → all in room)
  → heartbeat      (client ↔ server)
```

Reusable components:
- **Hub**: Generic pub/sub room manager (reusable for future real-time features)
- **Client**: WebSocket client wrapper with read/write pumps, heartbeat, reconnect
- **Message**: Typed message envelope (type + payload), serialized as JSON

#### 12. Multi-Device Sync

**Current**: No sync — if you open the same board on two devices, last-save-wins.
**Improved**: Same WebSocket infrastructure. When user is authenticated on multiple devices, all devices join the same board room and receive scene_update broadcasts. Etag concurrency control prevents conflicting saves.

---

## Image Storage Redesign

### Current State

```
Board Save → sceneJson.files[fileId].dataURL = "data:image/png;base64,iVBOR..."
                                                  └── 33% size overhead
                                                  └── Stored in PostgreSQL JSONB
                                                  └── Duplicated in every version
```

### New Design

```
Board Save → Extract new files from scene
           → Upload to MinIO (S3-compatible)
           → Replace dataURL with presigned URL reference
           → Store file metadata in board_assets table
           → Scene only stores fileId references

┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   Frontend   │────►│  Go Backend  │────►│    MinIO      │
│  (uploads)   │     │  (processes) │     │  (stores)     │
└──────────────┘     └──────┬───────┘     └──────────────┘
                            │
                     ┌──────▼───────┐
                     │  PostgreSQL  │
                     │ (metadata    │
                     │  + scene     │
                     │  references) │
                     └──────────────┘
```

### Storage Schema

**MinIO Bucket Structure**:
```
excalidraw/
  boards/
    {boardId}/
      files/
        {fileId}.{ext}     ← Original image
      thumbnails/
        {boardId}.png      ← Board thumbnail
        v{version}.png     ← Version thumbnail
```

**board_assets table** (enhanced):
```sql
CREATE TABLE board_assets (
    id          TEXT PRIMARY KEY,
    board_id    TEXT NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    file_id     TEXT NOT NULL,        -- Excalidraw's internal file ID
    mime_type   TEXT NOT NULL,
    size_bytes  BIGINT NOT NULL,
    storage_key TEXT NOT NULL,        -- MinIO object key
    sha256      TEXT NOT NULL,        -- Content hash for dedup
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(board_id, file_id)
);
```

### Migration Strategy for Existing Data

1. During the transition, the Go backend will accept both formats:
   - If `dataURL` starts with `data:` → extract, upload to MinIO, replace with reference
   - If `dataURL` starts with URL → already migrated, use as-is
2. Background migration job to process existing boards
3. Two-phase: write path updated first, then batch migration of historical data

### Benefits

| Metric | Before (Base64 in DB) | After (MinIO/S3) |
|---|---|---|
| 5MB image stored as | 6.7MB in JSONB | 5MB in S3 + 100B reference |
| 10 versions of same image | 67MB in DB | 5MB in S3 (shared) + 1KB refs |
| Database backup size | Includes all images | Metadata only |
| Image serving | Read from DB, decode, send | Presigned URL → direct S3 |
| CDN cacheable | No | Yes |
| Deduplication | Not possible | SHA256-based dedup |

---

## Security Hardening

This application will be exposed to the internet via Cloudflare Tunnel. All security measures follow OWASP Top 10 (2025) and current industry standards.

### Transport & Headers

| Control | Implementation | Standard |
|---|---|---|
| **HSTS** | `Strict-Transport-Security: max-age=31536000; includeSubDomains` | RFC 6797 |
| **CSP** | `script-src 'self'; connect-src 'self' {api_origin} wss://{ws_origin}; img-src 'self' {s3_origin} data:; style-src 'self' 'unsafe-inline'; frame-ancestors 'none'` | CSP Level 3 |
| **X-Content-Type-Options** | `nosniff` | Prevents MIME sniffing |
| **X-Frame-Options** | `DENY` | Clickjacking protection |
| **Referrer-Policy** | `strict-origin-when-cross-origin` | Limits referrer leakage |
| **Permissions-Policy** | `camera=(), microphone=(), geolocation=()` | Disable unused browser APIs |
| **X-XSS-Protection** | `0` (disabled, CSP is preferred) | Modern best practice |

> All headers applied via a single **reusable security middleware** component in `internal/middleware/security.go`.

### Authentication Security

| Control | Implementation |
|---|---|
| **Password storage** | bcrypt cost 12 (minimum, increase over time) |
| **JWT access tokens** | 15 min expiry, HMAC-SHA256, httpOnly cookie |
| **JWT refresh tokens** | 30 day expiry, rotated on use, stored in DB for revocation |
| **Cookie flags** | `httpOnly; Secure; SameSite=Lax; Path=/; __Host-` prefix when HTTPS |
| **CSRF defense** | SameSite cookie + Origin header validation on all mutations |
| **Brute-force** | Progressive delays: 1s after 3 failures, 5s after 5, lockout after 10 (15 min cooldown) |
| **User enumeration** | Same response timing and message for valid/invalid emails |
| **Token entropy** | Share tokens: `crypto/rand` 32 bytes → base64url (256-bit) |
| **OAuth state** | CSRF state parameter validated on OAuth callback |

### Input Validation

| Layer | Implementation |
|---|---|
| **Struct validation** | `go-playground/validator` on all request DTOs |
| **SQL injection** | pgx parameterized queries exclusively — never string concat |
| **File uploads** | Magic byte detection + MIME allowlist (png, jpeg, gif, webp, svg+xml) |
| **Request size** | Global: 1MB default. Per-route: 16MB (board save), 50MB (file upload) |
| **Path traversal** | Sanitize all file paths, validate board/org IDs as UUIDs |
| **HTML/XSS** | Sanitize user-provided text fields (title, description, labels) |
| **JSON depth** | Limit JSON nesting depth to prevent stack overflow |

### WebSocket Security

| Control | Implementation |
|---|---|
| **Origin validation** | Validate `Origin` header against `CORS_ALLOWED_ORIGINS` on WS upgrade |
| **Authentication** | JWT for owners, share token for viewers — validated before upgrade completes |
| **Message rate limiting** | Max 60 messages/sec per client, disconnect on violation |
| **Connection limits** | Max 50 concurrent connections per board |
| **Message size** | Max 64KB per WS message |
| **Heartbeat** | 30s ping interval, disconnect after 2 missed pongs |

### Operational Security

| Control | Implementation |
|---|---|
| **Dependency scanning** | `govulncheck` in CI pipeline |
| **Secret management** | Docker secrets support, never log sensitive values |
| **Error responses** | Never expose stack traces or internal details to clients |
| **Audit logging** | All auth events, permission changes, share link usage |
| **Container security** | Non-root user, read-only filesystem, minimal base image (Alpine) |
| **DB credentials** | Separate read/write users (future), connection via Unix socket or TLS |

---

## Performance Best Practices

### Database Performance

| Technique | Implementation |
|---|---|
| **Connection pooling** | pgxpool: min 2, max 25 connections, 30min max lifetime, 5min idle timeout |
| **Prepared statements** | pgx automatic statement cache (default 512 statements) |
| **Proper indexing** | GIN trigram on searchContent, composite indexes on frequently-joined columns, EXPLAIN ANALYZE audit on all queries |
| **Efficient queries** | SELECT only needed columns, avoid N+1 queries, use JOINs and CTEs |
| **Batch operations** | pgx.Batch for multiple related queries in one round-trip |
| **JSONB optimization** | Store file references (not data) in JSONB, index on JSONB keys where queried |

### HTTP Performance

| Technique | Implementation |
|---|---|
| **Response compression** | gzip (level 5) for JSON responses > 1KB, brotli where supported |
| **HTTP caching** | `ETag` on board reads, `Cache-Control: immutable` on S3 signed URLs, `no-cache` on mutable API responses |
| **Keep-Alive** | Go net/http default, `IdleTimeout: 120s` |
| **Request timeouts** | Per route group: reads 10s, writes 30s, uploads 120s, WebSocket no timeout |
| **Static assets** | `Cache-Control: public, max-age=31536000, immutable` for hashed frontend assets |
| **Presigned URLs** | S3 presigned URLs redirect clients directly to MinIO — backend never proxies image bytes |

### WebSocket Performance

| Technique | Implementation |
|---|---|
| **Cursor batching** | Accumulate cursor moves in 50ms window, send single batch message |
| **Compression** | permessage-deflate (RFC 7692) — coder/websocket built-in |
| **Write coalescing** | Buffer multiple messages, write in single syscall where possible |
| **Selective broadcast** | Only broadcast scene_update to viewers, not back to sender |
| **Idle cleanup** | Close connections with no activity for 5 minutes |

### Go-Specific Performance

| Technique | Implementation |
|---|---|
| **Zero-allocation logging** | zerolog — no heap allocations for log calls that don't match level |
| **sync.Pool** | Reuse byte buffers for JSON encoding/decoding hot paths |
| **Value types** | Use Go structs (value semantics) instead of pointer-heavy designs where appropriate |
| **Goroutine efficiency** | One goroutine per WS client (read pump + write pump), not per message |
| **Binary size** | CGO_ENABLED=0, strip debug info with `-ldflags="-s -w"` for smaller Docker images |

---

## Testing Strategy

All tests follow the **testing pyramid**: many unit tests, fewer integration tests, fewest E2E tests. Every feature must have tests before it's considered complete.

### Testing Pyramid

```
        ╱ ╲           E2E Tests (Playwright)
       ╱   ╲          Full browser, real backend + DB
      ╱─────╲         ~20 critical user flows
     ╱       ╲
    ╱ Integration╲    testcontainers-go
   ╱    Tests     ╲   Real PostgreSQL + MinIO in Docker
  ╱───────────────╲   ~100 tests across services
 ╱                 ╲
╱    Unit Tests     ╲  stdlib testing + testify
╱───────────────────╲  ~500+ tests, pure logic
```

### Backend Testing (Go)

| Layer | Tool | What It Tests | Pattern |
|---|---|---|---|
| **Unit** | `testing` + `testify` | Business logic, validation, token generation, JWT, pagination, error mapping | Table-driven tests, dependency injection via interfaces |
| **Repository** | `testcontainers-go` (PostgreSQL) | SQL queries, migrations, constraints, indexes | Real PG container, transaction rollback per test |
| **Service** | `testify/mock` + `testcontainers-go` | Service orchestration, access control, audit logging | Mock repos for unit, real repos for integration |
| **Handler** | `net/http/httptest` | Request parsing, response format, status codes, auth middleware | `httptest.NewRecorder()`, table-driven |
| **WebSocket** | `coder/websocket` + `httptest` | Hub/room lifecycle, message broadcast, cursor batching, presence | In-process WS server, multiple test clients |
| **Storage** | `testcontainers-go` (MinIO) | Upload, download, presigned URLs, dedup, cleanup | Real MinIO container |

#### Go Test Organization

```
backend/
├── internal/
│   ├── handler/
│   │   ├── board_handler.go
│   │   └── board_handler_test.go        # Unit: httptest
│   ├── service/
│   │   ├── board_service.go
│   │   ├── board_service_test.go         # Unit: mocked repos
│   │   └── board_service_integ_test.go   # Integration: real DB
│   ├── repository/
│   │   ├── board_repo.go
│   │   └── board_repo_test.go            # Integration: testcontainers PG
│   └── realtime/
│       ├── hub.go
│       └── hub_test.go                   # Unit: in-memory
├── tests/
│   └── integration/                      # Cross-cutting integration tests
│       ├── testutil/
│       │   ├── containers.go             # Reusable PG + MinIO container setup
│       │   └── fixtures.go               # Test data factories
│       ├── auth_flow_test.go
│       ├── board_lifecycle_test.go
│       └── share_flow_test.go
```

#### Go Test Best Practices (2026)

- **Table-driven tests** for all handler and validation logic
- **`t.Parallel()`** on all tests that don't share state
- **Build tags** to separate unit vs integration: `//go:build integration`
- **`testcontainers-go`** for real infrastructure — no SQLite fallback, no mocks for DB
- **Transaction rollback** pattern: wrap each test in a transaction, rollback after
- **Test fixtures via factories** — not SQL files, not ORM seeds
- **Golden files** for complex JSON responses (snapshot testing)
- **`t.Cleanup()`** for automatic resource cleanup
- **Race detector** enabled: `go test -race ./...`
- **Coverage target**: 80%+ on services layer, 90%+ on handlers

### Frontend Testing

| Layer | Tool | What It Tests | Pattern |
|---|---|---|---|
| **Unit** | Vitest | Utility functions, hooks, state logic, API client | Fast, in-process, no browser |
| **Component** | Vitest + React Testing Library | Individual component rendering, interaction, props | JSDOM environment, user-event |
| **Integration** | Vitest + MSW | Multi-component flows with mocked API | Mock Service Worker for API stubbing |
| **E2E** | Playwright | Full user flows (login → create board → save → share → view) | Real browser, real backend |

#### Frontend Test Organization

```
src/
├── components/
│   ├── ui/
│   │   ├── Button.tsx
│   │   └── Button.test.tsx              # Component: Vitest + RTL
│   ├── excalidraw/
│   │   ├── BoardEditor.tsx
│   │   └── BoardEditor.test.tsx
├── services/
│   ├── api.client.ts
│   └── api.client.test.ts               # Unit: Vitest + MSW
├── lib/
│   ├── utils.ts
│   └── utils.test.ts                     # Unit: pure Vitest
tests/
├── e2e/
│   ├── playwright.config.ts
│   ├── auth.spec.ts                      # E2E: login, register, OAuth
│   ├── board-crud.spec.ts                # E2E: create, edit, delete board
│   ├── board-sharing.spec.ts             # E2E: share link, view-only mode
│   ├── board-versioning.spec.ts          # E2E: save, history, restore
│   ├── org-management.spec.ts            # E2E: org CRUD, member mgmt
│   └── fixtures/
│       ├── auth.fixture.ts               # Reusable auth setup
│       └── board.fixture.ts              # Reusable board setup
├── mocks/
│   └── handlers.ts                       # MSW request handlers
```

### E2E Testing with Playwright

| Capability | Implementation |
|---|---|
| **Browser coverage** | Chromium + Firefox (WebKit optional) |
| **Parallel execution** | Worker-based sharding, isolated browser contexts |
| **Auth fixture** | Login once, reuse `storageState` across tests |
| **WebServer integration** | Playwright starts backend + frontend via `webServer` config |
| **Visual regression** | Optional: `toHaveScreenshot()` for critical pages |
| **Trace on failure** | `trace: 'retain-on-failure'` — full replay of failed tests |
| **Network interception** | Route stubbing for edge cases (rate limits, errors) |
| **CI/CD** | Run in Docker (headless), HTML report as artifact |

### Regression Testing Approach

| Trigger | Tests Run | Goal |
|---|---|---|
| Every commit | Unit + Component (Vitest + Go unit) | Fast feedback, <2 min |
| Every PR | Unit + Integration + E2E (Playwright) | Full confidence before merge |
| Nightly | Full suite + race detector + coverage report | Catch flaky tests, benchmark regressions |
| Before release | Full suite + visual regression + manual smoke | Final gate |

### CI Test Pipeline

```yaml
# Simplified — actual CI in .github/workflows/
jobs:
  backend-unit:
    runs-on: ubuntu-latest
    steps:
      - run: go test -race -count=1 ./internal/...

  backend-integration:
    runs-on: ubuntu-latest
    services: [postgres, minio]  # Or testcontainers
    steps:
      - run: go test -race -tags=integration ./...

  frontend-unit:
    runs-on: ubuntu-latest
    steps:
      - run: pnpm vitest run --coverage

  e2e:
    runs-on: ubuntu-latest
    needs: [backend-unit, frontend-unit]
    steps:
      - run: docker compose up -d
      - run: pnpm playwright test
      - uses: actions/upload-artifact@v4
        with:
          name: playwright-report
          path: tests/e2e/playwright-report/
```

---

## Database Backup & Restore

Since the application will be hosted on a Digital Ocean droplet (or similar) and data corruption is a real risk, automated backups are a first-class feature — not an afterthought.

### Architecture

```
┌──────────────┐     pg_dump      ┌──────────────┐     upload     ┌──────────────┐
│  PostgreSQL  │────────────────►│  Go Backend  │──────────────►│    MinIO      │
│              │                 │  (backup     │               │  (S3 bucket)  │
│              │◄────────────────│   service)   │◄──────────────│              │
│              │     psql restore │              │   download    │  backups/    │
└──────────────┘                 └──────┬───────┘               └──────────────┘
                                       │
                                       │ API
                                       ▼
                                ┌──────────────┐
                                │   Frontend   │
                                │  (settings   │
                                │   page)      │
                                └──────────────┘
```

### Backup Strategy

| Aspect | Implementation |
|---|---|
| **Tool** | `pg_dump` via `os/exec` in Go — industry standard, battle-tested |
| **Format** | Custom format (`-Fc`) for efficient compression + selective restore |
| **Storage** | MinIO bucket `backups/` — same S3-compatible storage we already run |
| **Schedule** | Configurable cron expression via API + frontend settings |
| **Rotation** | Keep N daily, N weekly, N monthly (configurable) |
| **Compression** | pg_dump custom format (built-in zlib compression) |
| **Encryption** | Optional: AES-256 encryption before upload to MinIO |
| **Naming** | `backups/{orgId}/excalidraw-{YYYYMMDD-HHmmss}.dump` |
| **Max size** | Track backup sizes, warn when approaching disk/bucket limits |

### Backup Service (Reusable Components)

```go
// internal/service/backup_service.go
type BackupService interface {
    CreateBackup(ctx context.Context) (*BackupMetadata, error)
    RestoreBackup(ctx context.Context, backupID string) error
    ListBackups(ctx context.Context, opts ListBackupsOpts) ([]BackupMetadata, error)
    GetBackup(ctx context.Context, backupID string) (*BackupMetadata, error)
    DeleteBackup(ctx context.Context, backupID string) error
    GetSchedule(ctx context.Context) (*BackupSchedule, error)
    UpdateSchedule(ctx context.Context, schedule BackupSchedule) error
    GetBackupDownloadURL(ctx context.Context, backupID string) (string, error)
}

type BackupMetadata struct {
    ID          string    `json:"id"`
    Filename    string    `json:"filename"`
    SizeBytes   int64     `json:"sizeBytes"`
    CreatedAt   time.Time `json:"createdAt"`
    Duration    string    `json:"duration"`
    Type        string    `json:"type"` // "scheduled" | "manual"
    Status      string    `json:"status"` // "completed" | "failed" | "in_progress"
    StorageKey  string    `json:"storageKey"`
}

type BackupSchedule struct {
    Enabled       bool   `json:"enabled"`
    CronExpr      string `json:"cronExpr"`      // e.g., "0 3 * * *" (daily 3AM)
    KeepDaily     int    `json:"keepDaily"`     // default: 7
    KeepWeekly    int    `json:"keepWeekly"`    // default: 4
    KeepMonthly   int    `json:"keepMonthly"`   // default: 6
}
```

### Frontend Settings UI

The settings page (`/settings`) will include a **Backup Management** section:

| Feature | Description |
|---|---|
| **Schedule toggle** | Enable/disable automated backups |
| **Cron picker** | User-friendly schedule selector (daily/weekly/custom cron) |
| **Retention config** | Configure how many daily/weekly/monthly backups to keep |
| **Backup list** | Table with: filename, size, date, type (manual/scheduled), status |
| **Manual backup** | "Backup Now" button for on-demand backups |
| **Download** | Download any backup file (presigned MinIO URL) |
| **Restore** | "Restore" button with confirmation dialog + warning |
| **Delete** | Delete individual old backups |
| **Storage usage** | Show total backup storage used vs available |

### API Endpoints

| Method | Endpoint | Description |
|---|---|---|
| GET | `/api/v1/backups` | List all backups (paginated) |
| POST | `/api/v1/backups` | Create manual backup |
| GET | `/api/v1/backups/{id}` | Get backup details |
| GET | `/api/v1/backups/{id}/download` | Get presigned download URL |
| POST | `/api/v1/backups/{id}/restore` | Restore from backup |
| DELETE | `/api/v1/backups/{id}` | Delete a backup |
| GET | `/api/v1/backups/schedule` | Get backup schedule config |
| PUT | `/api/v1/backups/schedule` | Update backup schedule config |

### Restore Process

```
1. User clicks "Restore" → confirmation dialog:
   "This will replace ALL current data with the backup from {date}.
    A pre-restore backup will be created automatically."

2. Backend:
   a. Create a pre-restore backup (safety net)
   b. Download the selected backup from MinIO
   c. Terminate all active DB connections
   d. pg_restore --clean --if-exists --no-owner
   e. Run any pending migrations (forward compatibility)
   f. Log audit event (backup.restore)
   g. Return success/failure
```

### Backup Audit Trail

All backup operations are logged as audit events:
- `backup.create` — manual or scheduled
- `backup.restore` — with backup ID and timestamp
- `backup.delete` — with backup ID
- `backup.schedule_update` — when schedule changes

---

## Documentation Standards

Documentation is a living artifact — updated with every feature, not after. Every PR that changes behavior must update relevant docs.

### README Structure

The project will maintain multiple README files:

| File | Audience | Content |
|---|---|---|
| `README.md` (root) | Everyone | Project overview, quick start, architecture diagram (Mermaid), feature list, screenshots |
| `backend/README.md` | Backend developers | Go setup, API docs, testing guide, env vars |
| `docs/API.md` | API consumers | Full API reference with request/response examples |
| `docs/DEPLOYMENT.md` | Operators | Docker setup, DigitalOcean deployment, Cloudflare Tunnel, backup config |
| `docs/ARCHITECTURE.md` | Contributors | System design, data flow diagrams, decision records |
| `CONTRIBUTING.md` (update) | Contributors | Updated dev setup, testing, PR process |

### Diagram Standards

All architecture and flow diagrams use **Mermaid** for version-controlled, reviewable diagrams:

```markdown
<!-- Example: rendered automatically on GitHub -->
```mermaid
flowchart LR
    Frontend -->|REST API| Backend
    Frontend -->|WebSocket| Backend
    Backend -->|pgx| PostgreSQL
    Backend -->|S3 API| MinIO
    Backend -->|pg_dump| Backups[MinIO backups/]
```
```

| Diagram Type | Where Used |
|---|---|
| **System architecture** | Root README.md, ARCHITECTURE.md |
| **Data flow** | ARCHITECTURE.md (board save, image upload, share flow) |
| **ER diagram** | ARCHITECTURE.md (database schema) |
| **Sequence diagrams** | API.md (auth flow, WebSocket handshake, backup restore) |
| **Deployment topology** | DEPLOYMENT.md (Docker services, networking) |

### Documentation Rules

1. **Every PR that changes API** → update `docs/API.md` with new/changed endpoints
2. **Every new feature** → update root README.md feature list
3. **Every env var added** → update `docs/DEPLOYMENT.md` and backend README
4. **Every migration** → update ER diagram in ARCHITECTURE.md
5. **Every Docker change** → update DEPLOYMENT.md and docker-compose comments
6. **Tables over prose** — use Markdown tables for configuration, env vars, API routes
7. **Mermaid over images** — version-controlled, diffable, no external tools needed
8. **Runnable examples** — docs include copy-paste-able commands

---

## Project Structure

```
backend/
├── cmd/
│   └── server/
│       └── main.go              # Entry point, wire up dependencies
├── internal/
│   ├── config/
│   │   └── config.go            # Environment configuration struct
│   ├── database/
│   │   ├── database.go          # pgxpool setup, health checks
│   │   └── migrations/          # SQL migration files (golang-migrate)
│   │       ├── 001_init.up.sql
│   │       ├── 001_init.down.sql
│   │       ├── 002_search_content.up.sql
│   │       └── 002_search_content.down.sql
│   ├── middleware/
│   │   ├── auth.go              # JWT validation, user context injection
│   │   ├── logging.go           # Request logging with zerolog
│   │   ├── ratelimit.go         # httprate wrappers
│   │   ├── recovery.go          # Panic recovery
│   │   └── security.go          # Security headers (CSP, HSTS, etc.)
│   ├── models/
│   │   ├── user.go
│   │   ├── organization.go
│   │   ├── membership.go
│   │   ├── board.go
│   │   ├── board_version.go
│   │   ├── board_permission.go
│   │   ├── board_asset.go
│   │   ├── audit_event.go
│   │   └── share_link.go
│   ├── repository/              # Data access layer (SQL queries)
│   │   ├── user_repo.go
│   │   ├── org_repo.go
│   │   ├── membership_repo.go
│   │   ├── board_repo.go
│   │   ├── board_version_repo.go
│   │   ├── board_permission_repo.go
│   │   ├── board_asset_repo.go
│   │   ├── share_link_repo.go
│   │   ├── audit_repo.go
│   │   └── storage_repo.go
│   ├── service/                 # Business logic layer
│   │   ├── auth_service.go
│   │   ├── access_service.go
│   │   ├── board_service.go
│   │   ├── org_service.go
│   │   ├── member_service.go
│   │   ├── version_service.go
│   │   ├── storage_service.go
│   │   ├── audit_service.go
│   │   ├── search_service.go
│   │   └── share_service.go     # Share link CRUD + validation
│   ├── handler/                 # HTTP handlers (thin, delegate to services)
│   │   ├── auth_handler.go
│   │   ├── board_handler.go
│   │   ├── org_handler.go
│   │   ├── member_handler.go
│   │   ├── version_handler.go
│   │   ├── storage_handler.go
│   │   ├── share_handler.go     # Share link + shared board view
│   │   ├── audit_handler.go
│   │   ├── health_handler.go
│   │   └── stats_handler.go
│   ├── realtime/                # WebSocket real-time system (reusable)
│   │   ├── hub.go               # Generic pub/sub room manager
│   │   ├── client.go            # WS client: read/write pumps, heartbeat
│   │   ├── message.go           # Typed message envelope (scene_update, cursor_move, presence)
│   │   └── handler.go           # WS upgrade handler, auth, room join/leave
│   ├── router/
│   │   └── router.go            # chi route definitions
│   ├── storage/                 # Object storage (MinIO/S3)
│   │   └── s3.go
│   ├── backup/                  # Database backup & restore
│   │   ├── backup_service.go    # Backup orchestration (pg_dump, upload, rotate)
│   │   ├── scheduler.go         # Cron scheduler for automated backups
│   │   └── handler.go           # Backup API handlers
│   └── pkg/                     # Shared utilities
│       ├── apierror/
│       │   └── errors.go        # Structured API errors
│       ├── pagination/
│       │   └── pagination.go    # Shared pagination helpers
│       ├── jwt/
│       │   └── jwt.go           # JWT creation/validation
│       ├── token/
│       │   └── token.go         # Crypto-secure token generation (share links)
│       └── response/
│           └── response.go      # Standard JSON response helpers
├── tests/
│   └── integration/             # Cross-cutting integration tests
│       ├── testutil/
│       │   ├── containers.go    # Reusable PG + MinIO testcontainers setup
│       │   └── fixtures.go      # Test data factories
│       ├── auth_flow_test.go
│       ├── board_lifecycle_test.go
│       └── share_flow_test.go
├── Dockerfile
├── go.mod
├── go.sum
└── README.md

tests/                           # E2E tests (project root)
├── e2e/
│   ├── playwright.config.ts
│   ├── auth.spec.ts
│   ├── board-crud.spec.ts
│   ├── board-sharing.spec.ts
│   ├── board-versioning.spec.ts
│   ├── org-management.spec.ts
│   ├── backup-restore.spec.ts
│   └── fixtures/
│       ├── auth.fixture.ts
│       └── board.fixture.ts
├── mocks/
│   └── handlers.ts              # MSW request handlers

docs/                            # Project documentation
├── API.md                       # Full API reference
├── ARCHITECTURE.md              # System design + diagrams
└── DEPLOYMENT.md                # Docker, DigitalOcean, tunnels, backups
```

---

## Migration Phases

### Phase 1: Foundation ✅
- [x] Initialize Go module and project structure
- [x] Set up configuration (env parsing)
- [x] Set up database connection (pgxpool)
- [x] Set up migrations (golang-migrate) — port existing Prisma schema
- [x] Set up structured logging (zerolog)
- [x] Set up chi router with base middleware
- [x] Health check endpoint
- [x] Docker setup for Go backend + MinIO

### Phase 2: Authentication ✅
- [x] User model and repository
- [x] Password hashing (bcrypt)
- [x] JWT creation and validation
- [x] Email/password login handler
- [x] Google OAuth2 flow
- [x] GitHub OAuth2 flow
- [x] Auth middleware (JWT validation)
- [x] Session cookie management

### Phase 3: Organizations & Members ✅
- [x] Organization CRUD
- [x] Membership management
- [x] Access control service (role hierarchy)
- [x] Organization API handlers
- [x] Member API handlers

### Phase 4: Boards & Versioning ✅
- [x] Board CRUD
- [x] Board version save (with search content extraction)
- [x] Optimistic concurrency control (etag)
- [x] Version history (list, get, restore)
- [x] Board search (ILIKE + GIN trigram)
- [x] Tag filtering

### Phase 5: Image Storage & Object Storage ✅
- [x] MinIO integration
- [x] Image upload/download handlers
- [x] Presigned URL generation
- [x] Image extraction from scene data
- [x] Thumbnail upload to MinIO
- [x] Orphaned file cleanup
- [x] Board asset tracking

### Phase 6: Audit & Storage ✅
- [x] Audit event logging service
- [x] Audit log query API
- [x] Audit stats API
- [x] Board storage calculation
- [x] Workspace storage calculation
- [x] Cleanup operations

### Phase 7: Sharing & Real-Time ✅
- [x] Share link CRUD (generate, revoke, list, validate expiry)
- [x] Shared board view endpoint (GET /api/v1/share/{token})
- [x] WebSocket hub/room infrastructure (reusable)
- [x] WebSocket upgrade handler with auth (JWT for owner, share token for viewers)
- [x] Real-time board state broadcast (scene_update on save)
- [x] Cursor position broadcasting (cursor_move messages)
- [x] Presence system (join/leave notifications, viewer list)
- [x] Heartbeat / ping-pong for stale connection cleanup
- [x] Multi-device sync (same user, multiple devices)
- [x] WS rate limiting (max connections per board, max message rate)
- [x] Frontend: share dialog, copy link, view-only mode
- [x] Frontend: live cursor rendering (colored cursors with names)
- [x] Frontend: presence indicator (who is viewing)

### Phase 8: Security & Performance ✅
- [x] Security headers middleware (CSP, HSTS, X-Frame-Options, etc.)
- [x] Cookie hardening (httpOnly, Secure, SameSite, __Host- prefix)
- [x] CSRF protection (SameSite + Origin validation)
- [x] Request size limits (global + per-route)
- [x] File upload validation (magic bytes + MIME allowlist)
- [x] Brute-force protection (progressive delays)
- [x] WebSocket origin validation
- [x] HTTP caching headers (ETag, Cache-Control)
- [x] Response compression (gzip/brotli)
- [x] Request timeout budgets per route group
- [x] Database query optimization (EXPLAIN ANALYZE audit)
- [x] Error information leakage audit
- [x] govulncheck integration

### Phase 9: Database Backup & Restore ✅
- [x] Backup service implementation (pg_dump wrapper, MinIO upload)
- [x] Backup scheduler (cron-based, configurable interval)
- [x] Backup rotation (daily/weekly/monthly retention)
- [x] Backup listing + metadata API endpoints
- [x] Manual backup trigger endpoint
- [x] Restore endpoint (pg_restore with pre-restore safety backup)
- [x] Backup download endpoint (presigned MinIO URL)
- [x] Backup schedule CRUD endpoints
- [x] Frontend: backup settings UI (schedule, retention config)
- [x] Frontend: backup list table (name, size, date, status)
- [x] Frontend: manual backup + restore + download buttons
- [x] Backup audit events (create, restore, delete, schedule_update)
- [x] Docker: ensure pg_dump/pg_restore available in Go container

### Phase 10: Testing (Backend) ✅
- [x] Set up Go test infrastructure (testcontainers-go for PG + MinIO)
- [x] Reusable test utilities (containers.go, fixtures.go, helpers.go)
- [x] Interface refactoring (12 repo + 1 storage interface, all 10 services)
- [x] Mock generation (mockgen → testutil/mocks/)
- [x] Unit tests for services (45 mock-based: auth 12, share 10, access 10, backup 13)
- [x] Unit tests for handlers (9 httptest: health 1, backup 8)
- [x] Integration tests for repositories (24 testcontainers: user 11, org 13)
- [x] Integration tests for storage (8 testcontainers: S3 upload/download/delete/presign/list)
- [x] Total: 222 tests passing, 0 failures (86 new this phase)
- [ ] WebSocket hub/client tests
- [ ] Set up Vitest for frontend (config, JSDOM environment)
- [ ] Frontend unit tests (utils, hooks, API client with MSW)
- [ ] Frontend component tests (React Testing Library)
- [ ] Set up Playwright for E2E (config, fixtures, webServer)
- [ ] E2E: auth flows (login, register, OAuth)
- [ ] E2E: board CRUD + versioning
- [ ] E2E: board sharing + view-only mode
- [ ] E2E: backup & restore flow
- [ ] E2E: organization management
- [ ] CI pipeline: unit → integration → E2E
- [ ] Coverage reporting (80%+ services, 90%+ handlers)

### Phase 11: Rate Limiting & Polish ✅
- [x] httprate integration per route group (4 tiers: general, auth, upload 20/min, WS 30/min)
- [x] System stats endpoint (health with version/build info, ready with DB+S3 check, /version endpoint)
- [x] Error response standardization (12 new domain-specific error codes)
- [x] Request validation (custom slug/xid tags, 20+ format messages, ValidateVar helper)
- [x] CORS strict configuration (already existed)
- [x] Graceful shutdown (configurable timeout via ShutdownTimeout, logs elapsed duration)

### Phase 12: Frontend Adaptation ✅
- [x] Update api.client.ts to point to Go backend
- [x] Update API URL paths (/api → /api/v1)
- [x] Update auth flow (cookie-based JWT)
- [x] Handle new image URL format (presigned URLs)
- [x] WebSocket client integration (board viewing, cursors)
- [x] Share UI components (generate link, copy, revoke)
- [x] View-only board mode (disable editing tools)
- [x] Live cursor overlay component
- [x] Presence sidebar/indicator component
- [x] Test full integration

### Phase 13: Cleanup ✅
- [x] Remove Prisma and related files
- [x] Remove Next.js API routes (keep pages/components)
- [x] Remove NextAuth
- [x] Remove server/ directory
- [x] Update docker-compose.yml
- [x] Update Dockerfile (separate frontend/backend)
- [x] Update README.md
- [x] Update CONTRIBUTING.md

### Phase 14: Documentation & Frontend Package Upgrades ✅
- [x] Write root README.md (overview, quick start, Mermaid architecture diagram)
- [x] Write backend/README.md (Go setup, API overview, testing guide)
- [x] Write docs/API.md (full API reference with examples)
- [x] Write docs/DEPLOYMENT.md (Docker, DigitalOcean, Cloudflare Tunnel, backups)
- [x] Write docs/ARCHITECTURE.md (system design, ER diagram, data flows)
- [x] Update CONTRIBUTING.md (new dev setup, Go + frontend testing)
- [x] Mermaid diagrams for all architecture views
- [x] Phase 14b: Frontend Package Upgrades
- [x] Audit all frontend packages for latest versions
- [x] Upgrade Next.js to latest
- [x] Upgrade React/React-DOM to latest
- [x] Upgrade Excalidraw to latest
- [x] Upgrade TipTap packages to latest
- [x] Upgrade Tailwind CSS to latest
- [x] Upgrade TypeScript to latest
- [x] Upgrade ESLint + config to latest
- [x] Upgrade remaining packages (mermaid, zod, sonner, etc.)
- [x] Run full test/build verification after upgrades

---

## Frontend Package Upgrades

All retained frontend packages should be upgraded to their latest versions as part of the migration.

### Dependencies

| Package | Current Version | Latest | Status | Notes |
|---|---|---|---|---|
| `next` | 16.0.5 | 16.2.1 | ✅ | Upgraded |
| `react` | ^19.2.0 | ^19.2.4 | ✅ | |
| `react-dom` | ^19.2.0 | ^19.2.4 | ✅ | |
| `@excalidraw/excalidraw` | ^0.18.0 | ^0.18.0 | ✅ | Latest stable |
| `@tiptap/starter-kit` | ^3.12.0 | ^3.21.0 | ✅ | All @tiptap/* upgraded together |
| `@tiptap/react` | ^3.12.0 | ^3.21.0 | ✅ | |
| `@tiptap/pm` | ^3.12.0 | ^3.21.0 | ✅ | |
| `@tiptap/extension-color` | ^3.12.0 | ^3.21.0 | ✅ | |
| `@tiptap/extension-highlight` | ^3.12.0 | ^3.21.0 | ✅ | |
| `@tiptap/extension-image` | ^3.12.0 | ^3.21.0 | ✅ | |
| `@tiptap/extension-link` | ^3.12.0 | ^3.21.0 | ✅ | |
| `@tiptap/extension-placeholder` | ^3.12.0 | ^3.21.0 | ✅ | |
| `@tiptap/extension-table` | ^3.12.0 | ^3.21.0 | ✅ | |
| `@tiptap/extension-table-cell` | ^3.12.0 | ^3.21.0 | ✅ | |
| `@tiptap/extension-table-header` | ^3.12.0 | ^3.21.0 | ✅ | |
| `@tiptap/extension-table-row` | ^3.12.0 | ^3.21.0 | ✅ | |
| `@tiptap/extension-task-item` | ^3.12.0 | ^3.21.0 | ✅ | |
| `@tiptap/extension-task-list` | ^3.12.0 | ^3.21.0 | ✅ | |
| `@tiptap/extension-text-align` | ^3.12.0 | ^3.21.0 | ✅ | |
| `@tiptap/extension-text-style` | ^3.12.0 | ^3.21.0 | ✅ | |
| `@tiptap/extension-typography` | ^3.12.0 | ^3.21.0 | ✅ | |
| `@tiptap/extension-underline` | ^3.12.0 | ^3.21.0 | ✅ | |
| `html2canvas` | ^1.4.1 | ^1.4.1 | ✅ | Latest available |
| `mermaid` | ^11.12.1 | ^11.13.0 | ✅ | |
| `react-markdown` | ^10.1.0 | ^10.1.0 | ✅ | Latest |
| `remark-gfm` | ^4.0.1 | ^4.0.1 | ✅ | Latest |
| `sonner` | ^2.0.7 | ^2.0.7 | ✅ | Latest |
| `uuid` | ^13.0.0 | ^13.0.0 | ✅ | Latest |
| `zod` | ^3.25.51 | ^4.3.6 | ✅ | Major version upgrade |

### Dev Dependencies

| Package | Current Version | Latest | Status | Notes |
|---|---|---|---|---|
| `typescript` | ^5 | ^6.0.2 | ✅ | TS 6.x upgrade |
| `tailwindcss` | ^4.1.17 | ^4.2.2 | ✅ | |
| `@tailwindcss/postcss` | ^4.1.17 | ^4.2.2 | ✅ | |
| `postcss` | ^8.4.49 | ^8.5.8 | ✅ | |
| `eslint` | ^9.39.1 | ^9.27.0 | ✅ | |
| `eslint-config-next` | 16.0.5 | 16.2.1 | ✅ | Matches Next.js version |
| `@types/node` | ^20 | ^25.5.0 | ✅ | |
| `@types/react` | ^19.2.7 | ^19.2.14 | ✅ | |
| `@types/react-dom` | ^19.2.3 | ^19.2.14 | ✅ | |
| `tsx` | ^4.20.6 | 🗑️ Removed | ✅ | Removed — seed.ts ported to Go |

### Packages to Remove (Backend-Only)

| Package | Current Version | Reason |
|---|---|---|
| `@auth/prisma-adapter` | ^1.6.0 | NextAuth Prisma adapter — both removed |
| `@prisma/client` | ^6.19.0 | Prisma ORM — replaced by pgx in Go |
| `bcryptjs` | ^3.0.2 | Password hashing — now in Go |
| `next-auth` | ^4.24.13 | Auth — replaced by Go JWT auth |
| `prisma` (dev) | ^6.19.0 | Prisma CLI — no longer needed |

---

## Cleanup Tracker

### Files/Directories to Remove After Migration

| Path | Type | Reason | Status |
|---|---|---|---|
| `prisma/` | Directory | Replaced by golang-migrate SQL files | ✅ |
| `prisma/schema.prisma` | File | Go models + SQL migrations | ✅ |
| `prisma/seed.ts` | File | Go seed command or SQL script | ✅ |
| `prisma/migrations/` | Directory | New migrations in backend/internal/database/migrations/ | ✅ |
| `src/pages/api/` | Directory | All API routes replaced by Go handlers | ✅ |
| `src/pages/api/auth/[...nextauth].ts` | File | Replaced by Go auth handlers | ✅ |
| `src/pages/api/boards/` | Directory | Replaced by Go board handlers | ✅ |
| `src/pages/api/orgs/` | Directory | Replaced by Go org handlers | ✅ |
| `src/pages/api/audit/` | Directory | Replaced by Go audit handlers | ✅ |
| `src/pages/api/health.ts` | File | Replaced by Go health handler | ✅ |
| `src/pages/api/stats.ts` | File | Replaced by Go stats handler | ✅ |
| `src/server/` | Directory | Entire server layer replaced by Go | ✅ |
| `src/server/db/prisma.ts` | File | Replaced by pgxpool | ✅ |
| `src/server/middleware/ratelimit.ts` | File | Replaced by httprate | ✅ |
| `src/server/middleware/withAuth.ts` | File | Replaced by Go auth middleware | ✅ |
| `src/server/services/access.service.ts` | File | Replaced by Go access service | ✅ |
| `src/server/services/audit.service.ts` | File | Replaced by Go audit service | ✅ |
| `src/server/services/boards.service.ts` | File | Replaced by Go board service | ✅ |
| `next-auth` dependency | Package | Completely replaced | ✅ |
| `@prisma/client` dependency | Package | Completely replaced | ✅ |
| `prisma` dependency | Package | Completely replaced | ✅ |
| `bcryptjs` dependency | Package | Go uses golang.org/x/crypto/bcrypt | ✅ |

### Package.json Dependencies to Remove

```
@auth/prisma-adapter
@prisma/client
bcryptjs
next-auth
prisma (devDep)
@types/bcryptjs (devDep)
```

### Configuration Files to Update

| File | Change | Status |
|---|---|---|
| `docker-compose.yml` | Add Go backend service, MinIO service, update network | ✅ |
| `Dockerfile` | Split into frontend-only Dockerfile | ✅ |
| `next.config.ts` | Remove API rewrites/config (no more API routes) | ✅ |
| `package.json` | Remove backend-only deps, remove db:* scripts | ✅ |

---

## Environment Variables

### Go Backend

```bash
# Server
PORT=8080
ENV=production  # development | production

# Database
DATABASE_URL=postgresql://user:pass@postgres:5432/excalidraw?sslmode=disable

# JWT
JWT_SECRET=<random_64_char_string>
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=720h  # 30 days

# OAuth2 (optional)
GOOGLE_CLIENT_ID=
GOOGLE_CLIENT_SECRET=
GOOGLE_REDIRECT_URL=http://localhost:3000/auth/callback/google
GITHUB_CLIENT_ID=
GITHUB_CLIENT_SECRET=
GITHUB_REDIRECT_URL=http://localhost:3000/auth/callback/github

# MinIO / S3
S3_ENDPOINT=minio:9000
S3_ACCESS_KEY=minioadmin
S3_SECRET_KEY=minioadmin
S3_BUCKET=excalidraw
S3_USE_SSL=false
S3_REGION=us-east-1
S3_PRESIGN_EXPIRY=1h

# Rate Limiting
RATE_LIMIT_REQUESTS_PER_MINUTE=60
RATE_LIMIT_AUTH_PER_MINUTE=10

# Logging
LOG_LEVEL=info  # debug | info | warn | error
LOG_FORMAT=json  # json | console (pretty for dev)

# CORS
CORS_ALLOWED_ORIGINS=http://localhost:3000

# WebSocket
WS_MAX_CONNECTIONS_PER_BOARD=50
WS_MAX_MESSAGE_SIZE=65536        # 64KB
WS_HEARTBEAT_INTERVAL=30s
WS_WRITE_TIMEOUT=10s
WS_CURSOR_BATCH_INTERVAL=50ms    # Batch cursor updates

# Backup
BACKUP_ENABLED=true
BACKUP_CRON=0 3 * * *            # Daily at 3 AM
BACKUP_KEEP_DAILY=7
BACKUP_KEEP_WEEKLY=4
BACKUP_KEEP_MONTHLY=6
BACKUP_S3_BUCKET=backups         # Separate MinIO bucket for backups
BACKUP_ENCRYPTION_KEY=            # Optional: AES-256 key for backup encryption
```

### Frontend (Updated)

```bash
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1
NEXT_PUBLIC_APP_URL=http://localhost:3000
NEXT_PUBLIC_AUTOSAVE_INTERVAL_MS=10000
```

---

## Docker Architecture

### docker-compose.yml (New)

```yaml
services:
  postgres:
    image: postgres:16-alpine
    ports:
      - "127.0.0.1:5432:5432"
    environment:
      POSTGRES_USER: ${POSTGRES_USER:-excalidraw}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD:-excalidraw_secret}
      POSTGRES_DB: ${POSTGRES_DB:-excalidraw}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER:-excalidraw}"]
      interval: 10s
      timeout: 5s
      retries: 5

  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    ports:
      - "127.0.0.1:9000:9000"
      - "127.0.0.1:9001:9001"  # MinIO Console
    environment:
      MINIO_ROOT_USER: ${S3_ACCESS_KEY:-minioadmin}
      MINIO_ROOT_PASSWORD: ${S3_SECRET_KEY:-minioadmin}
    volumes:
      - minio_data:/data
    healthcheck:
      test: ["CMD", "mc", "ready", "local"]
      interval: 10s
      timeout: 5s
      retries: 5

  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile
    ports:
      - "127.0.0.1:8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
      minio:
        condition: service_healthy
    env_file:
      - .env
    environment:
      DATABASE_URL: postgresql://${POSTGRES_USER:-excalidraw}:${POSTGRES_PASSWORD:-excalidraw_secret}@postgres:5432/${POSTGRES_DB:-excalidraw}?sslmode=disable
      S3_ENDPOINT: minio:9000
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/api/v1/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s

  frontend:
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "${APP_PORT:-3000}:3000"
    depends_on:
      backend:
        condition: service_healthy
    environment:
      NEXT_PUBLIC_API_URL: http://backend:8080/api/v1

  pgbackup:
    build:
      context: ./backend
      dockerfile: Dockerfile
    command: ["./server", "--backup-scheduler"]
    depends_on:
      postgres:
        condition: service_healthy
      minio:
        condition: service_healthy
    env_file:
      - .env
    environment:
      DATABASE_URL: postgresql://${POSTGRES_USER:-excalidraw}:${POSTGRES_PASSWORD:-excalidraw_secret}@postgres:5432/${POSTGRES_DB:-excalidraw}?sslmode=disable
      S3_ENDPOINT: minio:9000
      BACKUP_ENABLED: "true"
      BACKUP_CRON: "${BACKUP_CRON:-0 3 * * *}"
    restart: unless-stopped

  tunnel:
    image: cloudflare/cloudflared:latest
    command: tunnel run --token ${CLOUDFLARE_TUNNEL_TOKEN}
    depends_on:
      - frontend
      - backend
    profiles:
      - tunnel

volumes:
  postgres_data:
  minio_data:
```

### Go Backend Dockerfile

```dockerfile
# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/server

# Run stage
FROM alpine:3.21

RUN apk --no-cache add ca-certificates wget postgresql16-client

WORKDIR /app

COPY --from=builder /app/server .
COPY --from=builder /app/internal/database/migrations ./migrations

EXPOSE 8080

CMD ["./server"]
```

---

## API Route Mapping

| Current (Next.js) | New (Go) | Method | Notes |
|---|---|---|---|
| `/api/auth/[...nextauth]` | `/api/v1/auth/login` | POST | Email/password |
| | `/api/v1/auth/register` | POST | New user registration |
| | `/api/v1/auth/refresh` | POST | Refresh JWT token |
| | `/api/v1/auth/logout` | POST | Clear cookie |
| | `/api/v1/auth/oauth/{provider}` | GET | Start OAuth flow |
| | `/api/v1/auth/oauth/{provider}/callback` | GET | OAuth callback |
| `/api/boards` (GET) | `/api/v1/boards` | GET | List boards |
| `/api/boards` (POST) | `/api/v1/boards` | POST | Create board |
| `/api/boards/[id]` (GET) | `/api/v1/boards/{id}` | GET | Get board |
| `/api/boards/[id]` (PATCH) | `/api/v1/boards/{id}` | PATCH | Update board |
| `/api/boards/[id]` (DELETE) | `/api/v1/boards/{id}` | DELETE | Delete board |
| `/api/boards/[id]/save` | `/api/v1/boards/{id}/versions` | POST | Save new version |
| `/api/boards/[id]/versions` | `/api/v1/boards/{id}/versions` | GET | List versions |
| `/api/boards/[id]/versions/[v]` (GET) | `/api/v1/boards/{id}/versions/{v}` | GET | Get version |
| `/api/boards/[id]/versions/[v]` (POST) | `/api/v1/boards/{id}/versions/{v}/restore` | POST | Restore version |
| `/api/boards/[id]/storage` | `/api/v1/boards/{id}/storage` | GET | Storage info |
| `/api/boards/[id]/cleanup` | `/api/v1/boards/{id}/cleanup` | POST | Cleanup orphans |
| `/api/orgs` (GET) | `/api/v1/orgs` | GET | List orgs |
| `/api/orgs` (POST) | `/api/v1/orgs` | POST | Create org |
| `/api/orgs/[id]` (PATCH) | `/api/v1/orgs/{id}` | PATCH | Update org |
| `/api/orgs/[id]` (DELETE) | `/api/v1/orgs/{id}` | DELETE | Delete org |
| `/api/orgs/[id]/members` (GET) | `/api/v1/orgs/{id}/members` | GET | List members |
| `/api/orgs/[id]/members` (POST) | `/api/v1/orgs/{id}/members` | POST | Invite member |
| `/api/orgs/[id]/members` (PATCH) | `/api/v1/orgs/{id}/members/{mid}` | PATCH | Update role |
| `/api/orgs/[id]/members` (DELETE) | `/api/v1/orgs/{id}/members/{mid}` | DELETE | Remove member |
| `/api/orgs/[id]/storage` | `/api/v1/orgs/{id}/storage` | GET | Org storage |
| `/api/audit` | `/api/v1/audit` | GET | Audit logs |
| `/api/audit?stats=true` | `/api/v1/audit/stats` | GET | Audit stats |
| `/api/health` | `/api/v1/health` | GET | Health check |
| `/api/stats` | `/api/v1/stats` | GET | System stats |
| — | `/api/v1/ready` | GET | **NEW**: Readiness probe |
| — | `/api/v1/boards/{id}/files/{fileId}` | GET | **NEW**: Serve file (presigned redirect) |
| — | `/api/v1/boards/{id}/files` | POST | **NEW**: Upload file |
| — | `/api/v1/boards/{id}/share` | POST | **NEW**: Generate share link |
| — | `/api/v1/boards/{id}/share` | GET | **NEW**: List share links for board |
| — | `/api/v1/boards/{id}/share/{token}` | DELETE | **NEW**: Revoke share link |
| — | `/api/v1/share/{token}` | GET | **NEW**: Access shared board (read-only, no auth required) |
| — | `/api/v1/ws/boards/{id}` | WS | **NEW**: WebSocket — real-time board sync + cursors |
| — | `/api/v1/backups` | GET | **NEW**: List backups (paginated) |
| — | `/api/v1/backups` | POST | **NEW**: Create manual backup |
| — | `/api/v1/backups/{id}` | GET | **NEW**: Get backup details |
| — | `/api/v1/backups/{id}/download` | GET | **NEW**: Presigned download URL |
| — | `/api/v1/backups/{id}/restore` | POST | **NEW**: Restore from backup |
| — | `/api/v1/backups/{id}` | DELETE | **NEW**: Delete a backup |
| — | `/api/v1/backups/schedule` | GET | **NEW**: Get backup schedule |
| — | `/api/v1/backups/schedule` | PUT | **NEW**: Update backup schedule |

---

## Summary Counts

| Category | Total Features | Migrated | In Progress | New/Improved |
|---|---|---|---|---|
| Authentication | 10 | 0 | 0 | 2 new |
| Authorization | 14 | 0 | 0 | 1 improved |
| Boards | 10 | 0 | 0 | 2 improved |
| Versioning | 10 | 0 | 0 | 2 improved |
| Organizations | 6 | 0 | 0 | 0 |
| Members | 4 | 0 | 0 | 2 improved |
| Audit | 8 | 0 | 0 | 1 new |
| Storage | 5 | 0 | 0 | 0 |
| Rate Limiting | 6 | 0 | 0 | 1 improved |
| Health/Stats | 4 | 0 | 0 | 2 new |
| Database | 15 | 0 | 0 | 2 improved, 2 removed |
| **Real-Time & Sharing** | **13** | **0** | **0** | **11 new, 1 improved** |
| **Security Hardening** | **18** | **0** | **0** | **6 new, 10 improved** |
| **Performance** | **13** | **0** | **0** | **6 new, 5 improved** |
| Middleware | 11 | 0 | 0 | 5 new, 3 improved |
| **Testing** | **19** | **0** | **0** | **19 new** |
| **Backup & Restore** | **13** | **0** | **0** | **13 new** |
| **Documentation** | **7** | **0** | **0** | **5 new, 2 improved** |
| Pkg Upgrades (deps) | 27 | 0 | 0 | — |
| Pkg Upgrades (devDeps) | 10 | 0 | 0 | — |
| Pkg Removals | 5 | 0 | 0 | — |
| **TOTAL** | **228** | **0** | **0** | **101 new/improved** |
