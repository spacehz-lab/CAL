# CAL Release V1 Proposal

`proposal/` owns candidate proposal for acquisition.

It turns a loaded provider, provider observations, and the existing capability
catalog into executable candidate bindings plus probe plans. It does not prove
that a candidate works.

## Goal

`proposal/` is the acquisition candidate-generation boundary:

```text
Provider + Observations + Catalog
-> SurfaceItems
-> CapabilityPlans
-> parallel per-capability Binding/Evidence pipelines
-> Candidates + ProbePlans
-> select, dedupe, reindex
-> ProposalTrace diagnostics
```

The first V1 live path keeps the approved four-stage Proposal behavior:

```text
surface -> capability -> binding -> evidence
```

Prompt wording must not change unless a separate prompt-change document approves
it.

## Boundary

`proposal/` owns:

- Proposal request and result contracts.
- Four-stage Proposal orchestration.
- Per-capability binding/evidence concurrency.
- Candidate and probe-plan assembly.
- Final result selection, deduplication, limiting, and candidate-index reindexing.
- Proposal diagnostics and attempt records.
- Live progress diagnostics for proposal substages.
- Local policy application and validation.
- Replay and deterministic-rule proposal variants when implemented.

`proposal/` does not own:

- Provider registration or loading. That belongs to `entry`.
- Provider observation. That belongs to `observe`.
- Provider execution. That belongs to `execute`.
- Fixture materialization into a probe workdir. That belongs to `probe`.
- Deterministic `VerifySpec` evaluation. That belongs to `check`.
- Probe pass/fail classification. That belongs to `probe`.
- Capability promotion. That belongs to `promote`.
- Trace persistence. That belongs to `tracelog`.
- Store reads or writes.
- Config loading.
- Logging setup.
- CLI, HTTP, CLI daemon client, daemon, or DTO rendering.

`proposal` may describe probe materials. It must not create files or rewrite
probe inputs into real workdir paths.

## Dependency Rule

```text
proposal -> model, llm, progress, proposal/surface, proposal/capability, proposal/binding, proposal/evidence, proposal/policy
proposal/surface -> model, llm, proposal/policy
proposal/capability -> model, llm, proposal/policy
proposal/binding -> model, llm
proposal/evidence -> model, llm
proposal/policy -> model
proposal/replay -> proposal, model
proposal/rules -> proposal, model
```

Forbidden:

```text
proposal -> store
proposal -> execute
proposal -> check
proposal -> probe
proposal -> promote
proposal -> tracelog
proposal -> acquisition
proposal -> entry
proposal -> observe
proposal -> config
proposal -> logging
proposal -> contract/httpserver/cli/cald
```

The main `proposal` package may import `llm` only to expose
`NewLiveRunner(client, options)`. LLM calls, prompt text, schema parsing, and
per-stage prompt material live in the four stage packages.

The main `proposal` package may import `progress` to emit proposal substage
progress. Stage packages should not emit workflow progress directly in the first
version; they return `model.ProposalStage` and `model.ProposalAttempt`, and the
main runner turns those records into live diagnostics. This keeps callback,
logging, and streaming behavior centralized in the orchestration package.

The stage packages must not import the main `proposal` package; they define
their own stage-local request and result types. A stage package may import
`proposal/policy` when it needs local filtering policy.

## Callers

```text
acquisition -> proposal
cald/app -> proposal
probe -> proposal
```

Acquisition calls `proposal.Runner.Run` after `observe`.

`probe` may consume proposal result types such as `ProbePlan`. `tracelog`
consumes only `model.ProposalTrace` through `model.Trace`; it must not import
`proposal`. They must not call LLM stage packages directly.
`promote` consumes only `model.Candidate` and `model.Probe`; it must not import
`proposal`.

## Files

```text
proposal/
  runner.go
  pipeline.go
  request.go
  result.go
  contract.go
  select.go
  diagnostics.go
  options.go
  errors.go

  surface/
    runner.go
    prompt.go
    parse.go

  capability/
    runner.go
    prompt.go
    parse.go

  binding/
    runner.go
    prompt.go
    parse.go

  evidence/
    runner.go
    prompt.go
    parse.go

  policy/
    policy.go
    file.go

  replay/
    runner.go
    parse.go

  rules/
    runner.go
    rules.go
    fixtures.go
```

