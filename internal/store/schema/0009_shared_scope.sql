-- Schema version 9: allow the 'shared' scope in the recorded retention policies.
--
-- The other tables that hold a scope (rooms, messages, memories, memory_scope_activity) already
-- accept 'shared'. memory_retention was created with the three kinds that existed then.
--
-- SQLite cannot change a CHECK constraint, so the table is rebuilt with the procedure of
-- https://www.sqlite.org/lang_altertable.html#otheralter (the migration runner disables foreign
-- keys and checks them before commit).

CREATE TABLE memory_retention_v9 (
    scope        TEXT    PRIMARY KEY CHECK (scope IN ('user', 'repository', 'branch', 'shared')),
    half_life_ns INTEGER NOT NULL CHECK (half_life_ns >= 0),
    threshold    REAL    NOT NULL CHECK (threshold > 0 AND threshold < 1),
    grace_ns     INTEGER NOT NULL CHECK (grace_ns >= 0),
    recorded_at  INTEGER NOT NULL
) STRICT, WITHOUT ROWID;

INSERT INTO memory_retention_v9 (scope, half_life_ns, threshold, grace_ns, recorded_at)
SELECT scope, half_life_ns, threshold, grace_ns, recorded_at FROM memory_retention;

DROP TABLE memory_retention;
ALTER TABLE memory_retention_v9 RENAME TO memory_retention;
