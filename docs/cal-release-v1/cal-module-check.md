# CAL Release V1 Check

`check/` owns deterministic `VerifySpec` validation and evaluation.

It is shared by acquisition probe and promoted run verification. It is not a
workflow package.

## Goal

`check/` turns a stable `model.VerifySpec` plus execution outputs into local,
deterministic evidence:

```text
VerifySpec + inputs + stdout + stderr + exit_code
-> validate contract
-> resolve subjects
-> evaluate predicates
-> evidence refs + checked outputs
```

## Boundary

`check/` owns:

- `VerifySpec` semantic validation.
- Subject and predicate rule registration.
- Prompt-facing subject rule export.
- Subject resolution from runtime inputs and execution outputs.
- Built-in predicate implementations.
- Evidence item construction for passed checks.

`check/` does not own:

- Durable `VerifySpec` structs or enum strings. Those stay in `model`.
- Candidate probing, fixture creation, workdir cleanup, or probe
  classification. Those stay in `probe`.
- Command execution or input rendering. Those stay in `execute`.
- Binding selection or run persistence. Those stay in `run`.
- Prompt wording. Proposal may receive check rules, but `check` does not build
  prompts.
- API, HTTP, CLI, daemon, store, config, or LLM behavior.

## Dependency Rule

```text
check -> model
```

`check` may use only the Go standard library plus `internal/model`.

Callers:

```text
probe -> check
run   -> check
```

`proposal/evidence` should not import `check` directly. `cald/app` should build
the checker once and pass `checker.Rules()` into proposal as prompt material.

## Public Shape

Use one dependency-owning object:

```go
type Checker struct {
	// private subject rules and predicate handlers
}

func NewChecker() *Checker

func (c *Checker) Rules() []model.VerifySubjectRule
func (c *Checker) Validate(spec *model.VerifySpec) error
func (c *Checker) Run(ctx context.Context, req *Request) (*Result, error)
```

Package-level helpers are optional thin defaults:

```go
func Rules() []model.VerifySubjectRule
func Validate(spec *model.VerifySpec) error
func Run(ctx context.Context, req *Request) (*Result, error)
```

Production wiring should prefer an explicit `*Checker` from `cald/app`.

## Runtime Types

```go
type Request struct {
	Spec     *model.VerifySpec
	Inputs   map[string]any
	Stdout   string
	Stderr   string
	ExitCode int
}

type Result struct {
	Evidence []model.EvidenceRef
	Outputs  map[string]any
}
```

`Request` and `Result` are runtime-only and belong in `check`, not `model`.
They should be passed by pointer because they are non-trivial execution
contexts/results.

## Registry Model

`Checker` is the rule registry and evaluator. Do not split public
`Registry` and `Evaluator` objects in V1.

The checker is the single source of truth for:

- supported subject types;
- allowed predicates per subject;
- required predicate params;
- allowed param values;
- predicate handlers.

Each predicate file should register its own contract and implementation:

```text
predicate_file.go  -> exists, non_empty, format
predicate_text.go  -> equals, not_equals, contains, contains_any, regex
predicate_bytes.go -> bytes_equal_transform
predicate_hash.go  -> hash_line_matches
predicate_archive.go -> archive_contains_input
predicate_json.go -> json_query_matches, json_equivalent, json_field_equals,
                     json_field_matches_source
```

That keeps adding a predicate local to one file: params, allowed subjects, and
execution live together.

Suggested internal shape:

```go
type predicate struct {
	name     model.VerifyPredicate
	subjects []model.VerifySubjectType
	params   []paramRule
	run      predicateRunner
}

type paramRule struct {
	name          string
	required      bool
	allowedValues []string
}

type predicateRunner func(*predicateContext) error
```

## Supported Contract

V1 must preserve the existing `VerifySpec` JSON contract.

Subjects:

```text
file
stdout
stderr
exit_code
```

Predicates:

```text
equals
not_equals
exists
non_empty
format
contains
contains_any
regex
bytes_equal_transform
hash_line_matches
archive_contains_input
json_query_matches
json_equivalent
json_field_equals
json_field_matches_source
text_transform_matches
line_count_matches
text_filter_matches
delimited_column_matches
```

Allowed subject/predicate combinations:

```text
file:
  exists
  non_empty
  format
  contains
  contains_any
  regex
  bytes_equal_transform
  hash_line_matches
  archive_contains_input
  json_query_matches
  json_equivalent
  json_field_equals
  json_field_matches_source
  text_transform_matches
  line_count_matches
  text_filter_matches
  delimited_column_matches

stdout:
  equals
  not_equals
  non_empty
  contains
  contains_any
  regex
  hash_line_matches
  json_query_matches
  json_equivalent
  json_field_equals
  json_field_matches_source
  line_count_matches
  text_filter_matches
  delimited_column_matches

stderr:
  equals
  not_equals
  non_empty
  contains
  contains_any
  regex
  hash_line_matches
  json_query_matches
  json_equivalent
  json_field_equals
  json_field_matches_source
  line_count_matches
  text_filter_matches
  delimited_column_matches

exit_code:
  equals
  not_equals
```

