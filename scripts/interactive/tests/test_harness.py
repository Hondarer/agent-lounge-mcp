"""Self-tests of the harness that need no agent CLI and no server. They run on Linux, WSL and
Windows (Windows needs pywinpty):

  python3 -m unittest discover -s scripts/interactive/tests
"""

import http.server
import os
import queue
import random
import subprocess
import sys
import tempfile
import threading
import time
import unittest
from pathlib import Path
from unittest.mock import patch

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE.parent))

from harness import screen, terminal  # noqa: E402
from harness.agents import CTRL_U, DEFINITIONS, ENTER, Agent, AgentNotReady, Definition, clean_environment  # noqa: E402
from harness.agents import _Prompts  # noqa: E402
from harness.scenarios.common import answer, connections, new_lines, tools_called  # noqa: E402
from harness.lounge import Client  # noqa: E402
from harness.server import Server  # noqa: E402


class PlainTest(unittest.TestCase):
    def test_plain(self):
        raw = "\x1b(B\x1b[1mAcc\x1b[1Cws\x1b[0m\x1b7x\x1b8\x1b]0;title\x07ok\r\nnext\x07"
        self.assertEqual(screen.plain(raw), "Acc wsxok\nnext")

    def test_contains_allows_noise_inside_a_word(self):
        self.assertTrue(screen.contains("hello IVD ◎ONE1A2B", "IVDONE1A2B"))
        self.assertTrue(screen.contains("IVDONE\n1A2B", "IVDONE1A2B"))
        self.assertFalse(screen.contains("IVDONE" + "x" * 20 + "1A2B", "IVDONE1A2B"))
        self.assertFalse(screen.contains("ivdone1a2b", "IVDONE1A2B"))

    def test_completion_word_is_not_in_the_instruction(self):
        word, ask = screen.completion()
        self.assertTrue(word.startswith("IVDONE"))
        self.assertFalse(screen.contains(ask, word))

    def test_completion_token_does_not_begin_with_e(self):
        for _ in range(200):
            word, _ask = screen.completion()
            self.assertNotEqual(word[len("IVDONE")], "E")

    def test_words(self):
        self.assertEqual(screen.words("\x1b[31m◐◓\x1b[0m"), 0)
        self.assertEqual(screen.words("\x1b[31mab1\x1b[0m"), 3)


class PlainStreamTest(unittest.TestCase):
    RAW = ("\x1b[1mBold\x1b[0m text\r\nnext\x1b[3Cgap\x1b]0;a title\x1b\\after\x1b]8;;x\x07link\r"
           "over\x1b(Bcharset\x1bPstring\x1b\\end\x1b7saved\x9b31mred\r\n" * 3)

    def test_pieces_give_the_text_of_the_whole(self):
        whole = screen.plain(self.RAW)
        rnd = random.Random(1)
        for _ in range(200):
            s, i = screen.PlainStream(), 0
            while i < len(self.RAW):
                n = rnd.randint(1, 9)
                s.feed(self.RAW[i:i + n])
                i += n
            self.assertEqual(s.text(), whole)
            self.assertEqual(s.text(5), whole[5:])

    def test_carriage_return_before_an_escape_and_a_split(self):
        for pieces in (["a\r\x1b[K", "\nb"], ["a\r", "\x1b[K", "\nb"], ["a\r\x1b", "[K\n", "b"]):
            s = screen.PlainStream()
            for piece in pieces:
                s.feed(piece)
            self.assertEqual(s.text(), screen.plain("".join(pieces)), pieces)

    def test_escape_at_the_end_of_a_string_that_never_ends(self):
        pieces = ["x\x1b]" + "y" * 5000 + "\x1b", "[31mred"]
        s = screen.PlainStream()
        for piece in pieces:
            s.feed(piece)
        self.assertTrue(s.text().endswith("yred"), s.text()[-20:])

    def test_unfinished_end_is_held(self):
        s = screen.PlainStream()
        s.feed("one\x1b[3")
        self.assertEqual(s.text(), "one3")  # read as it stands, like plain of the output so far
        s.feed("1mtwo\r")
        self.assertEqual(s.text(), screen.plain("one\x1b[31mtwo\r"))
        s.feed("\nthree")
        self.assertEqual(s.text(), "onetwo\nthree")
        self.assertEqual(s.text(len("one")), "two\nthree")

    def test_a_string_that_never_ends_is_not_held_for_ever(self):
        s = screen.PlainStream()
        s.feed("\x1b]" + "x" * 5000)
        self.assertGreater(s.length, 4000)


