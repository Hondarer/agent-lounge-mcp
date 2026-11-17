"""The scenarios, by number."""

from . import guidance, interactive, unauthorized

# number: (name, function, CLIs it applies to (empty: all))
SCENARIOS = {
    "1": ("shutdown-notice", interactive.shutdown_notice, ()),
    "2": ("several-instructions", interactive.several_instructions, ()),
    "3": ("join-again", interactive.join_again, ()),
    "4": ("server-instructions", guidance.server_instructions, ()),
    "5": ("instruction-files", guidance.instruction_files, ()),
    "6": ("token-stopped", unauthorized.token_stopped, ("codex", "copilot", "agy")),
}
