# CAL Discovery LLM Integration

This document defines the production LLM boundary for Discovery.

The detailed prompt contract is defined in
`docs/design/cal-discovery-llm-prompt.md`.

## Role

The LLM is a proposer, not a verifier.

Allowed:

```text
Provider observations
selected existing Capability ids
optional debug capability hint
-> one proposal JSON object with one or more strongly supported candidates
```

Not allowed:

```text
decide verification success
promote a Binding
write durable Provider / Capability / Binding records
store or log API credentials
```

The proposal JSON is process material. CAL must still parse it, execute the
candidate through a safe probe, run a deterministic verifier, write Trace
evidence, and promote only passing candidates.

## SDK And Providers

CAL uses the official OpenAI Go SDK:

```text
github.com/openai/openai-go/v3
```

Provider transport lives in `internal/llm` behind the `llm.Client` interface.
The proposal adapter lives in `internal/proposal/llm`; it owns the CAL proposal
prompt and implements `proposal.Proposer` plus `proposal.ProbePlanner` by
calling `llm.Client` and parsing the returned proposal JSON. `discovery` and
`cli` do not import provider SDKs.

The Responses adapter calls the OpenAI Responses API with:

```text
instructions = bounded CAL system prompt
input        = serialized provider observations and capability ids
model        = CAL_LLM_MODEL
store        = false
```

The Chat Completions adapter calls an OpenAI-compatible Chat Completions API
with:

```text
base_url        = CAL_LLM_BASE_URL when set
system message  = bounded CAL system prompt
user message    = serialized provider observations and capability ids
model           = CAL_LLM_MODEL
response_format = json_object
```

The provider response text must be exactly one proposal JSON object. It may
contain multiple candidates when the observations strongly support multiple
capabilities, and every candidate must have a matching probe plan by
`candidate_index`. Markdown fences, prose, empty output, speculative
capabilities, and invalid schema fail before Verification.
Live provider response JSON does not own proposal provenance. The CAL adapter
sets source, prompt version, schema version, and model from local call context
before writing Trace.

The user prompt includes bounded helper inputs:

```text
existing_capability_ids
hint
```

`existing_capability_ids` is a bounded topK local lookup result for reuse. It
does not include bindings, probe history, vectors, or execution details.

The prompt must preserve this one-way chain:

```text
provider observations
-> candidate operations
-> capability ids
-> binding contracts
-> verifier requirements
-> generated harness packages
-> probe plans
```

The model includes `verifier_packages[]` when the inferred candidate outcome can
be checked with deterministic local evidence. The generated harness is still
executed by CAL locally; the model must not claim verification success.

## Configuration

Runtime secrets stay outside the repository, outside Trace artifacts, and outside
`CAL_HOME/config.json`.

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
  "provider_sources": [
    {"kind": "path", "value": "PATH"},
    {"kind": "path", "value": "/Applications"}
  ],
  "llm": {
    "api": "chat_completions",
    "base_url": "https://api.moonshot.cn/v1",
    "model": "kimi-k2.7-code",
    "api_key_ref": "env:CAL_LLM_API_KEY"
  }
}
```

OpenAI Responses environment-only example:

```text
CAL_LLM_API=responses
CAL_LLM_MODEL=gpt-5.5
CAL_LLM_API_KEY=...
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

For example, `env:CAL_LLM_API_KEY` reads the key from the explicit
`CAL_LLM_API_KEY` environment variable. Future productized builds may add local
secret-store references such as Keychain, Secret Service, or Credential Manager
without changing the durable config schema.

Explicit `CAL_LLM_*` environment variables override durable non-secret config
values for the current process. `CAL_LLM_API_KEY` also overrides `api_key_ref`.

When no LLM configuration is set, the default LLM proposer has no client and
targeted acquisition fails during proposal generation with `candidate_proposal_failed`.
Partial or unsupported explicit LLM configuration fails before scan execution
with `invalid_llm_config`. This keeps local tests, proposal replay, and rules
baseline runs offline by default.

Do not persist API keys or raw secret values in:

```text
repo files
Trace artifacts
CAL_HOME/config.json
proposal fixtures
logs
```

## Modes

Production default:

```text
calctl discovery run --provider-path <provider-path>
calctl discovery run --provider-id <provider-id>
```

These commands use the LLM proposer when valid LLM configuration and required
secrets are present.

Offline replay:

```text
calctl discovery run --provider-path <provider-path> --proposal-path <proposal.json>
```

This replays a proposal fixture through the same Verification and Promotion
path.

Rules baseline:

```text
calctl discovery run --provider-path <provider-path> --mode rules
```

`--mode` is hidden and kept for experiments and regression tests. Rules-only
proposal generation is not the production first attempt. Its implementation lives under
`internal/baseline/rules`, not under the production `proposal` or `discovery`
packages.

## Trace And Evaluation

Trace may record proposal provenance:

```text
source
prompt_version
model
schema_version
proposal_hash
```

These fields explain where a proposal came from. They are not proof of correctness.
The proof boundary remains deterministic Verification evidence.
For live LLM calls, CAL fills these fields locally; the model must not self-report
them. Replay proposal files may still carry metadata because replay provenance is
part of the fixture being reproduced.

Evaluation should count:

```text
LLM proposer calls
proposal parse failures
verification pass / fail / ambiguous outcomes
promotion outcomes
future low-level action reduction after reuse
```

## Testing

Unit tests use local fake clients or local HTTP servers. They must not require a
real provider API key.

Live LLM tests should be explicit and environment-gated. They should write no
secrets to Trace, logs, or fixtures.
