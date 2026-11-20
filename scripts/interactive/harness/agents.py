"""What the harness knows about each agent CLI, and driving one CLI in a terminal.

A definition holds what differs between CLIs: the commands, the dialogs a fresh folder shows
at start-up and the keys that answer them, the text that shows the CLI is ready for an
instruction, and the keys that end it. Where a value differs between operating systems, the
definition holds one per system. Update a definition when a new CLI version changes its
screen; the checks themselves (scenarios) do not change.

Permissions are given up front by options where a CLI has an option that is narrow enough.
Where it has none (the shell commands of Copilot CLI), the definition lists the permission
prompt and the keys that allow it (approvals); the prompt is answered while a reply is awaited,
and the number of answers is part of the reply. The start-up dialogs answered here are the
trust of the working folder and offers to change the terminal setup.

When a new CLI version changes its screen, these values of its definition may need an update:
ready (ready for an instruction), dialogs, approvals, busy (shown only while the CLI works: a
reply is not complete while it shows) and clear_keys (empty the input field).
"""

from __future__ import annotations

import json
import os
import shutil
import time
from dataclasses import dataclass, field
from pathlib import Path
from typing import Callable, Sequence

from . import screen
from .terminal import Terminal, TerminalClosed, open_terminal

SERVER = "agent-lounge"  # the name the CLIs register the server under

DOWN, ENTER, ESC, CTRL_C, CTRL_D, CTRL_U = "\x1b[B", "\r", "\x1b", "\x03", "\x04", "\x15"

# Variables of the session running the harness that change how a CLI behaves (for example a
# Claude Code session marks its children, and an editor's terminal makes a CLI offer to change
# the editor's key bindings). They are removed from the environment of every CLI.
_INHERITED_PREFIXES = ("CLAUDE", "VSCODE_", "TERM_PROGRAM", "COPILOT_DEBUG", "WT_SESSION")


def command(executable: str) -> str:
    """The full path of an executable: on Windows, a CLI installed by npm is a .cmd file, which
    is found only with its extension."""
    return shutil.which(executable) or executable


def clean_environment(cwd: str | os.PathLike | None = None) -> dict[str, str]:
    """The environment for a CLI. For a CLI that starts in cwd, PWD names that folder: a
    process started in another folder otherwise inherits the PWD of the harness, and a shell
    command of the agent that reads $PWD reports (and asks permission for) the wrong folder."""
    env = {k: v for k, v in os.environ.items() if not k.upper().startswith(_INHERITED_PREFIXES)}
    if cwd is not None and os.name == "posix":
        env["PWD"] = os.path.abspath(cwd)
        env.pop("OLDPWD", None)
    return env


@dataclass(frozen=True)
class Definition:
    name: str  # short name, used in session names (iv-<name>)
    title: str  # product name for reports
    executable: str
    interactive: Sequence[str]  # arguments of the interactive screen, after executable
    one_shot: Callable[[str], Sequence[str]]  # arguments that run one instruction and exit
    register: Callable[[str, str], Sequence[str]]  # (url, token) -> arguments that add the server
    unregister: Sequence[str]
    list_servers: Sequence[str]
    ready: Sequence[str]  # any of these on the screen: ready for an instruction
    dialogs: Sequence[tuple[str, Sequence[str]]] = ()  # (text on the screen, keys that answer it)
    # Permission prompts answered while a reply is awaited: (text on the screen, keys that allow).
    # The text is the question line; lines before it, separated by line feeds, must stand on
    # the screen above the question (see _Prompts.ABOVE), for a question that other prompts
    # also ask.
    approvals: Sequence[tuple[str, Sequence[str]]] = ()
    # Texts the screen shows only while the CLI works on an instruction.
    busy: Sequence[str] = ()
    # Keys that empty the input field, sent before an instruction is typed again.
    clear_keys: Sequence[str] = (CTRL_U,)
    quit_keys: Sequence[str] = (CTRL_C, CTRL_C, CTRL_D)
    # Instruction files the CLI is documented or expected to read in its working folder.
    instruction_files: Sequence[str] = ()
    version: Sequence[str] = ("--version",)
    env: dict[str, str] = field(default_factory=dict)
    # A settings file the CLI takes with settings_option (written next to the working folder).
    settings: dict | None = None
    settings_option: str = "--settings"

    def settings_args(self, cwd: str) -> list[str]:
        """Write the settings file for a run in cwd, and return the options that pass it."""
        if self.settings is None:
            return []
        path = Path(cwd).resolve().parent / f".{self.name}-settings.json"
        path.write_text(json.dumps(self.settings), encoding="utf-8")
        return [self.settings_option, str(path)]


