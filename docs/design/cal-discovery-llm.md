# CAL Discovery LLM Integration

This document defines the production LLM boundary for Discovery Proposal.

Proposal stage contracts are defined in:

```text
docs/design/cal-discovery-proposal.md
docs/design/cal-proposal-surface.md
docs/design/cal-proposal-capability.md
docs/design/cal-proposal-binding.md
docs/design/cal-proposal-evidence.md
```

Detailed prompt wording lives in code and tests. Design docs define stage
contracts, not full prompt text.

## Role

The LLM is a proposer, not a verification judge.

Allowed:

```text
Provider observations
selected existing Capability ids
optional debug capability filter
-> Proposal Surface / Capability / Binding / Evidence material
```

Not allowed:

```text
decide verification success
promote a Binding
write durable Provider / Capability / Binding records
store or log API credentials
```

The Proposal output is process material. CAL must still parse it, execute the
candidate through a safe probe, evaluate deterministic `verify.checks`, write
Trace evidence, and promote only passing candidates that satisfy promotion
policy.

## SDK And Providers

CAL uses the official OpenAI Go SDK:

```text
github.com/openai/openai-go/v3
```

Provider transport lives in `internal/llm` behind the `llm.Client` interface.
The live proposal adapter lives in `internal/proposalflow`; it owns the
four-stage Proposal orchestration and implements `proposalflow.Proposer` by
calling `llm.Client` and parsing strict JSON stage outputs. `discovery` and
`cli` do not import provider SDKs.

The Chat Completions adapter calls an OpenAI-compatible Chat Completions API
with:

```text
base_url        = CAL_LLM_BASE_URL when set
system message  = bounded CAL stage prompt
user message    = serialized stage context
model           = CAL_LLM_MODEL
response_format = json_object
```

The provider response text must be exactly one stage JSON object. Markdown
fences, prose, empty output, speculative capabilities, and invalid schema fail
before Verification.

Live provider response JSON does not own proposal provenance. The CAL adapter
sets source, prompt version, schema version, and model from local call context
before writing Trace.

## Stage Call Shape

The live LLM Proposal implementation may use four internal stages:

```text
Surface
-> Capability
-> Binding
-> Evidence
```

Surface and Capability are global for one provider acquisition. After
Capability planning, Binding and Evidence may run as bounded per-capability
pipelines.

Target call count depends on provider complexity:

```text
simple deterministic replay/parser: 0 LLM calls
simple live provider: 2-3 LLM calls
complex live provider: 2 global calls + bounded per-capability calls
```

Each stage must have its own compact schema and token budget. Complex CLI
providers may require larger max token budgets for Surface and Capability
planning because reasoning tokens can dominate even when final JSON is small.

## Configuration

Runtime secrets stay outside the repository, outside Trace artifacts, and
outside `CAL_HOME/config.json`.

The v0 product surface should let users configure non-secret LLM settings, such
as API type, model, and optional base URL, through CAL configuration or
equivalent `calctl` commands. API keys and other secrets should still come from
explicit environment variables or a later dedicated local secret mechanism.
`CAL_HOME/config.json` may store an `api_key_ref`, but it must be a reference to
a secret source, not the raw secret value.

CAL must not auto-read vendor-specific key names and must not infer a provider
from whichever key is present.

`CAL_HOME/config.json` example:

```json
{
  "llm": {
    "api": "chat_completions",
    "base_url": "https://api.moonshot.cn/v1",
    "model": "kimi-k2.7-code",
    "api_key_ref": "env:CAL_LLM_API_KEY"
  }
}
```

Supported API values:

```text
responses
chat_completions
```

Required when LLM proposal generation is enabled:

```text
an API key supplied by CAL_LLM_API_KEY or by llm.api_key_ref
```

LLM API type and model are required, but they may come from either durable
non-secret CAL configuration or explicit `CAL_LLM_*` environment variables.
`CAL_LLM_BASE_URL` is optional for providers that use the SDK default base URL
and required by deployment convention for non-default OpenAI-compatible
providers.

First-version `api_key_ref` support is intentionally narrow:

```text
env:<ENV_NAME>
```

Explicit `CAL_LLM_*` environment variables override durable non-secret config
values for the current process. `CAL_LLM_API_KEY` also overrides `api_key_ref`.

When no LLM configuration is set, the default LLM proposer has no client and
targeted acquisition fails during proposal generation with
`candidate_proposal_failed`. Partial or unsupported explicit LLM configuration
fails before scan execution with `invalid_llm_config`. This keeps local tests,
proposal replay, and rules baseline runs offline by default.

Do not persist API keys or raw secret values in:

```text
repo files
Trace artifacts
CAL_HOME/config.json
proposal fixtures
logs
```

## Streaming Diagnostics

`CAL_LLM_STREAM=1` may enable streaming for OpenAI-compatible Chat Completions
providers. Streaming does not change any Proposal stage contract; CAL still
parses only the final assistant content for each stage.

A separate local diagnostic switch may capture streamed reasoning/content
deltas for latency and prompt debugging. Stream diagnostics are not CAL
evidence and may contain provider observations or local paths.

## Modes

Production default:

```text
calctl discovery run --provider-id <provider-id>
```

These commands use the LLM proposer when valid LLM configuration and required
secrets are present.

Offline replay:

```text
calctl discovery run --provider-id <provider-id> --proposal-path <proposal.json>
```

This replays a proposal fixture through the same Verification and Promotion
path.

Rules baseline:

```text
calctl discovery run --provider-id <provider-id> --mode rules
```

`--mode` is hidden and kept for experiments and regression tests. Rules-only
proposal generation is not the production first attempt. Its implementation
lives under `internal/baseline/rules`, not under the production `proposal` or
`discovery` packages.

## Trace And Evaluation

Trace may record proposal provenance:

```text
source
prompt_version
model
schema_version
proposal_hash
proposal stage timings
```

These fields explain where a proposal came from. They are not proof of
correctness. The proof boundary remains deterministic Verification evidence.
For live LLM calls, CAL fills provenance locally; the model must not self-report
it.

Evaluation should count:

```text
LLM proposer calls by Proposal stage
Proposal stage parse failures
Verification pass / fail / ambiguous outcomes
verify level distribution
script fallback count
Promotion outcomes
future low-level action reduction after reuse
```

## Testing

Unit tests use local fake clients or local HTTP servers. They must not require a
real provider API key.

Live LLM tests should be explicit and environment-gated. They should write no
secrets to Trace, logs, or fixtures.
