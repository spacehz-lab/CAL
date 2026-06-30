# CAL Capability Use

Capability Use is the semantic entry point for later reuse of promoted
capabilities.

It closes the online path that sits above `run`:

```text
user or agent intent
-> select promoted Capability and Binding
-> complete binding-compatible inputs
-> execute through Run
```

`use` does not acquire new capabilities. It only routes an intent to records
that Discovery has already promoted.

## Role

Capability Use answers one question:

```text
Can CAL satisfy this user intent with an already promoted capability binding?
```

It is the user-facing semantic layer above Capability List and Capability Run.

The intended online path is:

```text
user or agent task
-> calctl use <intent> --json
-> CAL selects a promoted capability and binding
-> CAL completes provider-specific binding inputs when possible
-> CAL calls Run
-> caller receives selection, run result, and optional evidence
```

`list` exposes reusable capability records. `use` chooses among them for one
intent. `run` executes one already selected capability or binding.

## Input

Default command shape:

```bash
calctl use <intent> --json
```

Verified command shape for experiments or explicit outcome checks:

```bash
calctl use <intent> --verify --json
```

Structured inputs may be supplied when the caller already knows them:

```bash
calctl use --intent <intent> --inputs-json <json> --json
```

File-backed JSON may be used when shell quoting is awkward:

```bash
calctl use --intent <intent> --inputs-file <json-file> --json
```

Optional execution constraints:

```bash
calctl use <intent> --provider-id <provider-id> --json
calctl use <intent> --strategy default --json
```

The first HTTP shape is:

```json
{
  "intent": "compute the SHA-1 digest of this file",
  "inputs": {
    "source": "/tmp/a.txt",
    "target": "/tmp/a.sha1"
  },
  "provider_id": "provider_shasum",
  "strategy": "default",
  "verify": true
}
```

`intent` is required. `inputs` is optional; omitted inputs are treated as an
empty object. `provider_id`, `strategy`, and `verify` have the same meaning as
in Run, but they constrain semantic selection before execution.

## Capability And Binding Selection

Use selects from promoted records only.

The resolver receives a compact use catalog built from durable records:

```text
Capability.id
Capability.description
Binding.id
Binding.provider_id
Provider.name
Binding.execution.kind
required runtime inputs
Binding.verify.level
compact verification summary
```

It must not read Trace candidates, failed probes, raw observations, script
fallback source code, or provider documentation dumps as executable knowledge.

The resolver does not send the full durable catalog to an LLM. It first builds
a local high-recall shortlist over promoted bindings. If runtime LLM settings
are configured and multiple candidates remain or a candidate is missing inputs,
it sends only topK binding cards plus necessary execution and verification
summaries to an LLM.
Without LLM settings, it uses the local selector and local input planning. This
mirrors the progressive-disclosure shape used by skill systems: expose metadata
first, then expand details only for the likely relevant entries.

The v0 selection strategy is:

```text
promoted capabilities and bindings
-> local shortlist with high-recall topK scoring
-> optional LLM selection over shortlisted binding cards
-> local validation
-> Run
```

The local shortlist should prefer recall over precision and should keep
plausible bindings when the local score is uncertain. The LLM selector may only
choose among shortlisted promoted bindings. CAL still validates selected ids and
inputs before execution.

The initial local scoring can remain simple:

```text
intent overlap with capability id and description
user input keys covering required runtime inputs
optional provider_id match
binding execution support
verify request versus available verification level
verification strength, with L3 preferred over L2 and L1 excluded by default
```

The default shortlist size can be conservative, such as 20 bindings, and may be
skipped when the catalog is already small.

Selection and input planning have two jobs:

1. choose a promoted capability and binding that match the intent;
2. assign binding-specific inputs that are explicit in the intent or can be
   generated safely by CAL.

For example, a user may say:

```text
compute a SHA-1 digest
```

The user input may only contain:

```json
{"source": "a.txt", "target": "a.sha1"}
```

If a promoted binding requires an algorithm selector, Use may assign:

```json
{"algorithm": "1"}
```

for a `shasum` binding, or:

```json
{"algorithm": "sha1"}
```

for an `openssl` binding, but only when those values are explicit in the
selected binding execution or documented binding contract.

Use must not invent arbitrary business inputs. If the task requires a missing
business input, such as a source file, query string, or output format that
cannot be inferred from the user intent and inputs, return `missing_inputs`.

The only input CAL may generate locally in the first slice is `target`. If the
selected binding requires `target` through `{{target}}` or
`stdout_path_input:"target"` and the caller did not provide it, Use creates a
temporary artifact path:

```text
os.TempDir()/cal/artifacts/YYYY-MM-DD/<use-id>.out
```

This path is temporary. Callers that want a durable output location should pass
`target` explicitly through `--inputs-json` or `--inputs-file`.

## Resolver Boundary

The first semantic resolver may use an LLM over a compact catalog.

The resolver may:

```text
compare intent to capability ids and descriptions
compare user inputs to required binding inputs
select one promoted binding
extract input values explicitly present in the intent
return a reason for the selection
```

The resolver must not:

```text
create a Capability
create a Binding
create a verify spec
rewrite Binding.execution
trigger Discovery
call provider commands directly
read or write Trace
claim that execution or verification succeeded
choose from unpromoted candidates
```

This keeps LLM use in the online path limited to semantic routing. Execution and
verification remain deterministic CAL runtime work.

## Use LLM Prompt Contract

The Use LLM system prompt must be separate from the acquisition prompt.

Acquisition prompt:

```text
provider observations -> Proposal Surface -> Capability -> Binding -> Evidence
```

Use prompt:

```text
intent + input keys + promoted binding summaries -> selected binding + inputs_patch
```

The Use prompt should be short and restrictive:

