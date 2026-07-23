---
date: 2026-07-23T19:25:46.815551+00:00
git_commit: ""
branch: main
topic: "agentenv CLI"
tags: [plan, cli, go, agentenv, e2e]
status: ready
---

# PLAN: Build the agentenv CLI

Create `agentenv`, an agent-agnostic CLI that launches AI coding agents with a project-specific identity by running them with an isolated profile `HOME`. The first version starts from an empty repository and provides `run`, `wrap`, and `doctor` commands with end-to-end test coverage.

## Acceptance Criteria

- `agentenv run <agent> [args...]` launches any real agent binary found on `PATH` with a profile-specific `HOME`.
- `agentenv run <agent> [args...]` is agent-agnostic; `pi`, `claude`, `codex`, or any other executable name are examples only.
- `agentenv run` resolves the real agent by searching `PATH` while skipping the agentenv wrapper bin directory to avoid wrapper recursion.
- On first use in an unmapped project, `agentenv run` opens an interactive Bubble Tea TUI to select or create a profile, then stores the local project-to-profile mapping.
- Profiles have only a name in v1; no subscription label, account label, or secrets are stored by agentenv.
- Isolation in v1 sets only `HOME=$AGENTENV_HOME/profiles/<profile>/home`; no XDG variables are modified.
- `agentenv wrap <agent>` creates a wrapper in an explicit agentenv bin directory, without overwriting real agent binaries.
- The wrapper directory is intended to be placed before the real agent binary on `PATH`.
- `agentenv doctor [agent?]` checks local config, mapping, profile-home paths, wrapper/PATH state, and runs a light probe against the resolved real agent path such as `/path/to/real-agent --version` when an agent is provided.
- E2E tests cover `run`, `wrap`, and `doctor` using fake agents and temporary config/data directories.
- The repository contains Go project scaffolding, `mise` configuration, README documentation, and reproducible test commands.

## Technical Key Decisions and Tradeoffs

1. **Implementation language:** Go CLI developed via `mise`.
   - Why: Go fits process execution, filesystem operations, and single-binary distribution well while working naturally with Bubble Tea and Lip Gloss.
   - Impact: Add `go.mod`, Go package structure, and `.mise.toml` tasks/tools.

2. **TUI stack:** Bubble Tea for interactive flow and Lip Gloss for terminal styling.
   - Why: The desired UI is a terminal-first selector/banner experience.
   - Impact: Keep TUI logic isolated behind interfaces so E2E tests can avoid brittle terminal automation where possible.

3. **Agent model:** agent-agnostic executable names.
   - Why: The tool should isolate any coding agent, not just `pi`.
   - Impact: CLI syntax uses `<agent>` as a positional argument and command logic never hard-codes agent-specific paths or behavior.

4. **Profile discovery:** local path-to-profile mapping only.
   - Why: Customer/account associations should not be committed to project repositories.
   - Impact: Store mappings under the user config directory, not in `.agentenv.toml` inside the project.

5. **Isolation scope:** set only `HOME` in v1.
   - Why: Minimal, understandable, and directly addresses most agent auth/session state.
   - Impact: Do not set `XDG_CONFIG_HOME`, `XDG_CACHE_HOME`, `XDG_DATA_HOME`, or `XDG_STATE_HOME` in v1.

6. **Wrapper strategy:** explicit agentenv bin directory plus PATH ordering.
   - Why: Avoid overwriting package-manager-installed agent binaries.
   - Impact: `wrap` writes scripts into the agentenv bin dir and `doctor` validates that this directory appears before the real binary directory on `PATH`.

7. **Real binary resolution:** search `PATH` while skipping the agentenv wrapper bin directory.
   - Why: Prevent `pi -> wrapper -> agentenv run pi -> wrapper` recursion.
   - Impact: Implement a custom executable lookup rather than using `exec.LookPath` directly.

8. **Doctor scope:** local checks plus light resolved-agent probe.
   - Why: Useful for diagnosing wrapper/PATH/config issues while staying offline and privacy-conscious.
   - Impact: `doctor [agent?]` resolves the real agent while skipping the agentenv wrapper bin directory, then may execute `/resolved/path/to/agent --version`; command output must be summarized without requiring cloud login.

## Current State

The repository is empty except for Git metadata:

```text
agent-env/
└── .git/
```

There is no existing CLI, package manifest, README, test harness, or project guidance.

## Desired End State

```text
agent-env/
├── .mise.toml
├── README.md
├── go.mod
├── go.sum
├── cmd/agentenv/main.go
├── internal/
│   ├── app/
│   ├── cli/
│   ├── config/
│   ├── doctor/
│   ├── paths/
│   ├── runner/
│   ├── tui/
│   └── wrapper/
└── test/e2e/
    ├── run_test.go
    ├── wrap_test.go
    └── doctor_test.go
```

