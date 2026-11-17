-- Schema version 3: users identified by tokens, and scopes with identifiers.
--
-- users keeps its version 1 definition (name, token hash, creation time). Users are added with
-- the user subcommand; the existing data stays with the user local.
--
-- Every message records its scope and the identifier of that scope: the user name for scope
-- 'user', the repository unit for scope 'repository'. A message addressed to one session needs
-- the column, since it has no room to carry the identifier. A room message records the value too
-- (its room keeps its own user or repository as before), so that every message reads the same way.

ALTER TABLE messages ADD COLUMN scope_id TEXT NOT NULL DEFAULT '';

-- Existing messages: a repository message gets the unit of its room, a user message the name of
-- the sender's user (always local before version 3).
UPDATE messages SET scope_id = (SELECT r.repository FROM rooms r WHERE r.id = messages.room_id)
WHERE scope = 'repository';

UPDATE messages SET scope_id = (
    SELECT u.name FROM sessions s JOIN users u ON u.id = s.user_id WHERE s.id = messages.sender_session_id)
WHERE scope = 'user';
