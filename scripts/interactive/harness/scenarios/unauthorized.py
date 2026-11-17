"""Scenario 6: what a CLI tells the user when its token stopped working (HTTP 401).

It runs each CLI's one-shot mode, not the interactive screen: the question is what the agent
reports, and the one-shot output holds all of it.
"""

from __future__ import annotations

import os
import re
import subprocess
import time
from pathlib import Path

from ..agents import clean_environment, command
from .common import Context

USER = "iv-user"
_AUTH_WORDS = re.compile(r"\b401\b|unauthori[sz]ed|authenticat|\btoken\b", re.I)
_OAUTH_WORDS = re.compile(r"oauth|open (?:the|a|your) browser|sign in|log ?in to", re.I)

# Folders under the home folder where the CLIs keep settings, history and logs.
_CLI_HOMES = [".claude", ".claude.json", ".codex", ".copilot", ".grok", ".gemini", ".agy", ".antigravity",
              os.path.join("AppData", "Roaming", "agy"), os.path.join("AppData", "Local", "agy")]


def _one_shot(ctx: Context, token: str, prompt: str, cwd: Path, timeout: float = 180) -> dict:
    env = clean_environment()
    env["AGENT_LOUNGE_TOKEN"] = token  # Codex reads the token from this variable
    began = time.monotonic()
    try:
        out = subprocess.run([command(ctx.definition.executable), *ctx.definition.one_shot(prompt)], cwd=cwd, env=env,
                             capture_output=True, text=True, encoding="utf-8", errors="replace", timeout=timeout)
        text, code, timed_out = out.stdout + "\n" + out.stderr, out.returncode, False
    except subprocess.TimeoutExpired as e:
        text = (e.stdout or b"").decode("utf-8", "replace") if isinstance(e.stdout, bytes) else (e.stdout or "")
        code, timed_out = None, True
    return {"text": text.replace(token, "<TOKEN>"), "exit_code": code, "timed_out": timed_out,
            "seconds": round(time.monotonic() - began, 1)}


def _register(ctx: Context, token: str) -> None:
    d = ctx.definition
    env = clean_environment()
    subprocess.run([command(d.executable), *d.unregister], capture_output=True, env=env, timeout=60)
    out = subprocess.run([command(d.executable), *d.register(ctx.url, token)], capture_output=True, text=True, env=env, timeout=60)
    if out.returncode != 0:
        raise RuntimeError(f"registering with a token failed: {out.stderr.strip()}")


def _leftovers(token: str) -> list[str]:
    """Files under the CLI folders of the home folder that still hold token."""
    home = Path.home()
    found = []
    for name in _CLI_HOMES:
        root = home / name
        paths = [root] if root.is_file() else (root.rglob("*") if root.is_dir() else [])
        for path in paths:
            try:
                if path.is_file() and path.stat().st_size < 20_000_000 and token.encode() in path.read_bytes():
                    found.append("<home>/" + path.relative_to(home).as_posix())
            except OSError:
                continue
    return found


def token_stopped(ctx: Context) -> None:
    server = ctx.server(extra=("--require-token",))
    token = server.admin("user", "add", USER).strip()
    server.start()
    folder = ctx.folder("work")
    v = ctx.result.values
    try:
        _register(ctx, token)
        prompt = (f'Use the agent-lounge MCP server: call lounge_join with the name "{ctx.session}" and no profile '
                  "values, and tell me the result.")
        first = _one_shot(ctx, token, prompt, folder)
        v["joined_before"] = bool(server.tool_calls("lounge_join"))
        ctx.save("before.txt", first["text"])
        server.admin("user", "rotate", USER)
        mark = time.monotonic()
        second = _one_shot(ctx, token, prompt, folder)
        ctx.save("after.txt", second["text"])
        v["joined_after"] = bool(server.tool_calls("lounge_join", mark))
        v["reports_authentication"] = bool(_AUTH_WORDS.search(second["text"]))
        v["mentions_oauth"] = bool(_OAUTH_WORDS.search(second["text"]))
        v["timed_out"] = second["timed_out"]
        v["seconds"] = second["seconds"]
        v["exit_code"] = second["exit_code"]
    finally:
        d = ctx.definition
        subprocess.run([command(d.executable), *d.unregister], capture_output=True, env=clean_environment(), timeout=60)
        v["token_left_in"] = _leftovers(token)
