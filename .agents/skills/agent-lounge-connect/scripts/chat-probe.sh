#!/bin/sh
# Verify that connections to a running agent-lounge-mcp server can talk: they join with profiles,
# one sends a message to another's session and one to the lounge room of the user, and the other
# receives both. Then the completion of messages: probe-a checks the delivery of its message
# with chat_status (delivered, then acked after probe-b calls chat_ack, which gives the same
# result when repeated), another session's status request gets the error of a missing message,
# and chat_history returns the conversation in both directions, the read and unacked messages
# with unacked, and no unread message. Then the repository scope: probe-a and probe-b declare one repository in two
# forms of its origin, probe-c declares a folder without origin; a message to the repository room
# reaches probe-b and not probe-c, and a message to probe-b in scope repository reaches it. Then
# memories: a memory of the repository scope is found, read and deleted by another session of the
# repository, and not found from another repository. Then one connection waits in chat_inbox
# (wait_seconds) and receives a message sent while it waits;
# the probe sends only after /healthz reports the wait.
#
# With a second user's token, the probe also checks the repository scope across two users: a
# session of the second user in the same repository (also named probe-b) is listed by lounge_who
# with its user, address and shared scope, receives room and direct messages of scope
# repository, and replies with the scope and reply_to of the received message; a session of the
# second user in another repository is neither listed nor addressable.
#
# usage: chat-probe.sh [BASE_URL]      (default: http://127.0.0.1:7788)
# Tokens are read from the environment and sent as "Authorization: Bearer <token>"; they are
# never printed:
#   AGENT_LOUNGE_TOKEN    token of the first user (probe-a, probe-b, probe-c). Without it, the
#                         probe sends no token (the user local, on a server that does not
#                         require a token).
#   AGENT_LOUNGE_TOKEN_B  token of a second user, to run the checks across two users. Without it,
#                         those checks are skipped.
# Prints "OK: ..." lines and exits 0, or prints "NG: ..." to standard error and exits 1.
#
# The probe joins as the sessions probe-a, probe-b and probe-c (and, with a second user,
# probe-b and probe-e of that user), which stay in the database with made-up profile values,
# and its user room message is queued for every other session of the user too. Run it against
# a database used for checks.
set -eu

BASE=${1:-http://127.0.0.1:7788}
BASE=${BASE%/}
WAIT=${PROBE_WAIT:-5}
TOKEN_A=${AGENT_LOUNGE_TOKEN:-}
TOKEN_B=${AGENT_LOUNGE_TOKEN_B:-}

ng() {
	echo "NG: $*" >&2
	exit 1
}

command -v curl >/dev/null 2>&1 || ng "curl is required"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# Sessions of the second user; set when they connect. The token of a request is the second
# user's for these sessions, else the first user's. While connect opens a session, the session
# (pending) uses connect_token.
d=none
e=none
pending=none
connect_token=$TOKEN_A

# token_of SESSION_ID: print the token to send for the MCP session ("" for a new session).
token_of() {
	case $1 in
	"" | "$pending") printf '%s' "$connect_token" ;;
	"$d" | "$e") printf '%s' "$TOKEN_B" ;;
	*) printf '%s' "$TOKEN_A" ;;
	esac
}

# post SESSION_ID BODY: POST one JSON-RPC message; the response body goes to standard output.
post() {
	tok=$(token_of "$1")
	code=$(curl -sS --max-time "$WAIT" -D "$tmp/headers" -o "$tmp/body" -w '%{http_code}' \
		-H 'Content-Type: application/json' \
		-H 'Accept: application/json, text/event-stream' \
		${tok:+-H "Authorization: Bearer $tok"} \
		${1:+-H "Mcp-Session-Id: $1"} \
		--data "$2" "$BASE/mcp" 2>"$tmp/err") || ng "POST $BASE/mcp failed ($(cat "$tmp/err")). Is the server running?"
	case $code in
	2??) cat "$tmp/body" ;;
	401)
		if [ -n "$tok" ]; then
			ng "the server rejected a token (401: $(cat "$tmp/body")). Check AGENT_LOUNGE_TOKEN and AGENT_LOUNGE_TOKEN_B"
		else
			ng "the server requires a token: set AGENT_LOUNGE_TOKEN (401: $(cat "$tmp/body"))"
		fi
		;;
	*) ng "POST $BASE/mcp returned HTTP $code: $(cat "$tmp/body")" ;;
	esac
}

