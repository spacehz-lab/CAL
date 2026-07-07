# Implementation Plan

Release V1 is implemented from zero. The current `internal/` tree is a behavioral reference, not a migration source.

## Phase 0: Baseline Capture

1. Capture current CLI JSON outputs and structured errors for the accepted command surface.
2. Capture HTTP API request and response shapes used by cald clients.
3. Capture current durable record fixtures for providers, capabilities, bindings, traces, and runs.
4. Snapshot proposal prompt text and LLM selection prompt text.
5. Capture replay and rules baseline outputs for controlled CLI providers.

Exit criteria:

- Release V1 has golden fixtures for public command/API behavior.
- Prompt snapshots fail on accidental wording drift.
- Record fixtures cover required fields, optional omission, ID determinism, and validation failures.

## Phase 1: Foundation

1. Implement `model` records, IDs, validation, and `VerifySpec` data shape.
2. Implement `progress` context fanout for `model.ProgressEvent`.
3. Implement `store` against the Release V1 `CAL_HOME` record layout.
4. Implement `config`, `logging`, and `llm` enough to support local cald startup and LLM-backed proposal calls.
5. Add golden tests for persisted JSON compatibility with the current records.

Exit criteria:

- Existing record fixtures can be read.
- Invalid records fail with structured validation errors.
- Progress events can fan out to request-local and app-level handlers.
- `config.json` defaults and secret boundaries are preserved.

## Phase 2: Execute And Check

1. Implement `execute` input rendering, required input detection, and typed output contracts.
2. Implement `execute/cli` command execution, stdout/stderr capture, exit code output, and stdout target writing.
3. Implement `check` deterministic rule registration, validation, and predicate evaluation.
4. Add unit tests for every supported subject and predicate.

Exit criteria:

- CLI execution behavior matches current run/probe behavior through typed outputs.
- `check` supports all existing `VerifySpec` subjects and predicates.

## Phase 3: Proposal

1. Implement `proposal` contract and result selection rules.
2. Copy existing prompt wording into `proposal/surface`, `proposal/capability`, `proposal/binding`, and `proposal/evidence` without semantic edits.
3. Implement replay, rules baseline, policy loading, and policy validation.
4. Add prompt snapshot tests to catch accidental prompt drift.

Exit criteria:

- Replay proposal fixtures produce the same candidates and probe plans.
- LLM proposal stages preserve current prompt wording and JSON parsing behavior.

## Phase 4: Acquisition

1. Implement `entry` provider loading and scanning.
2. Implement `observe` and `observe/cli`.
3. Implement `probe` using `execute` and `check`.
4. Implement `promote` and `tracelog`.
5. Implement `acquisition.Runner` as explicit orchestration: `entry -> observe -> proposal -> probe -> promote -> tracelog`.

Exit criteria:

- Targeted acquisition can promote verified CLI bindings.
- Failed observation, proposal, probe, and promotion paths write equivalent trace failures.

## Phase 5: Run, Use, Eval

1. Implement `run/resolve` and `run`.
2. Implement `use/plan`, `use/select`, and `use`.
3. Implement `eval`.
4. Add closed-loop tests for acquisition, run, use, and eval over controlled CLI providers.

Exit criteria:

- `run` executes promoted bindings deterministically.
- `use` selects promoted bindings from intent and delegates to `run`.
- `eval` produces acquisition and reuse metrics compatible with current outputs.

## Phase 6: cald, HTTP, Client, CLI

1. Implement `contract` DTOs matching existing HTTP/JSON shape.
2. Implement `cald/app` assembly and use-case methods.
3. Implement `httpserver` handlers over `contract` and `cald/app`.
4. Implement the `cald` endpoint-file subpackage used by daemon and client.
5. Implement `cli/client`.
6. Implement `cald/daemon`.
7. Implement `cli`.
8. Add SSE streaming routes and `--stream` CLI support after blocking JSON paths
   pass.

Exit criteria:

- Existing CLI commands work against the Release V1 cald path.
- JSON stdout remains clean and structured logs stay on stderr/file logs.
- The CLI daemon client does not import `cald/app` or workflow implementation packages.
- Streaming mode emits progress without changing blocking JSON behavior.

## Phase 7: Entry Point Switch

Switch `cmd/calctl` and `cmd/cald` only after Release V1 passes the acceptance suite.

Exit criteria:

- Unit tests pass.
- Deterministic e2e tests pass.
- API compatibility checks pass.
- Prompt snapshot checks pass.
- No old implementation package is required by the new entrypoints.
