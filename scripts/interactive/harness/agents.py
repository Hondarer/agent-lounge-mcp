"""What the harness knows about each agent CLI, and driving one CLI in a terminal.

A definition holds what differs between CLIs: the commands, the dialogs a fresh folder shows
at start-up and the keys that answer them, the text that shows the CLI is ready for an
instruction, and the keys that end it. Where a value differs between operating systems, the
definition holds one per system. Update a definition when a new CLI version changes its
screen; the checks themselves (scenarios) do not change.

Permissions are given up front by options, so that no permission prompt appears. The
start-up dialogs answered here are the trust of the working folder and offers to change the
terminal setup; nothing else is answered.
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
from .terminal import Terminal, open_terminal

SERVER = "agent-lounge"  # the name the CLIs register the server under

DOWN, ENTER, ESC, CTRL_C, CTRL_D = "\x1b[B", "\r", "\x1b", "\x03", "\x04"

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


# Read-only commands that lounge_help tells agents to run for their profile.
_PROFILE_COMMANDS = ["cmd.exe", "git", "hostname", "whoami", "pwd", "uname", "echo", "id", "powershell.exe", "pwsh"]

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
        instruction_files=["CLAUDE.md", "AGENTS.md"],
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
    ),
    Definition(
        name="copilot", title="Copilot CLI", executable="copilot",
        # The commands lounge_help lists for the profile are allowed one by one.
        interactive=["--allow-tool", SERVER, "--allow-tool", "write",
                     *[arg for c in _PROFILE_COMMANDS for arg in ("--allow-tool", f"shell({c})")]],
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
        instruction_files=["AGENTS.md", "GROK.md", "CLAUDE.md"],
    ),
    Definition(
        name="agy", title="agy", executable="agy",
        interactive=[],
        one_shot=lambda prompt: ["-p", prompt],
        register=_agy_register, unregister=["mcp", "remove", SERVER], list_servers=["mcp", "list"],
        ready=["? for shortcuts"],
        dialogs=[("Yes, I trust this folder", [ENTER])],
        instruction_files=["GEMINI.md", "AGENTS.md"],
        settings={"permissions": {"allow": [f"mcp({SERVER}/*)"]}},
    ),
]}


class AgentNotReady(Exception):
    """The CLI did not become ready for an instruction."""


@dataclass
class Reply:
    complete: bool  # the completion word appeared and the screen settled
    text: str  # the screen and its history, then the plain output from the instruction on
    seconds: float


class Agent:
    """One CLI running in a terminal."""

    def __init__(self, definition: Definition, cwd: str, extra_args: Sequence[str] = (),
                 env: dict[str, str] | None = None):
        self.d = definition
        self.cwd = cwd
        self.extra_args = list(extra_args)
        self.env = env or {}
        self.term: Terminal | None = None

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

        The reply is complete when the word appeared and then no output arrived for settle
        seconds: some CLIs show the word in their reasoning before they reply, and keep
        drawing while they work.
        """
        assert self.term
        word, ask = screen.completion()
        start = self.term.mark()
        began = time.monotonic()
        # The lower-case token of the completion word shows that the instruction was typed.
        self.term.type_line(f"{instruction} {ask}", shown=word[len("IVDONE"):].lower())
        found = self.term.wait_for([word], timeout, start=start)
        complete = False
        if found:
            remaining = max(5.0, timeout - (time.monotonic() - began))
            complete = self.term.wait_quiet(settle, remaining)
        return Reply(complete, self.term.transcript() + "\n" + self.term.text(start), time.monotonic() - began)

    def quit(self) -> None:
        if self.term:
            self.term.stop(self.d.quit_keys)
            self.term = None