`runner.go` owns the top-level proposal flow.

`pipeline.go` owns the per-capability binding/evidence pipeline and concurrent
execution.

`request.go` owns proposal request inputs.

`result.go` owns proposal result, probe plan, and fixture contracts.

`contract.go` owns the stage runner interfaces consumed by the main runner.

`select.go` owns final mechanical cleanup: select, dedupe, limit, and reindex.

`diagnostics.go` owns `model.ProposalTrace` assembly.

`progress.go` may own proposal substage progress event construction if the
event-building logic grows beyond a few runner-local calls.

`options.go` owns timeout and concurrency defaults and normalization.

`errors.go` owns proposal error codes and coded errors.

`replay/runner.go` owns replay proposal generation from a fixed proposal file.

`replay/parse.go` owns replay proposal file parsing, metadata parsing, and
candidate/probe-plan DTO conversion.

`rules/runner.go` owns deterministic proposal generation from observations.

`rules/rules.go` owns built-in deterministic rule matching.

`rules/fixtures.go` owns small static probe fixtures used by deterministic
rules.

Do not add `proposal/materialize.go`. Materialization belongs in
`probe/materialize.go`.

## Public Shape

`request.go` owns:

```go
type Request struct {
	Provider     *model.Provider
	Observations []model.Observation
	Catalog      []model.Capability
	Hint         string
	TraceID      string
}
```

`Hint` is optional natural-language acquisition intent. Live proposal stages may
use it as soft relevance guidance, but must not treat it as a hard
`capability_id` filter.

`result.go` owns:

```go
type Result struct {
	Candidates  []model.Candidate
	ProbePlans  []ProbePlan
	Diagnostics *model.ProposalTrace
}

type ProbePlan struct {
	CandidateIndex int
	Inputs         map[string]any
	Fixtures       []Fixture
	Verify         model.VerifySpec
}

type Fixture struct {
	Input    string
	Filename string
	Content  string
}
```

`Result` must preserve this invariant after final selection:

```text
len(Candidates) == len(ProbePlans)
ProbePlans[i].CandidateIndex == i
```

`contract.go` owns:

```go
type SurfaceRunner interface {
	Run(context.Context, *surface.Request) (*surface.Result, error)
}

type CapabilityRunner interface {
	Run(context.Context, *capability.Request) (*capability.Result, error)
}

type BindingRunner interface {
	Run(context.Context, *binding.Request) (*binding.Result, error)
}

type EvidenceRunner interface {
	Run(context.Context, *evidence.Request) (*evidence.Result, error)
}
```

The interfaces live in `proposal` because they describe what the proposal use
case needs from its collaborators.

## Runner

`runner.go` owns:

```go
type Runner struct {
	surface    SurfaceRunner
	capability CapabilityRunner
	binding    BindingRunner
	evidence   EvidenceRunner
	policy     policy.Policy
	options    Options
}

func NewLiveRunner(client llm.Client, options Options) *Runner
func NewWithStages(surface SurfaceRunner, capability CapabilityRunner, binding BindingRunner, evidence EvidenceRunner, options Options) *Runner

func (runner *Runner) Run(ctx context.Context, req *Request) (*Result, error)
```

`NewLiveRunner` is the normal production constructor. It hides the four-stage
wiring from callers.

`NewWithStages` is for tests and advanced injection. Ordinary callers should
not assemble `proposal/surface`, `proposal/capability`, `proposal/binding`, and
`proposal/evidence` directly.

Use constructor wiring because `Runner` owns dependencies and non-trivial
initialization.

Use pointer parameters and pointer return values because proposal requests and
results are non-trivial workflow values.

## Progress Diagnostics

`proposal.Runner` may emit live diagnostics through the shared `progress`
context fanout. These events refine the acquisition `proposal` stage without
changing the acquisition stage model.

