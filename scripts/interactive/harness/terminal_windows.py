"""Terminal on Windows, with the pseudo console (ConPTY) through the pywinpty package."""

from __future__ import annotations

import signal
import subprocess
from typing import Mapping, Sequence

from .terminal import Terminal

try:
    from winpty import PtyProcess
except ImportError as e:  # pragma: no cover - depends on the machine
    raise ImportError("the harness needs pywinpty on Windows: python -m pip install --user pywinpty") from e


class WindowsTerminal(Terminal):
    def _spawn(self, argv: Sequence[str], cwd: str, env: Mapping[str, str], columns: int, rows: int) -> None:
        # ConPTY takes one command line; list2cmdline quotes as the C runtime expects.
        self._proc = PtyProcess.spawn(subprocess.list2cmdline(list(argv)), cwd=cwd, env=dict(env),
                                      dimensions=(rows, columns))

    def _read_chunk(self) -> str:
        try:
            data = self._proc.read(65536)
        except EOFError:
            return ""
        # pywinpty returns an empty string when nothing is available yet; keep blocking until
        # there is output or the program ends.
        while data == "":
            if not self._proc.isalive():
                return ""
            try:
                data = self._proc.read(65536)
            except EOFError:
                return ""
        return data

    def _write(self, data: str) -> None:
        self._proc.write(data)

    def alive(self) -> bool:
        return self._proc.isalive()

    def _kill(self) -> None:
        if self._proc.isalive():
            try:
                self._proc.terminate(force=True)
            except Exception:
                pass
        try:
            self._proc.close(force=True)
        except Exception:
            pass


def new_terminal() -> WindowsTerminal:
    return WindowsTerminal()


def new_process_group() -> dict:
    # CTRL_BREAK_EVENT reaches only a process group of its own; Go takes it as an interrupt.
    return {"creationflags": subprocess.CREATE_NEW_PROCESS_GROUP}


def interrupt_process(proc: subprocess.Popen) -> None:
    proc.send_signal(signal.CTRL_BREAK_EVENT)
