#!/bin/sh
# Verify that an agent-lounge-mcp server answers: the liveness check (with the schema version
# and the number of waiting calls), then MCP initialize and tools/list over Streamable HTTP,
# including the profile arguments of lounge_join and the scope argument of chat_send.
#
# usage: probe.sh [BASE_URL]      (default: http://127.0.0.1:7788)
# Prints "OK: ..." lines and exits 0, or prints "NG: ..." to standard error and exits 1.
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

health=$(curl -fsS --max-time "$WAIT" "$BASE/healthz" 2>"$tmp/err") ||
	ng "$BASE/healthz did not answer ($(cat "$tmp/err")). Is the server running?"
case $health in
*'"state":"ok"'*) echo "OK: health $health" ;;
*'"state":"stopping"'*) ng "the server is stopping: $health" ;;
*) ng "unexpected health response: $health" ;;
esac
# This version of the scripts expects schema version 2 or later (profiles, repository space).
schema=$(printf '%s' "$health" | sed -n 's/.*"schema_version":\([0-9][0-9]*\).*/\1/p')
[ -n "$schema" ] && [ "$schema" -ge 2 ] || ng "schema version 2 or later expected (an older server?): $health"
case $health in
*'"waiting":'*) ;;
*) ng "the health response has no waiting field (an older server?): $health" ;;
esac

# rpc BODY: POST one JSON-RPC message; the response body goes to standard output.
rpc() {
	curl -fsS --max-time "$WAIT" -D "$tmp/headers" \
		-H 'Content-Type: application/json' \
		-H 'Accept: application/json, text/event-stream' \
		${session:+-H "Mcp-Session-Id: $session"} \
		--data "$1" "$BASE/mcp" 2>"$tmp/err" || ng "POST $BASE/mcp failed ($(cat "$tmp/err"))"
}

session=
init=$(rpc '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"probe","version":"0"}}}')
case $init in
*'"serverInfo"'*) ;;
*) ng "unexpected initialize response: $init" ;;
esac
session=$(tr -d '\r' <"$tmp/headers" | awk -F': ' 'tolower($1) == "mcp-session-id" { print $2 }')
[ -n "$session" ] || ng "the server returned no Mcp-Session-Id header"

rpc '{"jsonrpc":"2.0","method":"notifications/initialized"}' >/dev/null
tools=$(rpc '{"jsonrpc":"2.0","id":2,"method":"tools/list"}')
for tool in lounge_help lounge_join lounge_who chat_send chat_inbox lounge_info; do
	case $tools in
	*"\"name\":\"$tool\""*) ;;
	*) ng "tools/list does not contain $tool: $tools" ;;
	esac
done
names=$(printf '%s' "$tools" | grep -o '"name":"[^"]*"' | cut -d'"' -f4 | tr '\n' ' ')
echo "OK: tools ${names% }"
for arg in os_user machine origin folder wsl_distro scope; do
	case $tools in
	*"\"$arg\":{"*) ;;
	*) ng "tools/list does not contain the argument $arg (an older server?)" ;;
	esac
done
echo "OK: profile arguments of lounge_join and scope of chat_send"

# End the probe's MCP session.
curl -fsS --max-time "$WAIT" -X DELETE -H "Mcp-Session-Id: $session" "$BASE/mcp" >/dev/null 2>&1 || true
echo "OK: MCP endpoint $BASE/mcp"
