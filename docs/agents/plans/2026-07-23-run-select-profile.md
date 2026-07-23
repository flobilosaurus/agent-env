---
date: 2026-07-23T19:54:25.732830+00:00
git_commit: ""
branch: main
topic: "Force profile selection for agentenv run"
tags: [plan, cli, tui, profiles]
status: ready
---

# PLAN: Force profile selection for `agentenv run`

Add a `--select` flag to `agentenv run` so users can intentionally reopen the existing profile selector for a project that already has a profile mapping. The requested call is `agentenv run --select pi`. The command should select or create a profile, update the current project mapping, and then launch the requested agent immediately with the selected profile `HOME`.

## Acceptance Criteria

- `agentenv run --select <agent> [args...]` opens profile selection even when the current project already has a mapping.
- The selected or newly created profile replaces the existing project mapping for the current project.
- After selection, the requested agent launches immediately with `HOME=$AGENTENV_HOME/profiles/<selected-profile>/home`.
- Existing `agentenv run <agent> [args...]` behavior remains unchanged.
- `--select` is treated as an agentenv `run` flag only when it appears before `<agent>`.
- `--select` after `<agent>` remains a passthrough argument to the agent.
- Non-interactive forced selection fails clearly when `AGENTENV_NONINTERACTIVE=1`, because the selector cannot be opened.
- CLI usage/help and README document the new syntax.
- Unit and E2E tests cover forced reselection, mapping replacement, launch behavior, and passthrough argument handling.

## Technical Key Decisions and Tradeoffs

1. **Flag syntax:** Support `agentenv run --select <agent> [args...]`.
   - Why: This matches the requested user-facing call exactly.
   - Impact: `internal/cli.App.run` needs a small pre-`<agent>` parser rather than always treating `args[0]` as the agent name.

2. **Reselect behavior:** Forced selection updates the mapping and then launches the agent.
   - Why: The user confirmed that `--select` should not be a mapping-only command; it should continue into the normal run path.
   - Impact: The existing missing-mapping prompt/save logic should be reused when `--select` is set.

3. **Flag scope:** Only parse `--select` before `<agent>` and do not introduce generic `run` flag parsing.
   - Why: `agentenv run pi --select` should remain valid for agents that have their own `--select` argument, and existing behavior for agent names/arguments should stay stable.
   - Impact: The parser should consume a single leading `--select` when present, then treat the next argument as `<agent>` and all later arguments as passthrough.

4. **Non-interactive mode:** `AGENTENV_NONINTERACTIVE=1` blocks both first-run prompting and forced prompting.
   - Why: `--select` requires interactive profile selection unless a future non-interactive selection mechanism is added.
   - Impact: Forced selection should return a clear error before launching or changing config when non-interactive mode is enabled.

## Current State

`agentenv run` currently treats the first argument after `run` as the agent name and all remaining arguments as passthrough args.

```text
agentenv run <agent> [args...]
      │
      ▼
internal/cli.App.run
      │
      ├─ load config
      ├─ normalize cwd as project key
      ├─ read cfg.Projects[project]
      │
      ├─ missing mapping
      │    └─ tui.ProfilePrompter.ChooseProfile(agent, cfg.Profiles)
      │       └─ validate/create profile, save mapping
      │
      ├─ existing mapping
      │    └─ skip selector and reuse mapped profile
      │
      ├─ ensure profile home
      ├─ runner.LookupAgent(agent, p.BinDir())
      ├─ print banner
      └─ runner.RunAgent(...)
```

Key files:

- `internal/cli/cli.go:82-153` - current `run` command parsing, mapping lookup, prompting, save, and launch flow.
- `internal/tui/tui.go:14-31` - `ProfilePrompter` interface and Bubble Tea implementation.
- `internal/cli/cli_test.go` - unit test seam using a fake `ProfilePrompter`.
- `test/e2e/run_test.go` - E2E tests for mapped runs, wrapper-bin skipping, exit propagation, and non-interactive unmapped failure.
- `README.md:14-28` - current command syntax and first-run selector documentation.

## Desired End State

`agentenv run --select pi` should take this path:

```text
agentenv run --select pi [args...]
      │
      ▼
parse run options
      ├─ forceSelect = true
      └─ agent = pi, pass = [args...]
      │
      ▼
load config + normalize project
      │
      ▼
forceSelect OR missing mapping?
      ├─ yes: open selector/create flow → save cfg.Projects[project] = selected
      └─ no: use existing mapping
      │
      ▼
ensure selected profile home
      │
      ▼
launch pi with selected HOME and passthrough args
```

User-facing examples:

```sh
agentenv run --select pi
agentenv run --select pi --version
agentenv run pi --select      # passes --select to pi, does not force agentenv selection
```

## Abstractions and Code Reuse

Reuse the existing `tui.ProfilePrompter` seam and the same selection/create validation behavior that already exists for unmapped projects. Avoid adding a separate command or a separate TUI mode.

- `internal/cli/cli.go`
  - `App.run` - parse `--select` before `<agent>` and route forced selection through the same prompt/save branch used for missing mappings.
  - Optional helper such as `parseRunArgs(args []string) (forceSelect bool, agent string, pass []string, err error)` - keeps flag parsing testable and prevents accidental passthrough changes.
  - Optional helper such as `chooseAndSaveProfile(...)` - reduces duplication between missing mapping and forced selection.