```text
You select one promoted binding for a user intent.
Choose only from the provided capabilities and bindings.
Only add inputs_patch values that are explicitly present in the intent.
Do not include target in inputs_patch; CAL creates missing target paths locally.
Do not create capability_id, binding_id, verify checks, execution, or proof.
Do not invent or overwrite user inputs.
Do not modify execution.
Do not claim execution success.
Return strict JSON only.
```

Expected resolver output:

```json
{
  "binding_id": "binding_abc123",
  "inputs_patch": {
    "source": "/tmp/a.txt",
    "algorithm": "1"
  },
  "reason": "The intent asks for SHA-1 and this binding supports SHA algorithm selector 1."
}
```

The service must validate the resolver output before execution.

## Local Validation

After resolver output, CAL must validate:

```text
capability_id exists
binding_id belongs to capability_id
binding state is promoted
provider_id constraint matches when supplied
final inputs satisfy runtime required inputs
selected execution kind is supported
```

If validation fails, Use returns a failed `UseResult` and must not execute the
provider command.

## Run Delegation

Use delegates execution to Run.

The final Run request is equivalent to:

```json
{
  "capability_id": "file.checksum",
  "binding_id": "binding_abc123",
  "inputs": {
    "source": "/tmp/a.txt",
    "target": "/tmp/a.sha1",
    "algorithm": "1"
  },
  "verify": true
}
```

Run remains the deterministic primitive for:

```text
binding resolution when only capability_id is supplied
input validation
provider execution
optional verify.checks evaluation
Run record persistence
```

Use may require Run to accept an optional `binding_id` so a semantic selection
can execute the exact binding it selected. `binding_id` is an execution
constraint, not a public binding management surface.

## Output

Successful output shape:

```json
{
  "id": "use_123",
  "intent": "compute the SHA-1 digest of this file",
  "selection": {
    "source": "llm",
    "capability_id": "file.checksum",
    "binding_id": "binding_abc123",
    "provider_id": "provider_shasum",
    "reason": "The intent asks for SHA-1 and the binding supports algorithm selector 1.",
    "candidates_considered": 4
  },
  "run": {
    "id": "run_456",
    "capability_id": "file.checksum",
    "binding_id": "binding_abc123",
    "provider_id": "provider_shasum",
    "status": "succeeded",
    "verified": true
  },
  "status": "succeeded",
  "duration_ms": 245
}
```

Failure output shape:

```json
{
  "id": "use_124",
  "intent": "encode this file as base64",
  "status": "failed",
  "error": {
    "code": "no_match",
    "message": "no promoted binding matched the intent"
  }
}
```

Required output fields:

```text
id
intent
status
```

Required when selected:

```text
selection.capability_id
selection.binding_id
selection.provider_id
selection.source
```

Required when execution starts:

```text
run
```

## Use Record

Use should write a durable Use record once the service supports it.

The record should capture safe operational facts:

```text
use id
intent summary or full intent according to privacy settings
selected capability id
selected binding id
selected provider id
assigned input keys and scalar control values
run id
status
timing
structured error code and message
```

Use records must not store API keys, raw LLM prompts, raw LLM responses, file
contents, or large user payloads.

For the first implementation, returning the Use result and storing the Run
record is sufficient if durable Use records would slow the slice down.

## Failure Model

Use failure codes:

```text
invalid_use_input
no_match
ambiguous
missing_inputs
llm_selection_failed
invalid_llm_selection
artifact_path_failed
invalid_run_input
execution_failed
verification_failed
```

Resolver failures and run failures must be distinguishable. A resolver failure
means CAL could not select a promoted binding. A run failure means a binding was
selected but execution or optional verification failed.

## Security And Privacy

Use should follow the control-plane logging boundary:

```text
do not log API keys
do not log raw LLM prompts or raw LLM responses
do not log file contents
do not log full user input payloads by default
log safe counts, ids, status, error codes, and timing
```

The compact catalog sent to an LLM should avoid local absolute paths unless a
provider id or provider name is needed for selection. User file paths in inputs
may be summarized by key and basename when possible.

## Evaluation Role

Use changes the benchmark interpretation.

Held-out reuse should exercise:

```text
task intent + held-out inputs
-> Use selection
-> Run
-> benchmark oracle
```

The benchmark should report separate metrics:

```text
capability_id_quality
intent_selection_success
use_success
oracle_reuse_success
use_llm_calls
use_latency
```

This separates naming quality from actual user-facing reuse. A generated
capability id may differ from the benchmark's preferred id while still being
usable when Use can route the intent to the promoted binding and the oracle
passes.

## Upgrade Path

The v0 strategy is intentionally small:

```text
local scoring over promoted bindings
-> optional LLM selection over topK binding cards
-> local validation
-> Run
```

Future versions may replace the shortlist stage without changing Run semantics:

```text
embedding-backed retrieval for large catalogs
two-step detail fetch where the resolver first requests binding details
cost-, latency-, and reliability-aware binding ranking
provider availability and recent run history in the local score
multi-binding fallback after execution failure
```

These upgrades should preserve the same trust boundary: retrieval and LLM
selection are advisory, while CAL-owned validation and Run execution remain
deterministic.

## Non-Goals

Use must not introduce:

```text
automatic discovery fallback
multi-step planning or composition
embedding-backed search in the first slice
provider command execution outside Run
new capability or binding generation
verify-check generation
raw Trace or candidate browsing as a public API
success claims from LLM output
```

## Relationship To List And Run

```text
Capability List
  deterministic catalog exposure

Capability Use
  semantic intent selection over promoted bindings

Capability Run
  deterministic execution of one selected capability or binding
```

Use is the default user-facing path. List and Run remain important primitives
for agents, debugging, reproducible tests, and callers that already know the
exact `capability_id`.
