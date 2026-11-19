"""Scenarios 4 and 5: whether a CLI passes the server's instructions to its model, which
instruction files it reads, and whether the snippet of lounge_help, pasted into one, makes
the agent use the lounge during unrelated work and read a message that arrives in the middle
of it."""

from __future__ import annotations

import re
import threading
import time

from .. import screen
from ..lounge import Client, ToolError
from .common import Context, FileWatch, answer, ask, connections

# Instruction files any of the CLIs may read; each gets its own code name in scenario 5.
CANDIDATE_FILES = ["AGENTS.md", "CLAUDE.md", "GEMINI.md", "GROK.md", ".github/copilot-instructions.md"]
RUNS = 5
PEER = "iv-peer"
TASK = ("Create a file named notes.txt in this folder with three lines: alpha, beta and gamma. "
        "Then read the file back and tell me how many lines it has.")


def server_instructions(ctx: Context) -> None:
    """Scenario 4: ask for a passphrase that only the server's instructions contain."""
    if not ctx.test_binary or not ctx.test_passphrase:
        ctx.result.status, ctx.result.reason = "skipped", "no test executable (see build-test-server)"
        return
    server = ctx.server(binary=ctx.test_binary).start()
    agent = ctx.agent(ctx.folder("work")).start()
    no = screen.token()
    reply, added = ask(agent, "Do not call any tool. When you connected to the agent-lounge MCP server, its "
                              "instructions may have given you a passphrase. Write that passphrase in capital "
                              f'letters. If you do not know one, write the word "{no}" in capital letters.')
    v = ctx.result.values
    v["reply_complete"] = reply.complete
    v["passes_instructions"] = answer(added, ctx.test_passphrase.upper(), no.upper()) if reply.complete else "undetermined"
    v["connections"] = connections(server)
    ctx.save("screen.txt", agent.term.transcript())


def _snippet(ctx: Context, server, session: str) -> str:
    # Taken from the text of the result: the structured result of lounge_help carries the guide
    # only on servers before the guide was taken out of it.
    peer = Client(server.url).connect()
    try:
        help_text = peer.call_text("lounge_help", {})
    finally:
        peer.close()
    m = re.search(r"~~~markdown\n(.*?)\n~~~", help_text, re.S)
    if not m:
        raise RuntimeError("lounge_help has no snippet for instruction files")
    return m.group(1).replace("<name>", session) + "\n"


class MidTaskMessage:
    """Sends the CLI's session a message once notes.txt appears and the session exists: a message
    that arrives in the middle of the task, which the agent reads only if it checks its inbox
    while it works."""

    def __init__(self, url: str, session: str, watch: FileWatch):
        self.word = screen.token().upper()
        self.message_id: int | None = None
        self.delay: float | None = None  # seconds from notes.txt to the message
        self.error = ""
        self._session = session
        self._watch = watch
        self._stop = threading.Event()
        # The peer joins before the CLI starts, so that the CLI's own session is the only new one.
        self._peer = Client(url).connect()
        self._peer.call("lounge_join", {"name": PEER})
        self._thread = threading.Thread(target=self._run, daemon=True)
        self._thread.start()

    def _run(self) -> None:
        while not self._stop.is_set() and self._watch.seen is None:
            time.sleep(0.1)
        # Some agents write the file before they join: the message is then sent as soon as the
        # CLI's session exists, until the reply ends.
        while not self._stop.is_set() and self._watch.seen is not None:
            try:
                sent = self._peer.call("chat_send", {"to": self._session, "scope": "user",
                                                     "body": f"The code word is {self.word}. No reply is needed."})
                self.message_id = sent["message_id"]
                self.delay = round(time.monotonic() - self._watch.seen, 1)
                return
            except ToolError as e:
                self.error = str(e)  # no such session yet
            except (ValueError, OSError) as e:
                self.error = str(e)
                return
            time.sleep(0.2)

    def stop(self) -> None:
        self._stop.set()
        self._thread.join()

    def state(self) -> str:
        """The delivery state of the message now: queued, delivered, acked or expired."""
        if self.message_id is None:
            return ""
        status = self._peer.call("chat_status", {"message_id": self.message_id})
        states = [r.get("state", "") for r in status.get("recipients", [])]
        return states[0] if states else ""

    def close(self) -> None:
        self._peer.close()


