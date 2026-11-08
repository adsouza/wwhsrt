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
    response_json TEXT NOT NULL,
    committed_store_revision INTEGER NOT NULL CHECK (committed_store_revision >= 0),
    created_at_utc TEXT NOT NULL,
    PRIMARY KEY (world_id, actor, operation_key)
) STRICT;

-- Operational identity data is excluded from portable game exports.
CREATE TABLE sessions (
    session_id TEXT NOT NULL PRIMARY KEY,
    world_id TEXT NOT NULL REFERENCES worlds(world_id),
    company_id TEXT NOT NULL,
    token_sha256 BLOB NOT NULL UNIQUE CHECK (length(token_sha256) = 32),
    expires_at_utc TEXT NOT NULL,
    revoked INTEGER NOT NULL DEFAULT 0 CHECK (revoked IN (0, 1))
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
