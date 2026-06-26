# CAL Engineering Directory

This document is the single directory contract for the CAL Go implementation.

It defines the target module layout and ownership boundaries. It is not a command contract; command semantics live in `docs/design/cal-control-plane.md`.

## Rules

The target tree is a direction for implementation slices, not a reason to create empty packages.

Add a directory only when it owns real behavior, direct tests, or a concrete command path.

Do not introduce old CapCLI package concepts:

- app-scoped packages
- proposal/apply flow
- profile records
- YAML package manifests
- public provider methods
- `LLM proposer`-generated durable ids

## Target Directory

```text
cmd/
  calctl/
    main.go

  cald/
    main.go

internal/
  baseline/
    rules/
      proposer.go
      planner.go

  core/
    model.go
    ids.go
    validate.go

  trace/
    model.go
    trace.go
    validate.go

  store/
    store.go
    home.go
    provider.go
    capability.go
    trace.go
    run.go

  config/
    config.go
    config_darwin.go
    config_linux.go
    config_windows.go
    helper.go
    logging.go
    llm.go

  logging/
    logging.go
    rotate.go
    dir_darwin.go
    dir_linux.go
    dir_windows.go

  cli/
    root.go
    json.go
    errors.go
    daemon.go
    discover.go
    discover_loop.go
    discover_path.go
    discover_scan.go
    discover_scanner.go
    discover_status.go
    capability.go
    run.go
    eval.go

  cald/
    command.go
    status.go

  discovery/
    job.go
    entry_scanner.go
    entry_scanner_darwin.go
    entry_scanner_linux.go
    entry_scanner_windows.go
    errors.go
    acquisition.go
    acquisition_result.go
    acquisition_verifier.go
    acquisition_promoter.go

  observe/
    observe.go
    result.go

    cli/
      observer.go
      help.go

  proposal/
    catalog.go
    materializer.go
    probe.go
    proposal.go
    prompt.go
    schema.go

    llm/
      chat.go
      factory.go
      options.go
      proposer.go
      client.go
      responses.go

  runtime/
    catalog.go
    runner.go
    runner_inputs.go
    runner_execute.go
    registry.go
    registry_install.go

  eval/
    eval.go
    records.go
    records_summary.go
    records_acquisition.go
    records_reuse.go
    metrics.go

  testsupport/
    e2e/
      support.go

tests/
  README.md

  e2e/
    functional/
      acquisition_fake_cli_test.go
      calctl_cli_test.go
      eval_metrics_test.go

    local_cli/
      cupsfilter_test.go
      sips_test.go

    live_llm/
      live_llm_test.go

evals/
  README.md

  experiments/
    cli-matrix/
      README.md
      run.py
      cases.json
      artifact.py
      report.py
      proposals/
        replay/
      templates/
        index.html

  benchmarks/
    cal-cli-v0/
      README.md
      scoring.md

  out/
    <generated, gitignored>

scripts/
  <reserved for non-runtime support scripts>
```

## Ownership

### `cmd/calctl`

Owns the executable entrypoint for the machine-facing CLI.

It should only assemble `internal/cli` and process exit behavior.

### `cmd/cald`

Owns the executable entrypoint for the local CAL service.

It should only assemble `internal/cald` and process exit behavior.

### `internal/core`

Owns shared model types and deterministic id shape rules.

Core-owned types:

```text
Provider
Capability
Binding
Verifier
EvidenceRef
Run
Eval
```

`core` must not import higher-level packages.

`core` does not own a verifier catalog. Verifier package ids are data owned by
local verifier packages, proposals, or tests.

### `internal/trace`

Owns Discovery Trace records and Trace-specific validation.

Trace is process material for explanation, debugging, and evaluation. It is not
executable capability knowledge and must not live in `core`.

Trace may reference core value types such as `Execution`, `Verifier`,
`EvidenceRef`, and `RecordError`.

### `internal/store`

Owns local JSON persistence under `CAL_HOME`.

Storage operations should be typed around CAL records, not path strings as the public API.

Business packages should define the narrow persistence interfaces they need.
`store` is the concrete file-backed adapter that implements those interfaces.

### `internal/config`

Owns user-editable local configuration, including provider sources, non-secret
LLM settings, and logging policy.

It does not own provider discovery, Trace writing, promotion, or run behavior.

### `internal/logging`

Owns process `slog` initialization, platform log/state paths, and rolling log
file retention.

It reads logging policy from config and applies `CAL_LOG_LEVEL` as a temporary
level override. It must not own business logging events.

### `internal/cli`

Owns command definitions, argument parsing, daemon process management, service
client calls, and output rendering.

It should stay a thin adapter over store, config, discovery, runtime, eval, and
cald service calls. Once a service API exists for a workflow, `internal/cli`
should call the typed `cald` client rather than duplicating service logic.

### `internal/cald`

Owns local service state.

Later responsibilities:

```text
local HTTP control API
WebUI static serving
future automatic discovery loop
manual discovery jobs
job status
provider locks
service lifecycle
```

Use `cald` for this service boundary. Do not introduce `internal/backend`.

### `internal/discovery`

Owns discovery jobs and the Discovery loop:

