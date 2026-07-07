# Release V1 Rewrite

Release V1 is a from-scratch implementation of CAL's current product behavior with a smaller, more cohesive package structure.

This is not a migration plan from the current `internal/` tree. The current implementation remains the behavioral reference until Release V1 passes the acceptance checks in this directory.

## Goals

- Preserve the approved Release V1 user-facing behavior, command surface, HTTP API shape, persisted records, and prompt behavior.
- Rebuild the implementation from zero under a new package structure before switching entrypoints.
- Reduce production implementation size by removing duplicate DTOs, scattered workflow state, and broad packages with mixed responsibilities.
- Make each package own one narrow responsibility with explicit dependency direction.
- Keep acquisition, run, use, and eval as readable use-case runners rather than generic framework abstractions.

## Non-Goals

- Do not introduce a remote hosted service, multi-user auth, package distribution, GUI acquisition, or broad plugin hosting.
- Do not redesign the CAL research claim, durable model, or CLI contract as part of the rewrite.
- Do not change prompt wording unless a separate prompt-change document approves it.
- Do not create a generic workflow framework for acquisition stages.
- Do not expose new public Go APIs during the first Release V1 pass.

## Invariants

Release V1 must preserve:

- CLI command names, flags, stdout JSON shape, and structured error behavior approved by this V1 design.
- Local `cald` daemon behavior and loopback HTTP API semantics.
- `CAL_HOME` durable layout for config, providers, capabilities, traces, and runs.
- Core records: `Provider`, `Capability`, `Binding`, `Trace`, and `Run`.
- `VerifySpec` JSON shape, predicate names, subject names, and validation rules.
- Proposal prompt wording and staged proposal behavior.
- Secret boundaries: API keys must stay out of config files, traces, logs, proposal fixtures, and normal CLI output.

## Design Shape

The rewrite separates the implementation into these groups:

- Foundation: `model`, `progress`, `store`, `config`, `logging`, and `llm`.
- Acquisition: `acquisition`, `entry`, `observe`, `proposal`, `probe`, `promote`, and `tracelog`.
- Shared execution and checking: `execute` and `check`.
- Reuse: `run`, `run/resolve`, `use`, `use/select`, `use/plan`, and `eval`.
- Adapters: `contract`, `httpserver`, `cli`, and `cald`.

See `cal-package-skeleton.md` for the concrete tree, `cal-module-model.md` for model boundaries, `cal-module-store.md` for local persistence design, `cal-module-acquisition.md` for the targeted acquisition orchestration boundary, `cal-module-entry.md` for explicit provider registration and loading, `cal-module-observe.md` for provider observation contracts and CLI usage observation, `cal-module-proposal.md` for proposal stages, concurrency, and candidate/probe-plan contracts, `cal-module-probe.md` for acquisition-time candidate verification, `cal-module-promote.md` for durable capability and binding promotion, `cal-module-tracelog.md` for acquisition trace assembly and persistence, `cal-progress-events.md` for live progress event semantics, `cal-progress-streaming.md` for HTTP SSE and CLI streaming design, `cal-module-execute.md` for provider execution and typed output collection, `cal-module-check.md` for deterministic verification design, `cal-module-run.md` for formal promoted capability execution, `cal-module-use.md` for intent-level reuse, `cal-module-eval.md` for read-only acquisition and reuse metrics, `cal-module-config.md` for configuration design, `cal-module-logging.md` for logging design, `cal-module-llm.md` for live LLM adapter design, `cal-module-contract.md` for daemon JSON contracts, `cal-module-httpserver.md` for HTTP server adapters, `cal-module-cli.md` for CLI commands and the CLI-internal daemon client, `cal-module-cald.md` for app, daemon, and endpoint ownership, `cal-dependency-map.md` for dependency rules, `cal-implementation-plan.md` for the from-zero build order, and `cal-acceptance.md` for release gates.