Runtime flow:

```text
agentenv run <agent> [args...]
      │
      ▼
load config from AGENTENV_CONFIG_HOME or OS user config dir
      │
      ▼
find mapping for current project path
      │
      ├─ found: use mapped profile
      │
      └─ missing: Bubble Tea profile select/create TUI -> save mapping
      ▼
ensure $AGENTENV_HOME/profiles/<profile>/home exists
      │
      ▼
resolve real <agent> from PATH, skipping agentenv bin dir
      │
      ▼
print compact banner
      │
      ▼
exec real agent with HOME set to profile home
```

Wrapper flow:

```text
agentenv wrap <agent>
      │
      ▼
write $AGENTENV_HOME/bin/<agent>
      │
      ▼
wrapper runs: /absolute/path/to/agentenv run <agent> "$@"
      │
      ▼
doctor validates PATH ordering and real binary resolution
```

Example banner for v1:

```text
┌─ agentenv ───────────────────────────────────┐
│ customer-a • pi • ✓                          │
└──────────────────────────────────────────────┘
```

Example first-run TUI:

```text
┌─ agentenv ───────────────────────────────────────────────
│ Profile: unmapped • Agent: pi
├──────────────────────────────────────────────────────────
│
│  Select Profile
│
│  > customer-a
│    customer-b
│    personal
│    + create new profile
│
└──────────────────────────────────────────────────────────
```

## Abstractions and Code Reuse

New abstractions:

- `internal/paths`
  - Resolves config directory, data/home directory, wrapper bin directory, and profile home directories.
  - Supports test overrides through environment variables such as `AGENTENV_CONFIG_HOME` and `AGENTENV_HOME`.
- `internal/config`
  - Loads/saves local config TOML with profiles and project mappings.
  - Normalizes project paths before lookup.
- `internal/runner`
  - Resolves real agent binaries with wrapper-bin skipping.
  - Executes the agent with isolated `HOME` and passthrough args/stdin/stdout/stderr.
- `internal/tui`
  - Provides profile selection and creation UI using Bubble Tea/Lip Gloss.
  - Exposes a narrow `ProfilePrompter` interface so command logic can be tested without an actual TUI.
- `internal/wrapper`
  - Creates portable shell wrappers safely.
  - Embeds the absolute path of the currently running `agentenv` executable to avoid depending on PATH lookup for `agentenv` itself.
- `internal/doctor`
  - Performs structured checks and returns machine-testable status.
- `test/e2e`
  - Builds the `agentenv` binary once per test package.
  - Creates fake agents in temp `PATH` directories.
  - Uses temp config/data directories to avoid touching the developer machine.

## Logging & Observability

- Normal `run` output should be quiet except for the short banner before handing control to the real agent.
- `doctor` should provide human-readable check rows with status markers:

```text
agentenv doctor pi

✓ config readable: /tmp/.../config/agentenv/config.toml
✓ project mapping: /repo/customer-a -> customer-a
✓ profile home: /tmp/.../profiles/customer-a/home
✓ wrapper bin in PATH before real agent
✓ real agent: /tmp/.../real-bin/pi
✓ probe: pi --version exited 0
```

- Failures should be actionable:

```text
✗ wrapper PATH: /tmp/.../agentenv/bin is not before /tmp/.../real-bin
  fix: add /tmp/.../agentenv/bin before the real agent directory in PATH
```

## Implementation

### Phase 1: Scaffold Go project, mise, and CLI skeleton

Dependencies: None.

Create the repository foundation and a minimal command dispatch layer.

**Tasks**:
- [x] Add `.mise.toml` with Go tool version and tasks for `test`, `test:e2e`, `fmt`, and `build`.
- [x] Add `go.mod` for the project module.
- [x] Add dependencies for Bubble Tea, Lip Gloss, and TOML parsing.
- [x] Create `cmd/agentenv/main.go` as the binary entry point.
- [x] Create `internal/cli` with command parsing for `run`, `wrap`, `doctor`, `help`, and version output.
- [x] Ensure unknown commands and missing arguments return non-zero with concise usage text.
- [x] Add a root `.gitignore` for build outputs, coverage files, and local temporary data.

**Automated Verification**:
- [x] `mise run fmt` succeeds.
- [x] `mise run build` creates an `agentenv` binary.
- [x] `mise run test` succeeds for initial unit tests.
- [x] Running `agentenv` without arguments prints usage and exits non-zero.

### Phase 2: Implement config, paths, and profile filesystem model

Dependencies: Phase 1.

Add local persistence for profiles and project mappings without touching project repositories.

