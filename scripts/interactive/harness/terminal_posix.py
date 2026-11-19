"""Terminal on Linux and WSL, with the standard pty module."""

from __future__ import annotations

import codecs
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
        try:
            self._proc = subprocess.Popen(list(argv), cwd=cwd, env=dict(env), stdin=slave, stdout=slave,
                                          stderr=slave, start_new_session=True, close_fds=True)
        except BaseException:
            os.close(master)
            raise
        finally:
            os.close(slave)
        self._fd = master
        # A character can be split between two reads.
        self._decoder = codecs.getincrementaldecoder("utf-8")("replace")

    def _read_chunk(self) -> str:
        try:
            data = os.read(self._fd, 65536)
        except OSError:
            # EIO: every process holding the terminal has ended.
            return ""
        return self._decoder.decode(data, final=not data)

    def _write(self, data: str) -> None:
        os.write(self._fd, data.encode("utf-8"))

    def alive(self) -> bool:
        return self._proc.poll() is None

    def _kill(self) -> None:
        # The whole process group, also when the program itself has ended: it may have left
        # children that hold the terminal.
        try:
            os.killpg(self._proc.pid, signal.SIGTERM)
            self._proc.wait(timeout=5)
        except (ProcessLookupError, PermissionError):
            pass
        except subprocess.TimeoutExpired:
            pass
        try:
            os.killpg(self._proc.pid, signal.SIGKILL)
        except (ProcessLookupError, PermissionError):
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
