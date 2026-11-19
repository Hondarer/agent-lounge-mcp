"""Self-tests of the harness that need no agent CLI and no server. They run on Linux, WSL and
Windows (Windows needs pywinpty):

  python3 -m unittest discover -s scripts/interactive/tests
"""

import os
import sys
import tempfile
import time
import unittest
from pathlib import Path

HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE.parent))

from harness import screen  # noqa: E402
from harness.agents import ENTER, Agent, AgentNotReady, Definition, clean_environment  # noqa: E402
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


def fake(dialog: bool = False) -> Definition:
    return Definition(
        name="fake", title="Fake", executable=sys.executable,
        interactive=["-u", str(HERE / "fake_agent.py")],
        one_shot=lambda p: [], register=lambda u, t: [], unregister=[], list_servers=[],
        ready=["? for shortcuts"], dialogs=[("Yes, I trust this folder", [ENTER])],
        quit_keys=["quit\r"], env={"FAKE_DIALOG": "1"} if dialog else {},
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