# connect: open an MCP session and print its ID.
connect() {
	post "" '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"chat-probe","version":"0"}}}' >/dev/null
	id=$(tr -d '\r' <"$tmp/headers" | awk -F': ' 'tolower($1) == "mcp-session-id" { print $2 }')
	[ -n "$id" ] || ng "the server returned no Mcp-Session-Id header"
	pending=$id
	post "$id" '{"jsonrpc":"2.0","method":"notifications/initialized"}' >/dev/null
	echo "$id"
}

# call SESSION_ID TOOL ARGUMENTS_JSON: call a tool and print the response; fail on a tool error.
call() {
	out=$(post "$1" "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"$2\",\"arguments\":$3}}")
	case $out in
	*'"isError":true'*) ng "$2 failed: $out" ;;
	*'"structuredContent"'*) printf '%s\n' "$out" ;;
	*) ng "unexpected response to $2: $out" ;;
	esac
}

# call_error SESSION_ID TOOL ARGUMENTS_JSON: call a tool that must fail and print the response.
call_error() {
	out=$(post "$1" "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"tools/call\",\"params\":{\"name\":\"$2\",\"arguments\":$3}}")
	expect '"isError":true' "$2 $3 must fail" "$out"
	printf '%s\n' "$out"
}

# expect TEXT WHAT RESPONSE: fail unless RESPONSE contains TEXT.
expect() {
	case $3 in
	*"$1"*) ;;
	*) ng "$2: expected $1 in: $3" ;;
	esac
}

# reject TEXT WHAT RESPONSE: fail if RESPONSE contains TEXT.
reject() {
	case $3 in
	*"$1"*) ng "$2: unexpected $1 in: $3" ;;
	esac
}

# field NAME RESPONSE: print the first string value of NAME in RESPONSE.
field() {
	printf '%s' "$2" | grep -o "\"$1\":\"[^\"]*\"" | head -n 1 | cut -d'"' -f4
}

# num_field NAME RESPONSE: print the first number value of NAME in RESPONSE.
num_field() {
	printf '%s' "$2" | grep -o "\"$1\":[0-9][0-9]*" | head -n 1 | cut -d: -f2
}

# The order of keys in a result is not fixed, so checks look at single fields of single objects.
# norm RESPONSE: print RESPONSE with each scope object ({"scope":S,"id":I} in any key order)
# replaced by <S:I>, so that sessions and messages are objects without nested objects.
norm() {
	printf '%s' "$1" |
		sed -e 's/{"scope":"\([^"]*\)","id":"\([^"]*\)"}/<\1:\2>/g' -e 's/{"id":"\([^"]*\)","scope":"\([^"]*\)"}/<\2:\1>/g'
}

# object_with TEXT RESPONSE: print the first object (session or message) of RESPONSE that
# contains TEXT, after norm. Only the structured result matches: its text copy escapes quotes.
object_with() {
	norm "$2" | grep -o "{[^{}]*$1[^{}]*}" | head -n 1
}

# top_level RESPONSE: print RESPONSE after norm without its session and message objects.
top_level() {
	norm "$1" | sed 's/{[^{}]*}//g'
}

# waiting: print the number of waiting chat_inbox calls reported by /healthz.
waiting() {
	curl -fsS --max-time "$WAIT" "$BASE/healthz" 2>"$tmp/err" | sed -n 's/.*"waiting":\([0-9][0-9]*\).*/\1/p'
}

mark=$(date +%s)-$$
a=$(connect)
b=$(connect)
c=$(connect)

