# CAL Discover Control

Discover Control is the control-plane module for provider registration and targeted
capability acquisition.

This document defines the target command semantics. The current implementation
may keep older direct `calctl` execution slices while the HTTP control service is
being connected endpoint by endpoint.

It exposes discovery state and commands through `calctl`, while `cald` owns
service state, provider locks, Trace writing, probes, verification, and
promotion.

## Role

Discover Control answers three questions:

```text
Is the discovery service available?
Which explicit CLI or app provider entries are registered?
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
provider registration
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

## Command Groups

Discover Control has three groups:

```text
status
provider registration
provider acquisition
```

Automatic or periodic discovery is a later service extension. It should not be
part of the first command contract.

Discovery status is not a separate CLI command in the first HTTP-aligned control
surface. Use `calctl daemon status --json` for service liveness and
`calctl providers list --json` for registered provider scope.

## Provider Registration

Provider registration records one explicit provider entry path. It does not run
`LLM proposer`, probes, verification, or promotion.

Commands:

```bash
calctl providers add --provider-path <provider-path> --json
calctl providers get --provider-path <provider-path> --json
```

Output shape:

```json
{
  "id": "provider_abc123",
  "kind": "cli",
  "name": "jq",
  "path": "/usr/bin/jq"
}
```

`providers get --provider-path` only checks stored Provider records; it does not
scan or register the path.

## Targeted Acquisition

Targeted acquisition observes one provider, proposes candidates, executes safe
probes, verifies outcomes, and promotes only passing Capability and Binding
records.

Commands:

```bash
calctl discovery run --provider-id <provider-id> [--capability-id <capability-id>] --json
calctl discovery run --provider-id <provider-id> --proposal-path <proposal.json> --json
```

`--mode rules` exists as a hidden experimental baseline and regression-test
switch. It is not part of the public discovery surface, and its implementation
belongs under `internal/baseline/rules`.

Meanings:

```text
calctl discovery run --provider-id <provider-id>
  Run targeted Discovery Proposal, Verification, and Promotion for one known provider.

calctl discovery run --provider-id <provider-id> --proposal-path <proposal.json>
  Replay candidate and probe-plan output from an SOP or `LLM proposer` fixture for one stored provider. CAL still executes the candidate and runs deterministic verification before promotion.
```

Provider path bootstrap belongs to `calctl providers add`; targeted acquisition
requires a stored `provider_id`.

`--capability-id` is an optional debug filter. Without it, the proposer may propose
any capability from the provider observations.

Current v1 acquisition is intentionally narrow: it supports CLI providers
observed through bounded help and usage commands, with local man output used
only as a fallback when those command observations do not return usable text;
live Proposal or replayed SOP/LLM-style proposal JSON for candidate, probe, and
verify-spec input; capability-specific safe probes; stdout-to-target CLI
execution when required; deterministic `verify.checks` evaluation with script
fallback only when checks cannot express the outcome. Deterministic rules for the fake CLI pattern, macOS
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

If no candidate passes Verification and Promotion, promotion counts should be
zero.

## Side Effects

Allowed side effects:

```text
providers add --provider-path
  writes Provider records only

discovery run
  writes Trace records and possibly promoted Capability or Binding records
```

Forbidden side effects:

```text
daemon status must not start cald
providers add must not run LLM proposal, probes, verification, or promotion
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
    "message": "provider_id is required"
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
provider registration commands
targeted acquisition commands
discovery status output
stable provider and job-result JSON
```

Discover Control does not own:

```text
capability catalog rendering
runtime binding resolution
capability execution
outcome verification policy
`LLM proposer` prompts for Proposal
Trace schema internals
agent task planning
automatic or periodic discovery in the first command contract
```

## Agent Policy

A skill or agent policy can use Discover Control like this:

```text
Use `calctl daemon status --json` to inspect service availability.
Use `calctl providers add --provider-path <provider-path> --json` when the task needs to register one explicit CLI provider entry.
Use `calctl discovery run --provider-id <provider-id>` when the task needs acquisition for one stored provider.
Do not trigger discovery as a hidden fallback from capabilities list or runs create.
After discovery promotes new capabilities, call `calctl capabilities list --json` again before selecting a capability.
```
