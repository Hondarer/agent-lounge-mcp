"""Starting and stopping an agent-lounge-mcp server for a check, and reading its diagnostics."""

from __future__ import annotations

import json
import re
import subprocess
import threading
import time
import urllib.request
from dataclasses import dataclass, field
from pathlib import Path
from typing import Sequence

from . import terminal

# A line of --log-connections, after the prefix the server writes before it.
_CONNECTION = re.compile(r"connection (\d+): (.*)$")


@dataclass
class Event:
    """A connection diagnostic line, with the time the harness read it (time.monotonic)."""

    at: float
    connection: int
    text: str


@dataclass
class Server:
    binary: str
    db: str
    listen: str = "127.0.0.1:7799"
    extra: Sequence[str] = ()
    log_path: str = ""
    _proc: subprocess.Popen | None = None
    _lines: list[tuple[float, str]] = field(default_factory=list)
    _lock: threading.Lock = field(default_factory=threading.Lock)

    @property
    def url(self) -> str:
        return f"http://{self.listen}/mcp"

    def admin(self, *args: str) -> str:
        """Run an administration command (such as user add) on the database; return its output."""
        out = subprocess.run([self.binary, *args, "--db", self.db], capture_output=True, text=True, timeout=60)
        if out.returncode != 0:
            raise RuntimeError(f"{' '.join(args)}: {out.stderr.strip()}")
        return out.stdout

    def start(self, timeout: float = 30) -> "Server":
        # A server left on the address (by a run that was killed) would answer the health check
        # below, and the scenario would drive it instead of the one started here.
        if self.health():
            raise RuntimeError(f"another server already answers at {self.listen}; stop it, or pass --listen")
        argv = [self.binary, "serve", "--listen", self.listen, "--db", self.db, "--log-connections", *self.extra]
        self._proc = subprocess.Popen(argv, stdout=subprocess.DEVNULL, stderr=subprocess.PIPE, text=True,
                                      encoding="utf-8", errors="replace", **terminal.new_process_group())
        threading.Thread(target=self._collect, daemon=True).start()
        deadline = time.monotonic() + timeout
        while time.monotonic() < deadline:
            if self._proc.poll() is not None:
                raise RuntimeError(f"the server ended at start-up:\n{self.log()}")
            if self.health().get("state") == "ok":
                return self
            time.sleep(0.2)
        raise RuntimeError(f"the server did not start:\n{self.log()}")

    def _collect(self) -> None:
        assert self._proc and self._proc.stderr
        sink = open(self.log_path, "a", encoding="utf-8") if self.log_path else None
        for line in self._proc.stderr:
            with self._lock:
                self._lines.append((time.monotonic(), line.rstrip("\n")))
            if sink:
                sink.write(line)
                sink.flush()
        if sink:
            sink.close()

    def health(self) -> dict:
        try:
            with urllib.request.urlopen(f"http://{self.listen}/healthz", timeout=5) as res:
                return json.loads(res.read())
        except (OSError, ValueError):
            return {}

    def log(self) -> str:
        with self._lock:
            return "\n".join(line for _, line in self._lines)

    def events(self, since: float = 0) -> list[Event]:
        """The connection diagnostics read at or after since (a time.monotonic value)."""
        with self._lock:
            lines = list(self._lines)
        out = []
        for at, line in lines:
            m = _CONNECTION.search(line)
            if m and at >= since:
                out.append(Event(at, int(m.group(1)), m.group(2)))
        return out

    def tool_calls(self, tool: str, since: float = 0) -> list[Event]:
        return [e for e in self.events(since) if e.text == f"request tools/call {tool}"]

    def interrupt(self) -> None:
        """Ask the server to stop, as Ctrl+C would. It keeps running for its grace period."""
        if self._proc and self._proc.poll() is None:
            terminal.interrupt_process(self._proc)

    def wait(self, timeout: float = 30) -> int | None:
        if not self._proc:
            return None
        try:
            return self._proc.wait(timeout=timeout)
        except subprocess.TimeoutExpired:
            return None

    def stop(self) -> None:
        if not self._proc or self._proc.poll() is not None:
            return
        try:
            self.interrupt()
        except OSError:
            pass  # go on to end the process
        if self.wait(15) is None:
            self._proc.kill()
            self._proc.wait()


def data_dir(base: str | Path, name: str) -> Path:
    path = Path(base) / name
    path.mkdir(parents=True, exist_ok=True)
    return path