Event shape:

```text
Scope  = acquisition
Stage  = proposal
Step   = surface | capability | binding | evidence
Status = started | succeeded | failed | canceled
```

Ownership:

- `runner.go` emits `surface` and `capability` step events.
- `pipeline.go` emits `binding` and `evidence` step events.
- Stage packages return attempts and summaries; they do not know about SSE,
  app logging, or CLI rendering.
- Replay and rules may skip substage progress because they are not live LLM
  substages.

Suggested diagnostic details:

```text
proposal_stage
model
capability_id
candidate_index
raw_response
raw_response_bytes
selected
raw
keep
skip
defer
```

Rules:

- Emit a `started` event before each live LLM stage call.
- Emit a `succeeded` event after parsing and local normalization succeeds.
- Emit a `failed` event when the LLM call, JSON parsing, or local normalization
  fails.
- `binding` and `evidence` events should include `capability_id`.
- `evidence` events should include `candidate_index` when known.
- `raw_response` means the model's final structured output text before local
  parsing. It is diagnostic output, not proof.
- Do not emit prompt text, API keys, hidden model reasoning, full user inputs,
  command stdout/stderr, or file contents.
- Do not treat raw LLM output as successful evidence. `probe` and `check`
  remain the proof boundary.

`model.ProposalTrace` remains the durable proposal diagnostic record. Progress
events are only live callbacks for logs and SSE/CLI streaming.

## Replay

`proposal/replay` owns deterministic replay of a proposal artifact. It is the
API-key-free acquisition path used by reproducible tests, experiments, and
release fixtures.

Replay input is a fixed JSON proposal file. Replay output is the normal
`*proposal.Result` consumed by `acquisition` and `probe`.

`proposal/replay` owns:

- Reading and parsing one replay proposal JSON file.
- Parsing replay metadata such as source, prompt version, model, and schema
  version.
- Converting replay candidates into `model.Candidate`.
- Converting replay probe plans into `proposal.ProbePlan`.
- Replacing or filling candidate `ProviderID` from the active acquisition
  request provider.
- Assigning stable replay candidate source and provenance.
- Validating candidate indexes and replay shape before returning a result.
- Normalizing result indexes so `ProbePlans[i].CandidateIndex == i`.
- Producing `model.ProposalTrace` diagnostics for trace visibility.

`proposal/replay` does not own:

- Provider registration or provider loading.
- Provider observation.
- LLM calls or prompt execution.
- Provider execution.
- Fixture materialization into real paths.
- Verify execution or deterministic check evaluation.
- Promotion.
- Trace persistence.
- Store reads or writes.
- CLI flag parsing or HTTP DTO parsing.

Replay must not trust provider identity from the replay file. The active
provider comes from `proposal.Request.Provider`; file-level provider fields, if
added later, are descriptive only.

Replay must preserve this output invariant:

```text
len(Candidates) == len(ProbePlans)
ProbePlans[i].CandidateIndex == i
Candidates[i].ProviderID == Request.Provider.ID
```

Replay should use stable source/provenance values:

```text
Candidate.Source = "proposal:replay"
Candidate.Provenance.Source = replay metadata source, or "replay"
Candidate.Provenance.PromptVersion = replay metadata prompt_version
Candidate.Provenance.Model = replay metadata model
Candidate.Provenance.SchemaVersion = replay metadata schema_version
Candidate.Provenance.ProposalHash = hash of the replay file content
```

Replay should expose a small constructor:

```go
func NewRunner(path string) *Runner

func (runner *Runner) Run(ctx context.Context, req *proposal.Request) (*proposal.Result, error)
```

The `path` belongs to the replay runner because replay owns file parsing. CLI,
HTTP, and contract layers only carry the path value.

Replay should reject:

- Missing or empty proposal path.
- Invalid JSON.
- Empty candidates.
- Missing matching probe plans.
- Invalid candidate indexes.
- Invalid capability ids.
- Missing candidate execution.
- Missing verify specs.
- Unsupported replay schema versions.

