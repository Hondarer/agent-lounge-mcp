-- Schema version 6: the expiry of a user's token and the last use of the token.
--
-- token_expires_at is the time (milliseconds) from which the token is no longer accepted; NULL
-- for a token without an expiry. The server compares it with its own clock on every request.
-- user expiry changes it without changing the token; user rotate replaces it with the expiry of
-- the new token (NULL when none is given).
--
-- last_used_at is the time (milliseconds) of the last request authenticated with the token,
-- also one that joined no session; NULL when none was recorded. The server writes it at most
-- once a minute for each user. The user local has no token, so its value stays NULL.
--
-- Existing users get NULL in both: no expiry, nothing recorded.

ALTER TABLE users ADD COLUMN token_expires_at INTEGER;
ALTER TABLE users ADD COLUMN last_used_at INTEGER;
