#!/bin/sh
# Verify that connections to a running agent-lounge-mcp server can talk: they join with profiles,
# one sends a message to another's session and one to the lounge room of the user, and the other
# receives both. Then the repository space: probe-a and probe-b declare one repository in two
# forms of its origin, probe-c declares a folder without origin; a message to the repository room
# reaches probe-b and not probe-c. Last, one connection waits in chat_inbox (wait_seconds) and
# receives a message sent while it waits; the probe sends only after /healthz reports the wait.
#
# usage: chat-probe.sh [BASE_URL]      (default: http://127.0.0.1:7788)
# Prints "OK: ..." lines and exits 0, or prints "NG: ..." to standard error and exits 1.
#
# The probe joins as the sessions probe-a, probe-b and probe-c, which stay in the database with
# made-up profile values, and its user room message is queued for every other session of the
# user too. Run it against a database used for checks.
set -eu

BASE=${1:-http://127.0.0.1:7788}
BASE=${BASE%/}
WAIT=${PROBE_WAIT:-5}

ng() {
	echo "NG: $*" >&2
	exit 1
}

command -v curl >/dev/null 2>&1 || ng "curl is required"

tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# post SESSION_ID BODY: POST one JSON-RPC message; the response body goes to standard output.
post() {
	curl -fsS --max-time "$WAIT" -D "$tmp/headers" \
		-H 'Content-Type: application/json' \
		-H 'Accept: application/json, text/event-stream' \
		${1:+-H "Mcp-Session-Id: $1"} \
		--data "$2" "$BASE/mcp" 2>"$tmp/err" || ng "POST $BASE/mcp failed ($(cat "$tmp/err")). Is the server running?"
}

# connect: open an MCP session and print its ID.
connect() {
	post "" '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"chat-probe","version":"0"}}}' >/dev/null
	id=$(tr -d '\r' <"$tmp/headers" | awk -F': ' 'tolower($1) == "mcp-session-id" { print $2 }')
	[ -n "$id" ] || ng "the server returned no Mcp-Session-Id header"
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

# waiting: print the number of waiting chat_inbox calls reported by /healthz.
waiting() {
	curl -fsS --max-time "$WAIT" "$BASE/healthz" 2>"$tmp/err" | sed -n 's/.*"waiting":\([0-9][0-9]*\).*/\1/p'
}

token=$(date +%s)-$$
a=$(connect)
b=$(connect)
c=$(connect)

# Tools other than lounge_join refuse a connection that has not joined.
out=$(post "$a" '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"chat_inbox","arguments":{}}}')
expect 'not joined' "chat_inbox before lounge_join" "$out"

# Profiles: values are passed as the commands print them (trailing newlines, a WSL path, an origin
# with credentials) and come back normalized. probe-a and probe-b give one origin in two forms.
repo="github.com/example-org/lounge-probe-$token"
out=$(call "$a" lounge_join "{\"name\":\"probe-a\",\"agent\":\"chat-probe\",\"os_user\":\"probe-user\\r\\n\",\"machine\":\"PROBE-HOST\\r\\n\",\"origin\":\"https://probe:SECRET@GitHub.com/Example-Org/Lounge-Probe-$token.git\\n\",\"folder\":\"/mnt/c/Probe/A\\n\",\"wsl_distro\":\"Ubuntu\\n\"}")
expect '"os_user":"probe-user"' "lounge_join with a profile" "$out"
expect '"machine":"probe-host"' "lounge_join with a profile" "$out"
expect "\"origin\":\"$repo\"" "lounge_join with a profile" "$out"
expect '"folder":"c:\\probe\\a"' "lounge_join with a profile" "$out"
expect "\"repository\":\"$repo\"" "lounge_join with a profile" "$out"
reject 'SECRET' "lounge_join with a credentialed origin" "$out"
out=$(call "$b" lounge_join "{\"name\":\"probe-b\",\"agent\":\"chat-probe\",\"os_user\":\"probe-user\",\"machine\":\"probe-host\",\"origin\":\"git@github.com:example-org/lounge-probe-$token.git\",\"folder\":\"/home/probe/b\",\"wsl_distro\":\"Ubuntu\"}")
expect "\"repository\":\"$repo\"" "lounge_join with the scp-like origin" "$out"
expect '"folder":"\\\\wsl.localhost\\ubuntu\\home\\probe\\b"' "lounge_join with a WSL folder" "$out"
# probe-c: a folder without origin; its repository is machine:folder.
out=$(call "$c" lounge_join "{\"name\":\"probe-c\",\"agent\":\"chat-probe\",\"machine\":\"probe-host\",\"origin\":\"\",\"folder\":\"/home/probe/c-$token\",\"wsl_distro\":\"Ubuntu\"}")
expect '"repository":"probe-host:\\\\wsl.localhost\\ubuntu\\home\\probe\\c-'"$token"'"' "lounge_join with a folder without origin" "$out"
# A folder without origin needs a machine.
out=$(post "$c" '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"lounge_join","arguments":{"name":"probe-c","machine":"","origin":"","folder":"/home/probe/c"}}}')
expect '"isError":true' "lounge_join with a folder and no machine" "$out"
expect 'machine' "lounge_join with a folder and no machine" "$out"
call "$b" chat_inbox '{"limit":100}' >/dev/null # drop what earlier runs left unread
call "$c" chat_inbox '{"limit":100}' >/dev/null
echo "OK: joined as probe-a, probe-b ($repo) and probe-c (folder without origin); profiles normalized"

out=$(call "$a" lounge_who '{}')
expect '"name":"probe-b"' "lounge_who" "$out"
# The order of keys is not fixed: check each session object, and the rest of the result without them.
expect "\"repository\":\"$repo\"" "lounge_who (own repository)" "$(printf '%s' "$out" | sed 's/{[^{}]*}//g')"
session_b=$(printf '%s' "$out" | grep -o '{[^{}]*"name":"probe-b"[^{}]*}' | head -n 1)
for field in '"os_user":"probe-user"' '"machine":"probe-host"' "\"repository\":\"$repo\"" '"same_repository":true'; do
	expect "$field" "lounge_who (probe-b)" "$session_b"
done
session_c=$(printf '%s' "$out" | grep -o '{[^{}]*"name":"probe-c"[^{}]*}' | head -n 1)
for field in '"machine":"probe-host"' '"repository":"probe-host:' '"same_repository":false'; do
	expect "$field" "lounge_who (probe-c)" "$session_c"
done

out=$(call "$a" chat_send "{\"to\":\"probe-b\",\"body\":\"direct-$token\"}")
expect '"recipients":["probe-b"]' "chat_send to a session" "$out"
out=$(call "$b" lounge_info '{}')
expect '"unread":1' "unread count after a message" "$out"
out=$(call "$b" chat_inbox '{}')
expect "\"body\":\"direct-$token\"" "chat_inbox" "$out"
expect '"from":"probe-a"' "chat_inbox" "$out"
expect '"unread":0' "unread count after reading" "$out"
echo "OK: message to a session delivered (probe-a -> probe-b)"

out=$(call "$b" chat_send "{\"room\":\"lounge\",\"body\":\"room-$token\"}")
expect '"probe-a"' "chat_send to the room" "$out"
out=$(call "$a" chat_inbox '{"limit":100}')
expect "\"body\":\"room-$token\"" "chat_inbox" "$out"
expect '"room":"lounge"' "chat_inbox" "$out"
echo "OK: message to the lounge room delivered (probe-b -> probe-a)"
call "$c" chat_inbox '{"limit":100}' >/dev/null # the user room message reached probe-c too

# Repository space: probe-a's message reaches probe-b (same repository) and not probe-c.
out=$(call "$a" chat_send "{\"room\":\"lounge\",\"scope\":\"repository\",\"body\":\"repo-$token\"}")
expect '"recipients":["probe-b"]' "chat_send to the repository room" "$out"
expect '"scope":"repository"' "chat_send to the repository room" "$out"
expect "\"repository\":\"$repo\"" "chat_send to the repository room" "$out"
out=$(call "$b" chat_inbox '{"limit":100}')
expect "\"body\":\"repo-$token\"" "chat_inbox of the repository room" "$out"
expect '"scope":"repository"' "chat_inbox of the repository room" "$out"
expect "\"repository\":\"$repo\"" "chat_inbox of the repository room" "$out"
expect '"from_os_user":"probe-user"' "chat_inbox of the repository room" "$out"
expect '"from_machine":"probe-host"' "chat_inbox of the repository room" "$out"
out=$(call "$c" chat_inbox '{"limit":100}')
reject "repo-$token" "chat_inbox of another repository" "$out"
echo "OK: message to the repository room delivered (probe-a -> probe-b), not to another repository (probe-c)"

out=$(call "$c" chat_send "{\"room\":\"lounge\",\"scope\":\"repository\",\"body\":\"folder-$token\"}")
expect '"recipients":[]' "chat_send to the room of a folder without origin" "$out"
out=$(post "$c" '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"chat_send","arguments":{"to":"probe-a","scope":"repository","body":"x"}}}')
expect '"isError":true' "chat_send with to and scope repository" "$out"
out=$(call "$a" chat_inbox '{"limit":100}')
reject "folder-$token" "chat_inbox of another repository" "$out"
echo "OK: probe-c's repository room reaches no other probe session; scope repository with to is refused"

# probe-b waits in chat_inbox in the background; probe-a sends while it waits. The waiting call
# uses its own headers file and a longer timeout than the wait.
wait_seconds=10
before=$(waiting)
[ -n "$before" ] || ng "/healthz reports no waiting field (an older server?)"
curl -fsS --max-time $((WAIT + wait_seconds)) -D "$tmp/wait-headers" \
	-H 'Content-Type: application/json' \
	-H 'Accept: application/json, text/event-stream' \
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
call "$a" chat_send "{\"to\":\"probe-b\",\"body\":\"waited-$token\"}" >/dev/null
wait "$waiter" || ng "chat_inbox with wait_seconds failed ($(cat "$tmp/wait-err"))"
out=$(cat "$tmp/wait-out")
case $out in
*'"isError":true'*) ng "chat_inbox with wait_seconds failed: $out" ;;
esac
expect "\"body\":\"waited-$token\"" "chat_inbox with wait_seconds" "$out"
expect '"waited_seconds"' "chat_inbox with wait_seconds (the call did not wait; is --inbox-wait-max 0?)" "$out"
echo "OK: waiting chat_inbox received a message sent while it waited (probe-a -> probe-b)"

# End the probe's MCP sessions.
for id in "$a" "$b" "$c"; do
	curl -fsS --max-time "$WAIT" -X DELETE -H "Mcp-Session-Id: $id" "$BASE/mcp" >/dev/null 2>&1 || true
done
echo "OK: chat works at $BASE/mcp"