# Tools other than lounge_join refuse a connection that has not joined.
out=$(post "$a" '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"chat_inbox","arguments":{}}}')
expect 'not joined' "chat_inbox before lounge_join" "$out"

# Profiles: values are passed as the commands print them (trailing newlines, a WSL path, an origin
# with credentials) and come back normalized. probe-a and probe-b give one origin in two forms.
repo="github.com/example-org/lounge-probe-$mark"
out=$(call "$a" lounge_join "{\"name\":\"probe-a\",\"agent\":\"chat-probe\",\"os_user\":\"probe-user\\r\\n\",\"machine\":\"PROBE-HOST\\r\\n\",\"origin\":\"https://probe:SECRET@GitHub.com/Example-Org/Lounge-Probe-$mark.git\\n\",\"folder\":\"/mnt/c/Probe/A\\n\",\"wsl_distro\":\"Ubuntu\\n\"}")
expect '"os_user":"probe-user"' "lounge_join with a profile" "$out"
expect '"machine":"probe-host"' "lounge_join with a profile" "$out"
expect "\"origin\":\"$repo\"" "lounge_join with a profile" "$out"
expect '"folder":"c:\\probe\\a"' "lounge_join with a profile" "$out"
expect "\"repository\":\"$repo\"" "lounge_join with a profile" "$out"
reject 'SECRET' "lounge_join with a credentialed origin" "$out"
user_a=$(field user "$out")
[ -n "$user_a" ] || ng "lounge_join returned no user: $out"
out=$(call "$b" lounge_join "{\"name\":\"probe-b\",\"agent\":\"chat-probe\",\"os_user\":\"probe-user\",\"machine\":\"probe-host\",\"origin\":\"git@github.com:example-org/lounge-probe-$mark.git\",\"folder\":\"/home/probe/b\",\"wsl_distro\":\"Ubuntu\"}")
expect "\"repository\":\"$repo\"" "lounge_join with the scp-like origin" "$out"
expect '"folder":"\\\\wsl.localhost\\ubuntu\\home\\probe\\b"' "lounge_join with a WSL folder" "$out"
# probe-c: a folder without origin; its repository is machine:folder.
out=$(call "$c" lounge_join "{\"name\":\"probe-c\",\"agent\":\"chat-probe\",\"machine\":\"probe-host\",\"origin\":\"\",\"folder\":\"/home/probe/c-$mark\",\"wsl_distro\":\"Ubuntu\"}")
expect '"repository":"probe-host:\\\\wsl.localhost\\ubuntu\\home\\probe\\c-'"$mark"'"' "lounge_join with a folder without origin" "$out"
# A folder without origin needs a machine.
out=$(call_error "$c" lounge_join '{"name":"probe-c","machine":"","origin":"","folder":"/home/probe/c"}')
expect 'machine' "lounge_join with a folder and no machine" "$out"
call "$b" chat_inbox '{"limit":100}' >/dev/null # drop what earlier runs left unread
call "$c" chat_inbox '{"limit":100}' >/dev/null
echo "OK: joined as user $user_a: probe-a, probe-b ($repo) and probe-c (folder without origin); profiles normalized"

# lounge_who: the caller's scopes with their ids, and for each session its user, address and
# the scopes shared with the caller.
out=$(call "$a" lounge_who '{}')
top=$(top_level "$out")
expect "\"repository\":\"$repo\"" "lounge_who (own repository)" "$top"
expect "\"scopes\":[<user:$user_a>,<repository:$repo>]" "lounge_who (own scopes)" "$top"
reject '"same_repository"' "lounge_who" "$out"
session_b=$(object_with '"name":"probe-b"' "$out")
[ -n "$session_b" ] || ng "lounge_who does not list probe-b: $out"
for f in "\"user\":\"$user_a\"" '"address":"probe-b"' '"os_user":"probe-user"' '"machine":"probe-host"' "\"repository\":\"$repo\"" \
	"\"scopes\":[<user:$user_a>,<repository:$repo>]"; do
	expect "$f" "lounge_who (probe-b)" "$session_b"
