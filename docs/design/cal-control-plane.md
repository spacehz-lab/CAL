# CAL Control Plane

The CAL control plane is the machine-facing interface used by agents and humans to inspect reusable capabilities, control discovery, request capability execution, and observe local service state.

It is not the Discovery loop itself and it is not an agent planner.

## Design Boundary

The control plane connects callers to CAL's service-owned model:

```text
agent / human / skill
-> calctl
-> CAL core or cald
-> Provider / Capability / Binding / Trace / Run
```

The control plane should make the online path simple:

```text
intent
-> use promoted capability
-> run selected binding
-> receive verified result and evidence
```

Discovery may run before, after, or in parallel with this path, but listing,
using, and running must not depend on open-ended discovery happening in the same
request.

## Entrypoints

CAL has two local entrypoints:

```text
calctl
  Machine-facing CLI for agents and humans.

cald
  Local CAL service for discovery state, provider observation, probes, verification, promotion, a local HTTP control API, future automatic discovery, and later shared runtime coordination.
```

`calctl` should stay a thin adapter. It manages the `cald` process shape, discovers the local endpoint, calls the local HTTP control API, and renders command output. It should not call discovery, runtime, eval, store, or promotion workflows directly.

The data model must stay the same either way.

## calctl And cald Relationship

`calctl` is the control client.

It owns:

```text
argument parsing
machine-readable output
human-readable rendering
daemon process management
HTTP calls to cald for service-owned work
```

It does not own:

```text
long-running discovery state
future automatic discovery loops
provider locks
safe probe scheduling
promotion decisions
background job history
```

`cald` is the local CAL service.

It owns:

```text
future automatic discovery worker state
manual discovery jobs that need observation, probes, or promotion
Trace writing
provider locks and concurrency
safe probe execution
deterministic verification
Promotion into Capability and Binding records
```

`cald` should be local-only in the first version.

The target relationship is:

```text
calctl command
-> local HTTP control API to cald for service-owned work
-> cald-owned CAL_HOME records
```

The local HTTP service listens only on loopback and writes endpoint material
under `CAL_HOME/cald/`. It is also the future host for the local WebUI so the
WebUI and `calctl` reuse the same `/v1` control API. It must not become a
remote HTTP service in v1.

`cald` may be unavailable. Commands that only read durable state should still work when possible. Commands that require service-owned state or long-running discovery jobs should return a structured service-unavailable error when `cald` is not running.

Current implementation slice: `calctl` routes provider registration, discovery
acquisition, capabilities listing, intent-level use, runtime runs, eval
summaries, traces, and service status through `cald`.
Future automatic discovery workers are not required for the current closed loop.

## Logging

`calctl` and `cald` write structured diagnostic logs to stderr. Logs must not
pollute machine-readable stdout output such as `--json`.

Durable logging configuration lives in `CAL_HOME/config.json`:

```json
{
  "logging": {
    "level": "info",
    "file": {
      "enabled": true,
      "max_bytes": 10485760,
      "max_files": 5
    }
  }
}
```

`CAL_LOG_LEVEL` is retained as a temporary process-level override for the log
level only:

```text
CAL_LOG_LEVEL=debug|info|warn|error
```

The effective level is:

```text
CAL_LOG_LEVEL
-> config.logging.level
-> info
```

File logging is enabled by default and writes rolling process logs to the
platform log/state directory, not to `CAL_HOME`:

```text
darwin:  ~/Library/Logs/cal/
linux:   ${XDG_STATE_HOME:-~/.local/state}/cal/logs/
windows: %LocalAppData%\cal\logs\
```

Process log files are named `calctl.log` and `cald.log`, with numeric rotated
files such as `calctl.log.1`. The default rotation policy is 10 MB per file and
5 retained rotated files.

Discovery debug logs may include provider ids, capability ids, selector counts,
verification levels, verify check counts, trace id, and structured error codes.
Logs must not include API keys, full prompts, full LLM responses, full
observation text, file contents, vectors, or user input payloads.

The current diagnostic event surface is intentionally limited to major workflow
boundaries:

- `discovery entry ...` records provider entry scan start, completion, and
  failure counts.
- `discovery acquisition ...` records provider load, observation, proposal,
  verification, promotion, completion, and structured failure stages.
- `proposal llm ...` records live proposal start, completion, and failure
  stages with safe counts such as observation, candidate, probe-plan, and
  verify-check counts.
- `proposal materializer ...` records replay/live proposal materialization with
  safe counts and proposal hashes, not raw proposal bodies.
- `observe cli documentation ...` records local CLI documentation source
  attempts, selected source, byte counts, duration, and timeout/empty-output
  classifications.
- `runtime binding ...`, `runtime execution ...`, `runtime verification ...`,
  and `runtime run ...` record binding resolution, provider execution,
  verify-check evaluation or script fallback execution, and top-level run
  completion or failure.
- `eval compute ...` records aggregate experiment record counts.

## Primary Agent Path

The primary v1 user or agent path has one semantic request:

```bash
calctl use <intent> --json
```

Flow:

```text
agent receives task
-> agent sends intent and optional inputs to CAL Use
-> CAL selects a promoted capability and binding
-> CAL completes binding-specific control inputs when possible
-> CAL executes the binding
-> CAL verifies the outcome when requested
-> CAL writes a Run record and evidence when available
-> agent receives structured result
```

The lower-level deterministic path remains available:

```bash
calctl capabilities list --json
calctl runs create --capability-id <capability-id> --inputs-json <json> --json
```

This keeps semantic intent routing in Use, capability catalog exposure in List,
and deterministic execution in Run.

## Command Surface

### Service