class ScreenTest(unittest.TestCase):
    def test_cursor_and_erase(self):
        s = screen.Screen(20, 4)
        s.feed("hello\r\nworld\x1b[1;3HXY\x1b[2;1H\x1b[2Kab\x1b[1;1H\x1b[K")
        self.assertEqual(s.lines(), ["", "ab", "", ""])

    def test_scrolling_keeps_history(self):
        s = screen.Screen(10, 2)
        s.feed("one\r\ntwo\r\nthree")
        self.assertEqual(s.history, ["one"])
        self.assertEqual(s.transcript(), "one\ntwo\nthree")

    def test_alternate_screen(self):
        s = screen.Screen(10, 2)
        s.feed("main\x1b[?1049hfull\r\nscreen\r\nmore\x1b[?1049l")
        self.assertEqual(s.lines(), ["main", ""])
        self.assertEqual(s.history, [])

    def test_wide_characters_and_split_sequences(self):
        s = screen.Screen(10, 2)
        s.feed("日本\x1b[")
        s.feed("1;9Hx")
        self.assertEqual(s.lines()[0], "日本    x")

    def test_long_string_sequence_split_between_reads(self):
        s = screen.Screen(40, 2)
        s.feed("a\x1b]8;;https://example.com/" + "x" * 200)
        s.feed("\x07link\x1b]8;;\x07b")
        self.assertEqual(s.lines()[0], "alinkb")

    def test_scroll_region(self):
        s = screen.Screen(10, 4)
        s.feed("top\x1b[2;3r\x1b[2;1Ha\r\nb\r\nc\x1b[4;1Hbottom")
        self.assertEqual(s.lines(), ["top", "b", "c", "bottom"])


class JudgingTest(unittest.TestCase):
    def test_answer(self):
        self.assertEqual(answer("the code is YES1", "YES1", "NO1"), "yes")
        self.assertEqual(answer("NO1", "YES1", "NO1"), "no")
        self.assertEqual(answer("YES1 NO1", "YES1", "NO1"), "undetermined")
        self.assertEqual(answer("nothing", "YES1", "NO1"), "undetermined")

    def test_new_lines(self):
        self.assertEqual(new_lines("a\nb", "a\nb\nc\nd"), "c\nd")


