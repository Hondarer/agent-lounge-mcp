"""A small MCP client over Streamable HTTP, for the harness's own sessions (the peer of an agent).

It speaks the protocol version with the initialize handshake, which every server version of
agent-lounge-mcp accepts, and uses only the standard library.
"""

from __future__ import annotations

import json
import urllib.error
import urllib.request
from typing import Any

PROTOCOL = "2025-06-18"


class ToolError(Exception):
    """A tool returned an error result."""


class Client:
    def __init__(self, url: str, token: str = "", timeout: float = 60):
        self.url = url
        self.token = token
        self.timeout = timeout
        self.session = ""
        self._id = 0

    def _post(self, message: dict) -> dict | None:
        headers = {"Content-Type": "application/json", "Accept": "application/json, text/event-stream"}
        if self.session:
            headers["Mcp-Session-Id"] = self.session
            headers["Mcp-Protocol-Version"] = PROTOCOL
        if self.token:
            headers["Authorization"] = "Bearer " + self.token
        req = urllib.request.Request(self.url, data=json.dumps(message).encode(), headers=headers, method="POST")
        with urllib.request.urlopen(req, timeout=self.timeout) as res:
            if not self.session:
                self.session = res.headers.get("Mcp-Session-Id", "")
            body = res.read().decode("utf-8")
            kind = res.headers.get("Content-Type", "")
        if "id" not in message:
            return None
        if kind.startswith("text/event-stream"):
            for line in body.splitlines():
                if line.startswith("data:"):
                    data = json.loads(line[5:])
                    if data.get("id") == message["id"]:
                        return data
            raise ValueError("no response in the event stream")
        return json.loads(body)

    def _request(self, method: str, params: dict) -> Any:
        self._id += 1
        res = self._post({"jsonrpc": "2.0", "id": self._id, "method": method, "params": params})
        if res is None or "error" in res:
            raise ValueError(f"{method}: {res}")
        return res["result"]

    def connect(self) -> "Client":
        self._request("initialize", {"protocolVersion": PROTOCOL, "capabilities": {},
                                     "clientInfo": {"name": "interactive-verification", "version": "1"}})
        self._post({"jsonrpc": "2.0", "method": "notifications/initialized"})
        return self

    def call(self, tool: str, arguments: dict) -> dict:
        """Call a tool and return its structured result; raise ToolError on a tool error."""
        result = self._request("tools/call", {"name": tool, "arguments": arguments})
        text = "".join(c.get("text", "") for c in result.get("content", []) if c.get("type") == "text")
        if result.get("isError"):
            raise ToolError(text)
        if "structuredContent" in result:
            return result["structuredContent"]
        return json.loads(text) if text.startswith("{") else {"text": text}

    def close(self) -> None:
        if not self.session:
            return
        headers = {"Mcp-Session-Id": self.session}
        if self.token:
            headers["Authorization"] = "Bearer " + self.token
        try:
            urllib.request.urlopen(urllib.request.Request(self.url, headers=headers, method="DELETE"), timeout=10)
        except (urllib.error.URLError, OSError):
            pass
        self.session = ""
