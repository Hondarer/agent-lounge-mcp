"""Scenarios 1 to 3: the interactive screen (the shutdown notice, several instructions in one
session, and joining again after a restart)."""

from __future__ import annotations

import time

from .. import screen
from ..lounge import Client
import re

from .common import Context, answer, ask, connections, join_instruction, new_lines

# A part of tools.ShutdownMessage that the instructions of the scenarios do not contain.
SHUTDOWN_TEXT = "is shutting down"
SERVER_WORDS = re.compile(r"agent-lounge|shutting down|disconnect", re.I)


def _joined(ctx: Context, server, since: float = 0) -> bool:
    return bool(server.tool_calls("lounge_join", since))


def shutdown_notice(ctx: Context) -> None:
    """Scenario 1: does an agent waiting for an instruction learn that the server stops?"""
    server = ctx.server().start()
    agent = ctx.agent(ctx.folder("work")).start()
    reply, _ = ask(agent, join_instruction(ctx.session))
    v = ctx.result.values
    v["join_reply_complete"] = reply.complete
    v["joined"] = _joined(ctx, server)
    if not v["joined"]:
        ctx.result.status, ctx.result.reason = "unverified", "the agent did not call lounge_join"
        ctx.save("screen.txt", agent.term.transcript())
        return
    mark = agent.term.mark()
    before = agent.term.transcript()
    server.interrupt()
    time.sleep(20)
    shown = agent.term.transcript()
    v["server_exit_code"] = server.wait(15)
    v["shown_on_screen"] = screen.contains(shown, SHUTDOWN_TEXT) or screen.contains(agent.term.text(mark), SHUTDOWN_TEXT)
    # What the CLI showed instead, such as its own message that the server went away.
    after = new_lines(before, shown) + "\n" + agent.term.text(mark)
    v["lines_mentioning_the_server"] = sorted({line.strip()[:200] for line in after.split("\n")
                                               if SERVER_WORDS.search(line)})[:10]
    ctx.save("after-stop.txt", agent.term.text(mark))
    no = screen.token()
    reply, added = ask(agent, "Do not call any tool. After your previous reply, did the agent-lounge MCP server send "
                              "you any notice? If it did, quote the notice. If it did not, write the word "
                              f'"{no}" in capital letters.', timeout=180)
    v["question_reply_complete"] = reply.complete
    # Judged on the lines the reply added: a notice that the CLI itself shows would otherwise
    # be taken for the model's answer when the screen is drawn again.
    said = "\n".join(line for line in added.split("\n") if line not in set(shown.split("\n")))
    v["conveyed_to_model"] = answer(said, SHUTDOWN_TEXT, no.upper()) if reply.complete else "undetermined"
    v["connections"] = connections(server)
    ctx.save("screen.txt", before + "\n----- after the stop -----\n" + agent.term.transcript())
    ctx.save("server.log", server.log())


def several_instructions(ctx: Context) -> None:
    """Scenario 2: an instruction, a message from another session, another instruction."""
    server = ctx.server().start()
    agent = ctx.agent(ctx.folder("work")).start()
    v = ctx.result.values
    reply, _ = ask(agent, join_instruction(ctx.session))
    v["joined"] = _joined(ctx, server)
    if not v["joined"]:
        ctx.result.status, ctx.result.reason = "unverified", "the agent did not call lounge_join"
        ctx.save("screen.txt", agent.term.transcript())
        return
    peer = Client(ctx.url).connect()
    peer.call("lounge_join", {"name": "iv-peer"})
    body = screen.token()
    sent = peer.call("chat_send", {"to": ctx.session, "body": f"The code word is {body}."})
    reply, added = ask(agent, "Check your agent-lounge inbox with chat_inbox and tell me the code word in the message "
                              "you received, written in capital letters.")
    v["reply_complete"] = reply.complete
    v["code_word_in_reply"] = screen.contains(added, body.upper())
    status = peer.call("chat_status", {"message_id": sent["message_id"]})
    v["delivery_state"] = [r.get("state") for r in status.get("recipients", [])]
    conns = connections(server)
    v["connections"] = conns
    join_conn = [n for n, c in conns.items() if "lounge_join" in c["tools"]]
    inbox_conn = [n for n, c in conns.items() if "chat_inbox" in c["tools"]]
    v["same_connection"] = bool(join_conn and inbox_conn and join_conn[0] == inbox_conn[-1])
    v["passed"] = v["code_word_in_reply"] and v["delivery_state"] in (["delivered"], ["acked"])
    if not reply.complete and not v["passed"]:
        ctx.result.status, ctx.result.reason = "unverified", "the reply did not finish in time"
    peer.close()
    ctx.save("screen.txt", agent.term.transcript())
    ctx.save("server.log", server.log())


def join_again(ctx: Context) -> None:
    """Scenario 3: quit the CLI, receive a message, start it again and join with the same name."""
    server = ctx.server().start()
    folder = ctx.folder("work")
    agent = ctx.agent(folder).start()
    v = ctx.result.values
    ask(agent, join_instruction(ctx.session))
    v["joined"] = _joined(ctx, server)
    first = agent.term.transcript()
    agent.quit()
    if not v["joined"]:
        ctx.result.status, ctx.result.reason = "unverified", "the agent did not call lounge_join"
        ctx.save("screen.txt", first)
        return
    peer = Client(ctx.url).connect()
    peer.call("lounge_join", {"name": "iv-peer"})
    body = screen.token()
    peer.call("chat_send", {"to": ctx.session, "body": f"The code word is {body}."})
    agent = ctx.agent(folder).start()
    reply, added = ask(agent, f'Call lounge_join of the agent-lounge MCP server again with the name "{ctx.session}" '
                              "and no profile values, then call chat_inbox and tell me the code word in the message "
                              "you received, written in capital letters.")
    v["reply_complete"] = reply.complete
    v["code_word_in_reply"] = screen.contains(added, body.upper())
    # session list prints addresses as user/name.
    sessions = [line.split("\t")[0].split("/")[-1] for line in server.admin("session", "list").splitlines()[1:]
                if line.strip()]
    v["sessions"] = sessions
    v["one_session"] = sessions.count(ctx.session) == 1 and len(sessions) == 2
    v["passed"] = v["code_word_in_reply"] and v["one_session"]
    if not reply.complete and not v["passed"]:
        ctx.result.status, ctx.result.reason = "unverified", "the reply did not finish in time"
    v["connections"] = connections(server)
    peer.close()
    ctx.save("screen.txt", first + "\n----- after the restart -----\n" + agent.term.transcript())
    ctx.save("server.log", server.log())