**Tasks**:
- [x] Implement `internal/paths` with OS defaults and environment overrides for tests.
- [x] Define default locations and document terminology clearly:
  - [x] config file: `${AGENTENV_CONFIG_HOME:-user config dir}/agentenv/config.toml`.
  - [x] agentenv data root: `${AGENTENV_HOME:-user data dir/agentenv}`.
  - [x] wrapper bin dir: `$AGENTENV_HOME/bin`.
  - [x] isolated profile `HOME`: `$AGENTENV_HOME/profiles/<profile>/home`.
  - [x] Note that `AGENTENV_HOME` is agentenv's data root, not the `HOME` value passed to agents.
- [x] Implement project path normalization for mapping keys.
- [x] Implement `internal/config` TOML schema with profiles and project mappings.
- [x] Implement atomic config save to reduce corruption risk.
- [x] Implement profile validation: non-empty, filesystem-safe names, duplicate detection.
- [x] Implement helpers to ensure profile home directories exist.
- [x] Add unit tests for path resolution, config load/save, path normalization, and profile validation.

**Automated Verification**:
- [x] `mise run test` passes config/path unit tests.
- [x] Config save/load roundtrips profiles and mappings.
- [x] Invalid profile names are rejected with clear errors.

### Phase 3: Implement `run <agent> [args...]` with HOME isolation and E2E coverage

Dependencies: Phase 2.

Deliver the core non-interactive execution path for already-mapped projects.

**Tasks**:
- [x] Implement `internal/runner.LookupAgent` that searches `PATH` and skips the agentenv wrapper bin directory.
- [x] Implement `internal/runner.RunAgent` that invokes the real agent with stdin/stdout/stderr connected and `HOME` set to the selected profile home.
- [x] Preserve passthrough arguments exactly after `<agent>`.
- [x] Preserve the agent exit code as the `agentenv run` exit code.
- [x] Print the compact v1 banner before execution.
- [x] Add explicit non-interactive mode via `AGENTENV_NONINTERACTIVE=1`.
- [x] Make `run` fail clearly when no project mapping exists and `AGENTENV_NONINTERACTIVE=1` is set, instead of launching the TUI.
- [x] Add E2E test harness that builds `agentenv` into a temp directory.
- [x] Add fake agent helper scripts that record `HOME`, argv, working directory, and environment to files.
- [x] Add `test/e2e/run_test.go` covering mapped project execution.
- [x] Add E2E coverage for PATH lookup skipping the wrapper bin directory.
- [x] Add E2E coverage for passthrough args and non-zero agent exit code propagation.

**Automated Verification**:
- [x] `mise run test:e2e` passes `run` E2E tests.
- [x] Fake agent observes `HOME=$AGENTENV_HOME/profiles/<profile>/home`.
- [x] Fake agent receives all passthrough args unchanged.
- [x] `agentenv run` returns the fake agent's exit code.

**Manual Verification**:
- [ ] From a shell with a test profile mapping, run `agentenv run pi --version` and confirm the banner appears and the real `pi` command runs.

### Phase 4: Implement first-run Bubble Tea profile selection/creation TUI

Dependencies: Phase 3.

Add the interactive path for unmapped projects.

**Tasks**:
- [x] Define a `ProfilePrompter` interface used by `run` when a project mapping is missing.
- [x] Implement Bubble Tea model for selecting an existing profile.
- [x] Add `+ create new profile` path with text input and validation errors.
- [x] Style the selection screen and banner with Lip Gloss according to the pitch aesthetic.
- [x] Save the chosen/new profile mapping to local config after confirmation.
- [x] Ensure the selected profile home directory is created before launching the agent.
- [x] Add unit tests for the command flow using a fake `ProfilePrompter`.
- [x] Add E2E or integration coverage for first-run behavior through a non-interactive test seam, avoiding brittle terminal key automation.

**Automated Verification**:
- [x] `mise run test` passes profile prompt flow tests.
- [x] A missing mapping plus fake prompter creates a profile, saves the mapping, and launches the fake agent with the new profile HOME.
- [x] Profile creation rejects invalid names before saving.

**Manual Verification**:
- [ ] In a new temp project, run `agentenv run pi`, create/select a profile in the TUI, and confirm the mapping is reused on the second run without prompting.

### Phase 5: Implement `wrap <agent>` with safe wrapper generation and E2E coverage

Dependencies: Phase 3.

Allow users to run the normal agent command directly through an agentenv wrapper.

**Tasks**:
- [x] Implement `internal/wrapper.Install(agent)` to create `$AGENTENV_HOME/bin/<agent>`.
- [x] Generate a POSIX shell wrapper that executes the absolute current `agentenv` binary path, e.g. `exec "/absolute/path/to/agentenv" run <agent> "$@"`.
- [x] Ensure wrapper directory creation with executable permissions.
- [x] Refuse unsafe agent names containing path separators or shell metacharacters.
- [x] Do not overwrite non-agentenv files in the wrapper directory unless a force option is explicitly added later; v1 should fail safely.
- [x] If an existing wrapper was generated by agentenv, update it idempotently.
- [x] Print PATH guidance after installation.
- [x] Add `test/e2e/wrap_test.go` covering wrapper creation.
- [x] Add E2E coverage that executing the wrapper invokes the embedded absolute `agentenv` path and then the real fake agent.
- [x] Add E2E coverage that real agent binaries in other PATH directories are not overwritten.