```bash
calctl daemon start
calctl daemon stop
calctl daemon status --json
```

Purpose:

```text
start, stop, or inspect the local CAL service
```

Current implementation slice: `daemon status` is safe to call even when `cald`
is not running. `daemon start` launches `cald serve`, waits for the local HTTP
endpoint to become ready, and returns service status. `daemon stop` calls the
running service through HTTP.

`daemon stop` stops `cald` and all service-owned work. It is different from future automatic discovery controls.

### Discovery

```bash
calctl providers add --provider-path <provider-path> --json
calctl providers get --provider-path <provider-path> --json
calctl discovery run --provider-id <provider-id> --json
```

Purpose:

```text
register CLI or app provider entries without running acquisition
run targeted Discovery Proposal, Verification, and Promotion for selected providers
promote verified Capability and Binding records through cald
```

Discovery writes process detail to Trace. It is not part of the normal online run path.

Detailed design:

```text
docs/design/cal-control-service.md
docs/design/cal-control-discover.md
```

### Query

```bash
calctl capabilities list --json
calctl capabilities list --provider-id <provider-id> --json
calctl capabilities list --capability-id <capability-id> --json
```

Purpose:

```text
expose the current promoted capability catalog to an agent or human
```

Detailed design:

```text
docs/design/cal-control-capability-list.md
```

The query surface is deterministic. It does not include natural-language intent
interpretation, semantic search, embeddings, or LLM calls. Semantic routing is
owned by Capability Use.

### Use

```bash
calctl use <intent> --json
calctl use <intent> --verify --json
calctl use --intent <intent> --inputs-json <json> --json
calctl use --intent <intent> --inputs-file <json-file> --json
calctl use <intent> --provider-id <provider-id> --json
```

Purpose:

```text
route a user or agent intent to one promoted capability binding
complete binding-specific control inputs when possible
execute through Run
record the selected capability, binding, and run result
```

Detailed design:

```text
docs/design/cal-control-use.md
```

`use` must not generate new capabilities, generate verify checks or script
fallbacks, trigger discovery, or claim execution success from LLM output. It
selects from promoted records and delegates execution to Run.

### Execution

```bash
calctl runs create --capability-id <capability-id> --inputs-json <json> --json
calctl runs create --capability-id <capability-id> --inputs-file <json-file> --json
calctl runs create --capability-id <capability-id> --binding-id <binding-id> --inputs-json <json> --json
calctl runs create --capability-id <capability-id> --provider-id <provider-id> --inputs-json <json> --json
calctl runs create --capability-id <capability-id> --inputs-json <json> --verify --json
```

Purpose:

```text
execute a promoted capability through an automatically selected binding
optionally verify the outcome
record Run evidence when verification is requested
```

Detailed design:

```text
docs/design/cal-control-capability-run.md
```

`run` must perform binding resolution internally when only `capability_id` is
supplied. `binding_id` is an optional execution constraint for Use, debugging,
and reproducible tests; it is not a public binding-management API.

### Evaluation

```bash
calctl eval --json
```

Purpose:

```text
summarize discovery and reuse outcomes for experiments
```

Evaluation is not required for the first online path, but Use and Run records
should preserve enough structure to support it later.

## Command Lifecycle

Online reuse:

```text
use
-> run
```

Deterministic reuse primitive:

```text
capabilities list
-> runs create
```

Background or targeted acquisition:

```text
discover
-> automatic worker or manual scan
-> Trace
-> Promotion
-> capabilities list sees new promoted records
```

Service inspection:

```text
daemon status
```

Evaluation:

```text
uses + runs + traces
-> eval
```

## JSON Contract Rules

All agent-facing commands must support `--json`.

JSON output should be:

```text
stable
deterministically ordered
schema-oriented
safe to parse without reading prose
explicit about success, failure, and empty results
```

Human-readable output may exist, but it is secondary.

Agent-facing commands must avoid hidden side effects:

```text
list must not trigger discovery
status must not start cald
run must not promote candidates
use must not trigger discovery
eval must not rewrite uses, traces, or runs
```

## Skill And MCP Integration

The first integration path can be skill plus CLI:

```text
CAL skill
-> tells agent when to call calctl
-> agent runs calctl use for intent-level reuse
-> fallback to capabilities list + runs create only when it already knows the exact capability_id
```

The first integration path can still use `calctl`, while `cald` provides the
local HTTP protocol behind it when service-owned operations are implemented.

A later MCP integration can expose the same control plane:

```text
cal.daemon_start
cal.daemon_status
cal.daemon_stop
cal.providers_sources_list
cal.providers_sources_add
cal.providers_sources_remove
cal.providers_find
cal.providers_list
cal.providers_get
cal.discovery_run
cal.capabilities_list
cal.capabilities_get
cal.use
cal.run
cal.runs_get
cal.eval
cal.traces_get
```

MCP should be a transport over the same command semantics, not a second data model.

## Non-Goals

The control plane should not introduce:

```text
an agent planner inside CAL
multi-step planning or composition in v1
embedding-backed semantic search in v1
public binding subcommands in v1
public resolve round trips in the normal path
remote HTTP control in v1
cross-device WebUI access in v1
automatic discovery as a side effect of list or run
Trace or candidate records as executable agent-facing APIs
raw clicks, coordinates, or transient UI nodes as public capability inputs
```

## References

Subcommand designs:

```text
docs/design/cal-control-capability-list.md
docs/design/cal-control-capability-run.md
docs/design/cal-control-use.md
docs/design/cal-control-discover.md
docs/design/cal-control-eval.md
```

Discovery designs:

```text
docs/design/cal-discovery-flow.md
docs/design/cal-discovery-promotion.md
```