class ServerEventsTest(unittest.TestCase):
    def test_events(self):
        s = Server("unused", "unused.db")
        s._lines = [(1.0, "agent-lounge-mcp: version x"),
                    (2.0, "agent-lounge-mcp: connection 1: request tools/call lounge_join"),
                    (3.0, "agent-lounge-mcp: connection 2: initialized test/1")]
        self.assertEqual([(e.connection, e.text) for e in s.events()],
                         [(1, "request tools/call lounge_join"), (2, "initialized test/1")])
        self.assertEqual(len(s.tool_calls("lounge_join", since=1.5)), 1)
        self.assertEqual(s.tool_calls("lounge_join", since=2.5), [])

    def test_connections(self):
        s = Server("unused", "unused.db")
        s._lines = [(1.0, "agent-lounge-mcp: connection 1: initialized claude-code/2"),
                    (2.0, "agent-lounge-mcp: connection 1: request tools/call lounge_join"),
                    (3.0, "agent-lounge-mcp: connection 2: request logging/setLevel")]
        self.assertEqual(connections(s), {1: {"client": "claude-code/2", "tools": ["lounge_join"]},
                                          2: {"client": "", "tools": []}})


    def test_tools_called_skips_own_connections_and_earlier_calls(self):
        s = Server("unused", "unused.db")
        s._lines = [(1.0, "agent-lounge-mcp: connection 1: request tools/call lounge_join"),
                    (2.0, "agent-lounge-mcp: connection 2: request tools/call lounge_send"),
                    (3.0, "agent-lounge-mcp: connection 2: request tools/list"),
                    (3.5, "agent-lounge-mcp: connection 2: initialized test/1"),
                    (4.0, "agent-lounge-mcp: connection 3: request tools/call lounge_help"),
                    (5.0, "agent-lounge-mcp: connection 2: request tools/call lounge_read")]
        self.assertEqual(tools_called(s, 0), ["lounge_join", "lounge_send", "lounge_help", "lounge_read"])
        self.assertEqual(tools_called(s, 0, own={1, 3}), ["lounge_send", "lounge_read"])
        self.assertEqual(tools_called(s, 2.5), ["lounge_help", "lounge_read"])  # not tools/list or initialized
        self.assertEqual(tools_called(s, 2.5, own={2}), ["lounge_help"])
        self.assertEqual(tools_called(s, 6), [])


class EnvironmentTest(unittest.TestCase):
    @unittest.skipUnless(os.name == "posix", "PWD is a POSIX convention")
    def test_pwd_names_the_working_folder(self):
        with tempfile.TemporaryDirectory() as tmp:
            self.assertEqual(clean_environment(tmp)["PWD"], os.path.abspath(tmp))
            self.assertNotIn("OLDPWD", clean_environment(tmp))


class TypedTwice(terminal.Terminal):
    """A terminal whose program drops the first text typed into it."""

    def __init__(self):
        super().__init__()
        self.written, self.field = [], ""

    def write(self, data):
        self.written.append(data)
        if data == CTRL_U:
            self.field = ""
        elif data != "\r":
            # The first text loses its end; later texts arrive whole.
            self.field += data[:4] if len(self.written) == 1 else data

    def display(self):
        return self.field

    def mark(self):
        return len(self.written)


class TypeLineTest(unittest.TestCase):
    def test_input_field_is_cleared_before_typing_again(self):
        t = TypedTwice()
        t.type_line("say token123", pause=0, confirm=0, shown="token123", clear=[CTRL_U])
        self.assertEqual(t.written[:3], ["say token123", CTRL_U, "say token123"])
        self.assertEqual(t.field, "say token123")  # not "say say token123"


class Fed(terminal.Terminal):
    """A terminal with no program: the test feeds the output through feed()."""

    def __init__(self):
        super().__init__()
        self._chunks: queue.Queue = queue.Queue()

    def feed(self, chunk: str) -> None:
        self._chunks.put(chunk)

    def _spawn(self, argv, cwd, env, columns, rows):
        pass

    def _read_chunk(self) -> str:
        return self._chunks.get()

    def alive(self) -> bool:
        return not self.closed()

    def _kill(self) -> None:
        self._chunks.put("")

    def settle(self) -> None:
        """Wait until the collector has taken everything fed so far."""
        deadline = time.monotonic() + 5
        while not self._chunks.empty() and time.monotonic() < deadline:
            time.sleep(0.01)
        time.sleep(0.05)


def _in_background(fn):
    """Run fn in a thread; return a function that waits for it and gives its result."""
    box = {}
    thread = threading.Thread(target=lambda: box.update(result=fn()), daemon=True)
    thread.start()

    def result():
        thread.join(10)
        assert not thread.is_alive(), "wait_for did not return"
        return box["result"]
    return result


