#!/bin/sh
# Verify that an agent-lounge-mcp server answers: the liveness check (with the schema version
# and the number of waiting calls), then MCP initialize and tools/list over Streamable HTTP,
# including the profile arguments of lounge_join, the scope argument of chat_send, and the
# outputs of scopes and addresses (scopes, address, scope_id, reply_to, from_user). It checks
# that all fourteen tools are listed, including chat_ack, chat_status, chat_history and the five
# memory tools.
#
# usage: probe.sh [BASE_URL]      (default: http://127.0.0.1:7788)
# The token of a user, when the server requires one or to probe as that user, is read from the
# environment variable AGENT_LOUNGE_TOKEN and sent as "Authorization: Bearer <token>". Without
# it, the probe sends no token (the user local, on a server that does not require a token).
# The token is never printed.
# Prints "OK: ..." lines and exits 0, or prints "NG: ..." to standard error and exits 1.
set -eu

BASE=${1:-http://127.0.0.1:7788}
BASE=${BASE%/}
WAIT=${PROBE_WAIT:-5}
TOKEN=${AGENT_LOUNGE_TOKEN:-}

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
# This version of the scripts expects schema version 3 or later (users, scopes with ids).
schema=$(printf '%s' "$health" | sed -n 's/.*"schema_version":\([0-9][0-9]*\).*/\1/p')
[ -n "$schema" ] && [ "$schema" -ge 3 ] || ng "schema version 3 or later expected (an older server?): $health"
case $health in
*'"waiting":'*) ;;
*) ng "the health response has no waiting field (an older server?): $health" ;;
esac

# rpc BODY: POST one JSON-RPC message; the response body goes to standard output.
rpc() {
	code=$(curl -sS --max-time "$WAIT" -D "$tmp/headers" -o "$tmp/body" -w '%{http_code}' \
		-H 'Content-Type: application/json' \
		-H 'Accept: application/json, text/event-stream' \
		${TOKEN:+-H "Authorization: Bearer $TOKEN"} \
		${session:+-H "Mcp-Session-Id: $session"} \
		--data "$1" "$BASE/mcp" 2>"$tmp/err") || ng "POST $BASE/mcp failed ($(cat "$tmp/err"))"
	case $code in
	2??) cat "$tmp/body" ;;
	401)
		if [ -n "$TOKEN" ]; then
			ng "the server rejected the token in AGENT_LOUNGE_TOKEN (401: $(cat "$tmp/body"))"
		else
			ng "the server requires a token: set AGENT_LOUNGE_TOKEN (401: $(cat "$tmp/body"))"
		fi
		;;
	*) ng "POST $BASE/mcp returned HTTP $code: $(cat "$tmp/body")" ;;
	esac
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
for tool in lounge_help lounge_join lounge_who chat_send chat_inbox chat_ack chat_status chat_history \
	memory_put memory_search memory_get memory_pin memory_forget lounge_info; do
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
# Output fields of scopes and addresses (in the output schemas of lounge_who, chat_send and chat_inbox).
for field in scopes address scope_id reply_to from_user skipped; do
	case $tools in
	*"\"$field\":{"*) ;;
	*) ng "tools/list does not contain the output field $field (an older server?)" ;;
	esac
done
case $tools in
*'"same_repository":{'*) ng "tools/list still has same_repository (an older server?)" ;;
esac
echo "OK: output fields of scopes and addresses"

# End the probe's MCP session.
curl -fsS --max-time "$WAIT" -X DELETE ${TOKEN:+-H "Authorization: Bearer $TOKEN"} -H "Mcp-Session-Id: $session" "$BASE/mcp" >/dev/null 2>&1 || true
if [ -n "$TOKEN" ]; then
	echo "OK: MCP endpoint $BASE/mcp (with the token of AGENT_LOUNGE_TOKEN)"
else
	echo "OK: MCP endpoint $BASE/mcp (without a token)"
fi
