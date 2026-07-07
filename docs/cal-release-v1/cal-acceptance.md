# Acceptance Criteria

Release V1 is accepted only when it preserves the approved V1 behavior through the public surfaces and durable records.

## Command Surface

The following commands define the approved V1 command names, flags, JSON output shape, and structured errors:

- `calctl daemon start`
- `calctl daemon stop`
- `calctl daemon status --json`
- `calctl providers add --provider-path <path> --json`
- `calctl providers list --json`
- `calctl acquisition run --provider-id <id> --json`
- `calctl acquisition run --provider-id <id> --proposal-path <path> --json`
- `calctl capabilities list --json`
- `calctl runs create --capability-id <id> --inputs-json <json> --json`
- `calctl use <intent> --json`
- `calctl eval --json`

## Durable Records

Release V1 must read and write the same durable record shapes for:

- `Provider`
- `Capability`
- `Binding`
- `Trace`
- `Run`

Compatibility checks must cover:

- Required fields.
- Optional field omission.
- ID determinism.
- Validation failure behavior.
- Release V1 `CAL_HOME` directory layout.

## Acquisition Behavior

Targeted acquisition must preserve:

- Provider entry loading.
- CLI observation strategy.
- Proposal replay behavior.
- Rules baseline behavior.
- LLM proposal stage order and prompt wording.
- Probe fixture materialization.
- Deterministic verify checks.
- Promotion rule: only passed probes create or update durable `Capability` and `Binding` records.
- Failure trace behavior after provider load.

## Run And Use Behavior

Run must preserve:

- Binding resolution by capability id, provider id, binding id, strategy, and minimum verify level.
- Supported execution kind handling.
- Required input detection.
- CLI execution stdout/stderr/exit code behavior.
- Optional verification and run record writing.

Use must preserve:

- Intent validation.
- Local binding selection behavior.
- Optional LLM binding selection behavior.
- Input planning, missing input handling, generated target path behavior, and prevention of overwriting caller inputs.
- Delegation to the promoted binding run path.

## Prompt Invariance

Prompt wording must be preserved unless an explicit prompt-change document updates this acceptance criterion.

Required checks:

- Snapshot all proposal stage prompts.
- Snapshot LLM selection prompt if used by `use/select`.
- Fail tests on accidental prompt changes.

## Secret And Logging Boundaries

Release V1 must not write API secrets to:

- repository files
- `CAL_HOME/config.json`
- traces
- proposal fixtures
- run records
- process logs
- CLI stdout JSON

Machine-readable stdout must not be polluted by diagnostic logs.

## Test Gates

Release V1 must pass:

- unit tests for every package with behavior
- record compatibility tests
- prompt snapshot tests
- deterministic acquisition closed-loop tests
- deterministic run/use/eval tests
- HTTP API contract tests
- CLI e2e tests

Known flaky behavior in the current implementation is not allowed to become an accepted Release V1 behavior. Timeout-sensitive observation tests must use controlled fixtures or configurable short test timeouts.
