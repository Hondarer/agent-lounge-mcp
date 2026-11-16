#!/bin/sh
# Verify that two connections to a running agent-lounge-mcp server can talk: both join, one
# sends a message to the other's session and one to the lounge room, and the other receives both.
# Then one connection waits in chat_inbox (wait_seconds) and receives a message sent while it waits.
#
# usage: chat-probe.sh [BASE_URL]      (default: http://127.0.0.1:7788)
# Prints "OK: ..." lines and exits 0, or prints "NG: ..." to standard error and exits 1.
#
# The probe joins as the sessions probe-a and probe-b, which stay in the database, and its room
# message is queued for every other session too. Run it against a database used for checks.
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

token=$(date +%s)-$$
a=$(connect)
b=$(connect)

# Tools other than lounge_join refuse a connection that has not joined.
out=$(post "$a" '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"chat_inbox","arguments":{}}}')
expect 'not joined' "chat_inbox before lounge_join" "$out"

call "$a" lounge_join '{"name":"probe-a","agent":"chat-probe"}' >/dev/null
call "$b" lounge_join '{"name":"probe-b","agent":"chat-probe"}' >/dev/null
call "$b" chat_inbox '{"limit":100}' >/dev/null # drop what earlier runs left unread
echo "OK: joined as probe-a and probe-b"

out=$(call "$a" lounge_who '{}')
expect '"name":"probe-b"' "lounge_who" "$out"

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

# probe-b waits in chat_inbox in the background; probe-a sends while it waits. The waiting call
# uses its own headers file and a longer timeout than the wait.
wait_seconds=10
curl -fsS --max-time $((WAIT + wait_seconds)) -D "$tmp/wait-headers" \
	-H 'Content-Type: application/json' \
	-H 'Accept: application/json, text/event-stream' \
	-H "Mcp-Session-Id: $b" \
	--data "{\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"tools/call\",\"params\":{\"name\":\"chat_inbox\",\"arguments\":{\"wait_seconds\":$wait_seconds}}}" \
	"$BASE/mcp" >"$tmp/wait-out" 2>"$tmp/wait-err" &
waiter=$!
sleep 1 # let the call start waiting
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
for id in "$a" "$b"; do
	curl -fsS --max-time "$WAIT" -X DELETE -H "Mcp-Session-Id: $id" "$BASE/mcp" >/dev/null 2>&1 || true
done
echo "OK: chat works at $BASE/mcp"