**Automated Verification**:
- [x] `mise run test:e2e` passes `wrap` E2E tests.
- [x] Installed wrapper is executable.
- [x] Running the wrapper reaches the fake real agent without recursion.
- [x] Existing non-agentenv files are not overwritten.

**Manual Verification**:
- [ ] Run `agentenv wrap pi`, place the printed wrapper dir before the real `pi` on `PATH`, then run `pi --version` and confirm it goes through agentenv.

### Phase 6: Implement `doctor [agent?]` diagnostics with E2E coverage

Dependencies: Phase 5.

Provide actionable debugging for config, mapping, profile homes, PATH, wrappers, and light agent probing.

**Tasks**:
- [x] Implement `internal/doctor` check model with status, label, detail, fix hint, and severity.
- [x] Check config file readability and parse errors.
- [x] Check current project mapping and referenced profile existence.
- [x] Check profile home existence and writability.
- [x] If an agent is provided, check wrapper existence in agentenv bin dir and report missing wrappers as warnings because `agentenv run <agent>` works without wrapping.
- [x] If an agent is provided, check PATH ordering: wrapper bin appears before the real agent directory, and report bad wrapper ordering as a warning unless it prevents real-agent resolution.
- [x] If an agent is provided, resolve the real agent by skipping wrapper bin.
- [x] If an agent is provided, run the resolved real agent path with `--version` using a timeout; do not run bare `<agent> --version`, because that can resolve to the wrapper.
- [x] Summarize successes, warnings, and failures with `✓`/`!`/`✗` rows and actionable fix hints.
- [x] Return exit code `0` when no failure-severity checks fail; warnings alone do not make `doctor` fail.
- [x] Add `test/e2e/doctor_test.go` covering healthy config, missing mapping, bad PATH ordering, missing real agent, and successful fake `--version` probe.

**Automated Verification**:
- [x] `mise run test:e2e` passes `doctor` E2E tests.
- [x] Healthy test environment returns exit code `0`.
- [x] Missing mapping returns non-zero and prints an actionable message.
- [x] Bad PATH ordering prints the wrapper PATH fix as a warning when `run` still works.
- [x] Fake agent probe is executed against the resolved real path and reported.

**Manual Verification**:
- [ ] Run `agentenv doctor pi` in a correctly wrapped shell and confirm all relevant checks pass.
- [ ] Temporarily remove the wrapper dir from the front of `PATH`, rerun `agentenv doctor pi`, and confirm it explains the PATH fix as a warning while still probing the real agent if possible.

### Phase 7: Documentation and final polish

Dependencies: Phases 1-6.

Document the product model, installation, usage, safety boundaries, and test workflow.

**Tasks**:
- [x] Add `README.md` with pitch, use cases, install instructions via `mise`, and basic commands.
- [x] Document `agentenv run <agent> [args...]` for generic agents with examples for `pi`, `claude`, and `codex`.
- [x] Document `agentenv wrap <agent>` and PATH ordering requirements.
- [x] Document `agentenv doctor [agent?]` checks and example output.
- [x] Document runtime directories, config file location, profile home layout, and environment overrides, explicitly distinguishing `AGENTENV_HOME` as agentenv's data root from the isolated `HOME` passed to agents.
- [x] Document v1 security model: only `HOME` is isolated; no secrets are managed; no repo-local mapping file.
- [x] Document E2E test strategy and commands.
- [x] Add a concise help text for each command that matches README terminology.

**Automated Verification**:
- [x] `mise run fmt` succeeds.
- [x] `mise run test` succeeds.
- [x] `mise run test:e2e` succeeds.
- [x] `mise run build` succeeds.
- [x] README command examples align with CLI help text.

## Implementation Notes

During implementation, document user feedback, problems, and decisions here.

Initial planning decisions from user collaboration:

- Use Go via `mise`.
- Use Bubble Tea and Lip Gloss for TUI/banner styling.
- Keep profile mapping local only.
- Isolate only `HOME` in v1.
- Install wrappers into an explicit agentenv bin directory and require PATH ordering.
- Keep profiles to names only.
- Make the tool agent-agnostic.
- Resolve real agents by searching `PATH` while skipping the agentenv wrapper bin directory.
- Add E2E tests for `run`, `wrap`, and `doctor`.

## References

- Bubble Tea README: https://github.com/charmbracelet/bubbletea
- Lip Gloss README: https://github.com/charmbracelet/lipgloss
- mise README/docs: https://mise.jdx.dev/