Replay rejection is a proposal failure. It must not create partial candidates
that later stages can promote accidentally.

## Rules

`proposal/rules` owns deterministic proposal generation from observations. It is
the no-LLM baseline acquisition path for stable local CLI cases.

Rules input is the already captured `proposal.Request.Observations`. Rules must
not run provider commands itself; provider observation belongs to `observe`.

`proposal/rules` owns:

- Reading observation text from `proposal.Request.Observations`.
- Matching deterministic rules over observation content.
- Producing `model.Candidate` values for matched capabilities.
- Producing probe inputs, fixtures, and `VerifySpec` values for matched
  candidates.
- Assigning stable rule source strings.
- Returning a normal `*proposal.Result`.
- Producing `model.ProposalTrace` diagnostics for trace visibility.

The initial rules are intentionally small:

```text
CAL_CAPABILITY / CAL_COMMAND marker -> declared capability and command
marker-free export-pdf help         -> document.convert
cupsfilter help/docs                -> document.convert
sips help                           -> image.resize
```

`proposal/rules` does not own:

- Provider registration or provider loading.
- Provider observation or help command execution.
- LLM calls.
- Provider execution.
- Probe workdir materialization.
- Verify execution or deterministic check evaluation.
- Promotion.
- Trace persistence.
- Store reads or writes.
- CLI, HTTP, or daemon mode dispatch.

Rules should expose a small constructor:

```go
func NewRunner() *Runner

func (runner *Runner) Run(ctx context.Context, req *proposal.Request) (*proposal.Result, error)
```

Rules should use stable source values. These values are part of trace/eval
semantics and must be constants in the `rules` package:

```text
rules:cli_help_marker
rules:cli_help_document_convert
rules:cli_docs_cupsfilter_pdf
rules:cli_help_sips_resize
```

Rules should generate probe plans with deterministic verification:

```text
document.convert -> L2 execute, target file format pdf
image.resize     -> L2 execute, target file format png
```

Rules may include small static fixtures needed by probes, such as a tiny text
file or tiny PNG. Rules must only describe those fixtures as
`proposal.Fixture`; it must not create files.

Rules should reject:

- Missing provider.
- No matching rule result.
- Empty `CAL_COMMAND` marker.
- Invalid capability id from a marker.
- Unsupported execution shape produced by a rule.

No-match should be a proposal failure with a clear error code. It should not be
reported as successful acquisition with zero candidates.

Do not expose a public `Rule` interface in the first implementation. Keep rule
matching as private runner-owned behavior until multiple external
implementations exist. If rule count grows, split private helpers by capability
or provider, not by acquisition stage.

## Alternate Proposal Sources

Live, replay, and rules are alternate proposal sources behind the same
acquisition boundary:

```text
acquisition -> proposer.Run(ctx, *proposal.Request) -> *proposal.Result
```

Mode dispatch belongs to `cald/app`, not to `acquisition`:

```text
mode=live   -> proposal.NewLiveRunner(...)
mode=replay -> replay.NewRunner(proposalPath)
mode=rules  -> rules.NewRunner()
```

`acquisition.Runner` must remain unaware of proposal source mode. It continues
to own only the stage order:

```text
entry -> observe -> proposer -> probe -> promote -> tracelog
```

This keeps replay and rules from becoming bypass paths. They still flow through
the same probe, promote, trace, run, use, and eval behavior as live proposal.

Shared mechanical cleanup should live in the parent `proposal` package. If
replay and rules need the same filtering, deduplication, limiting, and reindexing
as the live runner, expose a small parent-level function that performs only that
mechanical cleanup. Do not make replay or rules import live stage packages.

## Options

`options.go` owns:

```go
const (
	DefaultTimeout              = 10 * time.Minute
	DefaultPerCapabilityTimeout = 5 * time.Minute
	DefaultMaxSurfaceItems      = 40
	DefaultConcurrency          = 20
	MaxConcurrency              = 50
)

type Options struct {
	Timeout              time.Duration
	PerCapabilityTimeout time.Duration
	SurfaceLimit         int
	Concurrency          int
	CandidateLimit       int
}
```

