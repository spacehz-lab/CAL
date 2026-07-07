# CAL Release V1 Probe

`probe/` owns acquisition-time candidate verification.

It verifies proposed candidate bindings before promotion. It is not the formal
runtime path for already promoted capabilities.

## Goal

`probe/` turns proposal output into traceable verification results:

```text
Provider + Candidate + ProbePlan
-> create isolated workdir
-> materialize inputs and fixtures
-> execute candidate when verification method is execute
-> evaluate deterministic VerifySpec checks
-> model.Probe
```

The normal acquisition flow is:

```text
entry -> observe -> proposal -> probe -> promote -> tracelog
```

`proposal` describes candidates and probe material. `probe` is the first package
that touches the filesystem for probe fixtures or executes candidate bindings.

## Boundary

`probe/` owns:

- Probe request validation.
- Candidate and probe-plan alignment by `CandidateIndex`.
- Per-candidate probe workdir naming, creation, and optional cleanup.
- Fixture file creation inside the probe workdir.
- `{{workdir}}` materialization for probe inputs.
- Injection of fixture paths into materialized inputs.
- Calling `execute` for `VerifyMethodExecute` probes.
- Calling `check` for deterministic verification.
- Contract verification handling for `VerifyMethodContract`.
- `model.Probe` construction.
- Probe failure classification into stable reason and error codes.

`probe/` does not own:

- Provider registration or loading. That belongs to `entry`.
- Provider observation. That belongs to `observe`.
- Candidate generation or prompt behavior. That belongs to `proposal`.
- Durable `VerifySpec` structs or predicate semantics. Those belong to
  `model` and `check`.
- Provider adapter execution details. Those belong to `execute`.
- Capability or binding promotion decisions. That belongs to `promote`.
- Trace persistence. That belongs to `tracelog`.
- Formal execution of promoted bindings. That belongs to `run`.
- Intent selection. That belongs to `use`.
- Store reads or writes.
- Config loading, logging setup, LLM calls, API transport, HTTP handlers, CLI
  rendering, or daemon lifecycle.

Probe failures should be returned as `model.Probe` records whenever enough
context exists to record the failed candidate. A failed candidate probe should
not erase earlier proposal diagnostics or other candidate probe results.

## Dependency Rule

```text
probe -> model, proposal, execute, check
```

Forbidden:

```text
probe -> store
probe -> acquisition
probe -> entry
probe -> observe
probe -> promote
probe -> tracelog
probe -> run
probe -> use
probe -> eval
probe -> llm
probe -> config
probe -> logging
probe -> contract/httpserver/cli/cald
```

`probe` receives `WorkRoot` from its caller. It does not import `store` to
discover trace paths. This keeps store focused on durable records and keeps
probe workdir policy local to `probe`.

Callers:

```text
acquisition -> probe
cald/app    -> probe
```

`probe` may consume `proposal.ProbePlan`, `proposal.Fixture`, and
`model.Candidate`. It must not call proposal stage runners or LLM stage
packages.

## Files

```text
probe/
  runner.go
  request.go
  workdir.go
  fixture.go
  materialize.go
  classify.go
  errors.go
```

`runner.go` owns the main probe flow and delegates narrowly scoped work to the
other files.

`request.go` owns `Request`, `Result`, `Target`, `MaterializedPlan`, and
`Options`.

`workdir.go` owns probe workdir path construction, directory creation, and
cleanup policy.

`fixture.go` owns safe fixture file writes inside the probe workdir.

`materialize.go` owns `{{workdir}}` replacement and fixture input injection.

`classify.go` owns stable reason and error-code mapping for execution and
verification failures.

`errors.go` owns probe error codes and coded errors.

## Public Shape

Use one dependency-owning runner:

```go
type Runner struct {
	executor *execute.Runner
	checker  *check.Checker
	options  Options
}

func NewRunner(executor *execute.Runner, checker *check.Checker, options Options) *Runner
func (r *Runner) Run(ctx context.Context, req *Request) (*Result, error)
```

`Runner` owns runtime dependencies, so callers should use `NewRunner`. Plain
request/result values may still be assembled with literals.

`request.go` owns:

```go
type Request struct {
	Provider   *model.Provider
	Candidates []model.Candidate
	Plans      []proposal.ProbePlan
	TraceID    string
	WorkRoot   string
	Now        func() time.Time
}

type Result struct {
	Probes []model.Probe
}
```