class WaitForTest(unittest.TestCase):
    def setUp(self):
        self.t = Fed().start(["unused"], ".")
        self.addCleanup(self.t.stop, (), 1)

    def test_finds_text_that_arrives_later(self):
        result = _in_background(lambda: self.t.wait_for(["needle"], 5))
        time.sleep(0.2)
        self.t.feed("some text, then the needle\r\n")
        self.assertEqual(result(), "needle")

    def test_returns_none_at_the_timeout_and_when_the_program_ends(self):
        self.t.feed("nothing here\r\n")
        began = time.monotonic()
        self.assertIsNone(self.t.wait_for(["needle"], 0.6))
        self.assertGreaterEqual(time.monotonic() - began, 0.5)
        result = _in_background(lambda: self.t.wait_for(["needle"], 30))
        time.sleep(0.2)
        self.t.feed("")  # the end of the output
        self.assertIsNone(result())

    def test_later_searches_continue_from_where_the_previous_one_ended(self):
        searched = []
        text = self.t._plain.text
        self.t._plain.text = lambda start=0: (searched.append(start), text(start))[1]
        self.t.feed("x" * 20000 + "\r\n")
        self.t.settle()
        result = _in_background(lambda: self.t.wait_for(["needle"], 5))
        time.sleep(0.3)
        self.assertEqual(searched, [0])  # the first look covers everything
        self.t.feed("y" * 100 + " needle")
        self.assertEqual(result(), "needle")
        self.assertEqual(len(searched), 2)
        # The second look starts near the end of what the first one covered, not at 0.
        self.assertGreater(searched[1], 20000 - 5000)
        self.assertLess(searched[1], 20000 + 1)

    def test_finds_a_needle_split_between_two_pieces_of_output(self):
        # The screen is cleared between the pieces, so that only the plain output holds the
        # whole needle.
        self.t.feed("x" * 20000 + "ABC\x1b[2J\x1b[H")
        self.t.settle()
        result = _in_background(lambda: self.t.wait_for(["ABCDEF"], 5))
        time.sleep(0.3)
        self.t.feed("DEF")
        self.assertEqual(result(), "ABCDEF")
        self.assertFalse(screen.contains(self.t.transcript(), "ABCDEF"))

    def test_start_skips_the_output_before_it(self):
        self.t.feed("old needle\x1b[2J\x1b[H")  # gone from the screen, still in the output
        self.t.settle()
        self.assertEqual(self.t.wait_for(["old needle"], 2), "old needle")
        mark = self.t.mark()
        self.assertIsNone(self.t.wait_for(["old needle"], 0.6, start=mark))
        result = _in_background(lambda: self.t.wait_for(["old needle", "new needle"], 5, start=mark))
        time.sleep(0.2)
        self.t.feed("new needle")
        self.assertEqual(result(), "new needle")

    def test_start_in_the_middle_of_the_output_covers_the_rest(self):
        self.t.feed("a" * 20000 + "\x1b[2J\x1b[H")
        self.t.settle()
        mark = self.t.mark()
        self.t.feed("late needle")
        self.t.settle()
        self.assertEqual(self.t.wait_for(["late needle"], 2, start=mark), "late needle")

    def test_watch_judges_the_screen_and_its_text_is_the_result(self):
        seen = []

        def watch(display):
            seen.append(display)
            return "answered" if "PROMPT" in display else None

        result = _in_background(lambda: self.t.wait_for(["never shown"], 5, watch=watch))
        time.sleep(0.7)  # the watch runs also while nothing arrives
        self.assertGreaterEqual(len(seen), 1)
        self.t.feed("PROMPT")
        self.assertEqual(result(), "answered")

    def test_watch_does_not_see_the_history(self):
        self.t.feed("PROMPT\r\n" + "line\r\n" * 80)  # PROMPT scrolled into the history
        self.t.settle()
        self.assertIn("PROMPT", self.t.transcript())
        self.assertNotIn("PROMPT", self.t.display())
        calls = []

        def watch(display):
            calls.append(display)
            return "answered" if "PROMPT" in display else None

        self.assertIsNone(self.t.wait_for(["never shown"], 0.6, watch=watch))
        self.assertTrue(calls)

    def test_a_needle_wins_over_watch_and_watch_is_not_called_after_the_end(self):
        called = []
        self.t.feed("found needle")
        self.t.settle()
        self.assertEqual(self.t.wait_for(["needle"], 2, watch=lambda d: called.append(d) or "watched"), "needle")
        self.assertEqual(called, [])
        self.t.stop((), 1)
        self.assertIsNone(self.t.wait_for(["absent"], 2, watch=lambda d: called.append(d) or "watched"))
        self.assertEqual(called, [])


