"""Turning terminal output into plain text, and finding text in it.

There are two views of the output. plain is everything the program wrote, in order, without
control sequences. Full-screen programs redraw the screen with cursor movements and often write
only the cells that changed, so that text is not what a user sees; Screen is a small terminal
emulator that gives the screen as it looks, and the lines that scrolled off it.

A reply is known to be complete when it contains a completion word, written in capital
letters: the instruction gives it in small letters only, so the word appears only once the
agent writes it, never in the echo of the instruction (see completion). Some programs redraw a
spinner between the characters of a reply, so a word may arrive with a few other characters
inside it; contains allows for that.
"""

from __future__ import annotations

import re
import secrets
import unicodedata

# Cursor movements to the right (CSI n C): some programs draw the space between words this way.
_FORWARD = re.compile(r"\x1b\[(\d*)C")
# Other escape sequences: CSI (ESC [ ... final), OSC (ESC ] ... BEL or ESC \), DCS/SOS/PM/APC
# strings, character set designations (ESC ( B), and two-character escapes (ESC 7, ESC M).
_ESCAPES = re.compile(
    r"\x1b\[[0-?]*[ -/]*[@-~]"
    r"|\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)"
    r"|\x1b[PX^_][^\x1b]*\x1b\\"
    r"|\x1b[ -/]+[0-~]"
    r"|\x1b[0-~]"
    r"|\x9b[0-?]*[ -/]*[@-~]"
)
# Control characters other than tab and line feed.
_CONTROLS = re.compile(r"[\x00-\x08\x0b-\x1f\x7f]")


def plain(raw: str) -> str:
    """raw without escape sequences and control characters; carriage returns become line feeds."""
    text = _FORWARD.sub(lambda m: " " * int(m.group(1) or 1), raw)
    text = _ESCAPES.sub("", text)
    text = text.replace("\r\n", "\n").replace("\r", "\n")
    return _CONTROLS.sub("", text)


def words(raw: str) -> int:
    """The number of letters and digits raw output shows once control sequences are removed."""
    return sum(1 for c in plain(raw) if c.isalnum())


def squeeze(text: str) -> str:
    """text without white space, for finding words that a redraw split over lines or cells."""
    return re.sub(r"\s+", "", text)


# How many other characters a word may have inside it and still be found by contains.
SLACK = 8


def contains(text: str, needle: str, slack: int = SLACK) -> bool:
    """Whether plain text contains needle, also with up to slack other characters inside it."""
    if needle in text:
        return True
    return _scattered(squeeze(text), squeeze(needle), slack)


def _scattered(text: str, needle: str, slack: int) -> bool:
    if not needle:
        return True
    start = text.find(needle[0])
    while start != -1:
        limit = start + len(needle) + slack
        i, j = start, 0
        while i < len(text) and i < limit and j < len(needle):
            if text[i] == needle[j]:
                j += 1
            i += 1
        if j == len(needle):
            return True
        start = text.find(needle[0], start + 1)
    return False


def token() -> str:
    """A new random token of 8 small hexadecimal letters and digits, starting with a letter."""
    return "abcdef"[secrets.randbelow(6)] + secrets.token_hex(4)[:7]


def completion() -> tuple[str, str]:
    """A new completion word, and the sentence that asks the agent to end its reply with it."""
    t = token()
    word = "IVDONE" + t.upper()
    return word, (f'When you have finished, end your reply with the word "ivdone" followed by "{t}", '
                  f"all in capital letters, with nothing between them.")


def width(ch: str) -> int:
    """Columns a character takes on a terminal: 0 for combining marks, 2 for wide characters."""
    if unicodedata.combining(ch) or ch in "\u200b\u200c\u200d\ufe0f":
        return 0
    return 2 if unicodedata.east_asian_width(ch) in ("W", "F") else 1


