"""A stand-in for an agent CLI, for the harness's self-tests.

It behaves like the real interactive screens in the ways the harness must handle: an
optional trust dialog at start-up, a ready line, colours, a spinner drawn over the reply
line, a reply drawn out of order with cursor movements, and an animation while idle. It
answers "ivdone" followed by "<token>" with IVDONE<TOKEN>. An instruction containing
"never" gets no reply; "slow" waits before replying. It ends on "quit" or end of input.

"permission" asks for permission before the reply (a prompt that leaves the screen once it is
answered with Enter, as many times as the word occurs). "pause" shows the completion word,
then works on without output for longer than a short settle time under the status "esc to
interrupt", and writes LATEWORK when it is done. "stuck" shows a permission prompt that no
key answers.

"twin" asks for permission for the same command once per occurrence of the word, like two
calls made at once: after an answer the prompt is not redrawn at all, and only a line about
the remaining requests (written, then erased) and the state of the call above the prompt
change. "linger" asks once, and after the answer leaves the prompt on the screen for 5
seconds, drawing only symbols, before it removes it.
"""

import os
import re
import sys
import time


def out(text: str) -> None:
    sys.stdout.write(text)
    sys.stdout.flush()


def main() -> None:
    if os.environ.get("FAKE_DIALOG"):
        out("Do you trust this folder?\r\n  No, exit\r\n> Yes, I trust this folder\r\n")
        if not sys.stdin.readline():
            return
        out("\x1b[2J\x1b[H")
    out("\x1b[1mFake Agent\x1b[0m\r\n\x1b[2m? for shortcuts\x1b[0m\r\n")
    while True:
        out("\x1b[36m>\x1b[0m ")
        line = sys.stdin.readline()
        if not line or line.strip() == "quit":
            return
        if "never" in line:
            continue
        if "slow" in line:
            time.sleep(3)
        if "stuck" in line:
            out("Do you want to run this command?\r\n> 1. Yes\r\n")
            while sys.stdin.readline().strip() != "quit":
                pass
            return
        # Like a CLI that redraws only what changed: the question is written once, and each
        # further prompt rewrites only the line above it (the command).
        for n in range(line.count("permission")):
            if n == 0:
                out("\x1b[2J\x1b[H\x1b[1mRun a command\x1b[0m\r\ncommand 0\r\nDo you want to run this command?\r\n> 1. Yes\r\n  2. No\r\n")
            else:
                out(f"\x1b[2;1H\x1b[2Kcommand {n}\x1b[6;1H")
            if not sys.stdin.readline():
                return
        if "permission" in line:
            out("\x1b[2J\x1b[H")
        twins = line.count("twin")
        for n in range(twins):
            if n == 0:
                # The state of the call (line 2) is further above the question than the lines
                # that belong to the prompt.
                out("\x1b[2J\x1b[H\x1b[1mRun a command\x1b[0m\r\n" + "\r\n" * 11 + "Touch file\r\ntouch same.txt\r\n"
                    "Do you want to run this command?\r\n> 1. Yes\r\n  2. No\r\n")
            else:
                out(f"\x1b[20;1HPermission request ({twins - n} remaining)\x1b[20;1H\x1b[2K"
                    f"\x1b[2;1H$ No Content {n}\x1b[18;1H")
            if not sys.stdin.readline():
                return
        if twins:
            out("\x1b[2J\x1b[H")
        if "linger" in line:
            out("\x1b[2J\x1b[H\x1b[1mRun a command\x1b[0m\r\ncommand 0\r\nDo you want to run this command?\r\n> 1. Yes\r\n  2. No\r\n")
            if not sys.stdin.readline():
                return
            for i in range(50):
                out("\x1b7\x1b[20;1H" + "|/-\\"[i % 4] + "\x1b8")  # below the prompt
                time.sleep(0.1)
            out("\x1b[2J\x1b[H")
        m = re.search(r'followed by "([0-9a-f]+)"', line)
        word = "IVDONE" + m.group(1).upper() if m else "no word asked"
        # A spinner on the line the reply will take, then the reply drawn right half first.
        for frame in "|/-\\|/-\\":
            out(f"\r\x1b[33m{frame}\x1b[0m Working")
            time.sleep(0.05)
        half = len(word) // 2
        out("\r\x1b[2K")
        out(f"\x1b[{8 + half}G{word[half:]}")  # "hello: " takes 7 columns; columns count from 1
        out(f"\x1b[1Ghello: {word[:half]}")
        out("\r\n")
        if "pause" in line:
            out("esc to interrupt")
            time.sleep(3)
            out("\r\x1b[2KLATEWORK\r\n")
        # An idle animation of symbols only.
        for frame in "◐◓◑◒":
            out(f"\x1b7\x1b[1;40H{frame}\x1b8")
            time.sleep(0.05)


if __name__ == "__main__":
    main()
