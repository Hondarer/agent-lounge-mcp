#!/usr/bin/env python3
"""Drive agent CLIs on their interactive screens to check how they use agent-lounge-mcp.

This is a development and test tool of this repository; it is not part of the release.
See docs/ja/interactive-verification.md.

  verify.py run --server PATH [--agents claude,codex] [--scenarios 1,2] [--out DIR]
  verify.py build-test-server --out FILE [--goos windows]   (needs Go; for scenario 4)
  verify.py summary DIR

Every run works in a new check folder (under the system's temporary folder unless --out is
given): databases, working folders of the CLIs, transcripts and results stay there. Each CLI
is registered under the name agent-lounge before its scenarios and unregistered after them;
a CLI that already has that name registered is skipped, so that a user's registration is
never replaced.
"""

from __future__ import annotations

import argparse
import json
import os
import shutil
import subprocess
import sys
import tempfile
import time
import traceback
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parent))

from harness import screen, terminal  # noqa: E402
from harness.agents import DEFINITIONS, SERVER, AgentNotReady, clean_environment, command  # noqa: E402
from harness.report import Result, summary  # noqa: E402
from harness.scenarios import SCENARIOS  # noqa: E402
from harness.scenarios.common import Context, environment  # noqa: E402

REPO = Path(__file__).resolve().parents[2]


def _cli(definition, *args: str, timeout: float = 60) -> subprocess.CompletedProcess:
    return terminal.run([command(definition.executable), *args], timeout, text=True, encoding="utf-8",
                        errors="replace", env=clean_environment())


def _registered(definition) -> bool:
    """Whether the CLI has a server named agent-lounge. Raises when the CLI cannot list its
    servers: a registration of the user must not be replaced on a guess."""
    out = _cli(definition, *definition.list_servers)
    if out.returncode != 0:
        raise RuntimeError(f"listing the servers failed: {(out.stderr or out.stdout).strip()[:500]}")
    return SERVER in (out.stdout + out.stderr)


def run(args: argparse.Namespace) -> int:
    base = Path(args.out) if args.out else Path(tempfile.mkdtemp(prefix="agent-lounge-iv-"))
    base.mkdir(parents=True, exist_ok=True)
    binary = str(Path(args.server).resolve())
    test_binary, passphrase = "", ""
    if args.test_server:
        test_binary = str(Path(args.test_server).resolve())
        passphrase = json.loads(Path(test_binary + ".json").read_text(encoding="utf-8"))["passphrase"]
    names = args.agents.split(",") if args.agents else list(DEFINITIONS)
    numbers = args.scenarios.split(",") if args.scenarios else list(SCENARIOS)
    print(f"check folder: {base}", flush=True)
    unknown = [n for n in names if n not in DEFINITIONS] + [n for n in numbers if n not in SCENARIOS]
    if unknown:
        print(f"unknown agent or scenario: {', '.join(unknown)}", file=sys.stderr)
        return 2
    for name in names:
        d = DEFINITIONS[name]
        if shutil.which(d.executable) is None:
            print(f"{name}: not installed, skipped", flush=True)
            continue
        try:
            if _registered(d):
                print(f"{name}: a server named {SERVER} is already registered; skipped, nothing changed", flush=True)
                continue
        except (RuntimeError, subprocess.TimeoutExpired) as e:
            print(f"{name}: skipped, nothing changed: {e}", flush=True)
            continue
        url = f"http://{args.listen}/mcp"
        reg = _cli(d, *d.register(url, ""))
        if reg.returncode != 0:
            print(f"{name}: registering failed: {reg.stderr.strip()}", flush=True)
            continue
        try:
            for number in numbers:
                title, function, only = SCENARIOS[number]
                if only and name not in only:
                    continue
                result = Result(scenario=f"{number}-{title}", agent=name, environment=environment(d, binary))
                ctx = Context(definition=d, base=base, binary=binary, listen=args.listen, result=result,
                              test_binary=test_binary, test_passphrase=passphrase)
                began = time.monotonic()
                print(f"{name}: scenario {number} ({title}) ...", flush=True)
                try:
                    function(ctx)
                except AgentNotReady as e:
                    result.status, result.reason = "unverified", f"the harness could not drive the CLI: {e}"[:4000]
                except Exception as e:  # keep going with the other scenarios
                    # The text of an exception can hold the command line, and with it a token.
                    result.status, result.reason = "unverified", ctx.scrub(f"{type(e).__name__}: {e}")[:4000]
                    ctx.save("error.txt", traceback.format_exc())
                finally:
                    if d.approvals:
                        result.values["approvals"] = ctx.approvals()
                    ctx.close()
                    result.seconds = round(time.monotonic() - began, 1)
                    result.write(base)
                    print(f"{name}: scenario {number}: {result.status} {result.reason[:200]}", flush=True)
                # Scenario 6 registers with a token and unregisters; register again for the next.
                if not _registered(d):
                    _cli(d, *d.register(url, ""))
        finally:
            try:
                _cli(d, *d.unregister)
                if _registered(d):
                    print(f"{name}: warning: {SERVER} is still registered", flush=True)
            except (RuntimeError, subprocess.TimeoutExpired) as e:
                print(f"{name}: warning: could not confirm that {SERVER} was unregistered: {e}", flush=True)
    print(json.dumps(summary(base), ensure_ascii=False, indent=1))
    return 0


