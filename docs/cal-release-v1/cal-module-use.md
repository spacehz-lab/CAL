# CAL Release V1 Use

`use/` owns intent-level reuse.

It turns a user intent plus caller inputs into one formal promoted capability
run. It selects a binding, plans the final inputs, delegates execution to
`run`, and returns a use result.

## Goal

`use/` provides the semantic reuse entry point:

```text
intent + inputs
-> list promoted capabilities
-> select capability binding
-> plan final run inputs
-> run promoted capability
-> use result
```

`use` is above `run`. It does not execute providers, evaluate VerifySpec, or
save `model.Run` directly.

## Boundary

`use/` owns:

- Use request validation.
- Use result lifecycle and error mapping.
- Loading the capability catalog from store.
- Calling `use/select`.
- Calling `use/plan`.
- Delegating formal execution to `run`.
- Mapping failed run results into failed use results.

`use/` does not own:

- Provider execution. That belongs to `execute`.
- Deterministic verification. That belongs to `check`.
- Run record persistence. That belongs to `run`.
- Candidate proposal, probing, promotion, or acquisition traces.
- Provider scanning or observation.
- CLI, HTTP, CLI daemon client, daemon, or DTO rendering.
- Config loading, logging setup, or environment resolution.
- LLM prompt construction. That belongs to `use/select`.

## Dependency Rule

```text
use        -> model, store, llm, use/select, use/plan, run
use/select -> model, execute, llm
use/plan   -> standard library only
```

Forbidden:

```text
use -> execute
use -> check
use -> config
use -> logging
use -> acquisition/proposal/probe/promote/tracelog
use -> contract/httpserver/cli/cald
use/select -> run
use/select -> store
use/select -> check
use/plan -> run
use/plan -> execute
use/plan -> check
use/plan -> llm
use/plan -> store
```

`use` may import `llm` only to expose the default runner constructor. Prompt
construction and LLM selection behavior belong to `use/select`.

## Call Chain

```text
use.Run
  -> store.ListCapabilities
  -> use/select.Run
  -> use/plan.Run
  -> run.Run
```

Downstream:

```text
run.Run
  -> run/resolve
  -> execute
  -> check
  -> store.SaveRun
```

## Files

```text
use/
  request.go
  runner.go
  record.go

  select/
    runner.go
    local.go
    llm.go

  plan/
    runner.go
    target.go
```

Do not add more subpackages in V1.

## Public Shape

`request.go` owns use runtime request and result:

```go
type Request struct {
	Intent         string
	Inputs         map[string]any
	ProviderID     string
	Verify         bool
	MinVerifyLevel model.VerifyLevel
}

type Result struct {
	ID         string
	Intent     string
	Selection  *Selection
	Run        *model.Run
	Status     model.RunStatus
	StartedAt  string
	FinishedAt string
	DurationMS int64
	Error      *model.RecordError
}

type Selection struct {
	Source               string
	CapabilityID         string
	BindingID            string
	ProviderID           string
	Reason               string
	CandidatesConsidered int
}
```

`Request`, `Result`, and `Selection` are runtime DTOs. They belong in `use`, not
`model`.

Use pointer parameters and pointer return values because use requests and
results are non-trivial workflow values.

## Runner

`runner.go` owns orchestration:

```go
type Store interface {
	ListCapabilities() ([]model.Capability, error)
}

type Runner struct {
	store    Store
	selector *selector.Runner
	planner  *plan.Runner
	executor Executor
}

type Executor interface {
	Run(context.Context, *run.Request) (*run.Result, error)
}

func NewDefaultRunner(store Store, executor Executor, client llm.Client) *Runner
func NewRunner(store Store, selector *selector.Runner, planner *plan.Runner, executor Executor) *Runner

func (r *Runner) Run(ctx context.Context, req *Request) (*Result, error)
```

The run dependency should be a narrow interface so tests can use a fake runner.
`use` owns the default `use/select` and `use/plan` wiring. `NewRunner` remains
the narrow injection constructor for tests or non-standard selector/planner
variants.

Main flow:

```text
1. Validate use request.
2. Start use result.
3. Load capabilities from store.
4. Select binding with use/select.
5. Plan final inputs with use/plan.
6. Call run.Run.
7. If run failed, fail use result with run error.
8. If run succeeded, complete use result.
```

## Request Validation

Validation rules:

- `Intent` is required.
- `Inputs` may be nil and should normalize to an empty map.
- `ProviderID` is optional.
- `Verify` is passed through to `run`.
- Empty `MinVerifyLevel` defaults to `model.VerifyLevelL2` for selection.

Defaulting `MinVerifyLevel` to L2 preserves the existing behavior that semantic
reuse should prefer higher-confidence promoted bindings.

## Error Codes

Use error codes are part of the runtime result contract. Define constants in
`use` before production code writes them:

```text
invalid_use_input
no_match
missing_inputs
ambiguous
llm_selection_failed
invalid_llm_selection
artifact_path_failed
run_failed
```