def _claude_register(url: str, token: str) -> list[str]:
    args = ["mcp", "add", "--transport", "http", "--scope", "user", SERVER, url]
    return args + (["--header", f"Authorization: Bearer {token}"] if token else [])


def _codex_register(url: str, token: str) -> list[str]:
    args = ["mcp", "add", SERVER, "--url", url]
    return args + (["--bearer-token-env-var", "AGENT_LOUNGE_TOKEN"] if token else [])


def _copilot_register(url: str, token: str) -> list[str]:
    args = ["mcp", "add", "--transport", "http", SERVER, url]
    return args + (["--header", f"Authorization: Bearer {token}"] if token else [])


def _grok_register(url: str, token: str) -> list[str]:
    args = ["mcp", "add", "--transport", "http", SERVER, url]
    return args + (["-H", f"Authorization: Bearer {token}"] if token else [])


def _agy_register(url: str, token: str) -> list[str]:
    args = ["mcp", "add", "--type", "http"]
    return args + (["--header", f"Authorization: Bearer {token}"] if token else []) + [SERVER, url]


# Unquoted: Codex reads a value that is not TOML as a string, and quotes do not survive the
# .cmd launcher of Codex on Windows.
_CODEX_APPROVE = f"mcp_servers.{SERVER}.default_tools_approval_mode=approve"

DEFINITIONS: dict[str, Definition] = {d.name: d for d in [
    Definition(
        name="claude", title="Claude Code", executable="claude",
        interactive=["--allowedTools", f"mcp__{SERVER}", "Write", "Read", "Edit"],
        one_shot=lambda prompt: ["-p", prompt, "--allowedTools", f"mcp__{SERVER}"],
        register=_claude_register, unregister=["mcp", "remove", "--scope", "user", SERVER],
        list_servers=["mcp", "list"],
        ready=["? for shortcuts", "shift+tab to cycle", 'Try "'],
        dialogs=[("Yes, I trust this folder", [DOWN, ENTER])],
        instruction_files=["CLAUDE.md"],
        busy=["esc to interrupt"],
    ),
    Definition(
        name="codex", title="Codex", executable="codex",
        interactive=["-c", _CODEX_APPROVE, "--sandbox", "workspace-write", "--ask-for-approval", "never"],
        one_shot=lambda prompt: ["exec", "--skip-git-repo-check", "-s", "read-only", "-c", _CODEX_APPROVE, prompt],
        register=_codex_register, unregister=["mcp", "remove", SERVER], list_servers=["mcp", "list"],
        ready=["Ask Codex to do anything", "? for shortcuts"],
        dialogs=[("Do you trust the contents of this directory", [ENTER]),
                 ("Yes, allow Codex to work in this folder", [ENTER])],
        instruction_files=["AGENTS.md"],
        busy=["esc to interrupt"],
    ),
    Definition(
        name="copilot", title="Copilot CLI", executable="copilot",
        # Shell commands are not allowed by an option: for its profile the model runs commands
        # of its own choice (Get-Location, $env:USERNAME, git remote ...), which a list of
        # allowed commands does not cover, and allowing the whole shell would also allow what
        # the model never asked for. Each command's prompt is answered instead (approvals), and
        # the answers are counted. A command the model asks for still runs as the user who
        # runs the harness, in or outside the check folder. A command that reads or writes a
        # path outside the paths allowed for the session (such as cat /etc/hostname) asks
        # again for that path; it is allowed once (1. Yes is selected), not for the session.
        interactive=["--allow-tool", SERVER, "--allow-tool", "write"],
        approvals=[("Do you want to run this command?", [ENTER]),
                   ("Allow path access\nDo you want to allow this?", [ENTER])],
        busy=["ctrl+c interrupt"],
        one_shot=lambda prompt: ["-p", prompt, "--allow-tool", SERVER],
        register=_copilot_register, unregister=["mcp", "remove", SERVER], list_servers=["mcp", "list"],
        ready=["Type @ to mention", "shift+tab", "ctrl+c Exit", "/ commands"],
        dialogs=[("Yes, and remember this folder", [ENTER]), ("Set up terminal for multi-line", [ESC])],
        instruction_files=["AGENTS.md", ".github/copilot-instructions.md", "CLAUDE.md", "GEMINI.md"],
    ),
    Definition(
        name="grok", title="Grok", executable="grok",
        interactive=["--allow", f"mcp__{SERVER}"],
        one_shot=lambda prompt: ["-p", prompt, "--allow", f"mcp__{SERVER}"],
        register=_grok_register, unregister=["mcp", "remove", SERVER], list_servers=["mcp", "list"],
        ready=["Shift+Tab:mode", "Ctrl+x:shortcuts", "always-approve", "Resume session"],
        dialogs=[("Yes, I trust this folder", [ENTER]), ("Do you trust the contents of this directory", ["y"])],
        instruction_files=["AGENTS.md", "CLAUDE.md"],
        busy=["Ctrl+c:cancel"],
    ),
    Definition(
        name="agy", title="agy", executable="agy",
        interactive=[],
        one_shot=lambda prompt: ["-p", prompt],
        register=_agy_register, unregister=["mcp", "remove", SERVER], list_servers=["mcp", "list"],
        ready=["? for shortcuts"],
        dialogs=[("Yes, I trust this folder", [ENTER])],
        instruction_files=["GEMINI.md", "AGENTS.md"],
        busy=["esc to cancel"],
        settings={"permissions": {"allow": [f"mcp({SERVER}/*)"]}},
    ),
]}


