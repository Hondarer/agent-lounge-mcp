"""Self-tests of the harness that need no agent CLI and no server. They run on Linux, WSL and
Windows (Windows needs pywinpty):

  python3 -m unittest discover -s scripts/interactive/tests
"""

import os
import random
import subprocess
import sys
import tempfile
import time
import unittest
from pathlib import Path

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE.parent))

from harness import screen, terminal  # noqa: E402
from harness.agents import CTRL_U, ENTER, Agent, AgentNotReady, Definition, clean_environment  # noqa: E402
from harness.scenarios.common import answer, connections, new_lines  # noqa: E402
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


class AgentTest(unittest.TestCase):
    def setUp(self):
        self.tmp = tempfile.TemporaryDirectory()

    def tearDown(self):
        self.tmp.cleanup()

    def test_dialog_instruction_and_quit(self):
        a = Agent(fake(dialog=True), self.tmp.name).start(timeout=30)
        reply = a.ask("Say hello.", timeout=20, settle=1)
        self.assertTrue(reply.complete, reply.text)
        reply = a.ask("Say hello slow.", timeout=20, settle=1)
        self.assertTrue(reply.complete, reply.text)
        term = a.term
        a.quit()
        self.assertFalse(term.alive())

    def test_permission_prompts_are_answered_and_counted(self):
        a = Agent(fake(), self.tmp.name).start(timeout=30)
        # Three prompts in a row, of which only the command line is redrawn.
        began = time.monotonic()
        reply = a.ask("Say hello after permission and permission and permission.", timeout=40, settle=1)
        self.assertTrue(reply.complete, reply.text)
        self.assertEqual((reply.approvals, a.approvals), (3, 3))
        self.assertLess(time.monotonic() - began, 14)  # no prompt waited for the retry time
        reply = a.ask("Say hello.", timeout=20, settle=1)
        self.assertEqual((reply.approvals, a.approvals), (0, 3))
        a.quit()

    def test_a_prompt_that_stays_is_answered_once_and_the_timeout_holds(self):
        a = Agent(fake(), self.tmp.name).start(timeout=30)
        began = time.monotonic()
        reply = a.ask("This one is stuck.", timeout=5, settle=1)
        self.assertFalse(reply.complete)
        self.assertEqual(reply.approvals, 1)
        self.assertLess(time.monotonic() - began, 15)
        a.quit()

    def test_reply_is_not_complete_while_busy(self):
        a = Agent(fake(), self.tmp.name).start(timeout=30)
        reply = a.ask("Say hello and pause.", timeout=30, settle=1)
        self.assertTrue(reply.complete, reply.text)
        # The word came 3 seconds before the work ended: the reply covers the work.
        self.assertIn("LATEWORK", reply.text)
        a.quit()

    def test_busy_until_the_timeout_is_not_complete(self):
        d = fake()
        d = Definition(**{**d.__dict__, "busy": ["Fake Agent"]})  # a text that stays on the screen
        a = Agent(d, self.tmp.name).start(timeout=30)
        reply = a.ask("Say hello.", timeout=4, settle=1)
        self.assertFalse(reply.complete)
        a.quit()

    def test_no_reply_times_out(self):
        a = Agent(fake(), self.tmp.name).start(timeout=30)
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