_SEQUENCE = re.compile(
    r"\x1b\[([0-?]*)([ -/]*)([@-~])"  # CSI: parameters, intermediates, final
    r"|\x1b\][^\x07\x1b]*(?:\x07|\x1b\\)"  # OSC
    r"|\x1b[PX^_][^\x1b]*\x1b\\"  # DCS, SOS, PM, APC
    r"|\x1b([ -/]+)[0-~]"  # character set designation
    r"|\x1b(?![\[\]PX^_])([0-~])"  # two-character escape
)


class Screen:
    """A minimal terminal: enough of the VT100 and xterm controls to show what a full-screen
    program drew. Lines that scroll off the top go to history, so that the transcript keeps
    what an inline program printed earlier. Colours and modes other than the alternate screen
    are ignored."""

    def __init__(self, columns: int, rows: int):
        self.columns, self.rows = columns, rows
        self.history: list[str] = []
        self._main: list[list[str]] | None = None
        self.grid = self._blank_grid()
        self.x = self.y = 0
        self.top, self.bottom = 0, rows - 1
        self._saved = (0, 0)
        self._pending = ""

    def _blank_grid(self) -> list[list[str]]:
        return [[" "] * self.columns for _ in range(self.rows)]

    def lines(self) -> list[str]:
        """The visible screen, one string per row, without trailing spaces."""
        return ["".join(c for c in row if c).rstrip() for row in self.grid]

    def display(self) -> str:
        return "\n".join(self.lines())

    def transcript(self) -> str:
        """History followed by the visible screen."""
        return "\n".join(self.history + self.lines())

    def feed(self, data: str) -> None:
        data = self._pending + data
        self._pending = ""
        i = 0
        n = len(data)
        while i < n:
            ch = data[i]
            if ch == "\x1b":
                m = _SEQUENCE.match(data, i)
                if not m:
                    # Possibly a sequence split between two reads. A string sequence (a title,
                    # a hyperlink) can be long.
                    limit = 4096 if data[i + 1:i + 2] in ("]", "P", "X", "^", "_") else 64
                    if n - i < limit:
                        self._pending = data[i:]
                        return
                    i += 1
                    continue
                self._escape(m)
                i = m.end()
                continue
            if ch == "\r":
                self.x = 0
            elif ch == "\n" or ch == "\x0b" or ch == "\x0c":
                self._line_feed()
            elif ch == "\b":
                self.x = max(0, self.x - 1)
            elif ch == "\t":
                self.x = min(self.columns - 1, (self.x // 8 + 1) * 8)
            elif ch >= " " and ch != "\x7f":
                self._put(ch)
            i += 1

    def _put(self, ch: str) -> None:
        w = width(ch)
        if w == 0:
            if self.x > 0:
                self.grid[self.y][self.x - 1] += ch
            return
        if self.x + w > self.columns:
            self.x = 0
            self._line_feed()
        self.grid[self.y][self.x] = ch
        if w == 2 and self.x + 1 < self.columns:
            self.grid[self.y][self.x + 1] = ""
        self.x += w

    def _line_feed(self) -> None:
        if self.y == self.bottom:
            self._scroll_up(1)
        elif self.y < self.rows - 1:
            self.y += 1

    def _scroll_up(self, count: int) -> None:
        for _ in range(count):
            line = self.grid.pop(self.top)
            if self.top == 0 and self._main is None:
                self.history.append("".join(line).rstrip())
            self.grid.insert(self.bottom, [" "] * self.columns)

    def _scroll_down(self, count: int) -> None:
        for _ in range(count):
            self.grid.pop(self.bottom)
            self.grid.insert(self.top, [" "] * self.columns)

    def _escape(self, m: re.Match) -> None:
        if m.group(3) is not None:
            self._csi(m.group(1), m.group(2), m.group(3))
            return
        two = m.group(5)
        if two == "7":
            self._saved = (self.x, self.y)
        elif two == "8":
            self.x, self.y = self._saved
        elif two == "D":
            self._line_feed()
        elif two == "E":
            self.x = 0
            self._line_feed()
        elif two == "M":
            if self.y == self.top:
                self._scroll_down(1)
            else:
                self.y = max(0, self.y - 1)
        elif two == "c":
            self.grid = self._blank_grid()
            self.x = self.y = 0

    def _csi(self, params: str, inter: str, final: str) -> None:
        private = params[:1] in ("?", ">", "<", "=")
        nums = [int(p) if p.isdigit() else 0 for p in params.lstrip("?><=").split(";")] if params else []

        def arg(i: int = 0, default: int = 1) -> int:
            return nums[i] if i < len(nums) and nums[i] else default

        if private:
            if final in "hl" and set(nums) & {47, 1047, 1049}:
                self._alternate(final == "h")
            return
        if inter:
            return
        if final == "A":
            self.y = max(self.top if self.y >= self.top else 0, self.y - arg())
        elif final in "Be":
            self.y = min(self.bottom if self.y <= self.bottom else self.rows - 1, self.y + arg())
        elif final in "Ca":
            self.x = min(self.columns - 1, self.x + arg())
        elif final == "D":
            self.x = max(0, self.x - arg())
        elif final == "E":
            self.x, self.y = 0, min(self.rows - 1, self.y + arg())
        elif final == "F":
            self.x, self.y = 0, max(0, self.y - arg())
        elif final in "G`":
            self.x = min(self.columns - 1, arg() - 1)
        elif final in "Hf":
            self.y = min(self.rows - 1, arg(0) - 1)
            self.x = min(self.columns - 1, arg(1) - 1)
        elif final == "d":
            self.y = min(self.rows - 1, arg() - 1)
        elif final == "J":
            mode = arg(0, 0)
            if mode == 0:
                self.grid[self.y][self.x:] = [" "] * (self.columns - self.x)
                for r in range(self.y + 1, self.rows):
                    self.grid[r] = [" "] * self.columns
            elif mode == 1:
                for r in range(self.y):
                    self.grid[r] = [" "] * self.columns
                self.grid[self.y][: self.x + 1] = [" "] * (self.x + 1)
            else:
                if mode == 3:
                    self.history.clear()
                self.grid = self._blank_grid()
        elif final == "K":
            mode = arg(0, 0)
            row = self.grid[self.y]
            if mode == 0:
                row[self.x:] = [" "] * (self.columns - self.x)
            elif mode == 1:
                row[: self.x + 1] = [" "] * (self.x + 1)
            else:
                self.grid[self.y] = [" "] * self.columns
        elif final == "X":
            n = min(arg(), self.columns - self.x)
            self.grid[self.y][self.x:self.x + n] = [" "] * n
        elif final == "P":
            row = self.grid[self.y]
            n = min(arg(), self.columns - self.x)
            del row[self.x:self.x + n]
            row.extend([" "] * n)
        elif final == "@":
            row = self.grid[self.y]
            n = min(arg(), self.columns - self.x)
            row[self.x:self.x] = [" "] * n
            del row[self.columns:]
        elif final == "L" and self.top <= self.y <= self.bottom:
            for _ in range(arg()):
                self.grid.pop(self.bottom)
                self.grid.insert(self.y, [" "] * self.columns)
        elif final == "M" and self.top <= self.y <= self.bottom:
            for _ in range(arg()):
                self.grid.pop(self.y)
                self.grid.insert(self.bottom, [" "] * self.columns)
        elif final == "S":
            self._scroll_up(arg())
        elif final == "T":
            self._scroll_down(arg())
        elif final == "r":
            top, bottom = arg(0) - 1, arg(1, self.rows) - 1
            if 0 <= top < bottom < self.rows:
                self.top, self.bottom = top, bottom
                self.x = self.y = 0
        elif final == "s":
            self._saved = (self.x, self.y)
        elif final == "u":
            self.x, self.y = self._saved

    def _alternate(self, on: bool) -> None:
        if on and self._main is None:
            self._main = self.grid
            self.grid = self._blank_grid()
        elif not on and self._main is not None:
            self.grid = self._main
            self._main = None
