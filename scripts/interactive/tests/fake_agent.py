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
        for _ in range(line.count("permission")):
            out("\x1b[2J\x1b[H\x1b[1mRun a command\x1b[0m\r\nDo you want to run this command?\r\n> 1. Yes\r\n  2. No\r\n")
            if not sys.stdin.readline():
                return
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
