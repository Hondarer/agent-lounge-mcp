"""Terminal on Linux and WSL, with the standard pty module."""

from __future__ import annotations

import fcntl
import os
import pty
import signal
import struct
import subprocess
import termios
from typing import Mapping, Sequence

from .terminal import Terminal


class PosixTerminal(Terminal):
    def _spawn(self, argv: Sequence[str], cwd: str, env: Mapping[str, str], columns: int, rows: int) -> None:
        master, slave = pty.openpty()
        fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack("HHHH", rows, columns, 0, 0))
        # A new session makes the terminal the program's controlling terminal, so that keys
        # such as Ctrl+C reach it as they would from a user.
        self._proc = subprocess.Popen(list(argv), cwd=cwd, env=dict(env), stdin=slave, stdout=slave,
                                      stderr=slave, start_new_session=True, close_fds=True)
        os.close(slave)
        self._fd = master

    def _read_chunk(self) -> str:
        try:
            data = os.read(self._fd, 65536)
        except OSError:
            # EIO: every process holding the terminal has ended.
            return ""
        return data.decode("utf-8", errors="replace")

    def _write(self, data: str) -> None:
        os.write(self._fd, data.encode("utf-8"))

    def alive(self) -> bool:
        return self._proc.poll() is None

    def _kill(self) -> None:
        if self._proc.poll() is None:
            try:
                os.killpg(self._proc.pid, signal.SIGTERM)
                self._proc.wait(timeout=5)
            except (ProcessLookupError, subprocess.TimeoutExpired):
                try:
                    os.killpg(self._proc.pid, signal.SIGKILL)
                except ProcessLookupError:
                    pass
                self._proc.wait()
        try:
            os.close(self._fd)
        except OSError:
            pass


def new_terminal() -> PosixTerminal:
    return PosixTerminal()


def new_process_group() -> dict:
    return {"start_new_session": True}


def interrupt_process(proc: subprocess.Popen) -> None:
    proc.send_signal(signal.SIGINT)