- `internal/cli/cli_test.go`
  - Extend fake prompter to return configurable profile/create values and count calls.
  - Add tests for forced reselection and passthrough preservation.
- `test/e2e/run_test.go`
  - Add E2E coverage for parser-visible binary behavior: `--select` requires an agent, forced selection is rejected in non-interactive mode without changing the mapping, and `--select` after `<agent>` is passed through unchanged.
- `README.md`
  - Document `agentenv run [--select] <agent> [args...]` and examples.

## Logging & Observability

No new persistent logging is required. Error output should remain concise and consistent with existing messages.

Example non-interactive forced-selection error:

```text
agentenv: cannot select a profile in non-interactive mode for /path/to/project
```

Normal successful output remains the existing banner plus the real agent output:

```text
┌─ agentenv ───────────────────────────────────┐
│ selected-profile • pi • OpenAI Enterprise    │
└──────────────────────────────────────────────┘
```

## Implementation

### Phase 1: Add `--select` parsing and forced profile selection

Dependencies: None.

Implement the user-facing CLI behavior while preserving existing `run` semantics.

**Tasks**:
- [x] Update `internal/cli/cli.go` command usage for `run` from `agentenv run <agent> [args...]` to `agentenv run [--select] <agent> [args...]`.
- [x] Add a small parser for `run` arguments that recognizes only a single leading `--select` before `<agent>` and otherwise preserves existing positional behavior.
- [x] Ensure missing `<agent>` after `agentenv run --select` returns usage with exit code `2`.
- [x] Refactor the existing missing-mapping prompt/create/save block into reusable logic used when `profile == "" || forceSelect`.
- [x] Preserve existing missing-mapping behavior when `--select` is absent.
- [x] When `forceSelect` is true and `AGENTENV_NONINTERACTIVE=1`, fail before prompting, saving config, creating profile homes, or launching the agent.
- [x] Ensure the selected profile replaces any existing mapping through `cfg.SetProject(project, chosen)` and `config.Save(cfgPath, cfg)`.
- [x] Keep `agentenv run pi --select` parsed as agent `pi` with passthrough args `[]string{"--select"}`.
- [x] Avoid treating unrelated leading arguments such as `--foo` as agentenv options; if not equal to `--select`, the first argument remains the agent name as before.

**Automated Verification**:
- [x] `go test ./internal/cli` passes.
- [x] Unit test: `agentenv run --select pi` with an existing project mapping calls the fake prompter, saves the new mapping, and launches with the new profile home.
- [x] Unit test: `agentenv run pi --select` does not call the prompter when a mapping exists and passes `--select` to the fake agent.
- [x] Unit test: `agentenv run --select` exits `2` and prints run usage.
- [x] Unit test: `agentenv run --select pi` with `AGENTENV_NONINTERACTIVE=1` exits non-zero without changing the existing mapping.
- [x] Unit test: `agentenv run --foo` still attempts to run an agent named `--foo` or otherwise follows existing positional behavior rather than reporting an unknown agentenv option.

### Phase 2: Add E2E coverage for the new command syntax

Dependencies: Phase 1.

Exercise the built binary and guard against regressions in real command-line behavior.

**Tasks**:
- [x] Add E2E coverage that `agentenv run --select pi` with an existing mapping and `AGENTENV_NONINTERACTIVE=1` fails clearly without launching the fake agent.
- [x] In that E2E test, verify the config file still contains the original mapping after the failed forced-selection attempt.
- [x] Add E2E coverage that `agentenv run pi --select` treats `--select` as a real-agent argument when a mapping already exists.
- [x] Add E2E coverage that `agentenv run --select` without an agent exits with usage and code `2`.
- [x] Rely on `internal/cli` fake-prompter unit tests, not terminal key automation, for the successful forced-selection prompt/save/launch path.

**Automated Verification**:
- [x] `mise run test:e2e` passes.
- [x] `mise run test` passes.
- [x] E2E confirms non-interactive forced selection does not mutate the existing project mapping.
- [x] E2E confirms passthrough `--select` reaches the fake real agent when it appears after `<agent>`.

### Phase 3: Update documentation and polish

Dependencies: Phase 1.

Document the feature where users discover command syntax.

**Tasks**:
- [x] Update `README.md` command synopsis to show `agentenv run [--select] <agent> [args...]`.
- [x] Add an example for reselection: `agentenv run --select pi`.
- [x] Update the `run` description to explain that first use prompts automatically, and `--select` forces the selector even when a mapping already exists.
- [x] Ensure wording clarifies that agent arguments after `<agent>` are passed through unchanged.

**Automated Verification**:
- [x] `mise run fmt` succeeds.
- [x] `mise run test` succeeds.
- [x] `mise run test:e2e` succeeds.
- [x] `mise run build` succeeds.

**Manual Verification**:
- [ ] In a project with an existing mapping, run `agentenv run --select pi`, select or create a different profile in the TUI, and confirm `pi` launches with the newly selected profile banner.
- [ ] Run `agentenv run pi` again in the same project and confirm it reuses the newly selected profile without prompting.

## Implementation Notes

During implementation, document user feedback, problems, and decisions here.

## References

- `internal/cli/cli.go:82-153` - existing run flow.
- `internal/tui/tui.go:14-122` - existing profile selection UI and interface.
- `internal/cli/cli_test.go` - fake prompter unit test pattern.
- `test/e2e/run_test.go` - current run E2E tests.
- `README.md:14-28` - command syntax and run documentation.
