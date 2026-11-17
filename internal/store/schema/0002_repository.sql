-- Schema version 2: session profiles and the repository space.
--
-- Sessions get the profile values declared at lounge_join, normalized by internal/profile, and
-- the repository unit the server decided from them ('' when the session has none).
-- scope gets a third value, 'repository': the space of the sessions that declare the same
-- repository unit, whatever user they belong to.
--
-- SQLite cannot change a CHECK constraint, so rooms and messages are rebuilt with the procedure
-- of https://www.sqlite.org/lang_altertable.html#otheralter: create the new table, copy the rows
-- with their IDs, drop the old table and rename the new one. The migration runner applies this
-- script with foreign key enforcement off, so dropping a table does not cascade to the rows that
-- refer to it, and checks the foreign keys before it commits.

ALTER TABLE sessions ADD COLUMN os_user    TEXT NOT NULL DEFAULT ''; -- declared OS user name
ALTER TABLE sessions ADD COLUMN machine    TEXT NOT NULL DEFAULT ''; -- declared machine name, lowercase
ALTER TABLE sessions ADD COLUMN origin     TEXT NOT NULL DEFAULT ''; -- normalized Git origin, host/path
ALTER TABLE sessions ADD COLUMN folder     TEXT NOT NULL DEFAULT ''; -- normalized working folder
ALTER TABLE sessions ADD COLUMN repository TEXT NOT NULL DEFAULT ''; -- repository unit, '' for none

CREATE INDEX sessions_repository ON sessions (repository);

-- user_id is the owning user for scope 'user' and NULL otherwise. repository is the repository
-- unit for scope 'repository' and '' otherwise. A repository room has no user: it is created
-- at the first message sent to its unit.
CREATE TABLE rooms_v2 (
    id         INTEGER PRIMARY KEY,
    scope      TEXT    NOT NULL CHECK (scope IN ('user', 'shared', 'repository')),
    user_id    INTEGER REFERENCES users(id) ON DELETE CASCADE,
    repository TEXT    NOT NULL DEFAULT '',
    name       TEXT    NOT NULL,
    created_at INTEGER NOT NULL,
    CHECK ((scope = 'user') = (user_id IS NOT NULL)),
    CHECK ((scope = 'repository') = (repository <> ''))
) STRICT;

INSERT INTO rooms_v2 (id, scope, user_id, repository, name, created_at)
SELECT id, scope, user_id, '', name, created_at FROM rooms;

DROP TABLE rooms;
ALTER TABLE rooms_v2 RENAME TO rooms;

CREATE UNIQUE INDEX rooms_space_name ON rooms (scope, coalesce(user_id, 0), repository, name);

-- A message is addressed to exactly one of a session or a room. A message of scope
-- 'repository' is addressed to a repository room; its unit is the room's repository.
CREATE TABLE messages_v2 (
    id                INTEGER PRIMARY KEY,
    scope             TEXT    NOT NULL CHECK (scope IN ('user', 'shared', 'repository')),
    sender_session_id INTEGER NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    to_session_id     INTEGER REFERENCES sessions(id) ON DELETE CASCADE,
    room_id           INTEGER REFERENCES rooms(id) ON DELETE CASCADE,
    body              TEXT    NOT NULL,
    sent_at           INTEGER NOT NULL,
    CHECK ((to_session_id IS NULL) <> (room_id IS NULL))
) STRICT;

INSERT INTO messages_v2 (id, scope, sender_session_id, to_session_id, room_id, body, sent_at)
SELECT id, scope, sender_session_id, to_session_id, room_id, body, sent_at FROM messages;

DROP TABLE messages;
ALTER TABLE messages_v2 RENAME TO messages;

CREATE INDEX messages_room ON messages (room_id, id) WHERE room_id IS NOT NULL;
CREATE INDEX messages_direct ON messages (to_session_id, id) WHERE to_session_id IS NOT NULL;
CREATE INDEX messages_sender ON messages (sender_session_id, id);