Mapping:

```text
invalid_use_input      -> use request validation
no_match               -> select found no eligible candidate
ambiguous              -> local select found equally strong distinct capabilities
llm_selection_failed   -> LLM call failed
invalid_llm_selection  -> LLM selected invalid binding or invalid input patch
artifact_path_failed   -> plan could not create target path
missing_inputs         -> plan could not satisfy required non-target inputs
run_failed             -> run failed without a more specific run error
```

If `run.Result.Run.Error` is present, `use` may surface the run error code and
message directly.

## Select Subpackage

`use/select` owns semantic binding selection over the local promoted capability
catalog.

Input shape:

```go
type Request struct {
	Intent         string
	Inputs         map[string]any
	ProviderID     string
	MinVerifyLevel model.VerifyLevel
	Capabilities   []model.Capability
}
```

Output shape:

```go
type Result struct {
	Source               Source
	CapabilityID         string
	BindingID            string
	ProviderID           string
	RequiredInputs       []string
	InputsPatch          map[string]any
	Reason               string
	CandidatesConsidered int
}
```

Source constants:

```text
local
llm
```

`use/select` owns:

- Promoted binding filtering.
- Provider filtering.
- Minimum verify level filtering.
- Execution kind/spec support checks through `execute.RequiredInputs`.
- Intent token scoring.
- Candidate sorting.
- Local deterministic selection.
- Optional bounded LLM selection over topK candidates.
- LLM prompt construction and response parsing.
- LLM `inputs_patch` validation.

`use/select` does not own:

- Target generation.
- Temporary files.
- Final input planning.
- Store reads.
- Run execution.
- VerifySpec evaluation.

### Hard Filters

A binding is a candidate only when:

```text
binding.state == promoted
provider_id matches, when requested
binding.verify.level >= min_verify_level
execution kind/spec is supported by execute.RequiredInputs
intent tokens match capability id or description
```

V1 supported execution kind is CLI. Unsupported execution kinds are ignored by
selection until their adapters exist.

### Local Score

Use deterministic local scoring first:

```text
capability_id token hit in intent: +4
description token hit in intent: +2
each satisfied required input: +1
requested provider_id match: +4
```

Sort candidates:

```text
score descending
capability_id ascending
binding_id ascending
```

If local selection sees multiple equally strong distinct capabilities and no LLM
selector is available, return `ambiguous`.

### LLM TopK

`use/select` may use LLM only as a bounded selector over a local shortlist.

```text
topK = 5
```

Call LLM only when:

```text
multiple candidates are close or tied
the top candidate is missing non-target inputs
local token matching leaves semantic ambiguity
the caller explicitly enables semantic selection, if such a flag is added later
```

Do not call LLM when:

```text
one strong local candidate is selected
inputs are already satisfied
provider_id or binding constraints leave one obvious candidate
```

LLM input should include only:

```text
intent
caller input keys
topK candidate cards:
  capability_id
  capability_description
  binding_id
  provider_id
  execution_kind
  required_inputs
  execution args summary
```

LLM output:

```json
{"binding_id":"...","inputs_patch":{},"reason":"short reason"}
```

Validation rules:

```text
binding_id must be in topK
inputs_patch keys must be required by the selected binding
inputs_patch must not overwrite caller inputs
inputs_patch must not include target
inputs_patch must not invent input names
```

LLM must not generate execution args, verify specs, target paths, capabilities,
bindings, or success claims.

## Plan Subpackage

`use/plan` owns final input planning.

It uses primitive request fields to avoid a parent/child package import cycle:

```go
type Request struct {
	UseID          string
	Now            time.Time
	Inputs         map[string]any
	RequiredInputs []string
	InputsPatch    map[string]any
}

type Result struct {
	Inputs map[string]any
}
```

`use/plan` owns:

- Copying caller inputs.
- Merging selector `inputs_patch`.
- Rejecting `inputs_patch` that overwrites caller inputs.
- Creating a local target path when `target` is required and missing.
- Reporting remaining missing inputs.

`use/plan` does not own:

- Capability selection.
- LLM calls.
- Store reads.
- Run execution.
- VerifySpec evaluation.

Target generation:

```text
os.TempDir()/cal/artifacts/YYYY-MM-DD/<use-id>.out
```

Only `target` may be generated locally in V1. Missing business inputs such as
`source`, `query`, or `format` must return `missing_inputs`.

## Testing

Tests should cover:

- Use request validation.
- Nil inputs normalization.
- Default min verify level L2.
- Store list failure.
- No match.
- Ambiguous local selection.
- Local selection success.
- LLM topK selection success.
- LLM invalid binding id.
- LLM invalid inputs patch.
- Plan target generation.
- Plan missing non-target inputs.
- Plan rejects overwriting caller inputs.
- Use delegates to run with planned inputs.
- Failed run maps to failed use result.
- Successful run maps to completed use result.

Prompt snapshot tests are required once `use/select` LLM prompt text is
implemented.
