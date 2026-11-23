-- Schema version 8: record the effective policies so administrative memory listings
-- use the same settings as the server's retention decisions.
CREATE TABLE memory_retention (
    scope        TEXT    PRIMARY KEY CHECK (scope IN ('user', 'repository', 'branch')),
    half_life_ns INTEGER NOT NULL CHECK (half_life_ns >= 0),
    threshold    REAL    NOT NULL CHECK (threshold > 0 AND threshold < 1),
    grace_ns     INTEGER NOT NULL CHECK (grace_ns >= 0),
    recorded_at  INTEGER NOT NULL
) STRICT, WITHOUT ROWID;
