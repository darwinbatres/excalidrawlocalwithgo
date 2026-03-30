-- =============================================================================
-- 001_init.up.sql — Full schema for Excalidraw Go
-- Ported from Prisma schema with enhancements:
--   - TIMESTAMPTZ instead of TIMESTAMP(3) for timezone safety
--   - BIGINT for sizeBytes to support large files
--   - Refresh token tracking table (JWT rotation)
--   - pg_trgm extension + GIN indexes for full-text search
--   - Improved index strategy
-- =============================================================================
-- Extensions
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Enums
CREATE TYPE org_role AS ENUM ('OWNER', 'ADMIN', 'MEMBER', 'VIEWER');

CREATE TYPE board_role AS ENUM ('OWNER', 'EDITOR', 'VIEWER');

-- =============================================================================
-- Users
-- =============================================================================
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL,
    email_verified TIMESTAMPTZ,
    name TEXT,
    image TEXT,
    password_hash TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT users_email_unique UNIQUE (email)
);

CREATE INDEX idx_users_email ON users (email);

-- =============================================================================
-- OAuth Accounts (replaces NextAuth Account)
-- =============================================================================
CREATE TABLE accounts (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type TEXT NOT NULL,
    provider TEXT NOT NULL,
    provider_account_id TEXT NOT NULL,
    refresh_token TEXT,
    access_token TEXT,
    expires_at INTEGER,
    token_type TEXT,
    scope TEXT,
    id_token TEXT,
    session_state TEXT,
    CONSTRAINT accounts_provider_unique UNIQUE (provider, provider_account_id)
);

CREATE INDEX idx_accounts_user_id ON accounts (user_id);