Normalization rules:

```text
Timeout <= 0              -> DefaultTimeout
PerCapabilityTimeout <= 0 -> DefaultPerCapabilityTimeout
SurfaceLimit <= 0         -> DefaultMaxSurfaceItems
Concurrency <= 0          -> DefaultConcurrency
Concurrency > 50          -> MaxConcurrency
CandidateLimit <= 0       -> unlimited
```

Timeout semantics:

- Overall timeout bounds the full proposal run.
- Per-capability timeout bounds one binding/evidence pipeline.
- A pipeline's effective timeout is the smaller of the overall remaining time
  and `PerCapabilityTimeout`.

## Flow

Main flow:

```text
1. Validate request and runner dependencies.
2. Normalize options.
3. Create overall run context with Timeout.
4. Run surface stage serially.
5. Run capability stage serially.
6. Run one binding/evidence pipeline per capability, concurrently.
7. Merge pipeline results in original capability order.
8. Select, dedupe, limit, and reindex.
9. Build diagnostics.
10. Return Result.
```

Only this section is concurrent:

```text
for each CapabilityPlan:
  binding
  evidence
```

Do not run `surface`, `capability`, or final selection concurrently in V1.

Do not add second-level evidence concurrency inside one capability in V1.

Progress ordering under concurrency:

- `surface` and `capability` step events are serial and deterministic.
- `binding` and `evidence` step events may interleave across capability
  pipelines.
- Merged proposal results still preserve original capability order.
- Tests should not require a total event order across concurrent pipelines.

## Pipeline

`pipeline.go` owns:

```go
type capabilityPipelineResult struct {
	Index      int
	Capability capability.Plan
	Candidates []model.Candidate
	ProbePlans []ProbePlan
	Stages     []model.ProposalStage
	Attempts   []model.ProposalAttempt
	Err        error
}
```

Suggested core method:

```go
func (runner *Runner) runCapabilityPipeline(ctx context.Context, req *Request, capability capability.Plan, index int) capabilityPipelineResult
```

One pipeline does:

```text
binding.Run(capability)
for each candidate:
  evidence.Run(candidate)
assemble Candidate + ProbePlan
```

Concurrency implementation should use only standard library primitives:

```text
sem := make(chan struct{}, concurrency)
var wg sync.WaitGroup
results := make([]capabilityPipelineResult, len(capabilities))
```

Each goroutine writes only `results[index]`. Do not append to shared slices from
goroutines.

Merge in original capability order to keep output stable.

## Error Behavior

Error policy:

```text
surface failure      -> whole proposal fails
capability failure   -> whole proposal fails
one pipeline failure -> record diagnostics, continue
all pipelines fail   -> whole proposal fails
some pipelines pass  -> proposal succeeds
select failure       -> whole proposal fails
overall timeout      -> whole proposal fails
pipeline timeout     -> that pipeline fails only
```

Return partial diagnostics when possible, even when returning an error.

Do not treat LLM text as proof. LLM output creates candidates and verification
plans only. `probe` and `check` own proof.

## Select, Dedupe, Reindex

`select.go` owns the final mechanical cleanup before `probe`.

`select` filters candidates by:

- current provider id;
- candidate limit;
- presence of a matching probe plan.

`dedupe` removes duplicate candidates by:

```text
provider_id + capability_id + canonical_execution
```

`reindex` rewrites `ProbePlan.CandidateIndex` after filtering and deduplication
so the final invariant holds:

```text
ProbePlans[i].CandidateIndex == i
```

This is not semantic selection. It is deterministic result cleanup.

## Stage: Surface

`proposal/surface` owns:

```text
Observations -> SurfaceItems
```

It owns:

- surface prompt text;
- LLM request construction;
- JSON parsing;
- local normalization and filtering;
- optional provider-specific usage/invocation snippets copied from
  observations;
- `model.ProposalStage` summary for surface decisions.

It does not own:

- capability id generation;
- provider-specific execution plans;
- probe material;
- verify specs.

