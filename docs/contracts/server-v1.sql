-- First server database schema; apply only to an empty DB with user_version=0.
-- Operational PRAGMAs are also configured and read back on every connection.
PRAGMA foreign_keys = ON;
PRAGMA journal_mode = DELETE;
PRAGMA synchronous = EXTRA;
PRAGMA busy_timeout = 5000;

BEGIN IMMEDIATE;
CREATE TABLE worlds (
    world_id TEXT NOT NULL PRIMARY KEY,
    store_revision INTEGER NOT NULL CHECK (store_revision >= 0),
    save_json TEXT NOT NULL,
    host_json TEXT NOT NULL,
    updated_at_utc TEXT NOT NULL
) STRICT;

-- Actor is server-derived: "company:<id>" or "system". It is never a caller-
-- selected namespace. Keep receipts for the lifetime of an active world.
CREATE TABLE operations (
    world_id TEXT NOT NULL REFERENCES worlds(world_id),
    actor TEXT NOT NULL,
    operation_key TEXT NOT NULL,
    request_sha256 TEXT NOT NULL CHECK (length(request_sha256) = 64),
    request_json TEXT NOT NULL,
    response_json TEXT NOT NULL,
    committed_store_revision INTEGER NOT NULL CHECK (committed_store_revision >= 0),
    created_at_utc TEXT NOT NULL,
    PRIMARY KEY (world_id, actor, operation_key)
) STRICT;

CREATE TABLE audit_events (
    world_id TEXT NOT NULL REFERENCES worlds(world_id),
    store_revision INTEGER NOT NULL CHECK (store_revision >= 0),
    event_index INTEGER NOT NULL CHECK (event_index BETWEEN 0 AND 65535),
    kind TEXT NOT NULL,
    scope TEXT NOT NULL CHECK (scope IN ('public', 'company', 'admin')),
    company_id TEXT,
    correlation_id TEXT NOT NULL,
    business_year INTEGER NOT NULL CHECK (business_year BETWEEN 1 AND 9999),
    business_month INTEGER NOT NULL CHECK (business_month BETWEEN 1 AND 12),
    before_state_sha256 TEXT NOT NULL CHECK (length(before_state_sha256) = 64),
    after_state_sha256 TEXT NOT NULL CHECK (length(after_state_sha256) = 64),
    payload_version INTEGER NOT NULL CHECK (payload_version > 0),
    payload_json TEXT NOT NULL,
    occurred_at_utc TEXT NOT NULL,
    CHECK (scope <> 'company' OR company_id IS NOT NULL),
    PRIMARY KEY (world_id, store_revision, event_index)
) STRICT;

-- Operational identity data is excluded from portable game exports.
CREATE TABLE sessions (
    session_id TEXT NOT NULL PRIMARY KEY,
    world_id TEXT NOT NULL REFERENCES worlds(world_id),
    company_id TEXT NOT NULL,
    token_sha256 BLOB NOT NULL UNIQUE CHECK (length(token_sha256) = 32),
    expires_at_utc TEXT NOT NULL,
    revoked INTEGER NOT NULL DEFAULT 0 CHECK (revoked IN (0, 1)),
    -- Host sessions may also run world operations (start, pause, resume, end,
    -- company creation/removal); they still act on gameplay only as company_id.
    is_host INTEGER NOT NULL DEFAULT 0 CHECK (is_host IN (0, 1))
) STRICT;
CREATE INDEX sessions_by_world ON sessions(world_id);

CREATE TABLE invitations (
    invitation_id TEXT NOT NULL PRIMARY KEY,
    world_id TEXT NOT NULL REFERENCES worlds(world_id),
    company_id TEXT NOT NULL,
    token_sha256 BLOB NOT NULL UNIQUE CHECK (length(token_sha256) = 32),
    expires_at_utc TEXT NOT NULL,
    remaining_uses INTEGER NOT NULL CHECK (remaining_uses >= 0)
) STRICT;

PRAGMA application_id = 0x57574852;
PRAGMA user_version = 1;
COMMIT;
