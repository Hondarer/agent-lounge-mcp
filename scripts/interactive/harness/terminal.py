"""A program running in a pseudo-terminal, with its output collected in the background.

This is the only module that knows which operating system it runs on: open_terminal picks
the implementation (terminal_posix on Linux and WSL, terminal_windows on Windows). The
implementations provide the five primitives of Terminal; everything else (collecting output,
waiting for text, typing, stopping) is shared here.
"""

from __future__ import annotations

import os
import threading
import time
from typing import Mapping, Sequence

from . import screen

COLUMNS = 160
ROWS = 50


class TerminalClosed(Exception):
    """The program ended or closed its terminal."""


class Terminal:
    """A program in a pseudo-terminal. Use open_terminal to create one."""

    def __init__(self) -> None:
        self._lock = threading.Condition()
        self._raw: list[str] = []
        self._closed = False
        self._reader: threading.Thread | None = None
        self._screen = screen.Screen(COLUMNS, ROWS)

    # Primitives of each operating system.

    def _spawn(self, argv: Sequence[str], cwd: str, env: Mapping[str, str], columns: int, rows: int) -> None:
        raise NotImplementedError

    def _read_chunk(self) -> str:
        """Block until output is available and return it; return "" at the end of output."""
        raise NotImplementedError

    def _write(self, data: str) -> None:
        raise NotImplementedError

    def alive(self) -> bool:
        raise NotImplementedError

    def _kill(self) -> None:
        """End the program and release the terminal."""
        raise NotImplementedError

    # Shared behaviour.

    def start(self, argv: Sequence[str], cwd: str, env: Mapping[str, str] | None = None,
              columns: int = COLUMNS, rows: int = ROWS, inherit: bool = True) -> "Terminal":
        """Start argv in cwd. env is added to this process's environment, or replaces it when
        inherit is False."""
        merged = dict(os.environ) if inherit else {}
        merged.update(env or {})
        merged.setdefault("TERM", "xterm-256color")
        merged.setdefault("COLUMNS", str(columns))
        merged.setdefault("LINES", str(rows))
        self._screen = screen.Screen(columns, rows)
        self._spawn(argv, cwd, merged, columns, rows)
        self._reader = threading.Thread(target=self._collect, daemon=True)
        self._reader.start()
        return self

    def _collect(self) -> None:
        while True:
            try:
                chunk = self._read_chunk()
            except (OSError, EOFError):
                chunk = ""
            with self._lock:
                if not chunk:
                    self._closed = True
                    self._lock.notify_all()
                    return
                self._raw.append(chunk)
                self._screen.feed(chunk)
                self._lock.notify_all()

    def raw(self) -> str:
        """All output so far, with control sequences."""
        with self._lock:
            return "".join(self._raw)

    def text(self, start: int = 0) -> str:
        """All output so far from offset start of raw(), as plain text (see screen.plain)."""
        return screen.plain(self.raw()[start:])

    def display(self) -> str:
        """The screen as it looks now."""
        with self._lock:
            return self._screen.display()

    def transcript(self) -> str:
        """The lines that scrolled off the screen, followed by the screen as it looks now."""
        with self._lock:
            return self._screen.transcript()

    def mark(self) -> int:
        """The current end of the output, to pass as start to text and wait_for."""
        with self._lock:
            return sum(len(c) for c in self._raw)

    def closed(self) -> bool:
        with self._lock:
            return self._closed

    def wait_for(self, needles: Sequence[str], timeout: float, start: int = 0) -> str | None:
        """Wait until one of needles shows (see screen.contains), on the screen or its history,
        or in the plain output after start.

        Returns the needle found, or None when the timeout passed or the program ended first.
        """
        deadline = time.monotonic() + timeout
        with self._lock:
            while True:
                texts = (self._screen.transcript(), screen.plain("".join(self._raw)[start:]))
                for needle in needles:
                    if any(screen.contains(text, needle) for text in texts):
                        return needle
                remaining = deadline - time.monotonic()
                if remaining <= 0 or self._closed:
                    return None
                self._lock.wait(min(remaining, 0.5))

    def wait_quiet(self, quiet: float, timeout: float) -> bool:
        """Wait until no letters or digits were written for quiet seconds. Returns False at the
        timeout.

        Output of only symbols does not count: some programs animate a logo or a spinner while
        they wait for input, and would otherwise never be quiet. Work in progress shows letters
        or digits (a status word, a timer, the reply).
        """
        deadline = time.monotonic() + timeout
        last = self.mark()
        since = time.monotonic()
        while time.monotonic() < deadline:
            time.sleep(0.2)
            now = self.mark()
            if now != last:
                # Start a little earlier, so that an escape sequence split between two reads
                # is not taken for letters.
                raw = self.raw()
                back = max(0, last - 64)
                if screen.words(raw[back:now]) > screen.words(raw[back:last]):
                    since = time.monotonic()
                last = now
            if time.monotonic() - since >= quiet:
                return True
        return False

    def write(self, data: str) -> None:
        if self.closed():
            raise TerminalClosed()
        self._write(data)

    def type_line(self, text: str, submit: str = "\r", pause: float = 0.5, confirm: float = 3,
                  shown: str = "") -> None:
        """Type text and submit it.

        The pause before submit keeps the program from taking the whole input as one pasted
        block, in which the submit key would become a new line. The pause grows with the
        text, because a console can deliver a long text slowly.

        shown is a part of text that must be on the screen before the submit key is sent. A
        program that redraws its screen just after it started can drop what was typed; the
        text is then typed again, up to two times.

        When the program writes nothing for confirm seconds after the submit key, the key is
        sent once more: a program that took the key as part of a paste shows a still screen,
        while a program that started to work keeps drawing.
        """
        for _ in range(3):
            self.write(text)
            time.sleep(pause + len(text) / 400)
            if not shown or screen.contains(self.display(), shown):
                break
            time.sleep(2)
            if screen.contains(self.display(), shown):
                break
        self.write(submit)
        sent = self.mark()
        deadline = time.monotonic() + confirm
        while time.monotonic() < deadline:
            if self.mark() != sent:
                return
            time.sleep(0.2)
        self.write(submit)

    def stop(self, keys: Sequence[str] = (), timeout: float = 15) -> None:
        """Send keys (each followed by a short pause), wait up to timeout for the program to
        end, and end it if it did not."""
        for key in keys:
            if not self.alive():
                break
            try:
                self.write(key)
            except (OSError, TerminalClosed):
                break
            time.sleep(0.7)
        deadline = time.monotonic() + timeout
        while self.alive() and time.monotonic() < deadline:
            time.sleep(0.2)
        self._kill()
        if self._reader:
            self._reader.join(timeout=5)


def _platform():
    if os.name == "nt":
        from . import terminal_windows as platform
    else:
        from . import terminal_posix as platform
    return platform


def open_terminal() -> Terminal:
    """A new, not yet started Terminal for this operating system."""
    return _platform().new_terminal()


def new_process_group() -> dict:
    """Keyword arguments of subprocess.Popen that let interrupt_process reach the process alone."""
    return _platform().new_process_group()


def interrupt_process(proc) -> None:
    """Send the process the signal of Ctrl+C (SIGINT; on Windows, CTRL_BREAK_EVENT)."""
    _platform().interrupt_process(proc)
