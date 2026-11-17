-- Schema version 4: the sender's values at the time of sending, and indexes for the cleanup.
--
-- A message keeps the agent name, OS user name and machine name its sender declared when it
-- was sent, so that a later lounge_join of the sender does not change how earlier messages
-- read. The session name and the user name identify the session and never change, so they are
-- not copied. Existing messages get the values their sender's session has now.

ALTER TABLE messages ADD COLUMN sender_agent   TEXT NOT NULL DEFAULT ''; -- declared agent name when sent
ALTER TABLE messages ADD COLUMN sender_os_user TEXT NOT NULL DEFAULT ''; -- declared OS user name when sent
ALTER TABLE messages ADD COLUMN sender_machine TEXT NOT NULL DEFAULT ''; -- declared machine name when sent

UPDATE messages SET
    sender_agent   = (SELECT s.agent   FROM sessions s WHERE s.id = messages.sender_session_id),
    sender_os_user = (SELECT s.os_user FROM sessions s WHERE s.id = messages.sender_session_id),
    sender_machine = (SELECT s.machine FROM sessions s WHERE s.id = messages.sender_session_id);

-- The periodic cleanup (Store.Cleanup) looks for queued deliveries by the time they were
-- queued, for messages by the time they were sent, and for sessions by their last use.
CREATE INDEX deliveries_queued ON deliveries (queued_at) WHERE state = 'queued';
CREATE INDEX messages_sent_at ON messages (sent_at);
CREATE INDEX sessions_last_seen ON sessions (last_seen_at);