class _Prompts:
    """Tells a new permission prompt from one that was answered and is still on the screen.

    A CLI that redraws only the lines that changed does not write the question of the next
    prompt again when it stands on the same line as the question before, so "written after
    the answer" cannot be the test, and "left the screen in between" is never true for two
    prompts in a row. The lines above the question (what is asked for: the command) tell the
    prompts apart.

    When the same command is asked for twice in a row (two calls made at once), the second
    prompt has the same lines as the first, and the CLI does not redraw the prompt at all: it
    writes only elsewhere (Copilot CLI writes a line about the remaining requests and the
    state of the first call). A prompt with the same lines counts again once the CLI wrote
    letters or digits after the answer and then none for SETTLE seconds: a CLI that took the
    key and has an answered prompt to remove removes it in the output it writes for the key,
    not seconds later. A CLI that writes nothing after the answer (the key was not taken, or
    it has not yet redrawn) gets the same answer again only after RETRY seconds.
    """

    RETRY = 15.0
    SETTLE = 2.0
    CONTEXT = 8  # lines above the question that belong to the prompt
    ABOVE = 16  # lines above the question in which the other lines of a text must stand

    def __init__(self, texts: Sequence[str], letters: Callable[[], int]):
        self._texts = list(texts)
        self._letters = letters  # the number of letters and digits the CLI has written so far
        self._last: tuple[str, str] | None = None  # the prompt answered last, and its lines
        self._at = 0.0  # when it was answered
        self._count = 0  # letters and digits written when last looked
        self._wrote: float | None = None  # when letters or digits were last seen after the answer

    def _find(self, display: str) -> tuple[str, str] | None:
        lines = display.split("\n")
        for text in self._texts:
            *above, question = text.split("\n")
            for i, line in enumerate(lines):
                if question in line and all(any(a in l for l in lines[max(0, i - self.ABOVE):i]) for a in above):
                    return text, "\n".join(lines[max(0, i - self.CONTEXT):i])
        return None

    def new(self, display: str) -> str | None:
        """The text of a prompt on the screen that still needs an answer, or None."""
        seen = self._find(display)
        if seen is None:
            self._last = None  # the answered prompt left the screen: the next one is new
            return None
        if seen != self._last:
            return seen[0]
        now = time.monotonic()
        count = self._letters()
        if count != self._count:
            self._count, self._wrote = count, now
        if self._wrote is not None and now - self._wrote >= self.SETTLE:
            return seen[0]  # the CLI took the key and drew elsewhere: the same command again
        if now - self._at >= self.RETRY:
            return seen[0]  # the key was not taken
        return None

    def answered(self, display: str, text: str) -> None:
        """Call before the keys of the answer are sent."""
        seen = self._find(display)
        self._last = seen if seen and seen[0] == text else (text, "")
        self._at, self._count, self._wrote = time.monotonic(), self._letters(), None


class AgentNotReady(Exception):
    """The CLI did not become ready for an instruction."""


@dataclass
class Reply:
    complete: bool  # the completion word appeared and the screen settled
    text: str  # the screen and its history, then the plain output from the instruction on
    seconds: float
    approvals: int = 0  # permission prompts answered while waiting (see Definition.approvals)