done
session_c=$(object_with '"name":"probe-c"' "$out")
[ -n "$session_c" ] || ng "lounge_who does not list probe-c: $out"
for f in '"address":"probe-c"' '"machine":"probe-host"' '"repository":"probe-host:' "\"scopes\":[<user:$user_a>]"; do
	expect "$f" "lounge_who (probe-c)" "$session_c"
done
echo "OK: lounge_who lists the scopes with their ids, and each session's user, address and shared scopes"

out=$(call "$a" chat_send "{\"to\":\"probe-b\",\"body\":\"direct-$mark\"}")
direct_id=$(num_field message_id "$out")
[ -n "$direct_id" ] || ng "chat_send returned no message_id: $out"
expect '"recipients":["probe-b"]' "chat_send to a session" "$out"
expect '"scope":"user"' "chat_send to a session" "$out"
expect "\"scope_id\":\"$user_a\"" "chat_send to a session" "$out"
out=$(call "$b" lounge_info '{}')
expect '"unread":1' "unread count after a message" "$out"
out=$(call "$b" chat_inbox '{}')
expect '"unread":0' "unread count after reading" "$out"
msg=$(object_with "\"body\":\"direct-$mark\"" "$out")
[ -n "$msg" ] || ng "chat_inbox: no message direct-$mark in: $out"
for f in '"from":"probe-a"' "\"from_user\":\"$user_a\"" '"scope":"user"' "\"scope_id\":\"$user_a\"" '"reply_to":"probe-a"'; do
	expect "$f" "chat_inbox" "$msg"
done
reject '"room"' "chat_inbox (a message to a session)" "$msg"
echo "OK: message to a session delivered (probe-a -> probe-b), with scope, scope_id and reply_to"

# chat_status: the sender sees the delivery; chat_ack: the recipient records that it handled
# the message, and repeating the call gives the same result.
out=$(call "$a" chat_status "{\"message_id\":$direct_id}")
recipient=$(object_with '"address":"probe-b"' "$out")
[ -n "$recipient" ] || ng "chat_status lists no probe-b: $out"
expect '"state":"delivered"' "chat_status after chat_inbox" "$recipient"
expect '"delivered_at":"' "chat_status after chat_inbox" "$recipient"
reject '"acked_at"' "chat_status before chat_ack" "$recipient"
expect '"delivered":1' "chat_status counts" "$out"
missing_id=999999999
out=$(call "$b" chat_ack "{\"message_ids\":[$direct_id,$missing_id]}")
expect "\"acked\":[$direct_id]" "chat_ack" "$out"
expect "\"not_found\":[$missing_id]" "chat_ack of a missing message" "$out"
out=$(call "$b" chat_ack "{\"message_ids\":[$direct_id]}")
expect "\"acked\":[$direct_id]" "chat_ack repeated" "$out"
reject '"not_found"' "chat_ack repeated" "$out"
out=$(call "$a" chat_status "{\"message_id\":$direct_id}")
recipient=$(object_with '"address":"probe-b"' "$out")
expect '"state":"acked"' "chat_status after chat_ack" "$recipient"
expect '"acked_at":"' "chat_status after chat_ack" "$recipient"
expect '"acked":1' "chat_status counts after chat_ack" "$out"
# Another session's message and a missing message give the same error.
other=$(call_error "$b" chat_status "{\"message_id\":$direct_id}" | sed "s/$direct_id/<id>/g")
missing=$(call_error "$a" chat_status "{\"message_id\":$missing_id}" | sed "s/$missing_id/<id>/g")
[ "$other" = "$missing" ] || ng "chat_status of another session's message and of a missing one give different errors: $other / $missing"
echo "OK: chat_status shows delivered, then acked after chat_ack (repeatable); another session's message gives the error of a missing one"