class LoungeClientTest(unittest.TestCase):
    def test_close_ends_the_session_and_leaves_no_connection_open(self):
        seen = []

        class Handler(http.server.BaseHTTPRequestHandler):
            def do_DELETE(self):
                seen.append(self.headers.get("Mcp-Session-Id"))
                self.send_response(200)
                self.send_header("Content-Length", "0")
                self.end_headers()

            def log_message(self, *args):
                pass

        httpd = http.server.HTTPServer(("127.0.0.1", 0), Handler)
        thread = threading.Thread(target=httpd.serve_forever, daemon=True)
        thread.start()
        try:
            c = Client(f"http://127.0.0.1:{httpd.server_address[1]}/mcp", timeout=5)
            c.session = "s1"
            c.close()
            self.assertEqual((seen, c.session), (["s1"], ""))
        finally:
            httpd.shutdown()
            httpd.server_close()
            thread.join(5)


def _alive(pid: int) -> bool:
    if os.name == "nt":
        out = subprocess.run(["tasklist", "/FI", f"PID eq {pid}", "/NH"], capture_output=True, text=True).stdout
        return str(pid) in out
    try:
        os.kill(pid, 0)
    except ProcessLookupError:
        return False
    # A process that ended and was not yet collected by its parent still has an entry.
    try:
        with open(f"/proc/{pid}/stat") as f:
            return f.read().rsplit(")", 1)[1].split()[0] != "Z"
    except OSError:
        return True


class RunTest(unittest.TestCase):
    def test_timeout_ends_the_process_tree(self):
        with tempfile.TemporaryDirectory() as tmp:
            pid_file = Path(tmp) / "child.pid"
            began = time.monotonic()
            with self.assertRaises(subprocess.TimeoutExpired) as raised:
                terminal.run([sys.executable, str(HERE / "fake_tree.py"), str(pid_file)], 3, text=True)
            # Without ending the tree, the child keeps the output open for two minutes.
            self.assertLess(time.monotonic() - began, 30)
            self.assertIn("started", raised.exception.output or "")
            pid = int(pid_file.read_text())
            deadline = time.monotonic() + 10
            while _alive(pid) and time.monotonic() < deadline:
                time.sleep(0.2)
            self.assertFalse(_alive(pid), "the child of the timed-out process is still running")

    def test_result_without_timeout(self):
        out = terminal.run([sys.executable, "-c", "print('ok')"], 30, text=True)
        self.assertEqual((out.returncode, out.stdout.strip()), (0, "ok"))


def fake(dialog: bool = False) -> Definition:
    return Definition(
        name="fake", title="Fake", executable=sys.executable,
        interactive=["-u", str(HERE / "fake_agent.py")],
        one_shot=lambda p: [], register=lambda u, t: [], unregister=[], list_servers=[],
        ready=["? for shortcuts"], dialogs=[("Yes, I trust this folder", [ENTER])],
        quit_keys=["quit\r"], env={"FAKE_DIALOG": "1"} if dialog else {},
        approvals=[("Do you want to run this command?", [ENTER])], busy=["esc to interrupt"],
    )


