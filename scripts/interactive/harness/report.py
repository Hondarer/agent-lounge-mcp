"""Results of the scenarios, as one JSON file per scenario and CLI."""

from __future__ import annotations

import json
import time
from dataclasses import asdict, dataclass, field
from pathlib import Path
from typing import Any


@dataclass
class Result:
    scenario: str
    agent: str
    environment: dict[str, str] = field(default_factory=dict)
    # One of: done (every step ran; see values), unverified (the harness could not drive the
    # CLI or a step could not be judged; see reason), skipped.
    status: str = "done"
    reason: str = ""
    values: dict[str, Any] = field(default_factory=dict)
    started: str = field(default_factory=lambda: time.strftime("%Y-%m-%dT%H:%M:%S%z"))
    seconds: float = 0

    def write(self, base: Path) -> Path:
        path = base / self.scenario / self.agent / "result.json"
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(json.dumps(asdict(self), ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
        return path


def summary(base: Path) -> list[dict[str, Any]]:
    """Every result under base, in scenario and CLI order."""
    out = []
    for path in sorted(base.glob("*/*/result.json")):
        out.append(json.loads(path.read_text(encoding="utf-8")))
    return out
