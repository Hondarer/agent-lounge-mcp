-- Schema version 5: memories, their full-text index and the record of their references.
--
-- A memory belongs to one scope (kind and identifier, as a message does). Unlike a message it
-- does not belong to a session: it outlives the session that created it, so its creator and
-- last updater are kept as the values they had when they wrote it (user, session and agent
-- names), not as references to sessions. A memory of scope 'user' refers to its user, so that
-- removing the user removes its memories; a memory of another scope has no user and stays.
--
-- Whether a memory has faded, and when it is deleted, is not stored: the retention score
-- depends on the current time and on the startup options, so it is computed when it is needed
-- (see retention.go). The columns hold its inputs: updated_at, accessed_at and access_count.
--
-- AUTOINCREMENT keeps the ID of a deleted memory from being given to a new one, so that an ID
-- an agent kept never names a different memory.

CREATE TABLE memories (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    scope           TEXT    NOT NULL CHECK (scope IN ('user', 'shared', 'repository')),
    scope_id        TEXT    NOT NULL,            -- user name, or repository unit
    user_id         INTEGER REFERENCES users(id) ON DELETE CASCADE,
    title           TEXT    NOT NULL DEFAULT '',
    body            TEXT    NOT NULL,
    pinned          INTEGER NOT NULL DEFAULT 0 CHECK (pinned IN (0, 1)),
    created_at      INTEGER NOT NULL,
    updated_at      INTEGER NOT NULL,            -- last change of title, body or tags
    accessed_at     INTEGER NOT NULL DEFAULT 0,  -- last counted reference, 0 for none
    access_count    INTEGER NOT NULL DEFAULT 0,  -- counted references
    created_user    TEXT    NOT NULL,
    created_session TEXT    NOT NULL,
    created_agent   TEXT    NOT NULL DEFAULT '',
    updated_user    TEXT    NOT NULL,
    updated_session TEXT    NOT NULL,
    updated_agent   TEXT    NOT NULL DEFAULT '',
    CHECK ((scope = 'user') = (user_id IS NOT NULL))
) STRICT;

CREATE INDEX memories_scope ON memories (scope, scope_id, id);
CREATE INDEX memories_user ON memories (user_id) WHERE user_id IS NOT NULL;

-- Tags of a memory, in the order they were given (pos). Stored lowercase.
CREATE TABLE memory_tags (
    memory_id INTEGER NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    pos       INTEGER NOT NULL,
    tag       TEXT    NOT NULL,
    PRIMARY KEY (memory_id, pos),
    UNIQUE (memory_id, tag)
) STRICT, WITHOUT ROWID;

CREATE INDEX memory_tags_tag ON memory_tags (tag, memory_id);

-- Full-text index of title, body and tags (separated by spaces). Its rowid is the memory ID.
-- The trigram tokenizer indexes every run of three characters, which finds words in Japanese
-- text without word boundaries, ignoring the case of letters. The rows are written in the same
-- transaction as the memory (memory.go): a virtual table has no foreign keys or triggers here.
CREATE VIRTUAL TABLE memories_fts USING fts5(title, body, tags, tokenize = 'trigram');

-- A reference is counted at most once per session and memory in a day (memory.go). Each row
-- is the last counted reference of a session to a memory; rows older than a day are deleted
-- by the cleanup.
CREATE TABLE memory_refs (
    memory_id  INTEGER NOT NULL REFERENCES memories(id) ON DELETE CASCADE,
    session_id INTEGER NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    counted_at INTEGER NOT NULL,
    PRIMARY KEY (memory_id, session_id)
) STRICT, WITHOUT ROWID;

CREATE INDEX memory_refs_counted ON memory_refs (counted_at);
CREATE INDEX memory_refs_session ON memory_refs (session_id);
