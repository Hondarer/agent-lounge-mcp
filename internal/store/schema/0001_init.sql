-- Schema version 1: identification and chat (the "minimal configuration" stage of the proposal).
-- Times are Unix milliseconds (UTC). scope is 'user' (one user's space) or 'shared' (all users).

CREATE TABLE users (
    id         INTEGER PRIMARY KEY,
    name       TEXT    NOT NULL UNIQUE,
    token_hash TEXT    NOT NULL UNIQUE, -- only the hash of the token is stored
    created_at INTEGER NOT NULL
) STRICT;

CREATE TABLE sessions (
    id           INTEGER PRIMARY KEY,
    user_id      INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         TEXT    NOT NULL,            -- session name given at lounge_join
    agent        TEXT    NOT NULL DEFAULT '', -- declared agent name, e.g. claude-code@host-a
    created_at   INTEGER NOT NULL,
    last_seen_at INTEGER NOT NULL,
    UNIQUE (user_id, name)
) STRICT;

-- user_id is the owning user for scope 'user' and NULL for scope 'shared'.
CREATE TABLE rooms (
    id         INTEGER PRIMARY KEY,
    scope      TEXT    NOT NULL CHECK (scope IN ('user', 'shared')),
    user_id    INTEGER REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT    NOT NULL,
    created_at INTEGER NOT NULL,
    CHECK ((scope = 'user') = (user_id IS NOT NULL))
) STRICT;

CREATE UNIQUE INDEX rooms_space_name ON rooms (scope, coalesce(user_id, 0), name);

-- A message is addressed to exactly one of a session or a room.
CREATE TABLE messages (
    id                INTEGER PRIMARY KEY,
    scope             TEXT    NOT NULL CHECK (scope IN ('user', 'shared')),
    sender_session_id INTEGER NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    to_session_id     INTEGER REFERENCES sessions(id) ON DELETE CASCADE,
    room_id           INTEGER REFERENCES rooms(id) ON DELETE CASCADE,
    body              TEXT    NOT NULL,
    sent_at           INTEGER NOT NULL,
    CHECK ((to_session_id IS NULL) <> (room_id IS NULL))
) STRICT;

CREATE INDEX messages_room ON messages (room_id, id) WHERE room_id IS NOT NULL;
CREATE INDEX messages_direct ON messages (to_session_id, id) WHERE to_session_id IS NOT NULL;
CREATE INDEX messages_sender ON messages (sender_session_id, id);

-- One row per message and recipient session.
CREATE TABLE deliveries (
    message_id           INTEGER NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    recipient_session_id INTEGER NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    state                TEXT    NOT NULL DEFAULT 'queued'
                         CHECK (state IN ('queued', 'delivered', 'acked', 'expired')),
    queued_at            INTEGER NOT NULL,
    delivered_at         INTEGER,
    acked_at             INTEGER,
    expired_at           INTEGER,
    PRIMARY KEY (message_id, recipient_session_id)
) STRICT, WITHOUT ROWID;

CREATE INDEX deliveries_inbox ON deliveries (recipient_session_id, state, message_id);

-- Every space starts with a room named lounge. The one for the shared space is created here;
-- a user's own lounge is created together with the user.
INSERT INTO rooms (scope, user_id, name, created_at)
VALUES ('shared', NULL, 'lounge', CAST(unixepoch('subsec') * 1000 AS INTEGER));