Param rules:

```text
equals:
  value required

not_equals:
  value required

contains:
  value required

contains_any:
  values required

regex:
  pattern required and must compile

format:
  format required
  allowed values: pdf, png, json, text

bytes_equal_transform:
  source required
  transform required
  allowed values: base64_encode, base64_decode

hash_line_matches:
  source required
  algorithm required
  allowed values: sha1, sha256, sha-1, sha-256, sha_1, sha_256, sha 1, sha 256

archive_contains_input:
  source required
  format required
  allowed values: zip, tar

json_query_matches:
  source required
  query required

json_equivalent:
  source required

json_field_equals:
  query required
  value required

json_field_matches_source:
  query required
  source required
  property required
  allowed property values: basename, bytes, sha256

text_transform_matches:
  source required
  transform required
  allowed values: uppercase, lowercase

line_count_matches:
  source required

text_filter_matches:
  source required
  pattern required

delimited_column_matches:
  source required
  delimiter required
  column required
```

## Run Semantics

`Checker.Run` should reject:

- canceled context;
- nil request or nil spec;
- invalid `VerifySpec`;
- `verify.method != execute`;
- `verify.level == L0`.

For each check:

1. Resolve the subject.
2. Run the predicate.
3. Append one `model.EvidenceRef` when the predicate passes.
4. Merge checked outputs into `Result.Outputs`.

The first failed check stops evaluation and returns an error.

Evidence ID shape must remain:

```text
check_<1-based-index>_<subject-label>_<predicate>
```

Evidence content must preserve:

```json
{
  "subject": {},
  "predicate": ""
}
```

## Subject Resolution

`file` subjects use `VerifySubject.Input` as a key into `Request.Inputs`:

```text
subject.input = "target"
request.inputs["target"] = "/tmp/out.pdf"
```

`stdout`, `stderr`, and `exit_code` subjects read directly from `Request`.

Checked outputs:

```text
file target -> outputs["target"] = resolved path
stdout      -> outputs["stdout"] = stdout string
stderr      -> outputs["stderr"] = stderr string
exit_code   -> outputs["exit_code"] = exit code int
```

## Files

```text
check/
  checker.go
  request.go
  subject.go
  predicate.go
  predicate_file.go
  predicate_text.go
  predicate_bytes.go
  predicate_hash.go
  params.go
```

`checker.go` owns `Checker`, `NewChecker`, `Rules`, `Validate`, and `Run`.

`request.go` owns `Request` and `Result`.

`subject.go` owns subject resolution and subject text reading.

`predicate.go` owns the internal predicate registration type and dispatcher.

`predicate_*.go` files own concrete predicate contracts and evaluators.

`params.go` owns small param coercion helpers and should stay at the bottom of
the package dependency chain.

## Model Interaction

End-state ownership:

```text
model:
  VerifySpec data structs
  VerifyLevel, VerifyMethod, VerifyPredicate, VerifySubjectType enum strings
  VerifySubjectRule and VerifyParamRule DTOs
  VerifyLevelRank
  durable record validation that does not need predicate rules

check:
  VerifySpec semantic validation
  Rules prompt contract
  predicate params and allowed values
  deterministic evaluation
```

When `check` lands, remove duplicated semantic rule ownership from `model`.
`model` may keep structural record validation, but `check.Checker` should be the
only owner of subject/predicate/param semantics.

## Caller Responsibilities

`probe` should:

- execute candidate probes through `execute`;
- call `check.Checker.Run`;
- classify success or failure;
- write probe trace material.

`run` should:

- resolve a promoted binding;
- execute the binding through `execute`;
- call `check.Checker.Run` only when verification is requested;
- write the run record.

`proposal/evidence` should:

- receive `[]model.VerifySubjectRule` as prompt material;
- not evaluate checks;
- not claim pass/fail.

## Tests

Required unit tests:

- every registered predicate has one passing and one failing test;
- `Rules()` includes exactly the supported subject/predicate pairs;
- `Validate` rejects unsupported subject/predicate/param combinations;
- `Validate` rejects invalid regex patterns;
- `Run` rejects contract method and L0 execute specs;
- `Run` returns one evidence ref per passed check;
- `Run` returns checked outputs for file, stdout, stderr, and exit_code;
- file format checks cover pdf, png, json, text, and unsupported values;
- bytes transform covers base64 encode and decode;
- hash matching normalizes sha1 and sha256 spellings;
- context cancellation stops evaluation before work.

Compatibility tests should compare `check.Checker.Rules()` with the old contract
until V1 replaces the old implementation.

## Decision

Use `Checker`, not separate public `Registry` and `Evaluator` types.

Rationale:

- one object matches the current ownership boundary;
- rules, validation, and predicate execution stay together;
- adding a predicate touches one local area;
- the public API remains small enough for `probe`, `run`, and `cald/app`.
