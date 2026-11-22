-- Schema version 7: session branches and the branch scope.
--
-- Sessions get the branch name declared at lounge_join, normalized by internal/profile ('' when
-- the session has none). scope gets a fourth value, 'branch': the space of the sessions that
-- declare the same repository unit and the same branch, whatever user they belong to.
--
-- rooms.repository holds the scope identifier for both 'repository' (the repository unit) and
-- 'branch' (<repository>:<branch>). Since branch names never contain ':', distinct pairs of
-- repository unit and branch always produce distinct identifiers.
--
-- SQLite cannot change a CHECK constraint, so rooms, messages and memories are rebuilt with the
-- procedure of https://www.sqlite.org/lang_altertable.html#otheralter (the migration runner
-- disables foreign keys and checks them before commit).
--
-- memories uses AUTOINCREMENT so a deleted memory's ID is never reused. Rebuilding the table
-- copies the sqlite_sequence entry from memories to memories_v7 before dropping memories, so
-- that even when the highest-ID memory was deleted (or all memories were deleted), the next
-- inserted memory never reuses a deleted ID.

ALTER TABLE sessions ADD COLUMN branch TEXT NOT NULL DEFAULT ''; -- normalized Git branch name, lowercase

CREATE TABLE rooms_v7 (
    id         INTEGER PRIMARY KEY,
    scope      TEXT    NOT NULL CHECK (scope IN ('user', 'shared', 'repository', 'branch')),
    user_id    INTEGER REFERENCES users(id) ON DELETE CASCADE,
    repository TEXT    NOT NULL DEFAULT '',
    name       TEXT    NOT NULL,
    created_at INTEGER NOT NULL,
    CHECK ((scope = 'user') = (user_id IS NOT NULL)),
    CHECK ((scope IN ('repository', 'branch')) = (repository <> ''))
) STRICT;

INSERT INTO rooms_v7 (id, scope, user_id, repository, name, created_at)
SELECT id, scope, user_id, repository, name, created_at FROM rooms;

DROP TABLE rooms;
ALTER TABLE rooms_v7 RENAME TO rooms;

CREATE UNIQUE INDEX rooms_space_name ON rooms (scope, coalesce(user_id, 0), repository, name);

CREATE TABLE messages_v7 (
    id                INTEGER PRIMARY KEY,
    scope             TEXT    NOT NULL CHECK (scope IN ('user', 'shared', 'repository', 'branch')),
    sender_session_id INTEGER NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    to_session_id     INTEGER REFERENCES sessions(id) ON DELETE CASCADE,
    room_id           INTEGER REFERENCES rooms(id) ON DELETE CASCADE,
    body              TEXT    NOT NULL,
    sent_at           INTEGER NOT NULL,
    scope_id          TEXT    NOT NULL DEFAULT '',
    sender_agent      TEXT    NOT NULL DEFAULT '',
    sender_os_user    TEXT    NOT NULL DEFAULT '',
    sender_machine    TEXT    NOT NULL DEFAULT '',
    CHECK ((to_session_id IS NULL) <> (room_id IS NULL))
) STRICT;

INSERT INTO messages_v7 (
    id, scope, sender_session_id, to_session_id, room_id, body, sent_at,
    scope_id, sender_agent, sender_os_user, sender_machine
)
SELECT
    id, scope, sender_session_id, to_session_id, room_id, body, sent_at,
    scope_id, sender_agent, sender_os_user, sender_machine
FROM messages;

DROP TABLE messages;
ALTER TABLE messages_v7 RENAME TO messages;

CREATE INDEX messages_room ON messages (room_id, id) WHERE room_id IS NOT NULL;
CREATE INDEX messages_direct ON messages (to_session_id, id) WHERE to_session_id IS NOT NULL;
CREATE INDEX messages_sender ON messages (sender_session_id, id);
CREATE INDEX messages_sent_at ON messages (sent_at);

CREATE TABLE memories_v7 (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    scope           TEXT    NOT NULL CHECK (scope IN ('user', 'shared', 'repository', 'branch')),
    scope_id        TEXT    NOT NULL,            -- user name, repository unit, or <repository>:<branch>
    user_id         INTEGER REFERENCES users(id) ON DELETE CASCADE,
    title           TEXT    NOT NULL DEFAULT '',
    body            TEXT    NOT NULL,
    pinned          INTEGER NOT NULL DEFAULT 0 CHECK (pinned IN (0, 1)),
    created_at      INTEGER NOT NULL,
    updated_at      INTEGER NOT NULL,
    accessed_at     INTEGER NOT NULL DEFAULT 0,
    access_count    INTEGER NOT NULL DEFAULT 0,
    created_user    TEXT    NOT NULL,
    created_session TEXT    NOT NULL,
    created_agent   TEXT    NOT NULL DEFAULT '',
    updated_user    TEXT    NOT NULL,
    updated_session TEXT    NOT NULL,
    updated_agent   TEXT    NOT NULL DEFAULT '',
    CHECK ((scope = 'user') = (user_id IS NOT NULL))
) STRICT;

INSERT INTO memories_v7 (
    id, scope, scope_id, user_id, title, body, pinned, created_at, updated_at, accessed_at,
    access_count, created_user, created_session, created_agent, updated_user, updated_session, updated_agent
)
SELECT
    id, scope, scope_id, user_id, title, body, pinned, created_at, updated_at, accessed_at,
    access_count, created_user, created_session, created_agent, updated_user, updated_session, updated_agent
FROM memories;

-- Preserve sqlite_sequence for memories even when the highest-ID row (or every row) was deleted.
INSERT INTO sqlite_sequence (name, seq)
SELECT 'memories_v7', seq FROM sqlite_sequence
WHERE name = 'memories'
  AND NOT EXISTS (SELECT 1 FROM sqlite_sequence WHERE name = 'memories_v7');

UPDATE sqlite_sequence
SET seq = max(seq, coalesce((SELECT seq FROM sqlite_sequence WHERE name = 'memories'), 0))
WHERE name = 'memories_v7';

DROP TABLE memories;
ALTER TABLE memories_v7 RENAME TO memories;

CREATE INDEX memories_scope ON memories (scope, scope_id, id);
CREATE INDEX memories_user ON memories (user_id) WHERE user_id IS NOT NULL;