`Usage` is optional. It is a raw observation hint, not a trusted execution
schema. When present, it should be the closest documented invocation shape for
the surface, such as `make-pdf --in <input> --out <output>`. When absent, later
stages must fall back to the surface name, description, and original
observations.

Suggested stage-local shape:

```go
type Request struct {
	Provider     *model.Provider
	Observations []model.Observation
	Policy       policy.SurfacePolicy
}

type Result struct {
	Items   []Item
	Stage   model.ProposalStage
	Attempt model.ProposalAttempt
}
```

## Stage: Capability

`proposal/capability` owns:

```text
SurfaceItems + Catalog + Policy -> CapabilityPlans
```

It owns:

- provider-independent capability ids;
- capability descriptions;
- source surface references;
- acquisition hint narrowing when a hint is present;
- reused/created diagnostics.

It does not own:

- provider-specific command arguments;
- execution plans;
- probe inputs;
- verify specs.

Suggested stage-local shape:

```go
type Plan struct {
	CapabilityID     string
	Description      string
	SourceSurfaceIDs []string
	Confidence       string
}
```

When `Request.Hint` is present, capability planning treats it as a narrowing
constraint and returns the smallest directly relevant plan set. When no hint is
present, capability planning keeps broad discovery behavior.

## Stage: Binding

`proposal/binding` owns:

```text
CapabilityPlan + selected Source SurfaceItems + Provider + Observations
-> CandidateDrafts + ProbeMaterials
```

It owns:

- provider-specific `model.Candidate`;
- `model.Execution`;
- candidate description/source;
- probe inputs and fixture descriptions;
- candidate-level filtering;
- use of selected surface `Usage` as the strongest provider-specific
  invocation hint when available;
- use of `Request.Hint` only to choose concrete runtime-controlled values that
  are already supported by selected surfaces, such as format, algorithm, mode,
  or variant values;
- mapping of documented positional input/output operands to runtime-controlled
  placeholders such as `{{source}}` and `{{target}}`;
- stdout capture shape through `stdout_path_input` when the candidate chooses
  to write stdout to a runtime-controlled path;
- normalization of omitted or placeholder provider ids back to the active
  provider;
- rejection of candidates that do not have matching probe material;
- rejection of candidates whose `stdout_path_input` does not have a matching
  probe input;
- rejection of candidates whose `stdout_path_input` points to an input source
  such as `source`, `input`, or `stdin`.

It does not own:

- final `model.VerifySpec`;
- probe execution;
- fixture materialization.

Binding must return one `ProbeMaterial` for every kept candidate. A no-input
command still carries a material record with empty inputs. This keeps the next
Evidence stage deterministic: every candidate has an explicit probe context, and
missing probe material is reported as a binding failure rather than silently
promoting a candidate with unknown inputs.

Binding receives only the surfaces referenced by
`CapabilityPlan.SourceSurfaceIDs`. This keeps Stage 3 from reselecting across
unrelated surfaces and makes the Stage 2 -> Stage 3 relationship explicit:
Capability owns the semantic plan and source references; Binding materializes
those selected provider-specific surfaces into executable candidates.

Binding is not a filter stage. If selected surfaces contain a name, usage, mode,
option, command, or observed invocation detail, Binding should return at least
one best-effort candidate. It should not reject a selected surface because the
capability id or description is broader, generic, or named differently than the
concrete provider invocation.

For a non-empty selected surface payload, Binding must not return an empty
candidate array. Empty `candidates` and `probe_material` are reserved for an
empty selected surface payload.

Within one proposal result, CAL keeps a single default winner for each
`provider_id + capability_id`. Binding may return multiple raw candidates, but
the proposal selection step keeps the first valid candidate for a capability and
drops later variants. Future support for multiple promoted variants should be
modeled explicitly with variant metadata instead of promoting anonymous
alternatives.

Observations are supporting reference material for selected surfaces only; they
should not be used to discover unrelated surfaces, replace selected surfaces, or
re-evaluate the capability. They may still be used to recover invocation details
for the selected surface, including operand meaning, default behavior, and
stdout/file output behavior.

