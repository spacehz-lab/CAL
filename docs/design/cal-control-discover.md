# CAL Discover Control

Discover Control is the control-plane module for provider finding and targeted
capability acquisition.

This document defines the target command semantics. The current implementation
may keep older direct `calctl` execution slices while the HTTP control service is
being connected endpoint by endpoint.

It exposes discovery state and commands through `calctl`, while `cald` owns
service state, provider locks, Trace writing, probes, verification, and
promotion.

## Role

Discover Control answers four questions:

```text
Is the discovery service available?
What provider sources are configured?
Which CLI or app provider entries can CAL find?
What discovery job or Trace was created?
```

It is not the online capability catalog. Agents use Capability List for reusable
capabilities and Discover Control only when acquisition or provider management
is needed.

## calctl And cald Boundary

`calctl` is the command client.

`cald` is the local service.

Discovery operations that require service state must go through `cald`:

```text
provider finding
manual discovery jobs
provider observation
safe probe execution
deterministic verification
promotion
Trace writing
provider locks
```

The `calctl` to `cald` transport is the local-only HTTP control API defined in
`docs/design/cal-control-service.md`.

Provider source reads and writes go through `cald` so the service view and the
visible discovery state stay consistent.

## Command Groups

Discover Control has four groups:

```text
status
path
find
provider acquisition
```

Automatic or periodic discovery is a later service extension. It should not be
part of the first command contract.

Discovery status is not a separate CLI command in the first HTTP-aligned control surface. Use `calctl daemon status --json` for service liveness and `calctl providers sources list --json` for configured provider source scope.

## Provider Sources

Provider sources define where Provider Entry lookup searches for provider
entries. The first source kind is `path`, used for filesystem roots and `PATH`.
They are provider sources, not automatic-discovery triggers.

Commands:

```bash
calctl providers sources list --json
calctl providers sources add --kind path --value <path> --json
calctl providers sources remove --kind path --value <path> --json
```

Path values may include:

```text
PATH
/Applications
/System/Applications
$HOME/Applications
```

`PATH` is a symbolic value meaning command lookup path. Other values are
filesystem paths that may use `$HOME` or environment variables.

`path add` should validate path syntax but should not require every path to
exist at add time. A missing path can be reported during provider finding
without making configuration impossible across machines.

Output shape:

```json
{
  "sources": [
    {"kind": "path", "value": "PATH"},
    {"kind": "path", "value": "/Applications"}
  ]
}
```

Path changes affect future provider finding and targeted discovery. They must
not immediately run provider finding or capability acquisition unless the user
explicitly requests it.

## Provider Finding

Provider finding locates provider entries and writes Provider records. It does
not run `LLM proposer`, probes, verification, or promotion.

Commands:

```bash
calctl providers find --kind cli --json
calctl providers find --kind app --json
```

Optional root arguments may be added after the base shape is implemented:

```bash
calctl providers find --kind cli --root PATH --json
calctl providers find --kind app --root /Applications --json
```

Output shape:

```json
{
  "kind": "cli",
  "paths": ["PATH"],
  "providers_created": 1,
  "providers_updated": 2,
  "providers": [
    {
      "id": "provider_abc123",
      "kind": "cli",
      "name": "jq",
      "path": "/usr/bin/jq"
    }
  ]
}
```

No-provider is not an infrastructure failure. A provider-finding run may return
zero providers with a successful status.

## Targeted Acquisition

Targeted acquisition observes one provider, proposes candidates, executes safe
probes, verifies outcomes, and promotes only passing Capability and Binding
records.

Commands:

```bash
calctl discovery run --provider-path <provider-path> [--capability-id <capability-id>] --json
calctl discovery run --provider-id <provider-id> [--capability-id <capability-id>] --json
calctl discovery run --provider-path <provider-path> --proposal-path <proposal.json> --json
```

`--mode rules` exists as a hidden experimental baseline and regression-test
switch. It is not part of the public discovery surface, and its implementation
belongs under `internal/baseline/rules`.

Meanings:

```text
calctl discovery run --provider-path <provider-path>
  Inspect one explicit provider entry path, create or update the Provider record, then run targeted Discovery Inference, Verification, and Promotion for that provider.

calctl discovery run --provider-path <provider-path> --proposal-path <proposal.json>
  Replay candidate and probe-plan output from an SOP or `LLM proposer` fixture for one provider. CAL still executes the candidate and runs deterministic verification before promotion.

calctl discovery run --provider-id <provider-id>
  Run targeted Discovery Inference, Verification, and Promotion for one known provider.
```