class PromptsTest(unittest.TestCase):
    def test_rewriting_the_same_screen_does_not_answer_again_before_retry(self):
        question = "Do you want to run this command?"
        terminal_screen = screen.Screen(80, 24)
        terminal_screen.feed(f"command 0\r\n{question}\r\n2 requests remaining")
        original = terminal_screen.display()
        output = "\x1b[3;1H2 requests remaining"
        with patch("harness.agents.time.monotonic", return_value=0) as clock:
            mark = 0
            prompts = _Prompts([question], lambda: mark)
            self.assertEqual(prompts.new(original), question)
            prompts.answered(original, question)

            clock.return_value = 1
            terminal_screen.feed(output)
            mark += len(output)
            rewritten = terminal_screen.display()
            self.assertEqual(rewritten, original)
            self.assertIsNone(prompts.new(rewritten))

            clock.return_value = 1 + prompts.SETTLE
            self.assertIsNone(prompts.new(rewritten))
            clock.return_value = prompts.RETRY
            self.assertEqual(prompts.new(rewritten), question)


class DefinitionTest(unittest.TestCase):
    def test_codex_interactive_starts_with_the_update_check_off_and_one_shot_does_not(self):
        pair = ["-c", "check_for_update_on_startup=false"]
        codex = DEFINITIONS["codex"]
        args = codex.interactive
        self.assertTrue(any(args[i:i + 2] == pair for i in range(len(args) - 1)), args)
        one_shot = codex.one_shot("hello")
        self.assertFalse(any(one_shot[i:i + 2] == pair for i in range(len(one_shot) - 1)), one_shot)


class AgentTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()

    def tearDown(self):
        self.tmp.cleanup()

    def start(self, definition=None, **kwargs):
        """Start the fake agent, and end it at the end of the test even when the test fails:
        a terminal left open keeps its console (on Windows, its sockets) open."""
        a = Agent(definition or fake(**kwargs), self.tmp.name)
        self.addCleanup(a.quit)
        return a.start(timeout=30)

    def test_dialog_instruction_and_quit(self):
        a = self.start(dialog=True)
        reply = a.ask("Say hello.", timeout=20, settle=1)
        self.assertTrue(reply.complete, reply.text)
        reply = a.ask("Say hello slow.", timeout=20, settle=1)
        self.assertTrue(reply.complete, reply.text)
        term = a.term
        a.quit()
        self.assertFalse(term.alive())

    def test_permission_prompts_are_answered_and_counted(self):
        a = self.start()
        # Three prompts in a row, of which only the command line is redrawn.
        began = time.monotonic()
        reply = a.ask("Say hello after permission and permission and permission.", timeout=40, settle=1)
        self.assertTrue(reply.complete, reply.text)
        self.assertEqual((reply.approvals, a.approvals), (3, 3))
        self.assertLess(time.monotonic() - began, 14)  # no prompt waited for the retry time
        reply = a.ask("Say hello.", timeout=20, settle=1)
        self.assertEqual((reply.approvals, a.approvals), (0, 3))
        a.quit()

    def test_the_same_command_asked_for_in_a_row_is_answered_each_time_without_the_retry(self):
        a = self.start()
        # Three prompts with the same lines; between them, only lines outside the prompt change.
        began = time.monotonic()
        reply = a.ask("Say hello after twin and twin and twin.", timeout=40, settle=1)
        self.assertTrue(reply.complete, reply.text)
        self.assertEqual(reply.approvals, 3)
        self.assertLess(time.monotonic() - began, 14)  # no prompt waited for the retry time
        a.quit()

    def test_an_answered_prompt_that_stays_a_while_is_answered_once(self):
        a = self.start()
        # The prompt stays 5 seconds after the answer (longer than the settle time of prompts),
        # with only symbols drawn meanwhile.
        reply = a.ask("Say hello after linger.", timeout=30, settle=1)
        self.assertTrue(reply.complete, reply.text)
        self.assertEqual(reply.approvals, 1)
        a.quit()

    def test_copilot_path_access_is_allowed_and_the_same_question_elsewhere_is_not(self):
        d = Definition(**{**fake().__dict__, "approvals": DEFINITIONS["copilot"].approvals})
        a = self.start(d)
        reply = a.ask("Say hello after pathaccess.", timeout=20, settle=1)
        self.assertTrue(reply.complete, reply.text)
        self.assertEqual(reply.approvals, 1)
        # The same question under another title is not a prompt of the definition.
        reply = a.ask("Say hello after urlaccess.", timeout=4, settle=1)
        self.assertFalse(reply.complete)
        self.assertEqual(reply.approvals, 0)
        a.quit()

    def test_a_prompt_that_stays_is_answered_once_and_the_timeout_holds(self):
        a = self.start()
        began = time.monotonic()
        reply = a.ask("This one is stuck.", timeout=5, settle=1)
        self.assertFalse(reply.complete)
        self.assertEqual(reply.approvals, 1)
        self.assertLess(time.monotonic() - began, 15)
        a.quit()

    def test_reply_is_not_complete_while_busy(self):
        a = self.start()
        reply = a.ask("Say hello and pause.", timeout=30, settle=1)
        self.assertTrue(reply.complete, reply.text)
        # The word came 3 seconds before the work ended: the reply covers the work.
        self.assertIn("LATEWORK", reply.text)
        a.quit()

    def test_reply_waits_for_the_done_text(self):
        d = Definition(**{**fake().__dict__, "done": ["Worked for"]})
        a = self.start(d)
        reply = a.ask("Say hello and delay.", timeout=30, settle=1)
        self.assertTrue(reply.complete, reply.text)
        self.assertIn("LATEWORK", reply.text)
        self.assertGreaterEqual(reply.seconds, 3)
        a.quit()

    def test_without_the_done_text_the_reply_settles_early(self):
        a = self.start()
        reply = a.ask("Say hello and delay.", timeout=30, settle=1)
        self.assertTrue(reply.complete, reply.text)
        self.assertLess(reply.seconds, 3)
        a.quit()

    def test_done_text_that_never_appears_is_not_complete(self):
        d = Definition(**{**fake().__dict__, "done": ["a text it never shows"]})
        a = self.start(d)
        reply = a.ask("Say hello.", timeout=4, settle=1)
        self.assertFalse(reply.complete)
        a.quit()

    def test_done_text_of_an_earlier_instruction_does_not_count(self):
        d = Definition(**{**fake().__dict__, "done": ["Worked for"]})
        a = self.start(d)
        self.assertTrue(a.ask("Say hello and delay.", timeout=30, settle=1).complete)
        reply = a.ask("Say hello and delay.", timeout=30, settle=1)
        self.assertTrue(reply.complete, reply.text)
        self.assertGreaterEqual(reply.seconds, 3)
        a.quit()

    def test_busy_until_the_timeout_is_not_complete(self):
        d = fake()
        d = Definition(**{**d.__dict__, "busy": ["Fake Agent"]})  # a text that stays on the screen
        a = self.start(d)
        reply = a.ask("Say hello.", timeout=4, settle=1)
        self.assertFalse(reply.complete)
        a.quit()

    def test_no_reply_times_out(self):
        a = self.start()
        began = time.monotonic()
        reply = a.ask("never reply", timeout=3, settle=1)
        self.assertFalse(reply.complete)
        self.assertLess(time.monotonic() - began, 10)
        a.quit()

    def test_not_ready(self):
        d = fake()
        d = Definition(**{**d.__dict__, "ready": ["a text it never shows"]})
        a = Agent(d, self.tmp.name)
        with self.assertRaises(AgentNotReady):
            a.start(timeout=6)
        self.assertIsNone(a.term)  # the CLI was ended


if __name__ == "__main__":
    unittest.main()
