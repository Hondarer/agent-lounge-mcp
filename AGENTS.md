# AGENTS.md

Instructions for coding agents working in this repository.

## Build and test

- `make build` builds `dist/agent-lounge-mcp` for the local machine; `make dist` cross-compiles all targets (Go 1.26 or later, no C compiler).
- Run `go vet ./...` and `go test ./...` before reporting a change as complete.
- Keep all SQL in `internal/store/`. Add schema changes as a new script in `internal/store/schema/`; do not edit a released script.
- Write comments in Go files, shell scripts, Makefiles, workflows, and `SKILL.md` in English. Tool names, descriptions, and error messages are in English.
- The usage guide returned by `lounge_help` is the source of truth for how agents use the server: make usage changes in `internal/tools/help.go` first.

## Documentation policy

Documents describe the current, final specification. History belongs in `docs/prompts/`.

This applies to `docs/ja/`, `README.md`, `docs/README.md`, and every `SKILL.md`.

### Languages

- `docs/ja/` is the single source of truth.
- `docs/en/` will hold English translations. It does not exist yet: until it is created, edit `docs/ja/` only, keep `README.md` and `docs/README.md` in Japanese, and do not add partial English versions.
- After `docs/en/` exists, edit `docs/ja/` first and reflect the same change in `docs/en/` in the same commit. If they differ, the Japanese version takes precedence.
- `docs/agent-lounge-mcp-proposal.md` stays in Japanese only.

### What documents contain

- **Use cases**: what the feature is for and when to use it.
- **Procedures**: the steps, options, and commands to use it now.
- **Verified scope**: what has been verified, with the method (automated test, CI, real machine) and the environment (OS, CPU, product versions). List unverified items explicitly. Write the status as 検証済 / 未検証, not 成功.
- **Limitations**: what is out of scope and known constraints.

### What documents do not contain

- Dates of work, who confirmed what, or chronological narratives.
- Log excerpts, timestamps, and investigation steps.
- Corrections of earlier text or descriptions of superseded behavior.

Dates and numbers that are part of the specification stay: supported versions, measured values with their environment, and example values.

### Updating

- When a result changes, rewrite the specification in place. Do not append a dated section.
- Record the history of a task (discussions, designs, backlogs, validation records) in `docs/prompts/current/yymmdd-<topic>-<kind>.md`, in Japanese only. `<kind>` is `discussion`, `design`, `backlog`, or `validation`.
- Move the records of a task from `docs/prompts/current/` to `docs/prompts/past/` when every item of its backlog has been resolved or carried over to another backlog. Approval of the deliverables alone is not the condition.
- Records in `docs/prompts/past/` are snapshots. Do not update them to match the current state.
- Documents do not link to records in `docs/prompts/`; records may link to documents.
