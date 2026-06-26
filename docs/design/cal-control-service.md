# CAL Control Service

This document defines how `calctl`, `cald`, and a future local WebUI interact.

It is the service/protocol contract for the control plane. Command semantics
remain in the control documents such as `cal-control-discover.md`,
`cal-control-capability-list.md`, `cal-control-use.md`,
`cal-control-capability-run.md`, and `cal-control-eval.md`.

## Decision

CAL uses a local-only HTTP control API.

```text
calctl
  -> daemon process manager
  -> typed HTTP client
  -> cald /v1/*

WebUI
  -> same-origin browser client
  -> cald /v1/*

cald
  -> local HTTP server on loopback
  -> HTTP adapter
  -> CAL control service
  -> config, store, discovery, runtime, eval
```

The HTTP API exists so `calctl`, the future WebUI, and later adapters such as
MCP can share one local protocol. It is not a remote service interface.

The implementation keeps transport and control logic separate:

```text
internal/cald
  -> process lifecycle and loopback listener

internal/cald/httpapi
  -> HTTP routing, JSON decoding, response status codes

internal/cald/control
  -> provider sources, provider finding, discovery, runs, eval, trace reads

internal/use
  -> intent-level Use request contract and local promoted-binding selection
```

Future MCP support should add an MCP adapter that calls `internal/cald/control`.
It should not reuse HTTP handlers.

## Boundary

`cald` owns service state and service-side execution:

```text
daemon status
provider finding
provider source configuration
targeted discovery requests
Trace writing
verification and promotion
capability catalog reads
semantic capability use orchestration
runtime runs when routed through the service
eval summaries
```

Intent-to-binding matching is Use-domain behavior owned by `internal/use`.
`cald` should orchestrate that resolver with store reads and `Run`; it should
not accumulate scoring, tokenization, or binding-ranking logic in the daemon
package.

`calctl` owns command-line adaptation:

```text
argument parsing
daemon process start/stop
endpoint discovery
HTTP request construction
JSON and human-readable rendering
structured error rendering
```

`calctl` should not duplicate service workflows. Discovery, runtime execution,
eval reads, provider persistence, capability reads, and trace reads are
service-owned work reached through this local HTTP API.

## Local HTTP Transport

`cald serve` listens only on loopback:

```text
127.0.0.1:<random-port>
```

The first version must not listen on `0.0.0.0`, a LAN address, or a public
interface.

On startup, `cald` writes service connection material under `CAL_HOME/cald/`:

```text
endpoint.json
```

When `CAL_HOME` is not set, CAL uses the platform application data location:

```text
darwin:  ~/Library/Application Support/cal/
linux:   ${XDG_DATA_HOME:-~/.local/share}/cal/
windows: %LocalAppData%\cal\
```

`endpoint.json`:

```json
{
  "pid": 12345,
  "base_url": "http://127.0.0.1:54321",
  "started_at": "2026-06-24T10:00:00Z"
}
```

The first service slice does not add a local bearer token. The boundary is the
loopback-only listener. A token or UI session model can be added later when
`calctl` and WebUI routing move fully through the daemon.

## WebUI Access

The first WebUI should be served by `cald` from the same origin as the API:

```text
/ui/*
/v1/*
```

This avoids cross-origin browser access in the default product path.

`calctl ui` can be added later as a convenience command:

```text
calctl ui
-> ensure cald is running
-> request a short-lived UI launch session
-> open http://127.0.0.1:<port>/ui?launch_token=...
-> cald exchanges it for an HttpOnly SameSite cookie
```

The first WebUI session model is future work.

## Security Rules

The local HTTP API must follow these rules:

```text
listen on loopback only
use random ports by default
serve WebUI same-origin with session cookies
do not enable CORS by default
reject unsafe cross-origin mutation requests
do not log API keys, prompts, raw LLM responses, or user input payloads
do not support remote access in v0
do not introduce multi-user auth in v0
```

This keeps the system local-first while still allowing a browser UI.

## API Shape

Successful responses return the command payload directly. Errors use one stable
shape:

```json
{
  "error": {
    "code": "cald_unavailable",
    "message": "cald is not running"
  }
}
```

HTTP status guidance:

```text
400 invalid request shape
404 unknown endpoint or record
409 conflicting service state
422 semantic validation failure
500 internal error
503 unavailable dependency or service
```

## Initial API Surface

The initial local API should stay small:

```text
GET    /v1/daemon/status
POST   /v1/daemon/stop

GET    /v1/providers/sources
POST   /v1/providers/sources/add
POST   /v1/providers/sources/remove
POST   /v1/providers/find
GET    /v1/providers
GET    /v1/providers/{provider_id}

POST   /v1/discovery

GET    /v1/capabilities
GET    /v1/capabilities/{capability_id}

POST   /v1/uses

POST   /v1/runs
GET    /v1/runs/{run_id}

GET    /v1/eval
GET    /v1/traces/{trace_id}
```

`POST /v1/discovery` is synchronous in the first slice. It returns a full
`JobResult` and does not persist the job result as a separate record.

Request body must identify exactly one provider target:

```json
{"provider_id": "provider_abc123"}
```

or:

```json
{"provider_path": "/usr/bin/jq"}
```

`provider_path` must point to one provider entry, such as a CLI executable or
app bundle. Directory scanning belongs to provider finding, not discovery
acquisition.

`calctl daemon start` is not an HTTP call because it starts the process. It
launches `cald serve`, waits for `GET /v1/daemon/status` to report ready, then
returns the service status. If a live endpoint already exists, `daemon start`
returns that running status instead of launching a second service.

`POST /v1/uses` is synchronous in the first slice:

```text
POST /v1/uses
-> select one promoted capability binding for the intent
-> complete binding-compatible inputs
-> call Run
-> return UseResult with selection and Run result
```

It must not trigger Discovery or create new Capability, Binding, or Verifier
records. A later `GET /v1/uses/{use_id}` endpoint can be added once Use records
are persisted; until then, the synchronous response is the source of truth for
the first slice.

## Command Mapping

```text
calctl daemon status
  -> GET /v1/daemon/status

calctl daemon stop
  -> POST /v1/daemon/stop

calctl providers sources list
  -> GET /v1/providers/sources

calctl providers sources add --kind path --value <path>
  -> POST /v1/providers/sources/add

calctl providers sources remove --kind path --value <path>
  -> POST /v1/providers/sources/remove

calctl providers find --kind cli|app
  -> POST /v1/providers/find

calctl providers list
  -> GET /v1/providers

calctl providers get --provider-id <provider-id>
  -> GET /v1/providers/{provider_id}

calctl discovery run --provider-path <provider-path>
  -> POST /v1/discovery

calctl discovery run --provider-id <provider-id>
  -> POST /v1/discovery

calctl capabilities list
  -> GET /v1/capabilities

calctl capabilities get --capability-id <capability-id>
  -> GET /v1/capabilities/{capability_id}

calctl use <intent>
  -> POST /v1/uses

calctl runs create --capability-id <capability-id>
  -> POST /v1/runs

calctl runs get --run-id <run-id>
  -> GET /v1/runs/{run_id}

calctl eval
  -> GET /v1/eval

calctl traces get --trace-id <trace-id>
  -> GET /v1/traces/{trace_id}
```

The service API should use the same JSON contracts that `calctl --json` renders.
`calctl` is a transport adapter, not a different data model.

## Sync And Job Model

The first implementation may keep discovery calls synchronous:

```text
POST /v1/discovery
-> run one provider acquisition
-> return JobResult
```

If WebUI responsiveness or long-running live LLM discovery requires it, add a
job API later:

```text
POST /v1/jobs
GET  /v1/jobs/<id>
GET  /v1/jobs/<id>/events
```

Do not add the job protocol before the synchronous control path is working.

## Non-Goals

The local control service must not introduce:

```text
remote HTTP access
multi-user authentication
cloud sync
plugin hosting
remote workflow execution
public package distribution
a second data model for WebUI
an agent planner inside cald
```