class Agent:
    """One CLI running in a terminal."""

    def __init__(self, definition: Definition, cwd: str, extra_args: Sequence[str] = (),
                 env: dict[str, str] | None = None):
        self.d = definition
        self.cwd = cwd
        self.extra_args = list(extra_args)
        self.env = env or {}
        self.term: Terminal | None = None
        self.approvals = 0  # permission prompts answered so far, over all instructions

    def start(self, timeout: float = 120) -> "Agent":
        """Start the interactive screen, answer the start-up dialogs, and wait until ready."""
        env = clean_environment(self.cwd)
        env.update(self.d.env)
        env.update(self.env)
        self.term = open_terminal()
        # The environment replaces the inherited one, so that removed variables stay removed.
        self.term.start([command(self.d.executable), *self.d.interactive, *self.d.settings_args(self.cwd),
                         *self.extra_args], cwd=self.cwd, env=env,
                        inherit=False)
        deadline = time.monotonic() + timeout
        since = 0
        answered: set[str] = set()
        while time.monotonic() < deadline:
            self.term.wait_quiet(2, min(15, max(1, deadline - time.monotonic())))
            # The screen as it looks now, and the output since the last answer: a program may
            # leave a dialog on the screen for a moment after it was answered.
            text = self.term.display() + "\n" + self.term.text(since)
            dialog = next((d for d in self.d.dialogs if d[0] not in answered and screen.contains(text, d[0], 0)), None)
            if dialog:
                # Look for the next screen from here on: it may be drawn while the keys are sent.
                since = self.term.mark()
                for key in dialog[1]:
                    self.term.write(key)
                    time.sleep(0.7)
                answered.add(dialog[0])
                continue
            if any(screen.contains(text, r, 0) for r in self.d.ready):
                return self
            if self.term.closed():
                break
        last = self.term.text()[-2000:]
        # End the CLI: it holds the working folder, which cannot be deleted on Windows meanwhile.
        self.quit()
        raise AgentNotReady(f"{self.d.title} did not become ready; last output:\n{last}")

    def ask(self, instruction: str, timeout: float = 180, settle: float = 4) -> Reply:
        """Type an instruction that asks for the completion word, and wait for the reply.

        The reply is complete when the word appeared, then no output arrived for settle
        seconds, and the screen shows none of the texts of work in progress (busy): some CLIs
        show the word in their reasoning before they reply, and a CLI can pause its output for
        longer than settle while a tool runs. Permission prompts of the definition (approvals)
        are answered on the way and counted.
        """
        assert self.term
        word, ask = screen.completion()
        start = self.term.mark()
        began = time.monotonic()
        deadline = began + timeout
        prompts = _Prompts([text for text, _ in self.d.approvals], self.term.letters)
        approvals = 0
        found = complete = False
        try:
            # The lower-case token of the completion word shows that the instruction was typed.
            self.term.type_line(f"{instruction} {ask}", shown=word[len("IVDONE"):].lower(), clear=self.d.clear_keys)
            while not found and time.monotonic() < deadline:
                hit = self.term.wait_for([word], deadline - time.monotonic(), start=start, watch=prompts.new)
                if hit == word:
                    found = True
                elif hit:
                    approvals += self._approve(prompts, hit)
                else:
                    break
            if found:
                # At least one look after the word, also when it arrived just before the timeout.
                deadline = max(deadline, time.monotonic() + settle + 1)
            while found and not complete and time.monotonic() < deadline and not self.term.closed():
                quiet = self.term.wait_quiet(settle, max(settle + 1, deadline - time.monotonic()))
                now = self.term.display()
                hit = prompts.new(now)
                if hit:
                    approvals += self._approve(prompts, hit)
                elif quiet and not any(screen.contains(now, b, 0) for b in self.d.busy):
                    complete = True
                elif quiet:
                    time.sleep(1)  # still working without output: look again
        except TerminalClosed:
            complete = False  # the CLI ended while it was asked
        self.approvals += approvals
        return Reply(complete, self.term.transcript() + "\n" + self.term.text(start), time.monotonic() - began, approvals)

    def _approve(self, prompts: "_Prompts", prompt: str) -> int:
        """Answer a permission prompt that is on the screen. Returns the number answered (1)."""
        assert self.term
        prompts.answered(self.term.display(), prompt)
        keys = next(keys for text, keys in self.d.approvals if text == prompt)
        for key in keys:
            self.term.write(key)
            time.sleep(0.5)
        return 1

    def quit(self) -> None:
        if self.term:
            self.term.stop(self.d.quit_keys)
            self.term = None
