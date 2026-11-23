-- Schema version 10: members of created rooms.
--
-- Every scope has a room named lounge, which every session of the scope's range belongs to:
-- its membership is not recorded. Other rooms are created with the chat_room tool, and a
-- message posted to one reaches only the sessions of the range that are members of it.
--
-- A membership outlives a change of the session's repository unit or branch: while the session
-- is outside the room's range, messages to the room do not reach it, and they reach it again
-- once it is back. Memberships are deleted with their session or their room.

CREATE TABLE room_members (
    room_id    INTEGER NOT NULL REFERENCES rooms(id) ON DELETE CASCADE,
    session_id INTEGER NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    joined_at  INTEGER NOT NULL,
    PRIMARY KEY (room_id, session_id)
) STRICT, WITHOUT ROWID;

CREATE INDEX room_members_session ON room_members (session_id);
