"""Scenarios 4 and 5: whether a CLI passes the server's instructions to its model, which
instruction files it reads, and whether the snippet of lounge_help, pasted into one, makes
the agent use the lounge during unrelated work."""

from __future__ import annotations

import re
import time

from .. import screen
from ..lounge import Client
from .common import Context, FileWatch, answer, ask, connections

# Instruction files any of the CLIs may read; each gets its own code name in scenario 5.
CANDIDATE_FILES = ["AGENTS.md", "CLAUDE.md", "GEMINI.md", "GROK.md", ".github/copilot-instructions.md"]
RUNS = 3
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
    peer = Client(server.url).connect()
    try:
        result = peer.call("lounge_help", {})
        help_text = result.get("guide") or result.get("text", "")
    finally:
        peer.close()
    m = re.search(r"~~~markdown\n(.*?)\n~~~", help_text, re.S)
    if not m:
        raise RuntimeError("lounge_help has no snippet for instruction files")
    return m.group(1).replace("<name>", session) + "\n"


def instruction_files(ctx: Context) -> None:
    """Scenario 5: which files the CLI reads, then the snippet in that file, RUNS times."""
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
        agent = ctx.agent(folder).start()
        watch = FileWatch(folder / "notes.txt")
        began = time.monotonic()
        try:
            reply, _ = ask(agent, TASK, timeout=300)
        finally:
            watch.stop()
        events = [e for e in server.events(began)]
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
        }
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