# chat_history: both directions, unacked, and no unread message.
call "$a" chat_send "{\"to\":\"probe-b\",\"body\":\"pending-$mark\"}" >/dev/null
call "$b" chat_inbox '{"limit":100}' >/dev/null
call "$a" chat_send "{\"to\":\"probe-b\",\"body\":\"unread-$mark\"}" >/dev/null
out=$(call "$b" chat_history '{"with":"probe-a","limit":100}')
reject "unread-$mark" "chat_history (an unread message)" "$out"
msg=$(object_with "\"body\":\"direct-$mark\"" "$out")
[ -n "$msg" ] || ng "chat_history of probe-b: no message direct-$mark in: $out"
for f in '"direction":"received"' '"state":"acked"' '"reply_to":"probe-a"' '"to":"probe-b"' '"from_agent":"chat-probe"'; do
	expect "$f" "chat_history (received)" "$msg"
done
out=$(call "$b" chat_history '{"unacked":true,"limit":100}')
expect "\"body\":\"pending-$mark\"" "chat_history unacked" "$out"
reject "\"body\":\"direct-$mark\"" "chat_history unacked" "$out"
out=$(call "$a" chat_history '{"with":"probe-b","limit":1}')
msg=$(object_with "\"body\":\"unread-$mark\"" "$out")
[ -n "$msg" ] || ng "chat_history of probe-a: no message unread-$mark in: $out"
expect '"direction":"sent"' "chat_history (sent)" "$msg"
reject '"state"' "chat_history (sent)" "$msg"
expect '"next_before":' "chat_history with limit 1" "$out"
out=$(call "$b" chat_inbox '{"limit":100}')
expect "\"body\":\"unread-$mark\"" "chat_inbox after chat_history (history must not mark messages read)" "$out"
out=$(call "$b" chat_ack "{\"message_ids\":[$(num_field message_id "$(object_with "\"body\":\"unread-$mark\"" "$out")")]}")
expect '"acked":[' "chat_ack of unread-$mark" "$out"
echo "OK: chat_history returns sent and received messages, unacked ones with unacked, and leaves unread ones to chat_inbox"

out=$(call "$b" chat_send "{\"room\":\"lounge\",\"body\":\"room-$mark\"}")
expect '"probe-a"' "chat_send to the room" "$out"
out=$(call "$a" chat_inbox '{"limit":100}')
expect "\"body\":\"room-$mark\"" "chat_inbox" "$out"
expect '"room":"lounge"' "chat_inbox" "$out"
echo "OK: message to the lounge room delivered (probe-b -> probe-a)"
call "$c" chat_inbox '{"limit":100}' >/dev/null # the user room message reached probe-c too

# Repository scope: probe-a's message reaches probe-b (same repository) and not probe-c.
out=$(call "$a" chat_send "{\"room\":\"lounge\",\"scope\":\"repository\",\"body\":\"repo-$mark\"}")
expect '"recipients":["probe-b"]' "chat_send to the repository room" "$out"
expect '"scope":"repository"' "chat_send to the repository room" "$out"
expect "\"scope_id\":\"$repo\"" "chat_send to the repository room" "$out"
out=$(call "$a" chat_send "{\"to\":\"probe-b\",\"scope\":\"repository\",\"body\":\"repo-direct-$mark\"}")
expect '"recipients":["probe-b"]' "chat_send to a session in scope repository" "$out"
expect '"scope":"repository"' "chat_send to a session in scope repository" "$out"
expect "\"scope_id\":\"$repo\"" "chat_send to a session in scope repository" "$out"
out=$(call "$b" chat_inbox '{"limit":100}')
for body in "repo-$mark" "repo-direct-$mark"; do
	msg=$(object_with "\"body\":\"$body\"" "$out")
	[ -n "$msg" ] || ng "chat_inbox: no message $body in: $out"
	for f in '"scope":"repository"' "\"scope_id\":\"$repo\"" '"reply_to":"probe-a"' '"from_os_user":"probe-user"' '"from_machine":"probe-host"'; do
		expect "$f" "chat_inbox of $body" "$msg"
	done