-- =============================================================================
-- Refresh Tokens (NEW — for JWT rotation)
-- =============================================================================
CREATE TABLE refresh_tokens (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    replaced_by TEXT,
    user_agent TEXT,
    ip TEXT
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens (user_id);

CREATE INDEX idx_refresh_tokens_token_hash ON refresh_tokens (token_hash);

CREATE INDEX idx_refresh_tokens_expires_at ON refresh_tokens (expires_at);

-- =============================================================================
-- Organizations
-- =============================================================================
CREATE TABLE organizations (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT organizations_slug_unique UNIQUE (slug)
);

CREATE INDEX idx_organizations_slug ON organizations (slug);

-- =============================================================================
-- Memberships
-- =============================================================================
CREATE TABLE memberships (
    id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role org_role NOT NULL DEFAULT 'MEMBER',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT memberships_org_user_unique UNIQUE (org_id, user_id)
);

CREATE INDEX idx_memberships_user_id ON memberships (user_id);

CREATE INDEX idx_memberships_org_id ON memberships (org_id);

-- =============================================================================
-- Boards
-- =============================================================================
CREATE TABLE boards (
    id TEXT PRIMARY KEY,
    org_id TEXT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    owner_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    title TEXT NOT NULL,
    description TEXT,
    tags TEXT [] NOT NULL DEFAULT '{}',
    is_archived BOOLEAN NOT NULL DEFAULT FALSE,
    thumbnail TEXT,
    search_content TEXT,
    current_version_id TEXT,
    version_number INTEGER NOT NULL DEFAULT 0,
    etag TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_boards_org_updated ON boards (org_id, updated_at DESC);

CREATE INDEX idx_boards_org_archived ON boards (org_id, is_archived);

CREATE INDEX idx_boards_owner ON boards (owner_id);

CREATE INDEX idx_boards_title ON boards (title);

-- GIN trigram indexes for fast ILIKE search
CREATE INDEX idx_boards_search_trgm ON boards USING GIN (search_content gin_trgm_ops);

CREATE INDEX idx_boards_title_trgm ON boards USING GIN (title gin_trgm_ops);

CREATE INDEX idx_boards_desc_trgm ON boards USING GIN (description gin_trgm_ops);

-- =============================================================================
-- Board Versions
-- =============================================================================
CREATE TABLE board_versions (
    id TEXT PRIMARY KEY,
    board_id TEXT NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    created_by_id TEXT NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    label TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    scene_json JSONB NOT NULL,
    app_state_json JSONB,
    thumbnail_url TEXT,
    CONSTRAINT board_versions_board_version_unique UNIQUE (board_id, version)
);

CREATE INDEX idx_board_versions_board_created ON board_versions (board_id, created_at DESC);

CREATE INDEX idx_board_versions_board_version ON board_versions (board_id, version);

-- =============================================================================
-- Board Permissions
-- =============================================================================
CREATE TABLE board_permissions (
    id TEXT PRIMARY KEY,
    board_id TEXT NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    membership_id TEXT NOT NULL REFERENCES memberships(id) ON DELETE CASCADE,
    role board_role NOT NULL DEFAULT 'VIEWER',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT board_permissions_board_member_unique UNIQUE (board_id, membership_id)
);

CREATE INDEX idx_board_permissions_membership ON board_permissions (membership_id);

-- =============================================================================
-- Board Assets (S3 storage metadata)
-- =============================================================================
CREATE TABLE board_assets (
    id TEXT PRIMARY KEY,
    board_id TEXT NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
    file_id TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    storage_key TEXT NOT NULL,
    sha256 TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT board_assets_board_file_unique UNIQUE (board_id, file_id)
);

CREATE INDEX idx_board_assets_board ON board_assets (board_id);

CREATE INDEX idx_board_assets_sha256 ON board_assets (sha256);

-- =============================================================================
-- Audit Events (append-only)
-- =============================================================================
CREATE TABLE audit_events (
    id TEXT PRIMARY KEY,
    org_id TEXT REFERENCES organizations(id) ON DELETE CASCADE,
    actor_id TEXT,
    action TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL,
    ip TEXT,
    user_agent TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_events_org_created ON audit_events (org_id, created_at DESC);

CREATE INDEX idx_audit_events_actor_created ON audit_events (actor_id, created_at DESC);

CREATE INDEX idx_audit_events_target ON audit_events (target_type, target_id);

CREATE INDEX idx_audit_events_action ON audit_events (action);

-- =============================================================================
-- Share Links
-- =============================================================================
CREATE TABLE share_links (
    id TEXT PRIMARY KEY,
    board_id TEXT NOT NULL,
    token TEXT NOT NULL,
    role board_role NOT NULL DEFAULT 'VIEWER',
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by TEXT NOT NULL,
    CONSTRAINT share_links_token_unique UNIQUE (token)
);

CREATE INDEX idx_share_links_token ON share_links (token);

CREATE INDEX idx_share_links_board ON share_links (board_id);

-- =============================================================================
-- Backup Metadata (NEW)
-- =============================================================================
CREATE TABLE backup_metadata (
    id TEXT PRIMARY KEY,
    filename TEXT NOT NULL,
    size_bytes BIGINT NOT NULL DEFAULT 0,
    storage_key TEXT NOT NULL,
    type TEXT NOT NULL DEFAULT 'manual',
    status TEXT NOT NULL DEFAULT 'in_progress',
    duration_ms INTEGER,
    error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

CREATE INDEX idx_backup_metadata_created ON backup_metadata (created_at DESC);

CREATE INDEX idx_backup_metadata_status ON backup_metadata (status);

-- =============================================================================
-- Backup Schedule Config (NEW — singleton row)
-- =============================================================================
CREATE TABLE backup_schedule (
    id TEXT PRIMARY KEY DEFAULT 'default',
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    cron_expr TEXT NOT NULL DEFAULT '0 3 * * *',
    keep_daily INTEGER NOT NULL DEFAULT 7,
    keep_weekly INTEGER NOT NULL DEFAULT 4,
    keep_monthly INTEGER NOT NULL DEFAULT 6,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Insert default schedule
INSERT INTO
    backup_schedule (id)
VALUES
    ('default');