`WorkRoot` is required. The caller decides whether it is under a trace
directory, temp directory, or another acquisition workspace. `probe` decides the
per-candidate child layout under that root.

`Target` is the internal per-candidate unit:

```go
type Target struct {
	CandidateIndex int
	Candidate      *model.Candidate
	Plan           *proposal.ProbePlan
	WorkDir        string
}
```

`MaterializedPlan` is the internal output of materialization:

```go
type MaterializedPlan struct {
	CandidateIndex int
	Inputs         map[string]any
	Verify         model.VerifySpec
	WorkDir        string
}
```

`Options` owns runtime bounds:

```go
type Options struct {
	Timeout     time.Duration
	KeepWorkdir bool
}
```

First V1 should run candidate probes serially. `proposal` already has
per-capability proposal concurrency; probing executes real provider commands and
should default to the smallest safe behavior. Add probe concurrency only after
workdir isolation, side-effect policy, and cancellation behavior are proven.

## Probe Flow

For each selected candidate:

```text
validate target
-> create workdir
-> materialize plan
-> validate VerifySpec
-> if method=contract: return L1 contract evidence probe
-> execute candidate with materialized inputs
-> run checker against execution outputs
-> return passed or failed model.Probe
-> cleanup workdir unless KeepWorkdir
```

Probe should continue to the next candidate after a candidate-level execution or
verification failure. Structural request errors may stop the whole run because
they indicate bad wiring rather than one failed candidate.

## Materialization

`materialize.go` resolves only the probe material contract:

- Replace `{{workdir}}` inside string inputs.
- Reject unresolved `{{...}}` templates.
- Preserve non-string input values.
- Write each fixture under the candidate workdir.
- Reject absolute fixture filenames.
- Reject fixture paths that escape the workdir after cleaning.
- Add `inputs[fixture.Input] = <written fixture path>`.

Materialization must not render execution args. `execute` owns execution input
rendering.

## Verification Behavior

For `VerifyMethodExecute`:

```text
execute.Runner.Run
-> check.Checker.Run
-> passed model.Probe with check evidence
```

For `VerifyMethodContract`:

```text
check.Checker.Validate
-> no provider execution
-> passed L1 model.Probe with contract evidence
```

Contract verification must not exceed `L1`. The checker owns that rule.

For `VerifyLevelL0`:

```text
check.Checker.Validate
-> no provider execution required
-> non-passed model.Probe with an L0 reason
```

`L0` is recorded as insufficient evidence for promotion, not as a hard system
failure.

## Error Codes

Use stable string constants in `errors.go`:

```text
invalid_probe_input
probe_materialize_failed
verification_plan_invalid
execution_failed
execution_timeout
verification_failed
unsupported_verify_method
```

These codes are stored in `model.Probe.Error.Code` and may appear in trace and
API output, so they are contract values and must be constants.

## Probe Reasons

Use stable reason constants for `model.Probe.Reason`:

```text
verify_checks_passed
contract_evidence_recorded
verification_level_l0
verification_plan_invalid
execution_failed
execution_timeout
verification_failed
probe_materialize_failed
```

Reason strings should be short and machine-stable. Human-facing expansion
belongs to API/CLI rendering later.

## Output Mapping

`execute.Result` stores typed outputs. `probe` converts only the fields needed
by `check.Request`:

```text
execute.OutputStdout   -> check.Request.Stdout
execute.OutputStderr   -> check.Request.Stderr
execute.OutputExitCode -> check.Request.ExitCode
```

File checks read paths from materialized `Inputs`. `probe` should not copy file
contents into `model.Probe`.

## Completion Criteria

Probe implementation is complete when:

- It validates request shape and candidate/plan alignment.
- It materializes workdir inputs and fixtures safely.
- It records failed probes for materialization, execution, timeout, invalid
  verification, and failed checks.
- It records passed probes for execute verification and contract verification.
- It does not call LLM, store, promote, run, CLI, HTTP, CLI daemon client, or cald
  packages.
- Unit tests cover workdir safety, fixture escaping, unresolved templates,
  execute success, execute failure, timeout classification, contract evidence,
  invalid verify plan, and L0 behavior.