```text
Entry
Inference
Verification
Promotion
Trace
Probe
```

Discovery should coordinate `observe` and `proposal`, then promote only verified bindings.

`internal/proposal` defines the `Proposer` and `ProbePlanner` contracts.
Production planner implementations live with their proposal source, such as
`proposal/llm` or the proposal JSON materializer. Rules-only deterministic
planning is a baseline implementation under `internal/baseline/rules`, not a
default production fallback.

`ProbePlanRequest` should carry only the candidate and the temporary work
directory needed to materialize probe inputs. Provider execution stays in
Discovery Verification and should not be passed into planners unless a planner
has a concrete provider-specific planning need.

### `internal/observe`

Owns provider observation interfaces.

Driver subpackages:

```text
observe/cli
```

Keep observation separate from discovery because different providers require different drivers.

### `internal/proposal`

Owns the proposal contract for candidate generation and probe-plan materialization.

Implementation subpackages:

```text
proposal/llm
```

Keep proposal-domain adapters under `proposal`. Do not move proposal schema,
prompt, or materialization logic into generic package names such as `llm`,
`infer`, `model`, or `actor`.

`proposal/llm` is only the `LLM proposer` implementation detail.

The top-level `proposal` package owns offline proposal JSON parsing, loading,
and materialization. Live LLM output, SOP files, and fixtures use the same
proposal contract before Discovery executes and verifies anything.

Rules-only proposal generation is intentionally outside `internal/proposal` in
`internal/baseline/rules` so experiments and regression tests do not leak
hard-coded behavior into production discovery.

### `internal/baseline/rules`

Owns deterministic rules-only proposal generation and probe planning for evaluation baselines,
controlled fake CLI fixtures, and regression tests.

It may reference test-installed verifier package ids as fixture data, but
production discovery must import it only for hidden baseline mode.

### `internal/runtime`

Owns runtime handling for promoted bindings:

- reusable capability catalog read-model
- binding resolution for `run`
- supported execution kinds
- binding execution
- outcome verification
- run history summaries

It should not perform discovery or promote candidates.

### `internal/use`

Owns intent-level Use contracts and promoted-binding selection.

The first implementation uses local scoring over promoted capabilities and
bindings. `cald` calls this package while orchestrating service requests; it
should not own Use scoring, tokenization, or ranking logic.

### `internal/eval`

Owns acquisition and reuse metrics for experiments.

It reads Trace, Run, Capability, Binding, and Provider records without mutating them.

### `internal/testsupport/e2e`

Owns shared test support for black-box CAL tests and eval runners, such as
building `calctl`, running JSON commands, reading trace records, and writing
controlled provider fixtures.

It must stay test-oriented. Production packages must not import it.

### `tests`

Owns cross-package black-box tests. Unit tests remain next to the implementation
files they cover.

`tests/e2e/functional` contains deterministic closed-loop tests expected to run
by default. `tests/e2e/local_cli` contains local real-CLI end-to-end checks.
`tests/e2e/live_llm` contains API-key-gated live LLM checks.

### `evals`

Owns executable evaluation assets for CAL system claims.

`evals/experiments` contains exploratory evidence runners and fixtures.
`evals/benchmarks` contains stable task sets, scoring, and baselines once they
exist. Generated outputs belong under `evals/out/`, which is ignored by git.

Verifier packages live under `CAL_HOME/verifiers/`. Production runtime does not
load embedded verifier scripts from the repository.

## Naming Decisions

- `calctl` is the CLI.
- `cald` is the local service.
- `observe` is independent because it can use CLI, CUA, or later drivers.
- `proposal` owns candidate proposal generation and probe-plan materialization.
- `runtime` owns promoted binding execution, verification, catalog read-models, and binding selection.
- There is no `actor` module.
- There is no top-level `llm`, `infer`, `model`, `backend`, or `provider` module in the target tree.
- `provider` is a core record and store concern, not a top-level implementation package.

## Dependency Direction

Target direction:

```text
cmd/calctl -> internal/cli
cmd/cald   -> internal/cald

internal/cli         -> internal/cald
internal/cli         -> internal/cald/client
internal/cli         -> internal/cald/control
internal/cli         -> internal/config
internal/cli         -> internal/logging

internal/cald      -> internal/config
internal/cald      -> internal/discovery
internal/cald      -> internal/eval
internal/cald      -> internal/logging
internal/cald      -> internal/runtime
internal/cald      -> internal/store
internal/cald      -> internal/use

internal/discovery -> internal/core
internal/discovery -> internal/proposal
internal/discovery -> internal/observe
internal/discovery -> internal/runtime
internal/discovery -> internal/trace

internal/proposal  -> internal/core
internal/proposal  -> internal/runtime
internal/proposal  -> internal/trace
internal/observe   -> internal/core
internal/runtime   -> internal/core
internal/use       -> internal/core
internal/use       -> internal/runtime
internal/eval      -> internal/core
internal/eval      -> internal/trace
internal/store     -> internal/core
internal/store     -> internal/trace
internal/trace     -> internal/core

internal/core      -> standard library only
```

Avoid reverse imports from model and storage layers into `cli`, `cald`, or driver packages.
