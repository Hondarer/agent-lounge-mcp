"""A program running in a pseudo-terminal, with its output collected in the background.

This is the only module that knows which operating system it runs on: open_terminal picks
the implementation (terminal_posix on Linux and WSL, terminal_windows on Windows). The
implementations provide the five primitives of Terminal; everything else (collecting output,
waiting for text, typing, stopping) is shared here.
"""

from __future__ import annotations

import bisect
import os
import subprocess
import threading
import time
from typing import Mapping, Sequence

from . import screen

COLUMNS = 160
ROWS = 50

# How much text before the newly arrived output wait_for searches again: a needle can begin in
# the output of an earlier look and end in the new output.
_OVERLAP = 4000
_OVERLAP_LINES = 5


class TerminalClosed(Exception):
    """The program ended or closed its terminal."""


class Terminal:
    """A program in a pseudo-terminal. Use open_terminal to create one."""

    def __init__(self) -> None:
        self._lock = threading.Condition()
        self._raw: list[str] = []
        self._ends: list[int] = []  # length of the output through each piece of _raw
        self._plain = screen.PlainStream()
        self._plain_at: list[int] = []  # length of the plain text after each piece of _raw
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
                self._ends.append((self._ends[-1] if self._ends else 0) + len(chunk))
                self._plain.feed(chunk)
                self._plain_at.append(self._plain.length)
                self._screen.feed(chunk)
                self._lock.notify_all()

    def raw(self) -> str:
        """All output so far, with control sequences."""
        with self._lock:
            return "".join(self._raw)

    def _raw_slice(self, start: int, end: int) -> str:
        """raw()[start:end], without joining the whole output. Call with the lock held."""
        i = bisect.bisect_right(self._ends, start)
        j = bisect.bisect_left(self._ends, end)
        before = self._ends[i - 1] if i else 0
        return "".join(self._raw[i:j + 1])[start - before:end - before]

    def _plain_offset(self, start: int) -> int:
        """Where the plain text of the output from offset start of raw() begins. start is a
        value of mark, which lies between two pieces of output. Call with the lock held."""
        i = bisect.bisect_right(self._ends, start)
        return self._plain_at[i - 1] if i else 0

    def text(self, start: int = 0) -> str:
        """All output so far from offset start of raw() (a value of mark), as plain text (see
        screen.plain)."""
        with self._lock:
            return self._plain.text(self._plain_offset(start))

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
            return self._ends[-1] if self._ends else 0

    def closed(self) -> bool:
        with self._lock:
            return self._closed

    def wait_for(self, needles: Sequence[str], timeout: float, start: int = 0,
                 on_screen: Sequence[str] = (), on_screen_start: int = 0) -> str | None:
        """Wait until one of needles shows (see screen.contains), on the screen or its history,
        or in the plain output after start; or until one of on_screen shows on the screen as it
        looks now and was written after on_screen_start (text that asks for an answer, such as
        a permission prompt: neither its old copies in the history nor a prompt that was
        answered and is still on the screen must count).

        Returns the text found, or None when the timeout passed or the program ended first.

        The first look covers everything; later looks cover only what arrived since, with some
        of the text before it, so that a long session does not search its whole output again
        for every piece of output.
        """
        deadline = time.monotonic() + timeout
        with self._lock:
            begin = self._plain_offset(start)
            asked = self._plain_offset(on_screen_start)
            searched, history = begin, None
            while True:
                plain = self._plain.text(max(begin, searched - _OVERLAP))
                if history is None:
                    shown = self._screen.transcript()
                else:
                    shown = "\n".join(self._screen.history[max(0, history - _OVERLAP_LINES):] + self._screen.lines())
                searched, history = self._plain.length, len(self._screen.history)
                for needle in needles:
                    if screen.contains(shown, needle) or screen.contains(plain, needle):
                        return needle
                remaining = deadline - time.monotonic()
                if on_screen and remaining > 0 and not self._closed:
                    now, written = self._screen.display(), self._plain.text(asked)
                    for needle in on_screen:
                        if screen.contains(now, needle, 0) and screen.contains(written, needle):
                            return needle
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
                back = max(0, last - 64)
                with self._lock:
                    old, new = self._raw_slice(back, last), self._raw_slice(back, now)
                if screen.words(new) > screen.words(old):
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
                  shown: str = "", clear: Sequence[str] = ()) -> None:
        """Type text and submit it.

        The pause before submit keeps the program from taking the whole input as one pasted
        block, in which the submit key would become a new line. The pause grows with the
        text, because a console can deliver a long text slowly.

        shown is a part of text that must be on the screen before the submit key is sent. A
        program that redraws its screen just after it started can drop what was typed; the
        text is then typed again, up to two times. clear are the keys that empty the input
        field, sent before the text is typed again: a program that dropped only a part of the
        text would otherwise get the rest of it twice.

        When the program writes nothing for confirm seconds after the submit key, the key is
        sent once more: a program that took the key as part of a paste shows a still screen,
        while a program that started to work keeps drawing.
        """
        for attempt in range(3):
            if attempt:
                for key in clear:
                    self.write(key)
                    time.sleep(0.3)
            self.write(text)
            time.sleep(pause + len(text) / 400)
            if not shown or screen.contains(self.display(), shown):
                break
            time.sleep(2)
            if screen.contains(self.display(), shown):
                break
        sent = self.mark()  # before the key, so that output that follows it at once is seen
        self.write(submit)
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


def run(argv: Sequence[str], timeout: float, **kwargs) -> subprocess.CompletedProcess:
    """subprocess.run with captured output that, at the timeout, ends the whole process tree.

    subprocess.run ends only the process it started. A CLI started through a launcher (a .cmd
    file on Windows, a shell script elsewhere) is a child of that process: it survives, keeps
    the output pipes open, and subprocess.run then waits for it without a limit.

    Raises subprocess.TimeoutExpired like subprocess.run, with the output collected so far.
    """
    kwargs.setdefault("stdin", subprocess.DEVNULL)
    proc = subprocess.Popen(argv, stdout=subprocess.PIPE, stderr=subprocess.PIPE, **new_process_group(), **kwargs)
    try:
        out, err = proc.communicate(timeout=timeout)
    except subprocess.TimeoutExpired:
        try:
            _platform().kill_tree(proc)
        except (OSError, subprocess.SubprocessError):
            pass  # the process itself is ended below
        try:
            out, err = proc.communicate(timeout=15)
        except subprocess.TimeoutExpired:
            # A process that left the tree still holds the pipes: give the output up.
            proc.kill()
            out, err = None, None
            for pipe in (proc.stdout, proc.stderr):
                if pipe:
                    pipe.close()
            try:
                proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                pass
        raise subprocess.TimeoutExpired(argv, timeout, output=out, stderr=err) from None
    return subprocess.CompletedProcess(argv, proc.returncode, out, err)
