"""What every scenario uses: the run context, instructions, and judging replies."""

from __future__ import annotations

import os
import platform
import subprocess
import threading
import time
from dataclasses import dataclass, field
from pathlib import Path
from typing import Any

from .. import screen
from ..agents import Agent, Definition, Reply, clean_environment, command
from ..report import Result
from ..server import Server


@dataclass
class Context:
    """One scenario for one CLI."""

    definition: Definition
    base: Path  # the check folder of this run
    binary: str  # the agent-lounge-mcp executable
    listen: str
    result: Result
    test_binary: str = ""  # executable built with a passphrase in its instructions (scenario 4)
    test_passphrase: str = ""
    secrets: list[str] = field(default_factory=list)  # tokens that must not reach the result files
    _servers: list[Server] = field(default_factory=list)
    _agents: list[Agent] = field(default_factory=list)

    @property
    def session(self) -> str:
        return f"iv-{self.definition.name}"

    @property
    def url(self) -> str:
        return f"http://{self.listen}/mcp"

    def folder(self, name: str) -> Path:
        path = self.base / self.result.scenario / self.definition.name / name
        path.mkdir(parents=True, exist_ok=True)
        return path

    def server(self, name: str = "server", binary: str = "", extra: tuple[str, ...] = ()) -> Server:
        folder = self.folder(name)
        s = Server(binary or self.binary, db=str(folder / "lounge.db"), listen=self.listen, extra=extra,
                   log_path=str(folder / "server.log"))
        self._servers.append(s)
        return s

    def agent(self, cwd: Path) -> Agent:
        a = Agent(self.definition, str(cwd))
        self._agents.append(a)
        return a

    def save(self, name: str, text: str) -> None:
        """Keep a transcript next to the result."""
        path = self.base / self.result.scenario / self.definition.name / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(self.scrub(text), encoding="utf-8")

    def scrub(self, text: str) -> str:
        for secret in self.secrets:
            text = text.replace(secret, "<TOKEN>")
        return text

    def close(self) -> None:
        for a in self._agents:
            try:
                a.quit()
            except Exception:
                pass
        for s in self._servers:
            try:
                s.stop()
            except Exception:
                pass  # stop the other servers too


def join_instruction(session: str) -> str:
    return (f'Use the agent-lounge MCP server: call lounge_join with the name "{session}" and no profile '
            f"values, then tell me the session name you joined as.")


def new_lines(before: str, after: str) -> str:
    """Lines of after that before does not have: what a reply added to the screen."""
    old = set(before.split("\n"))
    return "\n".join(line for line in after.split("\n") if line not in old)


def ask(agent: Agent, instruction: str, timeout: float = 180) -> tuple[Reply, str]:
    """Ask, and return the reply with only what it added (new screen lines and new output)."""
    assert agent.term
    before = agent.term.transcript()
    start = agent.term.mark()
    reply = agent.ask(instruction, timeout=timeout)
    added = new_lines(before, agent.term.transcript()) + "\n" + agent.term.text(start)
    return reply, added


def answer(text: str, yes: str, no: str) -> str:
    """Judge a reply that was asked for one of two words in capital letters."""
    has_yes, has_no = screen.contains(text, yes), screen.contains(text, no)
    if has_yes and not has_no:
        return "yes"
    if has_no and not has_yes:
        return "no"
    return "undetermined"


def connections(server: Server) -> dict[int, dict[str, Any]]:
    """What the diagnostics say about each connection: client and tools."""
    out: dict[int, dict[str, Any]] = {}
    for e in server.events():
        c = out.setdefault(e.connection, {"client": "", "tools": []})
        if e.text.startswith("initialized "):
            c["client"] = e.text[len("initialized "):]
        elif e.text.startswith("request tools/call "):
            c["tools"].append(e.text[len("request tools/call "):])
    return out


def environment(definition: Definition, binary: str) -> dict[str, str]:
    def version(argv: list[str]) -> str:
        try:
            out = subprocess.run(argv, capture_output=True, text=True, timeout=60, env=clean_environment())
            return (out.stdout or out.stderr).strip().splitlines()[0]
        except (OSError, subprocess.TimeoutExpired, IndexError):
            return ""
    return {
        "os": f"{platform.system()} {platform.release()} {platform.machine()}" + (" (WSL)" if "microsoft" in platform.release().lower() else ""),
        "python": platform.python_version(),
        "cli": version([command(definition.executable), *definition.version]),
        "server": version([binary, "version"]),
    }


class FileWatch:
    """Notes the time (time.monotonic) a file first exists."""

    def __init__(self, path: Path):
        self.path = path
        self.seen: float | None = None
        self._stop = threading.Event()
        self._thread = threading.Thread(target=self._run, daemon=True)
        self._thread.start()

    def _run(self) -> None:
        while not self._stop.is_set():
            if self.seen is None and self.path.exists():
                self.seen = time.monotonic()
            time.sleep(0.1)

    def stop(self) -> None:
        self._stop.set()
        self._thread.join()


def is_wsl() -> bool:
    return os.name != "nt" and "microsoft" in platform.release().lower()