done
expect '"room":"lounge"' "chat_inbox of the repository room" "$(object_with "\"body\":\"repo-$mark\"" "$out")"
out=$(call "$c" chat_inbox '{"limit":100}')
reject "repo-$mark" "chat_inbox of another repository" "$out"
echo "OK: messages of scope repository delivered (probe-a -> probe-b, room and session), not to another repository (probe-c)"

out=$(call "$c" chat_send "{\"room\":\"lounge\",\"scope\":\"repository\",\"body\":\"folder-$mark\"}")
expect '"recipients":[]' "chat_send to the room of a folder without origin" "$out"
# probe-a is not in probe-c's repository: the same error as a session that does not exist.
outside=$(call_error "$c" chat_send '{"to":"probe-a","scope":"repository","body":"x"}' | sed 's/probe-a/<to>/g')
missing=$(call_error "$c" chat_send "{\"to\":\"probe-none-$mark\",\"scope\":\"repository\",\"body\":\"x\"}" | sed "s/probe-none-$mark/<to>/g")
[ "$outside" = "$missing" ] || ng "a session outside the scope and a missing session give different errors: $outside / $missing"
expect 'lounge_who' "chat_send to a session outside the scope" "$outside"
out=$(call "$a" chat_inbox '{"limit":100}')
reject "folder-$mark" "chat_inbox of another repository" "$out"
echo "OK: probe-c's repository room reaches no other probe session; a session outside the scope gives the error of a missing one"

# Memories: probe-a saves one in the repository scope; probe-b (the same repository) finds it by
# a two-character Japanese term and the run's mark, with a tag in another case, reads it whole
# and deletes it; probe-c (another repository) does not find it. Nothing of it stays.
out=$(call "$a" memory_put "{\"scope\":\"repository\",\"body\":\"認証 memo-$mark\",\"tags\":[\"Probe\"]}")
# The text copy of the result keeps the order of the fields (escaped): the memory's id comes first.
mem=$(printf '%s' "$out" | grep -o '\\"memory\\":{\\"id\\":[0-9]*' | head -n 1 | grep -o '[0-9]*$')
[ -n "$mem" ] || ng "memory_put returned no memory ID: $out"
expect '"created":true' "memory_put" "$out"
out=$(call "$b" memory_search "{\"query\":\"認証 memo-$mark\",\"tags\":[\"PROBE\"]}")
expect "{\\\"id\\\":$mem,\\\"scope\\\":\\\"repository\\\"" "memory_search of probe-b" "$out"
expect '"count":1' "memory_search of probe-b" "$out"
out=$(call "$c" memory_search "{\"query\":\"memo-$mark\"}")
expect '"count":0' "memory_search of probe-c (another repository)" "$out"
out=$(call "$b" memory_get "{\"id\":$mem}")
expect "\"body\":\"認証 memo-$mark\"" "memory_get" "$out"
expect '"access_count":1' "memory_get (the search counted one reference)" "$out"
call "$b" memory_forget "{\"id\":$mem}" >/dev/null
out=$(call_error "$a" memory_get "{\"id\":$mem}")
expect 'no memory with ID' "memory_get after memory_forget" "$out"
echo "OK: a repository memory saved by probe-a is found and read by probe-b, not by probe-c, and deleted"

# probe-b waits in chat_inbox in the background; probe-a sends while it waits. The waiting call
# uses its own headers file and a longer timeout than the wait.
wait_seconds=10
before=$(waiting)
[ -n "$before" ] || ng "/healthz reports no waiting field (an older server?)"
curl -fsS --max-time $((WAIT + wait_seconds)) -D "$tmp/wait-headers" \
	-H 'Content-Type: application/json' \
	-H 'Accept: application/json, text/event-stream' \
	${TOKEN_A:+-H "Authorization: Bearer $TOKEN_A"} \
	-H "Mcp-Session-Id: $b" \
	--data "{\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"tools/call\",\"params\":{\"name\":\"chat_inbox\",\"arguments\":{\"wait_seconds\":$wait_seconds}}}" \
	"$BASE/mcp" >"$tmp/wait-out" 2>"$tmp/wait-err" &