`Request.Hint` is supporting narrowing context for runtime values only. For
example, if selected surfaces already expose a format or algorithm operand,
Binding may use the hinted value as probe material instead of choosing an
arbitrary default. The hint must not be used to invent unsupported arguments,
replace selected surfaces, or re-evaluate the capability.

Binding does not reject a selected surface because `CapabilityPlan.Description`
is broader than the concrete provider invocation. The candidate keeps the plan's
capability id and may describe the narrower observed execution. Empty candidates
are reserved for empty selected surfaces. Binding must not return empty
candidates merely because the invocation uses positional operands, optional
operands, defaults, stdout output, or a narrower execution than the capability
description.

Suggested stage-local shape:

```go
type ProbeMaterial struct {
	CandidateIndex int
	Inputs         map[string]any
	Fixtures       []Fixture
}

type Result struct {
	Candidates []model.Candidate
	Materials  []ProbeMaterial
	Stage      model.ProposalStage
	Attempt    model.ProposalAttempt
}
```

## Stage: Evidence

`proposal/evidence` owns:

```text
Candidate + ProbeMaterial + Observations -> VerifySpec
```

It owns:

- verify-spec prompt text;
- JSON parsing;
- `model.VerifySpec` normalization;
- verification level derivation;
- validation that referenced probe inputs exist;
- predicate parameter rules shared by prompt payload and local parsing.

It does not own:

- executing verification;
- calling `check`;
- judging pass/fail.

`proposal/evidence` should not import `check`. `cald/app` can pass deterministic
rule summaries into evidence prompt material when needed.

Evidence injects `verify_predicate_rules` into the prompt and uses the same
rules to filter LLM checks locally. Prompt text should describe the generic rule
contract, not enumerate every predicate's params. Adding a new evidence
predicate should update the evidence rule table first.

Evidence derives `VerifySpec.Level` locally after filtering checks. Contract
verification is L1. Execute verification with only process checks is L1,
artifact-shape checks such as `non_empty`, `format`, or `regex` are L2, and
semantic checks such as `contains`, `contains_any`, `bytes_equal_transform`,
`hash_line_matches`, `archive_contains_input`, or `json_query_matches` are L3.

Evidence checks for structured formats must avoid brittle serialization details.
For JSON key/value content, Evidence should prefer a regex with optional
whitespace or `contains_any` with compact and spaced forms instead of a single
whitespace-sensitive `contains` string.

## Policy

`proposal/policy` owns local proposal policy:

- allowed surface kinds;
- skipped names and patterns;
- preferred capability subjects;
- preferred capability operations;
- proposal limits when needed.

The first implementation may support only `DefaultPolicy` and validation.
`file.go` can be deferred until a real caller needs policy-file parsing.

`policy` must not import `llm` or `proposal` main package.

## Replay And Rules

`proposal/replay` owns replay JSON to `proposal.Result`.

`proposal/rules` owns deterministic baseline proposal.

Both are proposal sources, not four-stage LLM stages. They must not import
`llm`, `probe`, `promote`, `tracelog`, `acquisition`, or adapters.

Replay and rules can be implemented after the live LLM path if the goal is the
fastest first closed loop.

## Tests

Add direct unit tests for:

- option normalization;
- missing runner dependency validation;
- invalid request validation;
- surface failure fails whole proposal;
- capability failure fails whole proposal;
- one pipeline failure with another successful pipeline returns success;
- all pipeline failures return error with diagnostics;
- overall timeout fails the whole proposal;
- per-capability timeout fails only that pipeline;
- final select/dedupe/reindex preserves invariants;
- result ordering is stable despite concurrent pipeline completion order;
- proposal emits surface and capability step progress on success and failure;
- proposal emits binding and evidence step progress without requiring stable
  cross-pipeline ordering;
- proposal progress does not include prompt text or hidden model reasoning.

Add stage package tests for prompt-free parsing and normalization behavior.
Snapshot prompt text separately so prompt changes are explicit.