def instruction_files(ctx: Context) -> None:
    """Scenario 5: which files the CLI reads, then the snippet in that file, RUNS times, each
    with a message sent to the agent's session in the middle of the task."""
    v = ctx.result.values
    # Which instruction files the CLI reads.
    server = ctx.server("discovery").start()
    folder = ctx.folder("discovery-work")
    phrases = {}
    for name in CANDIDATE_FILES:
        phrases[name] = screen.token()
        path = folder / name
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(f"# Project notes\n\nThe code name of this project, from {name}, is {phrases[name]}.\n",
                        encoding="utf-8")
    no = screen.token()
    # Asked as code names of the project: some models refuse to repeat a "passphrase". A model
    # can still refuse to tell what its instructions say, so the question is asked in a new
    # session once more when the reply names no file.
    for attempt in (1, 2):
        agent = ctx.agent(folder).start()
        reply, added = ask(agent, "Do not call any tool. What code names of this project do your instructions for "
                                  "this folder give? Write each code name in capital letters. If they give none, "
                                  f'write the word "{no}" in capital letters.')
        read = [name for name, p in phrases.items() if screen.contains(added, p.upper())]
        ctx.save("discovery-screen.txt" if attempt == 1 else "discovery-screen-2.txt", agent.term.transcript())
        agent.quit()
        if read or (reply.complete and screen.contains(added, no.upper())):
            break
    v["discovery_attempts"] = attempt
    v["discovery_reply_complete"] = reply.complete
    v["files_read"] = read
    server.stop()
    chosen = next((f for f in ctx.definition.instruction_files if f in read), read[0] if read else "")
    v["file_used"] = chosen
    if not chosen:
        ctx.result.status, ctx.result.reason = "unverified", "the CLI read none of the candidate instruction files"
        return
    runs = []
    for i in range(1, RUNS + 1):
        session = f"{ctx.session}-{i}"
        server = ctx.server(f"run{i}").start()
        folder = ctx.folder(f"run{i}-work")
        path = folder / chosen
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(_snippet(ctx, server, session), encoding="utf-8")
        watch = FileWatch(folder / "notes.txt")
        message = MidTaskMessage(server.url, session, watch)
        # The harness's own connections (the snippet's and the peer's) are left out of the
        # agent's tool calls: they are all the connections before the CLI starts.
        deadline = time.monotonic() + 10
        while not server.tool_calls("lounge_join") and time.monotonic() < deadline:
            time.sleep(0.1)
        own = {e.connection for e in server.events()}
        agent = ctx.agent(folder).start()
        began = time.monotonic()
        try:
            reply, added = ask(agent, TASK, timeout=300)
        finally:
            message.stop()
            watch.stop()
        # The main result: whether the agent read the message by the time its reply ended.
        try:
            state = message.state()
        finally:
            message.close()
        events = [e for e in server.events(began) if e.connection not in own]
        tools = [(e.at, e.text[len("request tools/call "):]) for e in events if e.text.startswith("request tools/call ")]
        created = watch.seen
        run = {
            "reply_complete": reply.complete,
            "notes_created": created is not None,
            "tools": [t for _, t in tools],
            "joined": any(t == "lounge_join" for _, t in tools),
            # None when notes.txt was never seen: before and after the work cannot be told apart.
            "inbox_at_start": None if created is None else any(t == "chat_inbox" and at < created for at, t in tools),
            "inbox_after_work": None if created is None else any(t == "chat_inbox" and at >= created for at, t in tools),
            "memory_search": any(t == "memory_search" for _, t in tools),
            "message_sent": message.message_id is not None,
            "message_delay": message.delay,
            "message_state": state,
            # None when no message was sent: there was nothing to read.
            "message_read": None if message.message_id is None else state in ("delivered", "acked"),
            "code_word_in_reply": screen.contains(added.upper(), message.word),
        }
        if message.message_id is None and message.error:
            run["message_error"] = message.error
        runs.append(run)
        v["runs"] = runs  # kept when a later run fails
        ctx.save(f"run{i}-screen.txt", agent.term.transcript())
        agent.quit()
        server.stop()
    v["runs"] = runs
    # The inbox counts cover only these runs: without notes.txt, before and after are unknown.
    v["runs_with_notes"] = sum(1 for r in runs if r["notes_created"])
    for key in ("joined", "inbox_at_start", "inbox_after_work", "memory_search"):
        v[f"{key}_count"] = sum(1 for r in runs if r[key])
    # The read count covers only the runs whose message was sent.
    v["runs_with_message"] = sum(1 for r in runs if r["message_sent"])
    v["message_read_count"] = sum(1 for r in runs if r["message_read"])