waiter=$!
# Send only after /healthz reports one more waiting call than before (another client may wait too).
tries=$((WAIT * 10))
until [ "$(waiting)" -gt "$before" ] 2>/dev/null; do
	tries=$((tries - 1))
	[ "$tries" -gt 0 ] || ng "/healthz did not report the waiting chat_inbox within ${WAIT}s"
	sleep 0.1
done
call "$a" chat_send "{\"to\":\"probe-b\",\"body\":\"waited-$mark\"}" >/dev/null
wait "$waiter" || ng "chat_inbox with wait_seconds failed ($(cat "$tmp/wait-err"))"
out=$(cat "$tmp/wait-out")
case $out in
*'"isError":true'*) ng "chat_inbox with wait_seconds failed: $out" ;;
esac
expect "\"body\":\"waited-$mark\"" "chat_inbox with wait_seconds" "$out"
expect '"waited_seconds"' "chat_inbox with wait_seconds (the call did not wait; is --inbox-wait-max 0?)" "$out"
echo "OK: waiting chat_inbox received a message sent while it waited (probe-a -> probe-b)"

# Two users: run only with the second user's token.
if [ -n "$TOKEN_B" ]; then
	connect_token=$TOKEN_B
	d=$(connect)
	e=$(connect)
	connect_token=$TOKEN_A
	# The second user's probe-b has the name of the first user's probe-b, in the same repository;
	# probe-e works in another repository.
	out=$(call "$d" lounge_join "{\"name\":\"probe-b\",\"agent\":\"chat-probe\",\"origin\":\"https://github.com/example-org/lounge-probe-$mark\"}")
	user_b=$(field user "$out")
	[ -n "$user_b" ] || ng "lounge_join returned no user: $out"
	[ "$user_b" != "$user_a" ] || ng "AGENT_LOUNGE_TOKEN and AGENT_LOUNGE_TOKEN_B belong to the same user ($user_a)"
	expect "\"repository\":\"$repo\"" "lounge_join of the second user" "$out"
	out=$(call "$e" lounge_join "{\"name\":\"probe-e\",\"agent\":\"chat-probe\",\"origin\":\"https://github.com/example-org/lounge-probe-other-$mark\"}")
	call "$d" chat_inbox '{"limit":100}' >/dev/null
	call "$e" chat_inbox '{"limit":100}' >/dev/null
	echo "OK: joined as user $user_b: probe-b ($repo) and probe-e (another repository)"

	# lounge_who of probe-a lists the second user's probe-b with its user, address and the
	# repository scope only, and not probe-e.
	out=$(call "$a" lounge_who '{}')
	other_b=$(object_with "\"address\":\"$user_b/probe-b\"" "$out")
	[ -n "$other_b" ] || ng "lounge_who of probe-a does not list $user_b/probe-b: $out"
	for f in '"name":"probe-b"' "\"user\":\"$user_b\"" "\"scopes\":[<repository:$repo>]"; do
		expect "$f" "lounge_who ($user_b/probe-b)" "$other_b"
	done
	reject '"name":"probe-e"' "lounge_who of probe-a (another user's session of another repository)" "$out"
	echo "OK: lounge_who lists $user_b/probe-b with the shared repository scope, not $user_b/probe-e"

	# Room and session messages of scope repository reach the second user.
	out=$(call "$a" chat_send "{\"room\":\"lounge\",\"scope\":\"repository\",\"body\":\"two-room-$mark\"}")
	expect "\"$user_b/probe-b\"" "chat_send to the repository room (two users)" "$out"
	expect '"probe-b"' "chat_send to the repository room (two users)" "$out"
	out=$(call "$a" chat_send "{\"to\":\"$user_b/probe-b\",\"scope\":\"repository\",\"body\":\"two-direct-$mark\"}")
	expect "\"recipients\":[\"$user_b/probe-b\"]" "chat_send to a session of another user" "$out"
	expect '"scope":"repository"' "chat_send to a session of another user" "$out"
	expect "\"scope_id\":\"$repo\"" "chat_send to a session of another user" "$out"
	# Without scope repository, another user's address is refused; the scope is not inferred.
	out=$(call_error "$a" chat_send "{\"to\":\"$user_b/probe-b\",\"body\":\"x\"}")
	expect 'scope repository' "chat_send to another user's session without scope" "$out"
	# probe-e (another repository) and a missing session give the same error.
	outside=$(call_error "$a" chat_send "{\"to\":\"$user_b/probe-e\",\"scope\":\"repository\",\"body\":\"x\"}" | sed "s|$user_b/probe-e|<to>|g")
	missing=$(call_error "$a" chat_send "{\"to\":\"$user_b/probe-none\",\"scope\":\"repository\",\"body\":\"x\"}" | sed "s|$user_b/probe-none|<to>|g")
	[ "$outside" = "$missing" ] || ng "another repository's session and a missing session give different errors: $outside / $missing"
	echo "OK: room and session messages of scope repository sent to $user_b/probe-b; $user_b/probe-e is not addressable"

	out=$(call "$d" chat_inbox '{"limit":100}')
	for body in "two-room-$mark" "two-direct-$mark"; do
		msg=$(object_with "\"body\":\"$body\"" "$out")
		[ -n "$msg" ] || ng "chat_inbox of the second user: no message $body in: $out"
		for f in "\"from_user\":\"$user_a\"" '"from":"probe-a"' '"scope":"repository"' "\"scope_id\":\"$repo\"" "\"reply_to\":\"$user_a/probe-a\""; do
			expect "$f" "chat_inbox of $body" "$msg"
		done
	done
	scope=$(field scope "$msg")
	reply_to=$(field reply_to "$msg")
	# Reply with the received scope and reply_to.
	out=$(call "$d" chat_send "{\"to\":\"$reply_to\",\"scope\":\"$scope\",\"body\":\"two-reply-$mark\"}")
	expect "\"recipients\":[\"$user_a/probe-a\"]" "reply with scope and reply_to" "$out"
	out=$(call "$a" chat_inbox '{"limit":100}')
	msg=$(object_with "\"body\":\"two-reply-$mark\"" "$out")
	[ -n "$msg" ] || ng "chat_inbox of probe-a: no reply in: $out"
	for f in "\"from_user\":\"$user_b\"" "\"reply_to\":\"$user_b/probe-b\"" '"scope":"repository"'; do
		expect "$f" "chat_inbox of the reply" "$msg"
	done
	out=$(call "$e" chat_inbox '{"limit":100}')
	reject "two-" "chat_inbox of another repository's session of the second user" "$out"
	out=$(call_error "$e" chat_send "{\"to\":\"$user_a/probe-a\",\"scope\":\"repository\",\"body\":\"x\"}")
	expect 'no session' "chat_send from another repository to the first user" "$out"
	call "$b" chat_inbox '{"limit":100}' >/dev/null # the repository room message reached probe-b too
	echo "OK: $user_b/probe-b received both with from_user and reply_to, and its reply reached probe-a; $user_b/probe-e received nothing"
else
	echo "OK: checks across two users skipped (set AGENT_LOUNGE_TOKEN_B to run them)"
fi

# End the probe's MCP sessions.
for id in "$a" "$b" "$c" "$d" "$e"; do
	[ "$id" != none ] || continue
	tok=$(token_of "$id")
	curl -fsS --max-time "$WAIT" -X DELETE ${tok:+-H "Authorization: Bearer $tok"} -H "Mcp-Session-Id: $id" "$BASE/mcp" >/dev/null 2>&1 || true
done
echo "OK: chat works at $BASE/mcp"