`--provider-path` must point to a provider entry, not an arbitrary input file or a
directory to scan. In v0 this usually means a CLI executable path. Unsupported
provider kinds return a structured unsupported-provider error.

`--capability-id` is an optional debug filter. Without it, the proposer may propose
any capability from the provider observations.

Current v1 acquisition is intentionally narrow: it supports CLI providers
observed through bounded help and usage commands, with local man output used
only as a fallback when those command observations do not return usable text;
live `LLM proposer` or replayed SOP/LLM-style proposal JSON for candidate,
probe-plan, and generated verifier-harness input; capability-specific safe
probes; stdout-to-target CLI execution when required; deterministic script
verifier execution using local trusted or generated scripts under
`CAL_HOME/verifiers/`. Production runtime does not load embedded default
verifier scripts. Deterministic rules for the fake CLI pattern, macOS
`cupsfilter`, and macOS `sips` are retained only behind the hidden
baseline/regression mode. Broader app, AX, menu, API, and marker-free `soffice`
help parsing are later extensions.

Targeted acquisition returns a job result.

For an already known provider target:

```json
{
  "job_id": "disc_123",
  "state": "succeeded",
  "target": {
    "type": "provider",
    "provider_id": "provider_soffice"
  },
  "providers_created": 0,
  "providers_updated": 1,
  "capabilities_promoted": 1,
  "bindings_promoted": 1,
  "trace_id": "trace_123"
}
```

For an explicit provider entry path target:

```json
{
  "job_id": "disc_124",
  "state": "succeeded",
  "target": {
    "type": "provider_path",
    "value": "/usr/sbin/cupsfilter",
    "provider_id": "provider_abc123"
  },
  "providers_created": 1,
  "providers_updated": 0,
  "capabilities_promoted": 1,
  "bindings_promoted": 1,
  "trace_id": "trace_124"
}
```

If no candidate passes Verification and Promotion, promotion counts should be
zero.

## Side Effects

Allowed side effects:

```text
providers sources add/remove
  updates path-type provider source configuration

providers find --kind cli|app
  writes Provider records only

discovery run
  writes Provider records, Trace records, and possibly promoted Capability or Binding records
```

Forbidden side effects:

```text
daemon status must not start cald
providers sources list must not run provider finding
providers sources add/remove must not immediately run provider finding or acquisition
providers find must not run LLM proposal, probes, verification, or promotion
capabilities list must not trigger discovery
runs create must not trigger discovery automatically
```

## Error Semantics

If `cald` is required but unavailable, return a structured error:

```json
{
  "error": {
    "code": "cald_unavailable",
    "message": "cald is not running"
  }
}
```

If a target is invalid:

```json
{
  "error": {
    "code": "invalid_discovery_target",
    "message": "supply either a provider id or one explicit provider entry path"
  }
}
```

If an explicit provider entry path exists but cannot be recognized as a provider
entry, return a structured error because the user requested one concrete target:

```json
{
  "error": {
    "code": "target_provider_not_found",
    "message": "no provider found for provider path"
  }
}
```

Unsupported app acquisition should return a structured unsupported-provider
error until app acquisition is implemented.

## Boundary

Discover Control owns:

```text
discovery path configuration commands
provider finding commands
targeted acquisition commands
discovery status output
stable provider-find and job-result JSON
```

Discover Control does not own:

```text
capability catalog rendering
runtime binding resolution
capability execution
outcome verification policy
`LLM proposer` prompts for Inference
Trace schema internals
agent task planning
automatic or periodic discovery in the first command contract
```

## Agent Policy

A skill or agent policy can use Discover Control like this:

```text
Use `calctl daemon status --json` to inspect service availability.
Use `calctl providers find --kind cli --json` when the task needs to locate CLI providers.
Use `calctl discovery run --provider-path <provider-path>` when the task needs acquisition for one explicit CLI provider entry.
Use `calctl providers sources add/remove` only when the user wants CAL to change future provider source scope.
Do not trigger discovery as a hidden fallback from capabilities list or runs create.
After discovery promotes new capabilities, call `calctl capabilities list --json` again before selecting a capability.
```
