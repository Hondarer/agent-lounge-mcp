"""A stand-in for a CLI started through a launcher, for the self-test of terminal.run: it
starts a child that inherits its output, writes the child's process ID to the file named by
its argument, and then both run for a long time."""

import subprocess
import sys
import time

child = subprocess.Popen([sys.executable, "-c", "import time; time.sleep(120)"])
with open(sys.argv[1], "w") as f:
    f.write(str(child.pid))
print("started", flush=True)
time.sleep(120)