def build_test_server(args: argparse.Namespace) -> int:
    """Build agent-lounge-mcp with a passphrase added to the instructions it sends on connect.

    The source is copied to a temporary folder and changed there: Instructions is renamed and
    wrapped by one that appends the sentence, set at link time. The repository is not changed.
    """
    passphrase = screen.token()
    out = Path(args.out).resolve()
    with tempfile.TemporaryDirectory(prefix="agent-lounge-iv-src-") as tmp:
        src = Path(tmp) / "src"
        files = subprocess.run(["git", "ls-files", "-z"], cwd=REPO, capture_output=True, check=True).stdout
        for rel in filter(None, files.decode().split("\0")):
            target = src / rel
            target.parent.mkdir(parents=True, exist_ok=True)
            if (REPO / rel).is_file():
                shutil.copy2(REPO / rel, target)
        tools = src / "internal" / "tools" / "tools.go"
        text = tools.read_text(encoding="utf-8")
        if "func Instructions(" not in text:
            print("internal/tools/tools.go has no func Instructions", file=sys.stderr)
            return 1
        tools.write_text(text.replace("func Instructions(", "func instructionsWithoutPassphrase(", 1), encoding="utf-8")
        (tools.parent / "zz_passphrase.go").write_text(
            "package tools\n\n// passphrase is set with -ldflags -X by verify.py build-test-server.\n"
            "var passphrase string\n\n"
            "func Instructions(d *Deps) string {\n"
            '\treturn instructionsWithoutPassphrase(d) + " The passphrase of this lounge is " + passphrase + "."\n'
            "}\n", encoding="utf-8")
        env = dict(os.environ, CGO_ENABLED="0")
        if args.goos:
            env["GOOS"] = args.goos
            env["GOARCH"] = args.goarch
        subprocess.run(["go", "build", "-trimpath", "-ldflags", f"-X github.com/hondarer/agent-lounge-mcp/internal/tools.passphrase={passphrase}",
                        "-o", str(out), "./cmd/agent-lounge-mcp"], cwd=src, env=env, check=True)
    Path(str(out) + ".json").write_text(json.dumps({"passphrase": passphrase}) + "\n", encoding="utf-8")
    print(f"built {out} (the passphrase is in {out}.json)")
    return 0


def main() -> int:
    p = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    sub = p.add_subparsers(dest="command", required=True)
    r = sub.add_parser("run", help="run scenarios")
    r.add_argument("--server", required=True, help="agent-lounge-mcp executable")
    r.add_argument("--test-server", help="executable from build-test-server, for scenario 4")
    r.add_argument("--agents", help=f"comma-separated, from {','.join(DEFINITIONS)} (default: all)")
    r.add_argument("--scenarios", help=f"comma-separated, from {','.join(SCENARIOS)} (default: all)")
    r.add_argument("--listen", default="127.0.0.1:7799", help="address of the check server")
    r.add_argument("--out", help="check folder (default: a new temporary folder)")
    b = sub.add_parser("build-test-server", help="build the executable of scenario 4 (needs Go)")
    b.add_argument("--out", required=True)
    b.add_argument("--goos", help="target operating system, such as windows")
    b.add_argument("--goarch", default="amd64")
    s = sub.add_parser("summary", help="print the results under a check folder")
    s.add_argument("folder")
    args = p.parse_args()
    if args.command == "run":
        return run(args)
    if args.command == "build-test-server":
        return build_test_server(args)
    print(json.dumps(summary(Path(args.folder)), ensure_ascii=False, indent=1))
    return 0


if __name__ == "__main__":
    sys.exit(main())
